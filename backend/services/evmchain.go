package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/internal/config"
	"github.com/ringecosystem/degov-apps/internal/utils"
	"gorm.io/gorm"
)

type etherscanAbiResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}
type etherscanSourceCodeResult struct {
	Proxy          string `json:"Proxy"`
	ABI            string `json:"ABI"`
	Implementation string `json:"Implementation"`
}
type etherscanSourceCodeResponse struct {
	Status  string                      `json:"status"`
	Message string                      `json:"message"`
	Result  []etherscanSourceCodeResult `json:"result"`
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
		return nil, err
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
	cfg := config.GetConfig()
	apiUrl := cfg.GetString("ETHERSCAN_API_URL")
	apiKey := cfg.GetString("ETHERSCAN_API_KEY")
	if apiUrl == "" {
		return nil, fmt.Errorf("ETHERSCAN_API_URL is not set in environment variables")
	}
	sourceCodeUrl := fmt.Sprintf("%s/v2/api?apikey=%s&chainid=%d&module=contract&action=getsourcecode&address=%s", apiUrl, apiKey, chainId, address)
	resp, err := s.httpClient.Get(sourceCodeUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to call explorer source code API: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var sourceCodeResp etherscanSourceCodeResponse
	if err := json.Unmarshal(body, &sourceCodeResp); err != nil {
		return nil, fmt.Errorf("failed to parse explorer source code response: %w", err)
	}
	if sourceCodeResp.Status == "1" && len(sourceCodeResp.Result) > 0 && sourceCodeResp.Result[0].Proxy == "1" && sourceCodeResp.Result[0].Implementation != "" {
		slog.Info("Explorer identified contract as a proxy", "address", address, "implementation", sourceCodeResp.Result[0].Implementation)
		return &dbmodels.ContractsAbi{
			ChainId:        chainId,
			Address:        address,
			Type:           dbmodels.ContractsAbiTypeProxy,
			Abi:            sourceCodeResp.Result[0].ABI,
			Implementation: strings.ToLower(sourceCodeResp.Result[0].Implementation),
		}, nil
	}
	abiUrl := fmt.Sprintf("%s/v2/api?apikey=%s&chainid=%d&module=contract&action=getabi&address=%s", apiUrl, apiKey, chainId, address)
	resp, err = s.httpClient.Get(abiUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to call explorer ABI API: %w", err)
	}
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)
	var abiResp etherscanAbiResponse
	if err := json.Unmarshal(body, &abiResp); err != nil {
		return nil, fmt.Errorf("failed to parse explorer ABI response: %w", err)
	}
	if abiResp.Status == "1" && abiResp.Result != "Contract source code not verified" && abiResp.Result != "" {
		slog.Info("Explorer found ABI for contract", "address", address)
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
