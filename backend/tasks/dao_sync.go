package tasks

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/internal"
	"github.com/ringecosystem/degov-apps/internal/database"
	"github.com/ringecosystem/degov-apps/models"
)

type DaoSyncTask struct {
	db *gorm.DB
}

// DaoRegistryConfig represents the structure of the config.yml file
type DaoRegistryConfig map[string][]struct {
	Name   string `yaml:"name"`
	Code   string `yaml:"code"`
	Config string `yaml:"config"`
}

// DaoConfig represents the structure of individual DAO config files
type DaoConfig struct {
	ChainID int    `yaml:"chain_id"`
	Name    string `yaml:"name"`
	// Add other fields from DAO config as needed
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
	registryConfig, err := t.fetchRegistryConfig()
	if err != nil {
		return fmt.Errorf("failed to fetch registry config: %w", err)
	}

	slog.Info("Successfully fetched registry config", "chains", len(registryConfig))

	// Get chain ID mappings
	chainIDMap := t.getChainIDMap()

	// Track active DAO codes for marking inactive ones
	activeDaoCodes := make(map[string]bool)

	// Process each chain and its DAOs
	for chainName, daos := range registryConfig {
		chainID, exists := chainIDMap[chainName]
		if !exists {
			slog.Warn("Unknown chain name, skipping", "chain", chainName)
			continue
		}

		for _, daoInfo := range daos {
			activeDaoCodes[daoInfo.Code] = true

			// Fetch DAO config details
			daoConfig, err := t.fetchDaoConfig(daoInfo.Config)
			if err != nil {
				slog.Error("Failed to fetch DAO config", "dao", daoInfo.Code, "error", err)
				continue
			}

			// Upsert DAO in database
			dao := &models.Dao{
				ID:         internal.NextIDString(),
				ChainID:    chainID,
				ChainName:  chainName,
				Name:       daoInfo.Name,
				Code:       daoInfo.Code,
				Seq:        0,
				State:      "ACTIVE",
				ConfigLink: daoInfo.Config,
				TimeSync:   timePtr(time.Now()),
			}

			// Override with config values if available
			if daoConfig.ChainID != 0 {
				dao.ChainID = daoConfig.ChainID
			}
			if daoConfig.Name != "" {
				dao.Name = daoConfig.Name
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
func (t *DaoSyncTask) fetchRegistryConfig() (DaoRegistryConfig, error) {
	const configURL = "https://raw.githubusercontent.com/ringecosystem/degov-registry/refs/heads/daos-config/config.yml"

	resp, err := http.Get(configURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config DaoRegistryConfig
	decoder := yaml.NewDecoder(resp.Body)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return config, nil
}

// fetchDaoConfig fetches and parses individual DAO configuration
func (t *DaoSyncTask) fetchDaoConfig(configURL string) (*DaoConfig, error) {
	resp, err := http.Get(configURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DAO config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config DaoConfig
	decoder := yaml.NewDecoder(resp.Body)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse DAO config YAML: %w", err)
	}

	return &config, nil
}

// getChainIDMap returns a mapping of chain names to chain IDs
func (t *DaoSyncTask) getChainIDMap() map[string]int {
	return map[string]int{
		"darwinia": 46,    // Darwinia chain ID
		"ethereum": 1,     // Ethereum mainnet
		"polygon":  137,   // Polygon
		"arbitrum": 42161, // Arbitrum One
		"optimism": 10,    // Optimism
		"bsc":      56,    // Binance Smart Chain
		// Add more chains as needed
	}
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
	dao.UTime = timePtr(time.Now())

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
			dao.UTime = timePtr(time.Now())
			if err := t.db.Save(&dao).Error; err != nil {
				slog.Error("Failed to mark DAO as inactive", "dao", dao.Code, "error", err)
			}
		}
	}

	return nil
}

// timePtr returns a pointer to the given time
func timePtr(t time.Time) *time.Time {
	return &t
}
