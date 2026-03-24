package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/internal/utils"
	"gorm.io/gorm"
)

type explorerProvider string

const (
	explorerProviderEtherscanV2 explorerProvider = "etherscan-v2"
	explorerProviderBlockscout  explorerProvider = "blockscout"
)

type etherscanAbiResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}
type explorerSourceCodeResult struct {
	Proxy                 string `json:"Proxy"`
	IsProxy               string `json:"IsProxy"`
	ABI                   string `json:"ABI"`
	Implementation        string `json:"Implementation"`
	ImplementationAddress string `json:"ImplementationAddress"`
}

func (r explorerSourceCodeResult) IsProxyContract() bool {
	return r.Proxy == "1" || r.IsProxy == "1" || strings.EqualFold(r.IsProxy, "true")
}

func (r explorerSourceCodeResult) ImplementationValue() string {
	if r.Implementation != "" {
		return r.Implementation
	}
	return r.ImplementationAddress
}

type explorerSourceCodeResponse struct {
	Status  string                     `json:"status"`
	Message string                     `json:"message"`
	Result  []explorerSourceCodeResult `json:"result"`
}
type swissKnifeResponse struct {
	Status string `json:"status"`
	Data   struct {
		IsProxy               bool   `json:"isProxy"`
		ImplementationAddress string `json:"implementationAddress"`
		Abi                   string `json:"abi"`
	} `json:"data"`
}

type EvmChainService struct {
	db         *gorm.DB
	httpClient *http.Client
}

type explorerAPIConfig struct {
	Provider explorerProvider
	APIURL   string
	APIKey   string
}

type explorerChainMapCache struct {
	mu                sync.Mutex
	blockscoutURLsRaw string
	blockscoutKeysRaw string
	blockscoutURLs    map[int]string
	blockscoutKeys    map[int]string
	err               error
}

var cachedExplorerChainMaps explorerChainMapCache

func NewEvmChainService() *EvmChainService {
	return &EvmChainService{
		db: database.GetDB(),
		httpClient: &http.Client{
			Timeout: 15 * time.Second, // Set a 15-second timeout.
		},
	}
}

func (s *EvmChainService) GetAbi(input gqlmodels.EvmAbiInput) ([]*gqlmodels.EvmAbiOutput, error) {
	initialAddress := strings.ToLower(input.Contract)
	chainId := int(input.Chain)

	// Use a map to detect circular dependencies in proxy contracts.
	visited := make(map[string]bool)

	// 1. Parse the contract chain (which may be a single implementation or multiple levels of proxy).
	fullChain, err := s.resolveProxyChain(chainId, initialAddress, visited)
	if err != nil {
		return nil, fmt.Errorf("could not resolve contract chain for %s on chain %d: %w", initialAddress, chainId, err)
	}

	// 2. Converts a database model to a GQL output model.
	var results []*gqlmodels.EvmAbiOutput
	for _, contract := range fullChain {
		var gqlType gqlmodels.AbiType
		switch contract.Type {
		case dbmodels.ContractsAbiTypeProxy:
			gqlType = gqlmodels.AbiTypeProxy
		case dbmodels.ContractsAbiTypeImplementation:
			gqlType = gqlmodels.AbiTypeImplementation
		default:
			return nil, fmt.Errorf("unknown contract type '%s' for address %s", contract.Type, contract.Address)
		}

		output := &gqlmodels.EvmAbiOutput{
			Address:        contract.Address,
			Abi:            contract.Abi,
			Type:           gqlType,
			Implementation: &contract.Implementation,
		}
		results = append(results, output)
	}

	return results, nil
}

// resolveProxyChain Recursively walk the proxy chain to find and return all contracts in the chain (including intermediate proxies and final implementations).
func (s *EvmChainService) resolveProxyChain(chainId int, address string, visited map[string]bool) ([]*dbmodels.ContractsAbi, error) {
	// Prevent circular proxy references.
	if visited[address] {
		return nil, fmt.Errorf("circular proxy dependency detected for address %s", address)
	}
	visited[address] = true

	// Get the contract information of the current address.
	contractInfo, err := s.getContractInfo(chainId, address)
	if err != nil {
		return nil, err
	}

	// Check the type to decide whether to continue recursing.
	if contractInfo.Type == dbmodels.ContractsAbiTypeImplementation {
		// The end of the chain is reached (the contract is fulfilled), and a slice containing only itself is returned.
		return []*dbmodels.ContractsAbi{contractInfo}, nil
	}

	if contractInfo.Type == dbmodels.ContractsAbiTypeProxy {
		if contractInfo.Implementation == "" {
			return nil, fmt.Errorf("proxy contract %s has no implementation address", address)
		}
		slog.Info("Following proxy chain...", "from", address, "to", contractInfo.Implementation)

		// Call recursively to get the rest of the chain.
		remainingChain, err := s.resolveProxyChain(chainId, contractInfo.Implementation, visited)

		results := []*dbmodels.ContractsAbi{contractInfo}
		if err != nil {
			if contractInfo.Address == contractInfo.Implementation {
				slog.Warn("Circular proxy detected", "address", address, "err", err)
				return results, nil
			}
		}

		// Prepend the current contract to the front of the result chain and return it.
		return append(results, remainingChain...), err
	}

	return nil, fmt.Errorf("contract %s has an unknown or invalid type: %s", address, contractInfo.Type)
}

// getContractInfo retrieves information about a single contract address from various sources.
// It checks the database -> blockchain explorer -> Swiss-knife in this order, and caches the results. It is not recursive.
func (s *EvmChainService) getContractInfo(chainId int, address string) (*dbmodels.ContractsAbi, error) {
	// 1. Try reading from the database.
	dbContract, err := s.getAbiFromDB(chainId, address)
	if err != nil {
		slog.Error("Error querying database", "chainId", chainId, "address", address, "error", err)
		return nil, err
	}
	if dbContract != nil {
		return dbContract, nil
	}

	// 2. If it's not in the database, try getting it from the browser.
	slog.Info("Contract not in DB, trying Explorer...", "chainId", chainId, "address", address)
	explorerContract, err := s.getAbiFromExplorer(chainId, address)
	if err != nil {
		slog.Warn("Failed to get info from explorer", "chainId", chainId, "address", address, "error", err)
	}
	if explorerContract != nil {
		if err := s.saveAbiToDB(explorerContract); err != nil {
			slog.Error("Failed to save explorer data to DB", "error", err)
		}
		return explorerContract, nil
	}

	// 3. If it's not available in your browser, try getting it from Swiss-knife.
	slog.Info("Contract not in Explorer, trying Swiss-knife...", "chainId", chainId, "address", address)
	swissKnifeContract, err := s.getAbiFromSwissKnife(chainId, address)
	if err != nil {
		slog.Warn("Failed to get info from Swiss-knife", "chainId", chainId, "address", address, "error", err)
	}
	if swissKnifeContract != nil {
		if err := s.saveAbiToDB(swissKnifeContract); err != nil {
			slog.Error("Failed to save Swiss-knife data to DB", "error", err)
		}
		return swissKnifeContract, nil
	}

	// 4. Not found in any data source.
	return nil, fmt.Errorf("contract info not found for address %s on chain %d", address, chainId)
}

func (s *EvmChainService) getAbiFromDB(chainId int, address string) (*dbmodels.ContractsAbi, error) {
	var contract dbmodels.ContractsAbi
	err := s.db.Where("chain_id = ? AND address = ?", chainId, address).First(&contract).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &contract, nil
}

func (s *EvmChainService) getAbiFromExplorer(chainId int, address string) (*dbmodels.ContractsAbi, error) {
	explorerConfig, err := resolveExplorerAPIConfig(chainId)
	if err != nil {
		return nil, err
	}

	switch explorerConfig.Provider {
	case explorerProviderBlockscout:
		return s.getAbiFromBlockscout(chainId, address, explorerConfig)
	case explorerProviderEtherscanV2:
		return s.getAbiFromEtherscanV2(chainId, address, explorerConfig)
	default:
		return nil, fmt.Errorf("unsupported explorer provider %q", explorerConfig.Provider)
	}
}

func resolveExplorerAPIConfig(chainId int) (*explorerAPIConfig, error) {
	cfg := config.GetConfig()

	blockscoutURLs, blockscoutKeys, err := loadCachedExplorerChainMaps(cfg)
	if err != nil {
		return nil, err
	}

	if apiURL, ok := blockscoutURLs[chainId]; ok && apiURL != "" {
		return &explorerAPIConfig{
			Provider: explorerProviderBlockscout,
			APIURL:   normalizeBlockscoutAPIURL(apiURL),
			APIKey:   strings.TrimSpace(blockscoutKeys[chainId]),
		}, nil
	}

	apiURL := strings.TrimRight(strings.TrimSpace(cfg.GetString("ETHERSCAN_API_URL")), "/")
	apiKey := strings.TrimSpace(cfg.GetString("ETHERSCAN_API_KEY"))
	if apiURL == "" {
		return nil, fmt.Errorf("no explorer API configured for chain %d", chainId)
	}

	return &explorerAPIConfig{
		Provider: explorerProviderEtherscanV2,
		APIURL:   apiURL,
		APIKey:   apiKey,
	}, nil
}

func loadCachedExplorerChainMaps(cfg *config.Config) (map[int]string, map[int]string, error) {
	blockscoutURLsRaw := strings.TrimSpace(cfg.GetString("BLOCKSCOUT_API_URLS"))
	blockscoutKeysRaw := strings.TrimSpace(cfg.GetString("BLOCKSCOUT_API_KEYS"))

	cachedExplorerChainMaps.mu.Lock()
	defer cachedExplorerChainMaps.mu.Unlock()

	if cachedExplorerChainMaps.blockscoutURLsRaw == blockscoutURLsRaw &&
		cachedExplorerChainMaps.blockscoutKeysRaw == blockscoutKeysRaw {
		return cachedExplorerChainMaps.blockscoutURLs, cachedExplorerChainMaps.blockscoutKeys, cachedExplorerChainMaps.err
	}

	blockscoutURLs, err := loadChainStringMap(blockscoutURLsRaw)
	if err != nil {
		cachedExplorerChainMaps.blockscoutURLsRaw = blockscoutURLsRaw
		cachedExplorerChainMaps.blockscoutKeysRaw = blockscoutKeysRaw
		cachedExplorerChainMaps.blockscoutURLs = nil
		cachedExplorerChainMaps.blockscoutKeys = nil
		cachedExplorerChainMaps.err = fmt.Errorf("failed to parse BLOCKSCOUT_API_URLS: %w", err)
		return nil, nil, cachedExplorerChainMaps.err
	}

	blockscoutKeys, err := loadChainStringMap(blockscoutKeysRaw)
	if err != nil {
		cachedExplorerChainMaps.blockscoutURLsRaw = blockscoutURLsRaw
		cachedExplorerChainMaps.blockscoutKeysRaw = blockscoutKeysRaw
		cachedExplorerChainMaps.blockscoutURLs = blockscoutURLs
		cachedExplorerChainMaps.blockscoutKeys = nil
		cachedExplorerChainMaps.err = fmt.Errorf("failed to parse BLOCKSCOUT_API_KEYS: %w", err)
		return nil, nil, cachedExplorerChainMaps.err
	}

	cachedExplorerChainMaps.blockscoutURLsRaw = blockscoutURLsRaw
	cachedExplorerChainMaps.blockscoutKeysRaw = blockscoutKeysRaw
	cachedExplorerChainMaps.blockscoutURLs = blockscoutURLs
	cachedExplorerChainMaps.blockscoutKeys = blockscoutKeys
	cachedExplorerChainMaps.err = nil

	return blockscoutURLs, blockscoutKeys, nil
}

func loadChainStringMap(raw string) (map[int]string, error) {
	if strings.TrimSpace(raw) == "" {
		return map[int]string{}, nil
	}

	var valueByChain map[string]string
	if err := json.Unmarshal([]byte(raw), &valueByChain); err != nil {
		return nil, err
	}

	result := make(map[int]string, len(valueByChain))
	for rawChainID, value := range valueByChain {
		chainID, err := strconv.Atoi(strings.TrimSpace(rawChainID))
		if err != nil {
			return nil, fmt.Errorf("invalid chain id %q", rawChainID)
		}
		result[chainID] = strings.TrimSpace(value)
	}
	return result, nil
}

func normalizeBlockscoutAPIURL(raw string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "/api") {
		return trimmed
	}
	return trimmed + "/api"
}

func buildExplorerQuery(baseURL string, params url.Values) string {
	return fmt.Sprintf("%s?%s", strings.TrimRight(baseURL, "?"), params.Encode())
}

func hasVerifiedExplorerABI(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return trimmed != "" && trimmed != "[]" && trimmed != "Contract source code not verified"
}

func (s *EvmChainService) getAbiFromEtherscanV2(chainId int, address string, explorerConfig *explorerAPIConfig) (*dbmodels.ContractsAbi, error) {
	sourceCodeParams := url.Values{
		"chainid": []string{strconv.Itoa(chainId)},
		"module":  []string{"contract"},
		"action":  []string{"getsourcecode"},
		"address": []string{address},
	}
	if explorerConfig.APIKey != "" {
		sourceCodeParams.Set("apikey", explorerConfig.APIKey)
	}

	sourceCodeURL := buildExplorerQuery(explorerConfig.APIURL+"/v2/api", sourceCodeParams)
	abiParams := url.Values{
		"chainid": []string{strconv.Itoa(chainId)},
		"module":  []string{"contract"},
		"action":  []string{"getabi"},
		"address": []string{address},
	}
	if explorerConfig.APIKey != "" {
		abiParams.Set("apikey", explorerConfig.APIKey)
	}

	abiURL := buildExplorerQuery(explorerConfig.APIURL+"/v2/api", abiParams)
	return s.getAbiFromExplorerEndpoints(chainId, address, sourceCodeURL, abiURL, explorerConfig)
}

func (s *EvmChainService) getAbiFromBlockscout(chainId int, address string, explorerConfig *explorerAPIConfig) (*dbmodels.ContractsAbi, error) {
	sourceCodeParams := url.Values{
		"module":  []string{"contract"},
		"action":  []string{"getsourcecode"},
		"address": []string{address},
	}
	if explorerConfig.APIKey != "" {
		sourceCodeParams.Set("apikey", explorerConfig.APIKey)
	}

	sourceCodeURL := buildExplorerQuery(explorerConfig.APIURL, sourceCodeParams)
	abiParams := url.Values{
		"module":  []string{"contract"},
		"action":  []string{"getabi"},
		"address": []string{address},
	}
	if explorerConfig.APIKey != "" {
		abiParams.Set("apikey", explorerConfig.APIKey)
	}

	abiURL := buildExplorerQuery(explorerConfig.APIURL, abiParams)
	return s.getAbiFromExplorerEndpoints(chainId, address, sourceCodeURL, abiURL, explorerConfig)
}

func (s *EvmChainService) getAbiFromExplorerEndpoints(chainId int, address string, sourceCodeURL string, abiURL string, explorerConfig *explorerAPIConfig) (*dbmodels.ContractsAbi, error) {
	resp, err := s.httpClient.Get(sourceCodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to call explorer source code API: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var sourceCodeResp explorerSourceCodeResponse
	if err := json.Unmarshal(body, &sourceCodeResp); err != nil {
		return nil, fmt.Errorf("failed to parse explorer source code response: %w", err)
	}
	if sourceCodeResp.Status == "1" && len(sourceCodeResp.Result) > 0 {
		result := sourceCodeResp.Result[0]
		implementationAddress := strings.ToLower(result.ImplementationValue())
		if result.IsProxyContract() && implementationAddress != "" {
			slog.Info("Explorer identified contract as a proxy", "address", address, "implementation", implementationAddress)
			return &dbmodels.ContractsAbi{
				ChainId:        chainId,
				Address:        address,
				Type:           dbmodels.ContractsAbiTypeProxy,
				Abi:            result.ABI,
				Implementation: implementationAddress,
			}, nil
		}

		if hasVerifiedExplorerABI(result.ABI) {
			slog.Info("Explorer found ABI in source code response", "address", address, "provider", explorerConfig.Provider)
			return &dbmodels.ContractsAbi{
				ChainId: chainId,
				Address: address,
				Type:    dbmodels.ContractsAbiTypeImplementation,
				Abi:     result.ABI,
			}, nil
		}
	}

	resp, err = s.httpClient.Get(abiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to call explorer ABI API: %w", err)
	}
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)
	var abiResp etherscanAbiResponse
	if err := json.Unmarshal(body, &abiResp); err != nil {
		return nil, fmt.Errorf("failed to parse explorer ABI response: %w", err)
	}
	if abiResp.Status == "1" && hasVerifiedExplorerABI(abiResp.Result) {
		slog.Info("Explorer found ABI for contract", "address", address, "provider", explorerConfig.Provider)
		return &dbmodels.ContractsAbi{
			ChainId: chainId,
			Address: address,
			Type:    dbmodels.ContractsAbiTypeImplementation,
			Abi:     abiResp.Result,
		}, nil
	}
	return nil, nil
}

func (s *EvmChainService) getAbiFromSwissKnife(chainId int, address string) (*dbmodels.ContractsAbi, error) {
	url := fmt.Sprintf("https://swiss-knife.xyz/api/evm/contract/%s?network=%d", address, chainId)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "degov-app/1.0")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Swiss-knife API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("swiss-knife API returned non-200 status: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var swissResp swissKnifeResponse
	if err := json.Unmarshal(body, &swissResp); err != nil {
		return nil, fmt.Errorf("failed to parse Swiss-knife response: %w", err)
	}
	if swissResp.Status == "success" {
		if swissResp.Data.IsProxy && swissResp.Data.ImplementationAddress != "" {
			slog.Info("Swiss-knife identified contract as a proxy", "address", address, "implementation", swissResp.Data.ImplementationAddress)
			return &dbmodels.ContractsAbi{
				ChainId:        chainId,
				Address:        address,
				Type:           dbmodels.ContractsAbiTypeProxy,
				Implementation: strings.ToLower(swissResp.Data.ImplementationAddress),
			}, nil
		}
		if swissResp.Data.Abi != "" && swissResp.Data.Abi != "[]" {
			slog.Info("Swiss-knife found ABI for contract", "address", address)
			return &dbmodels.ContractsAbi{
				ChainId: chainId,
				Address: address,
				Type:    dbmodels.ContractsAbiTypeImplementation,
				Abi:     swissResp.Data.Abi,
			}, nil
		}
	}
	return nil, nil
}

func (s *EvmChainService) saveAbiToDB(contract *dbmodels.ContractsAbi) error {
	var existing dbmodels.ContractsAbi
	err := s.db.Where("chain_id = ? AND address = ?", contract.ChainId, contract.Address).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == nil {
		existing.Type = contract.Type
		existing.Implementation = contract.Implementation
		if contract.Abi != "" {
			existing.Abi = contract.Abi
		}
		now := time.Now()
		existing.UTime = &now
		return s.db.Save(&existing).Error
	}
	contract.ID = utils.NextIDString()
	contract.CTime = time.Now()
	now := time.Now()
	contract.UTime = &now
	return s.db.Create(contract).Error
}
