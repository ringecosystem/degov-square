package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/internal"
	"github.com/ringecosystem/degov-apps/internal/config"
	"github.com/ringecosystem/degov-apps/internal/database"
	"github.com/ringecosystem/degov-apps/internal/utils"
	"github.com/ringecosystem/degov-apps/models"
	"github.com/ringecosystem/degov-apps/types"
)

// GitHubTag represents a GitHub tag response
type GitHubTag struct {
	Name string `json:"name"`
}

type GithubConfigLink struct {
	BaseLink   string
	ConfigLink string
}

type DaoSyncTask struct {
	db *gorm.DB
}

type DaoRegistryConfigResult struct {
	RemoteLink GithubConfigLink
	Result     DaoRegistryConfig
}

// DaoRegistryConfig represents the structure of the config.yml file
type DaoRegistryConfig map[string][]struct {
	Code   string `yaml:"code"`
	Config string `yaml:"config"`
}

// NewDaoSyncTask creates a new DAO sync task
func NewDaoSyncTask() *DaoSyncTask {
	return &DaoSyncTask{
		db: database.GetDB(),
	}
}

// Name returns the task name
func (t *DaoSyncTask) Name() string {
	return "dao-sync"
}

// Execute performs the DAO synchronization
func (t *DaoSyncTask) Execute() error {
	return t.SyncDaos()
}

// SyncDaos fetches the latest DAO configuration and syncs it with the database
func (t *DaoSyncTask) SyncDaos() error {
	startTime := time.Now()
	slog.Info("Starting DAO synchronization", "timestamp", startTime.Format(time.RFC3339))

	// Fetch the registry config
	registryConfigResult, err := t.fetchRegistryConfig()
	if err != nil {
		return fmt.Errorf("failed to fetch registry config: %w", err)
	}

	slog.Info("Successfully fetched registry config", "chains", len(registryConfigResult.Result))

	// Track active DAO codes for marking inactive ones
	activeDaoCodes := make(map[string]bool)

	// Process each chain and its DAOs
	for chainName, daos := range registryConfigResult.Result {
		for _, daoInfo := range daos {
			// Fetch DAO config details first to get the actual DAO information
			daoConfig, err := t.fetchDaoConfig(registryConfigResult.RemoteLink, daoInfo.Config, daoInfo.Code)
			if err != nil {
				slog.Error("Failed to fetch DAO config", "config_url", daoInfo.Config, "error", err)
				continue
			}

			// Skip if essential fields are missing
			if daoInfo.Code == "" || daoConfig.Name == "" {
				slog.Warn("DAO config missing essential fields", "config_url", daoInfo.Config)
				continue
			}

			activeDaoCodes[daoInfo.Code] = true

			// Upsert DAO in database
			dao := &models.Dao{
				ID:         internal.NextIDString(),
				ChainID:    daoConfig.Chain.ID,
				ChainName:  daoConfig.Chain.Name,
				Name:       daoConfig.Name,
				Code:       daoInfo.Code,
				Seq:        0,
				State:      "ACTIVE",
				ConfigLink: daoInfo.Config,
				TimeSyncd:  utils.TimePtrNow(),
			}

			if err := t.upsertDao(dao); err != nil {
				slog.Error("Failed to upsert DAO", "dao", daoInfo.Code, "error", err)
				continue
			}

			slog.Debug("Successfully synced DAO", "dao", daoInfo.Code, "chain", chainName)
		}
	}

	// Mark DAOs as inactive if they're not in the current config
	if err := t.markInactiveDAOs(activeDaoCodes); err != nil {
		return fmt.Errorf("failed to mark inactive DAOs: %w", err)
	}

	duration := time.Since(startTime)
	slog.Info("DAO synchronization completed",
		"active_daos", len(activeDaoCodes),
		"duration", duration.String(),
		"timestamp", time.Now().Format(time.RFC3339))
	return nil
}

// fetchRegistryConfig fetches and parses the main registry configuration
func (t *DaoSyncTask) fetchRegistryConfig() (DaoRegistryConfigResult, error) {
	// Get config mode and refs from config
	mode := config.GetString("REGISTRY_CONFIG_MODE")
	refs := config.GetString("REGISTRY_CONFIG_REFS")

	var configURLs []GithubConfigLink

	if mode != "" && refs != "" {
		// Mode is explicitly set, construct URL directly
		configURL := t.buildConfigURL(mode, refs)
		configURLs = []GithubConfigLink{configURL}
		slog.Debug("Using explicit config mode", "mode", mode, "refs", refs, "url", configURL)
	} else if refs != "" {
		// Refs is set but mode is not, try tag first then branch
		configURLs = []GithubConfigLink{
			t.buildConfigURL("tag", refs),
			t.buildConfigURL("branch", refs),
		}
		slog.Debug("Using explicit refs with fallback", "refs", refs, "urls", configURLs)
	} else {
		// No config set, fetch latest tag or use main branch
		latestTag, err := t.getLatestTag()
		if err != nil {
			slog.Warn("Failed to get latest tag, will use main branch", "error", err)
			configURLs = []GithubConfigLink{t.buildConfigURL("branch", "main")}
		} else if latestTag == "" {
			slog.Info("No tags found, using main branch")
			configURLs = []GithubConfigLink{t.buildConfigURL("branch", "main")}
		} else {
			slog.Info("Using latest tag", "tag", latestTag)
			configURLs = []GithubConfigLink{t.buildConfigURL("tag", latestTag)}
		}
	}

	// Try each URL until one succeeds
	for i, configURL := range configURLs {
		slog.Debug("Attempting to fetch registry config", "url", configURL, "attempt", i+1)

		resp, err := http.Get(configURL.ConfigLink)
		if err != nil {
			slog.Warn("Failed to fetch config", "url", configURL, "error", err)
			if i == len(configURLs)-1 {
				return DaoRegistryConfigResult{}, fmt.Errorf("failed to fetch config from all URLs: %w", err)
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// Read response body for additional error context
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				slog.Error("Failed to fetch registry config",
					"url", configURL,
					"status_code", resp.StatusCode,
					"status", resp.Status,
					"body_read_error", readErr)
			} else {
				slog.Error("Failed to fetch registry config",
					"url", configURL,
					"status_code", resp.StatusCode,
					"status", resp.Status,
					"response_body", string(body))
			}

			if i == len(configURLs)-1 {
				return DaoRegistryConfigResult{}, fmt.Errorf("failed to fetch config from %s: unexpected status %d (%s)", configURL, resp.StatusCode, resp.Status)
			}
			continue
		}

		var config DaoRegistryConfig
		decoder := yaml.NewDecoder(resp.Body)
		if err := decoder.Decode(&config); err != nil {
			slog.Error("Failed to parse YAML", "url", configURL, "error", err)
			if i == len(configURLs)-1 {
				return DaoRegistryConfigResult{}, fmt.Errorf("failed to parse YAML from %s: %w", configURL, err)
			}
			continue
		}

		slog.Debug("Successfully fetched registry config", "url", configURL, "chains_count", len(config))
		return DaoRegistryConfigResult{
			RemoteLink: configURL,
			Result:     config,
		}, nil
	}

	return DaoRegistryConfigResult{}, fmt.Errorf("failed to fetch config from any URL")
}

func (t *DaoSyncTask) baseRawGithubLink(mode, refs string) string {
	baseURL := "https://raw.githubusercontent.com/ringecosystem/degov-registry"
	if mode == "tag" {
		return fmt.Sprintf("%s/tags/%s", baseURL, refs)
	}
	return fmt.Sprintf("%s/heads/%s", baseURL, refs)
}

// buildConfigURL constructs the config URL based on mode and refs
func (t *DaoSyncTask) buildConfigURL(mode, refs string) GithubConfigLink {
	baseURL := t.baseRawGithubLink(mode, refs)
	return GithubConfigLink{
		BaseLink:   baseURL,
		ConfigLink: fmt.Sprintf("%s/config.yml", baseURL),
	}
}

// getLatestTag fetches the latest tag from GitHub API
func (t *DaoSyncTask) getLatestTag() (string, error) {
	apiURL := "https://api.github.com/repos/ringecosystem/degov-registry/tags"

	slog.Debug("Fetching tags from GitHub API", "url", apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var tags []GitHubTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	if len(tags) == 0 {
		return "", nil
	}

	// Return the latest (first) tag
	return tags[0].Name, nil
}

// fetchDaoConfig fetches and parses individual DAO configuration
func (t *DaoSyncTask) fetchDaoConfig(remoteConfigLink GithubConfigLink, configURL string, daoCode string) (*types.DaoConfig, error) {
	slog.Debug("Fetching DAO config", "url", configURL)

	if !strings.HasPrefix(configURL, "http://") && !strings.HasPrefix(configURL, "https://") {
		configURL = fmt.Sprintf("%s/%s", remoteConfigLink.BaseLink, configURL)
	}

	resp, err := http.Get(configURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DAO config from %s: %w", configURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body for additional error context
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			slog.Error("Failed to fetch DAO config",
				"url", configURL,
				"status_code", resp.StatusCode,
				"status", resp.Status,
				"body_read_error", readErr)
		} else {
			slog.Error("Failed to fetch DAO config",
				"url", configURL,
				"status_code", resp.StatusCode,
				"status", resp.Status,
				"response_body", string(body))
		}
		return nil, fmt.Errorf("failed to fetch DAO config from %s: unexpected status %d (%s)", configURL, resp.StatusCode, resp.Status)
	}

	var config types.DaoConfig
	decoder := yaml.NewDecoder(resp.Body)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse DAO config YAML from %s: %w", configURL, err)
	}

	slog.Debug("Successfully fetched DAO config", "url", configURL, "dao_code", daoCode)
	return &config, nil
}

// upsertDao inserts or updates a DAO in the database
func (t *DaoSyncTask) upsertDao(dao *models.Dao) error {
	var existingDao models.Dao
	result := t.db.Where("code = ?", dao.Code).First(&existingDao)

	if result.Error == gorm.ErrRecordNotFound {
		// Insert new DAO
		dao.CTime = time.Now()
		return t.db.Create(dao).Error
	} else if result.Error != nil {
		return result.Error
	}

	// Update existing DAO
	dao.ID = existingDao.ID
	dao.CTime = existingDao.CTime
	dao.UTime = utils.TimePtrNow()

	return t.db.Save(dao).Error
}

// markInactiveDAOs marks DAOs as inactive if they're not in the active list
func (t *DaoSyncTask) markInactiveDAOs(activeCodes map[string]bool) error {
	var allDaos []models.Dao
	if err := t.db.Find(&allDaos).Error; err != nil {
		return err
	}

	for _, dao := range allDaos {
		if !activeCodes[dao.Code] && dao.State != "INACTIVE" {
			slog.Info("Marking DAO as inactive", "dao", dao.Code)
			dao.State = "INACTIVE"
			dao.UTime = utils.TimePtrNow()
			if err := t.db.Save(&dao).Error; err != nil {
				slog.Error("Failed to mark DAO as inactive", "dao", dao.Code, "error", err)
			}
		}
	}

	return nil
}
