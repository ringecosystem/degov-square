package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/types"
	"github.com/wealdtech/go-ens/v3"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

const defaultENSCacheTTL = 3 * time.Hour
const defaultENSCacheMaxEntries = 1000
const mainnetChainID = 1

type ENSRecord struct {
	Address *string
	Name    *string
}

type ensCacheEntry struct {
	record ENSRecord
	timer  *time.Timer
}

type ENSService struct {
	db    *gorm.DB
	mu    sync.Mutex
	cache map[string]ensCacheEntry
}

func NewENSService() *ENSService {
	return &ENSService{
		db:    database.GetDB(),
		cache: make(map[string]ensCacheEntry),
	}
}

func (s *ENSService) Resolve(ctx context.Context, daoCode *string, address *string, name *string) (*ENSRecord, error) {
	normalizedDaoCode := ""
	if daoCode != nil {
		normalizedDaoCode = strings.ToLower(strings.TrimSpace(*daoCode))
	}

	normalizedAddress := ""
	if address != nil {
		normalizedAddress = strings.ToLower(strings.TrimSpace(*address))
	}

	normalizedName := ""
	if name != nil {
		normalizedName = strings.ToLower(strings.TrimSpace(*name))
	}

	if (normalizedAddress == "" && normalizedName == "") || (normalizedAddress != "" && normalizedName != "") {
		return nil, fmt.Errorf("ens query requires exactly one of address or name")
	}

	if normalizedAddress != "" && !common.IsHexAddress(normalizedAddress) {
		return nil, fmt.Errorf("invalid ens address")
	}

	cacheKey := normalizedDaoCode + ":name:" + normalizedAddress
	if normalizedName != "" {
		cacheKey = normalizedDaoCode + ":address:" + normalizedName
	}

	if record, ok := s.getCached(cacheKey); ok {
		return &record, nil
	}

	rpcURLs := s.ensRPCURLs(daoCode)
	if len(rpcURLs) == 0 {
		return nil, fmt.Errorf("no ENS RPC URL configured")
	}

	var record ENSRecord
	var err error
	if normalizedAddress != "" {
		record.Address = &normalizedAddress
		record.Name, err = resolveENSNameWithRPCs(ctx, rpcURLs, normalizedAddress)
	} else {
		record.Name = &normalizedName
		record.Address, err = resolveENSAddressWithRPCs(ctx, rpcURLs, normalizedName)
	}
	if err != nil {
		return nil, err
	}

	s.setCached(cacheKey, record)
	return &record, nil
}

func (s *ENSService) getCached(key string) (ENSRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.cache[key]
	if !ok {
		return ENSRecord{}, false
	}
	return entry.record, true
}

func (s *ENSService) setCached(key string, record ENSRecord) {
	ttl := ensCacheTTL()

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.cache[key]
	if exists && entry.timer != nil {
		entry.timer.Stop()
	}

	for !exists && len(s.cache) >= ensCacheMaxEntries() {
		for oldestKey, oldestEntry := range s.cache {
			if oldestEntry.timer != nil {
				oldestEntry.timer.Stop()
			}
			delete(s.cache, oldestKey)
			break
		}
	}

	var timer *time.Timer
	timer = time.AfterFunc(ttl, func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		entry, ok := s.cache[key]
		if ok && entry.timer == timer {
			delete(s.cache, key)
		}
	})
	s.cache[key] = ensCacheEntry{record: record, timer: timer}
}

func (s *ENSService) ensRPCURLs(daoCode *string) []string {
	rpcURLs := envENSRPCURLs()
	if daoCode != nil && strings.TrimSpace(*daoCode) != "" {
		rpcURLs = append(rpcURLs, s.daoRPCURLs(strings.TrimSpace(*daoCode))...)
	}
	if len(rpcURLs) == 0 {
		rpcURLs = append(rpcURLs, s.activeDaoRPCURLs()...)
	}
	return compactStrings(rpcURLs)
}

func (s *ENSService) daoRPCURLs(daoCode string) []string {
	var rawConfig dbmodels.DgvDaoConfig
	err := s.db.Where("dao_code = ?", daoCode).First(&rawConfig).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		slog.Warn("Failed to load DAO config for ENS RPC fallback", "daoCode", daoCode, "err", err)
		return nil
	}

	var daoConfig types.DaoConfig
	if err := yaml.Unmarshal([]byte(rawConfig.Config), &daoConfig); err != nil {
		slog.Warn("Failed to parse DAO config for ENS RPC fallback", "daoCode", daoCode, "err", err)
		return nil
	}

	if daoConfig.Chain.ID != mainnetChainID {
		return nil
	}

	return daoConfig.Chain.RPCs
}

func (s *ENSService) activeDaoRPCURLs() []string {
	var rows []struct {
		Config string
	}
	if err := s.db.Table("dgv_dao_config").
		Select("dgv_dao_config.config").
		Joins("join dgv_dao on dgv_dao.code = dgv_dao_config.dao_code").
		Where("dgv_dao.state = ?", dbmodels.DaoStateActive).
		Find(&rows).Error; err != nil {
		slog.Warn("Failed to load active DAO configs for ENS RPC fallback", "err", err)
		return nil
	}

	rpcURLs := make([]string, 0)
	for _, row := range rows {
		var daoConfig types.DaoConfig
		if err := yaml.Unmarshal([]byte(row.Config), &daoConfig); err != nil {
			continue
		}
		if daoConfig.Chain.ID != mainnetChainID {
			continue
		}
		rpcURLs = append(rpcURLs, daoConfig.Chain.RPCs...)
	}
	return rpcURLs
}

func resolveENSNameWithRPCs(ctx context.Context, rpcURLs []string, address string) (*string, error) {
	var lastErr error
	for _, rpcURL := range rpcURLs {
		ensName, err := resolveENSNameViaRPC(ctx, rpcURL, address)
		if err == nil {
			return ensName, nil
		}
		if isNoENSResolutionError(err) {
			return nil, nil
		}
		lastErr = err
		slog.Warn("Failed to resolve ENS name", "rpc", safeRPCLabel(rpcURL), "address", address, "errorName", errorName(err))
	}
	if lastErr != nil {
		return nil, errors.New("all ENS RPCs failed")
	}
	return nil, nil
}

func resolveENSAddressWithRPCs(ctx context.Context, rpcURLs []string, name string) (*string, error) {
	var lastErr error
	for _, rpcURL := range rpcURLs {
		address, err := resolveENSAddressViaRPC(ctx, rpcURL, name)
		if err == nil {
			return address, nil
		}
		if isNoENSResolutionError(err) {
			return nil, nil
		}
		lastErr = err
		slog.Warn("Failed to resolve ENS address", "rpc", safeRPCLabel(rpcURL), "name", name, "errorName", errorName(err))
	}
	if lastErr != nil {
		return nil, errors.New("all ENS RPCs failed")
	}
	return nil, nil
}

var resolveENSNameViaRPC = func(ctx context.Context, rpcURL string, address string) (*string, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, errors.New("failed to connect to ENS RPC")
	}
	defer client.Close()

	ensName, err := ens.ReverseResolve(client, common.HexToAddress(address))
	if err != nil {
		return nil, err
	}

	return &ensName, nil
}

var resolveENSAddressViaRPC = func(ctx context.Context, rpcURL string, name string) (*string, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, errors.New("failed to connect to ENS RPC")
	}
	defer client.Close()

	address, err := ens.Resolve(client, name)
	if err != nil {
		return nil, err
	}
	if address == (common.Address{}) {
		return nil, errors.New("no resolution")
	}

	resolvedAddress := strings.ToLower(address.Hex())
	return &resolvedAddress, nil
}

func envENSRPCURLs() []string {
	cfg := config.GetConfig()
	return splitCSV(cfg.GetString("RPC_URL_1"))
}

func ensCacheTTL() time.Duration {
	cfg := config.GetConfig()
	if raw := strings.TrimSpace(cfg.GetString("DEGOV_ENS_CACHE_TTL")); raw != "" {
		if ttl, err := time.ParseDuration(raw); err == nil && ttl > 0 {
			return ttl
		}
	}
	if raw := strings.TrimSpace(cfg.GetString("DEGOV_ENS_CACHE_TTL_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultENSCacheTTL
}

func ensCacheMaxEntries() int {
	cfg := config.GetConfig()
	if raw := strings.TrimSpace(cfg.GetString("DEGOV_ENS_CACHE_MAX_ENTRIES")); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			return value
		}
	}
	return defaultENSCacheMaxEntries
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func compactStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func safeRPCLabel(rpcURL string) string {
	parsed, err := url.Parse(rpcURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "invalid_rpc_url"
	}
	return parsed.Scheme + "://" + parsed.Host
}

func errorName(err error) string {
	if err == nil {
		return ""
	}

	return fmt.Sprintf("%T", err)
}

func isNoENSResolutionError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no resolution") ||
		strings.Contains(message, "unregistered name") ||
		strings.Contains(message, "no address")
}
