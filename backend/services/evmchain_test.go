package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
)

func TestResolveExplorerAPIConfig_BlockscoutOverride(t *testing.T) {
	t.Setenv("ETHERSCAN_API_URL", "https://api.etherscan.io")
	t.Setenv("BLOCKSCOUT_API_URLS", `{"1135":"https://blockscout.lisk.com"}`)
	t.Setenv("BLOCKSCOUT_API_KEYS", `{"1135":"lisk-key"}`)

	config, err := resolveExplorerAPIConfig(1135)
	if err != nil {
		t.Fatalf("resolveExplorerAPIConfig returned error: %v", err)
	}

	if config.Provider != explorerProviderBlockscout {
		t.Fatalf("expected provider %q, got %q", explorerProviderBlockscout, config.Provider)
	}
	if config.APIURL != "https://blockscout.lisk.com/api" {
		t.Fatalf("expected normalized blockscout api url, got %q", config.APIURL)
	}
	if config.APIKey != "lisk-key" {
		t.Fatalf("expected blockscout api key to be propagated, got %q", config.APIKey)
	}
}

func TestResolveExplorerAPIConfig_FallsBackToEtherscan(t *testing.T) {
	t.Setenv("BLOCKSCOUT_API_URLS", "")
	t.Setenv("BLOCKSCOUT_API_KEYS", "")
	t.Setenv("ETHERSCAN_API_URL", "https://api.etherscan.io")
	t.Setenv("ETHERSCAN_API_KEY", "etherscan-key")

	config, err := resolveExplorerAPIConfig(1)
	if err != nil {
		t.Fatalf("resolveExplorerAPIConfig returned error: %v", err)
	}

	if config.Provider != explorerProviderEtherscanV2 {
		t.Fatalf("expected provider %q, got %q", explorerProviderEtherscanV2, config.Provider)
	}
	if config.APIURL != "https://api.etherscan.io" {
		t.Fatalf("expected etherscan api url, got %q", config.APIURL)
	}
	if config.APIKey != "etherscan-key" {
		t.Fatalf("expected etherscan api key, got %q", config.APIKey)
	}
}

func TestGetAbiFromExplorer_BlockscoutProxy(t *testing.T) {
	address := "0x58a61b1807a7bda541855daaeaee89b1dda48568"
	requests := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, fmt.Sprintf("%s?%s", r.URL.Path, r.URL.RawQuery))
		module := r.URL.Query().Get("module")
		action := r.URL.Query().Get("action")
		if module != "contract" {
			t.Fatalf("unexpected module %q", module)
		}

		switch action {
		case "getsourcecode":
			_, _ = w.Write([]byte(`{"status":"1","message":"OK","result":[{"IsProxy":"true","ImplementationAddress":"0x18A0B8C653C291D69F21A6Ef9A1000335F71618E","ABI":"[{\"type\":\"fallback\"}]"}]}`))
		case "getabi":
			_, _ = w.Write([]byte(`{"status":"1","message":"OK","result":"[{\"type\":\"fallback\"}]"}`))
		default:
			t.Fatalf("unexpected action %q", action)
		}
	}))
	defer server.Close()

	t.Setenv("ETHERSCAN_API_URL", "https://api.etherscan.io")
	t.Setenv("BLOCKSCOUT_API_URLS", fmt.Sprintf(`{"1135":"%s"}`, server.URL))
	t.Setenv("BLOCKSCOUT_API_KEYS", "")

	service := &EvmChainService{httpClient: server.Client()}
	contract, err := service.getAbiFromExplorer(1135, address)
	if err != nil {
		t.Fatalf("getAbiFromExplorer returned error: %v", err)
	}
	if contract == nil {
		t.Fatal("expected contract info, got nil")
	}
	if contract.Type != dbmodels.ContractsAbiTypeProxy {
		t.Fatalf("expected proxy contract type, got %q", contract.Type)
	}
	if contract.Implementation != "0x18a0b8c653c291d69f21a6ef9a1000335f71618e" {
		t.Fatalf("expected normalized implementation address, got %q", contract.Implementation)
	}
	if contract.ChainId != 1135 {
		t.Fatalf("expected chain id 1135, got %d", contract.ChainId)
	}
	if len(requests) != 1 {
		t.Fatalf("expected only getsourcecode request for proxy, got %d requests", len(requests))
	}
	if !strings.HasPrefix(requests[0], "/api?") ||
		!strings.Contains(requests[0], "action=getsourcecode") ||
		!strings.Contains(requests[0], "module=contract") ||
		!strings.Contains(requests[0], "address="+address) {
		t.Fatalf("unexpected blockscout request %q", requests[0])
	}
}

func TestGetAbiFromExplorer_BlockscoutImplementationUsesSourceCodeABI(t *testing.T) {
	address := "0x18a0b8c653c291d69f21a6ef9a1000335f71618e"
	requests := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, fmt.Sprintf("%s?%s", r.URL.Path, r.URL.RawQuery))
		module := r.URL.Query().Get("module")
		action := r.URL.Query().Get("action")
		if module != "contract" {
			t.Fatalf("unexpected module %q", module)
		}

		switch action {
		case "getsourcecode":
			_, _ = w.Write([]byte(`{"status":"1","message":"OK","result":[{"IsProxy":"false","ABI":"[{\"type\":\"function\",\"name\":\"propose\"},{\"type\":\"function\",\"name\":\"queue\"}]"}]}`))
		case "getabi":
			t.Fatalf("getabi should not be called when getsourcecode already contains a verified ABI")
		default:
			t.Fatalf("unexpected action %q", action)
		}
	}))
	defer server.Close()

	t.Setenv("ETHERSCAN_API_URL", "https://api.etherscan.io")
	t.Setenv("BLOCKSCOUT_API_URLS", fmt.Sprintf(`{"1135":"%s"}`, server.URL))
	t.Setenv("BLOCKSCOUT_API_KEYS", "")

	service := &EvmChainService{httpClient: server.Client()}
	contract, err := service.getAbiFromExplorer(1135, address)
	if err != nil {
		t.Fatalf("getAbiFromExplorer returned error: %v", err)
	}
	if contract == nil {
		t.Fatal("expected contract info, got nil")
	}
	if contract.Type != dbmodels.ContractsAbiTypeImplementation {
		t.Fatalf("expected implementation contract type, got %q", contract.Type)
	}
	if contract.Abi != `[{"type":"function","name":"propose"},{"type":"function","name":"queue"}]` {
		t.Fatalf("expected ABI from sourcecode response, got %q", contract.Abi)
	}
	if contract.Implementation != "" {
		t.Fatalf("expected empty implementation address, got %q", contract.Implementation)
	}
	if len(requests) != 1 {
		t.Fatalf("expected only getsourcecode request, got %d requests", len(requests))
	}
	if !strings.HasPrefix(requests[0], "/api?") ||
		!strings.Contains(requests[0], "action=getsourcecode") ||
		!strings.Contains(requests[0], "module=contract") ||
		!strings.Contains(requests[0], "address="+address) {
		t.Fatalf("unexpected blockscout request %q", requests[0])
	}
}

func TestGetAbiFromExplorer_BlockscoutLiskLive(t *testing.T) {
	if os.Getenv("BLOCKSCOUT_LIVE_TEST") == "" {
		t.Skip("set BLOCKSCOUT_LIVE_TEST=1 to run the live Lisk Blockscout regression")
	}

	const (
		liskChainID                   = 1135
		governorProxyAddress          = "0x58a61b1807a7bda541855daaeaee89b1dda48568"
		expectedImplementationAddress = "0x18a0b8c653c291d69f21a6ef9a1000335f71618e"
	)

	t.Setenv("ETHERSCAN_API_URL", "https://api.etherscan.io")
	t.Setenv("BLOCKSCOUT_API_URLS", `{"1135":"https://blockscout.lisk.com"}`)
	t.Setenv("BLOCKSCOUT_API_KEYS", "")

	service := &EvmChainService{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	proxyContract, err := service.getAbiFromExplorer(liskChainID, governorProxyAddress)
	if err != nil {
		t.Fatalf("getAbiFromExplorer returned error for governor proxy: %v", err)
	}
	if proxyContract == nil {
		t.Fatal("expected proxy contract info, got nil")
	}
	if proxyContract.Type != dbmodels.ContractsAbiTypeProxy {
		t.Fatalf("expected proxy contract type, got %q", proxyContract.Type)
	}
	if proxyContract.Implementation != expectedImplementationAddress {
		t.Fatalf("expected implementation %q, got %q", expectedImplementationAddress, proxyContract.Implementation)
	}
	assertABIContainsEntry(t, proxyContract.Abi, "fallback", "")

	implementationContract, err := service.getAbiFromExplorer(liskChainID, expectedImplementationAddress)
	if err != nil {
		t.Fatalf("getAbiFromExplorer returned error for implementation: %v", err)
	}
	if implementationContract == nil {
		t.Fatal("expected implementation contract info, got nil")
	}
	if implementationContract.Type != dbmodels.ContractsAbiTypeImplementation {
		t.Fatalf("expected implementation contract type, got %q", implementationContract.Type)
	}
	assertABIContainsEntry(t, implementationContract.Abi, "function", "initialize")
	assertABIContainsEntry(t, implementationContract.Abi, "function", "propose")
	assertABIContainsEntry(t, implementationContract.Abi, "function", "queue")
	assertABIContainsEntry(t, implementationContract.Abi, "function", "castVoteWithReason")
}

type abiEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func assertABIContainsEntry(t *testing.T, rawABI string, entryType string, entryName string) {
	t.Helper()

	var entries []abiEntry
	if err := json.Unmarshal([]byte(rawABI), &entries); err != nil {
		t.Fatalf("failed to parse ABI JSON: %v", err)
	}

	for _, entry := range entries {
		if entry.Type != entryType {
			continue
		}
		if entryName == "" || entry.Name == entryName {
			return
		}
	}

	if entryName == "" {
		t.Fatalf("expected ABI to contain an entry of type %q", entryType)
	}
	t.Fatalf("expected ABI to contain %s %q", entryType, entryName)
}
