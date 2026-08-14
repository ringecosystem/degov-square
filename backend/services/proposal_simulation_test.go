package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	testSimulationDescriptionHash = "0x1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8"
	testSimulationProposalID      = "45104572425689951088238789760954264672210632875420191946694406891418830968826"
	testSimulationExecuteData     = "0x2656227d000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001001c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac800000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000021234000000000000000000000000000000000000000000000000000000000000"
)

func testSimulationRequest() ProposalSimulationRequest {
	return ProposalSimulationRequest{
		Caller:          "0x0000000000000000000000000000000000000002",
		Targets:         []string{"0x0000000000000000000000000000000000000001"},
		Values:          []string{"0"},
		Calldatas:       []string{"0x1234"},
		DescriptionHash: testSimulationDescriptionHash,
	}
}

func newTestProposalSimulationDB(t *testing.T, rpcURL, features string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE dgv_dao (id TEXT PRIMARY KEY, code TEXT NOT NULL UNIQUE, features TEXT)`,
		`CREATE TABLE dgv_dao_config (id TEXT PRIMARY KEY, dao_code TEXT NOT NULL UNIQUE, config TEXT NOT NULL)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create simulation table: %v", err)
		}
	}
	if err := db.Exec(`INSERT INTO dgv_dao (id, code, features) VALUES (?, ?, ?)`, "dao-1", "demo", features).Error; err != nil {
		t.Fatalf("seed DAO: %v", err)
	}
	configYAML := fmt.Sprintf(`
code: demo
chain:
  id: 1
  rpcs: [%q]
contracts:
  governor: "0x0000000000000000000000000000000000000003"
`, rpcURL)
	if err := db.Exec(`INSERT INTO dgv_dao_config (id, dao_code, config) VALUES (?, ?, ?)`, "config-1", "demo", configYAML).Error; err != nil {
		t.Fatalf("seed DAO config: %v", err)
	}
	return db
}

func TestValidateSimulationRequestMatchesOpenZeppelinVector(t *testing.T) {
	validated, err := validateSimulationRequest(testSimulationProposalID, testSimulationRequest())
	if err != nil {
		t.Fatalf("validateSimulationRequest: %v", err)
	}
	if got := hexutil.Encode(validated.ExecuteData); got != testSimulationExecuteData {
		t.Fatalf("execute calldata = %s, want %s", got, testSimulationExecuteData)
	}

	_, err = validateSimulationRequest("1", testSimulationRequest())
	var simulationErr *ProposalSimulationError
	if !errors.As(err, &simulationErr) || simulationErr.Code != "proposal_mismatch" {
		t.Fatalf("mismatch error = %v, want proposal_mismatch", err)
	}

	overlong := testSimulationRequest()
	overlong.Values[0] = strings.Repeat("9", 79)
	_, err = validateSimulationRequest(testSimulationProposalID, overlong)
	if !errors.As(err, &simulationErr) || simulationErr.Code != "invalid_request" {
		t.Fatalf("overlong value error = %v, want invalid_request", err)
	}
}

func TestValidateSimulationRequestDetectsXAccountDispatch(t *testing.T) {
	request := testSimulationRequest()
	request.Calldatas[0] = hexutil.Encode(xAccountSelector)
	arguments, err := governorSimulationABI.Methods["hashProposal"].Inputs.Pack(
		[]common.Address{common.HexToAddress(request.Targets[0])},
		[]*big.Int{new(big.Int)},
		[][]byte{xAccountSelector},
		common.HexToHash(request.DescriptionHash),
	)
	if err != nil {
		t.Fatalf("pack proposal: %v", err)
	}
	proposalID := new(big.Int).SetBytes(crypto.Keccak256(arguments)).String()
	validated, err := validateSimulationRequest(proposalID, request)
	if err != nil {
		t.Fatalf("validateSimulationRequest: %v", err)
	}
	if !validated.HasXAccount {
		t.Fatal("XAccount dispatch was not detected")
	}
}

func TestProposalSimulationCapabilityRequiresFeatureAndProvider(t *testing.T) {
	db := newTestProposalSimulationDB(t, "https://rpc.example", `[]`)
	service := newProposalSimulationService(db, proposalSimulationConfig{NativeFallback: true})

	capability, err := service.Capability(context.Background(), "missing")
	if err != nil || capability.Enabled || capability.Reason != "dao_not_found" {
		t.Fatalf("missing capability = %#v, err %v", capability, err)
	}
	capability, err = service.Capability(context.Background(), "demo")
	if err != nil || capability.Enabled || capability.Reason != "dao_feature_disabled" {
		t.Fatalf("disabled capability = %#v, err %v", capability, err)
	}

	if err := db.Exec(`UPDATE dgv_dao SET features = ? WHERE code = ?`, `["proposal-simulation"]`, "demo").Error; err != nil {
		t.Fatalf("enable feature: %v", err)
	}
	service = newProposalSimulationService(db, proposalSimulationConfig{})
	capability, err = service.Capability(context.Background(), "demo")
	if err != nil || capability.Enabled || capability.Reason != "provider_unavailable" {
		t.Fatalf("provider-disabled capability = %#v, err %v", capability, err)
	}

	service = newProposalSimulationService(db, proposalSimulationConfig{NativeFallback: true})
	capability, err = service.Capability(context.Background(), "demo")
	if err != nil || !capability.Enabled || capability.Fidelity != "basic" || capability.Provider != "native" {
		t.Fatalf("native capability = %#v, err %v", capability, err)
	}

	service = newProposalSimulationService(db, proposalSimulationConfig{
		TenderlyAccount: "account", TenderlyProject: "project", TenderlyKey: "secret", TenderlyChains: map[int]struct{}{1: {}},
	})
	capability, err = service.Capability(context.Background(), "demo")
	if err != nil || !capability.Enabled || capability.Fidelity != "rich" || capability.Provider != "tenderly" {
		t.Fatalf("rich capability = %#v, err %v", capability, err)
	}
}

func TestProposalSimulationRunsNativeBeforeTenderlyAtSameBlockAndCaches(t *testing.T) {
	var mu sync.Mutex
	rpcCalls := map[string]int{}
	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var payload struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode RPC request: %v", err)
			return
		}
		mu.Lock()
		rpcCalls[payload.Method]++
		mu.Unlock()
		result := "0x"
		if payload.Method == "eth_blockNumber" {
			result = "0x123"
		}
		if payload.Method == "eth_call" {
			var transaction map[string]any
			if err := json.Unmarshal(payload.Params[0], &transaction); err != nil {
				t.Errorf("decode eth_call transaction: %v", err)
			}
			if transaction["to"] != "0x0000000000000000000000000000000000000003" || transaction["value"] != "0x0" || transaction["input"] != testSimulationExecuteData {
				t.Errorf("eth_call transaction = %#v", transaction)
			}
			var block string
			_ = json.Unmarshal(payload.Params[1], &block)
			if block != "0x123" {
				t.Errorf("eth_call block = %q, want 0x123", block)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, payload.ID, result)
	}))
	defer rpcServer.Close()

	tenderlyCalls := 0
	tenderlyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		tenderlyCalls++
		if request.Header.Get("X-Access-Key") != "secret" {
			t.Errorf("Tenderly access key not sent")
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode Tenderly request: %v", err)
		}
		if payload["block_number"] != float64(0x123) || payload["value"] != "0" || payload["input"] != testSimulationExecuteData {
			t.Errorf("Tenderly payload = %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transaction":{"status":true,"gas_used":123,"call_trace":{"calls":[]},"transaction_info":{"logs":[],"asset_changes":[],"state_diff":[]}},"simulation":{"id":"sim-1"}}`))
	}))
	defer tenderlyServer.Close()

	db := newTestProposalSimulationDB(t, rpcServer.URL, `["proposal-simulation"]`)
	service := newProposalSimulationService(db, proposalSimulationConfig{
		NativeFallback: true, TenderlyAccount: "account", TenderlyProject: "project", TenderlyKey: "secret",
		TenderlyChains: map[int]struct{}{1: {}}, TenderlyURL: tenderlyServer.URL + "/account/%s/project/%s/simulate",
	})

	for range 2 {
		result, err := service.Simulate(context.Background(), "demo", testSimulationProposalID, testSimulationRequest())
		if err != nil {
			t.Fatalf("Simulate: %v", err)
		}
		if result.Status != "success" || result.Fidelity != "rich" || result.Provider != "tenderly" || result.BlockNumber != "291" || result.GasUsed != "123" || result.ProviderReference != "sim-1" {
			t.Fatalf("simulation result = %#v", result)
		}
	}
	if rpcCalls["eth_call"] != 1 || tenderlyCalls != 1 {
		t.Fatalf("cached calls: eth_call=%d tenderly=%d, want 1 each", rpcCalls["eth_call"], tenderlyCalls)
	}
}

func TestProposalSimulationReturnsNativeRevertWithoutTenderly(t *testing.T) {
	tenderlyCalls := 0
	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var payload struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		_ = json.NewDecoder(request.Body).Decode(&payload)
		w.Header().Set("Content-Type", "application/json")
		if payload.Method == "eth_blockNumber" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x123"}`, payload.ID)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":3,"message":"execution reverted: denied","data":{"data":"0x08c379a0"}}}`, payload.ID)
	}))
	defer rpcServer.Close()
	tenderlyServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { tenderlyCalls++ }))
	defer tenderlyServer.Close()

	db := newTestProposalSimulationDB(t, rpcServer.URL, `["proposal-simulation"]`)
	service := newProposalSimulationService(db, proposalSimulationConfig{
		NativeFallback: true, TenderlyAccount: "account", TenderlyProject: "project", TenderlyKey: "secret",
		TenderlyChains: map[int]struct{}{1: {}}, TenderlyURL: tenderlyServer.URL + "/%s/%s",
	})
	result, err := service.Simulate(context.Background(), "demo", testSimulationProposalID, testSimulationRequest())
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}
	if result.Status != "reverted" || result.Fidelity != "basic" || result.Revert == nil || tenderlyCalls != 0 {
		t.Fatalf("revert result = %#v, tenderly calls = %d", result, tenderlyCalls)
	}
}

func TestProposalMismatchDoesNotCallRPC(t *testing.T) {
	rpcCalls := 0
	rpcServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { rpcCalls++ }))
	defer rpcServer.Close()
	db := newTestProposalSimulationDB(t, rpcServer.URL, `["proposal-simulation"]`)
	service := newProposalSimulationService(db, proposalSimulationConfig{NativeFallback: true})

	_, err := service.Simulate(context.Background(), "demo", "1", testSimulationRequest())
	var simulationErr *ProposalSimulationError
	if !errors.As(err, &simulationErr) || simulationErr.Code != "proposal_mismatch" || rpcCalls != 0 {
		t.Fatalf("mismatch err = %v, RPC calls = %d", err, rpcCalls)
	}
}
