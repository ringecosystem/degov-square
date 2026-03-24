package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
