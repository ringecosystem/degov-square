package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	"github.com/ringecosystem/degov-apps/internal"
	"github.com/ringecosystem/degov-apps/internal/config"
	"github.com/ringecosystem/degov-apps/services"
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
	daoService     *services.DaoService
	daoChipService *services.DaoChipService
}

// DaoRegistryConfig represents the structure of individual DAO configuration
type DaoRegistryConfig struct {
	Tags   []string          `yaml:"tags,omitempty"`  // Optional tags field
	State  dbmodels.DaoState `yaml:"state,omitempty"` // Optional state field
	Config string            `yaml:"config"`
}

type DaoRegistryConfigResult struct {
	RemoteLink GithubConfigLink
	Result     map[string][]DaoRegistryConfig
}

type DaoConfigResult struct {
	Raw    string           // Original YAML content as string
	Config *types.DaoConfig // Parsed YAML content
}

// NewDaoSyncTask creates a new DAO sync task
func NewDaoSyncTask() *DaoSyncTask {
	return &DaoSyncTask{
		daoService:     services.NewDaoService(),
		daoChipService: services.NewDaoChipService(),
	}
}

// Name returns the task name
func (t *DaoSyncTask) Name() string {
	return "dao-sync"
}

// Execute performs the DAO synchronization
func (t *DaoSyncTask) Execute() error {
	return t.syncDaos()
}

// SyncDaos fetches the latest DAO configuration and syncs it with the database
func (t *DaoSyncTask) syncDaos() error {
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

	agentDaos, adErr := t.agentDaos()
	if adErr != nil {
		// return fmt.Errorf("failed to fetch agent DAOs: %w", err)
		slog.Warn("Failed to fetch agent DAOs, continuing without them", "error", adErr)
	}

	// Process each chain and its DAOs
	for chainName, daos := range registryConfigResult.Result {
		for _, daoInfo := range daos {
			daoConfig, err := t.processSingleDao(registryConfigResult.RemoteLink, daoInfo, chainName, activeDaoCodes)
			if err != nil {
				slog.Error("Failed to process DAO", "dao", daoConfig.Code, "chain", chainName, "error", err)
				continue
			}
		}
	}

	// Process chip agents if agent DAOs were fetched successfully
	if adErr == nil {
		err := t.daoChipService.SyncAgentChips(agentDaos)
		if err != nil {
			slog.Error("Failed to sync agent chips", "error", err)
		} else {
			slog.Info("Agent chips synced successfully", "count", len(agentDaos))
		}
	}

	// Mark DAOs as inactive if they're not in the current config
	if err := t.daoService.MarkInactiveDAOs(activeDaoCodes); err != nil {
		return fmt.Errorf("failed to mark inactive DAOs: %w", err)
	}

	duration := time.Since(startTime)
	slog.Info("DAO synchronization completed",
		"active_daos", len(activeDaoCodes),
		"duration", duration.String(),
		"timestamp", time.Now().Format(time.RFC3339))
	return nil
}

// processSingleDao processes a single DAO configuration
func (t *DaoSyncTask) processSingleDao(remoteLink GithubConfigLink, daoInfo DaoRegistryConfig, chainName string, activeDaoCodes map[string]bool) (types.DaoConfig, error) {
	configURL := daoInfo.Config
	// Convert relative URL to absolute if needed
	if !strings.HasPrefix(configURL, "http://") && !strings.HasPrefix(configURL, "https://") {
		configURL = fmt.Sprintf("%s/%s", remoteLink.BaseLink, configURL)
	}

	// Fetch DAO config details
	daoConfig, err := t.fetchDaoConfig(configURL)
	if err != nil {
		return types.DaoConfig{}, fmt.Errorf("failed to fetch DAO config: %w", err)
	}

	// Skip if essential fields are missing
	if daoConfig.Config.Code == "" || daoConfig.Config.Name == "" {
		slog.Warn("DAO config missing essential fields", "config_url", daoInfo.Config)
		return types.DaoConfig{}, fmt.Errorf("missing essential fields in DAO config for code: %s", daoConfig.Config.Code)
	}

	activeDaoCodes[daoConfig.Config.Code] = true

	indexer := internal.NewDegovIndexer(daoConfig.Config.Indexer.Endpoint)

	// Query data metrics from the indexer with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var state = dbmodels.DaoStateActive
	if daoInfo.State != "" {
		state = daoInfo.State
	}

	// Prepare base input
	input := types.RefreshDaoAndConfigInput{
		Code:       daoConfig.Config.Code,
		Tags:       daoInfo.Tags,
		State:      state,
		ConfigLink: configURL,
		Config:     *daoConfig.Config,
		Raw:        daoConfig.Raw,
	}

	// Try to get metrics data
	metrics, err := indexer.QueryGlobalDataMetrics(ctx)
	if err != nil {
		slog.Warn("Failed to query data metrics", "dao", daoConfig.Config.Code, "error", err)
		// Metrics fields will be nil, indicating no update needed
	} else if metrics != nil {
		// Set metrics data if available
		input.MetricsCountProposals = &metrics.ProposalsCount
		input.MetricsCountMembers = &metrics.MemberCount
		input.MetricsSumPower = &metrics.PowerSum
		input.MetricsCountVote = &metrics.VotesCount
	}

	t.daoService.RefreshDaoAndConfig(input)

	slog.Debug("Successfully synced DAO", "dao", daoConfig.Config.Code, "chain", chainName)

	return *daoConfig.Config, nil
}

func (t *DaoSyncTask) agentDaos() ([]types.AgentDaoConfig, error) {
	const (
		maxRetries = 3
		baseDelay  = time.Second
		timeout    = 15 * time.Second
	)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: wait baseDelay * 2^(attempt-1)
			delay := baseDelay * time.Duration(1<<(attempt-1))
			slog.Debug("Retrying agent DAOs fetch", "attempt", attempt+1, "delay", delay)
			time.Sleep(delay)
		}

		// Create request with context for timeout
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel() // Defer cancel to ensure cleanup after request completes
		req, err := http.NewRequestWithContext(ctx, "GET", "https://agent.degov.ai/degov/daos", nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Set User-Agent header for better API compatibility
		req.Header.Set("User-Agent", "degov-apps/1.0")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to fetch agent DAOs (attempt %d/%d): %w", attempt+1, maxRetries, err)
			slog.Warn("Failed to fetch agent DAOs", "attempt", attempt+1, "error", err)
			continue
		}
		defer resp.Body.Close()

		// Check for HTTP errors
		if resp.StatusCode != http.StatusOK {
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				lastErr = fmt.Errorf("HTTP %d from agent DAOs API (attempt %d/%d), failed to read error body: %w",
					resp.StatusCode, attempt+1, maxRetries, readErr)
			} else {
				lastErr = fmt.Errorf("HTTP %d from agent DAOs API (attempt %d/%d): %s",
					resp.StatusCode, attempt+1, maxRetries, string(body))
			}
			slog.Warn("Agent DAOs API returned error", "attempt", attempt+1, "status_code", resp.StatusCode, "body", string(body))
			continue
		}

		var agentDaos types.Resp[[]types.AgentDaoConfig]
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read agent DAOs response body (attempt %d/%d): %w", attempt+1, maxRetries, err)
			slog.Warn("Failed to read response body", "attempt", attempt+1, "error", err)
			continue
		}

		if err := json.Unmarshal(body, &agentDaos); err != nil {
			lastErr = fmt.Errorf("failed to unmarshal agent DAOs (attempt %d/%d): %w", attempt+1, maxRetries, err)
			slog.Warn("Failed to unmarshal response", "attempt", attempt+1, "error", err)
			continue
		}

		if agentDaos.Code != 0 {
			lastErr = fmt.Errorf("agent DAOs response error (attempt %d/%d): %s", attempt+1, maxRetries, agentDaos.Message)
			slog.Warn("Agent DAOs API returned error response", "attempt", attempt+1, "code", agentDaos.Code, "message", agentDaos.Message)
			continue
		}

		// Success - return the data
		slog.Info("Successfully fetched agent DAOs", "count", len(agentDaos.Data), "attempt", attempt+1)
		return agentDaos.Data, nil
	}

	// All retries failed
	return nil, fmt.Errorf("failed to fetch agent DAOs after %d attempts: %w", maxRetries, lastErr)
}

// fetchRegistryConfig fetches and parses the main registry configuration
func (t *DaoSyncTask) fetchRegistryConfig() (DaoRegistryConfigResult, error) {
	configURLs := t.buildConfigURLs()

	for i, configURL := range configURLs {
		slog.Debug("Attempting to fetch registry config", "url", configURL, "attempt", i+1)

		var config map[string][]DaoRegistryConfig
		_, err := t.fetchAndParseYAML(configURL.ConfigLink, &config)
		if err != nil {
			if i == len(configURLs)-1 {
				return DaoRegistryConfigResult{}, fmt.Errorf("failed to fetch config from all URLs: %w", err)
			}
			slog.Warn("Failed to fetch config, trying next URL", "url", configURL, "error", err)
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

// fetchAndParseYAML fetches content from URL and parses it as YAML
func (t *DaoSyncTask) fetchAndParseYAML(url string, target interface{}) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			slog.Error("Failed to fetch content", "url", url, "status_code", resp.StatusCode, "status", resp.Status, "body_read_error", readErr)
		} else {
			slog.Error("Failed to fetch content", "url", url, "status_code", resp.StatusCode, "status", resp.Status, "response_body", string(body))
		}
		return "", fmt.Errorf("unexpected status %d (%s) from %s", resp.StatusCode, resp.Status, url)
	}

	// Read the raw content first
	rawContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from %s: %w", url, err)
	}

	// Parse the YAML content
	if err := yaml.Unmarshal(rawContent, target); err != nil {
		return "", fmt.Errorf("failed to parse YAML from %s: %w", url, err)
	}

	return string(rawContent), nil
}

// buildConfigURLs constructs the list of config URLs to try based on configuration
func (t *DaoSyncTask) buildConfigURLs() []GithubConfigLink {
	mode := config.GetString("REGISTRY_CONFIG_MODE")
	refs := config.GetString("REGISTRY_CONFIG_REFS")

	switch {
	case mode != "" && refs != "":
		configURL := t.buildConfigURL(mode, refs)
		slog.Debug("Using explicit config mode", "mode", mode, "refs", refs, "url", configURL)
		return []GithubConfigLink{configURL}

	case refs != "":
		urls := []GithubConfigLink{
			t.buildConfigURL("tag", refs),
			t.buildConfigURL("branch", refs),
		}
		slog.Debug("Using explicit refs with fallback", "refs", refs, "urls", urls)
		return urls

	default:
		return t.buildDefaultConfigURLs()
	}
}

// buildDefaultConfigURLs builds URLs when no explicit config is provided
func (t *DaoSyncTask) buildDefaultConfigURLs() []GithubConfigLink {
	latestTag, err := t.getLatestTag()
	if err != nil || latestTag == "" {
		if err != nil {
			slog.Warn("Failed to get latest tag, will use main branch", "error", err)
		} else {
			slog.Info("No tags found, using main branch")
		}
		return []GithubConfigLink{t.buildConfigURL("branch", "main")}
	}

	slog.Info("Using latest tag", "tag", latestTag)
	return []GithubConfigLink{t.buildConfigURL("tag", latestTag)}
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
func (t *DaoSyncTask) fetchDaoConfig(configURL string) (DaoConfigResult, error) {
	slog.Debug("Fetching DAO config", "url", configURL)

	var config types.DaoConfig
	rawContent, err := t.fetchAndParseYAML(configURL, &config)
	if err != nil {
		return DaoConfigResult{}, err
	}

	slog.Debug("Successfully fetched DAO config", "url", configURL, "dao_code", config.Code)
	return DaoConfigResult{
		Raw:    rawContent,
		Config: &config,
	}, nil
}
