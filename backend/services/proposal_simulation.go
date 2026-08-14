package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/patrickmn/go-cache"
	"github.com/ringecosystem/degov-square/database"
	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/types"
	"gorm.io/gorm"
)

const (
	proposalSimulationFeature = "proposal-simulation"
	maxSimulationActions      = 50
	maxSimulationCalldata     = 256 << 10
	tenderlySimulationURL     = "https://api.tenderly.co/api/v1/account/%s/project/%s/simulate"
	xAccountWarning           = "Source-chain dispatch simulated; destination-chain execution was not simulated."
	snapshotWarning           = "Simulation uses a pinned snapshot and cannot guarantee later execution."
)

const governorSimulationABIJSON = `[
  {"inputs":[{"type":"address[]","name":"targets"},{"type":"uint256[]","name":"values"},{"type":"bytes[]","name":"calldatas"},{"type":"bytes32","name":"descriptionHash"}],"name":"execute","outputs":[{"type":"uint256"}],"stateMutability":"payable","type":"function"},
  {"inputs":[{"type":"address[]","name":"targets"},{"type":"uint256[]","name":"values"},{"type":"bytes[]","name":"calldatas"},{"type":"bytes32","name":"descriptionHash"}],"name":"hashProposal","outputs":[{"type":"uint256"}],"stateMutability":"pure","type":"function"}
]`

var governorSimulationABI = func() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(governorSimulationABIJSON))
	if err != nil {
		panic(err)
	}
	return parsed
}()

var xAccountSelector = crypto.Keccak256([]byte("send(uint256,address,bytes,bytes)"))[:4]

type ProposalSimulationRequest struct {
	Caller          string   `json:"caller"`
	Targets         []string `json:"targets"`
	Values          []string `json:"values"`
	Calldatas       []string `json:"calldatas"`
	DescriptionHash string   `json:"descriptionHash"`
}

type ProposalSimulationCapability struct {
	Enabled  bool     `json:"enabled"`
	Reason   string   `json:"reason,omitempty"`
	Modes    []string `json:"modes,omitempty"`
	Fidelity string   `json:"fidelity,omitempty"`
	Provider string   `json:"provider,omitempty"`
	ChainID  int      `json:"chainId,omitempty"`
}

type ProposalSimulationRevert struct {
	Reason string `json:"reason,omitempty"`
	Data   string `json:"data,omitempty"`
}

type ProposalSimulationResult struct {
	Status            string                    `json:"status"`
	Fidelity          string                    `json:"fidelity"`
	Provider          string                    `json:"provider"`
	ChainID           int                       `json:"chainId"`
	BlockNumber       string                    `json:"blockNumber"`
	SimulatedAt       time.Time                 `json:"simulatedAt"`
	Caller            string                    `json:"caller"`
	Governor          string                    `json:"governor"`
	GasUsed           string                    `json:"gasUsed,omitempty"`
	Revert            *ProposalSimulationRevert `json:"revert,omitempty"`
	Calls             json.RawMessage           `json:"calls,omitempty"`
	Logs              json.RawMessage           `json:"logs,omitempty"`
	AssetChanges      json.RawMessage           `json:"assetChanges,omitempty"`
	StateChanges      json.RawMessage           `json:"stateChanges,omitempty"`
	Warnings          []string                  `json:"warnings"`
	ProviderReference string                    `json:"providerReference,omitempty"`
}

type ProposalSimulationError struct {
	Code string
	Err  error
}

func (e *ProposalSimulationError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *ProposalSimulationError) Unwrap() error { return e.Err }

type proposalSimulationConfig struct {
	NativeFallback  bool
	TenderlyAccount string
	TenderlyProject string
	TenderlyKey     string
	TenderlyChains  map[int]struct{}
	TenderlyURL     string
}

type ProposalSimulationService struct {
	daoService       *DaoService
	daoConfigService *DaoConfigService
	config           proposalSimulationConfig
	httpClient       *http.Client
	cache            *cache.Cache
	now              func() time.Time
}

type proposalSimulationDAO struct {
	ChainID  int
	RPCURL   string
	Governor common.Address
}

type validatedSimulationRequest struct {
	Caller          common.Address
	Targets         []common.Address
	Values          []*big.Int
	Calldatas       [][]byte
	DescriptionHash [32]byte
	ExecuteData     []byte
	HasXAccount     bool
}

func NewProposalSimulationService() *ProposalSimulationService {
	cfg := config.GetConfig()
	return newProposalSimulationService(database.GetDB(), proposalSimulationConfig{
		NativeFallback:  cfg.GetBool("SIMULATION_NATIVE_RPC_FALLBACK"),
		TenderlyAccount: strings.TrimSpace(cfg.GetString("SIMULATION_TENDERLY_ACCOUNT")),
		TenderlyProject: strings.TrimSpace(cfg.GetString("SIMULATION_TENDERLY_PROJECT")),
		TenderlyKey:     strings.TrimSpace(cfg.GetString("SIMULATION_TENDERLY_ACCESS_KEY")),
		TenderlyChains:  parseSimulationChainIDs(cfg.GetString("SIMULATION_TENDERLY_CHAIN_IDS")),
		TenderlyURL:     tenderlySimulationURL,
	})
}

func newProposalSimulationService(db *gorm.DB, cfg proposalSimulationConfig) *ProposalSimulationService {
	return &ProposalSimulationService{
		daoService:       &DaoService{db: db},
		daoConfigService: &DaoConfigService{db: db},
		config:           cfg,
		httpClient:       &http.Client{},
		cache:            cache.New(15*time.Second, 30*time.Second),
		now:              time.Now,
	}
}

func parseSimulationChainIDs(value string) map[int]struct{} {
	chainIDs := make(map[int]struct{})
	for _, raw := range strings.Split(value, ",") {
		chainID, err := strconv.Atoi(strings.TrimSpace(raw))
		if err == nil && chainID > 0 {
			chainIDs[chainID] = struct{}{}
		}
	}
	return chainIDs
}

func (s *ProposalSimulationService) Capability(ctx context.Context, daoCode string) (ProposalSimulationCapability, error) {
	dao, reason, err := s.resolveDAO(ctx, daoCode)
	if err != nil {
		return ProposalSimulationCapability{}, err
	}
	if reason != "" {
		return ProposalSimulationCapability{Enabled: false, Reason: reason}, nil
	}

	capability := ProposalSimulationCapability{Enabled: true, Modes: []string{"execute"}, ChainID: dao.ChainID}
	if s.tenderlySupports(dao.ChainID) {
		capability.Fidelity = "rich"
		capability.Provider = "tenderly"
	} else {
		capability.Fidelity = "basic"
		capability.Provider = "native"
	}
	return capability, nil
}

func (s *ProposalSimulationService) Simulate(ctx context.Context, daoCode, proposalID string, request ProposalSimulationRequest) (*ProposalSimulationResult, error) {
	dao, reason, err := s.resolveDAO(ctx, daoCode)
	if err != nil {
		return nil, err
	}
	if reason != "" {
		return nil, simulationError("simulation_disabled", errors.New(reason))
	}

	validated, err := validateSimulationRequest(proposalID, request)
	if err != nil {
		return nil, err
	}

	client, err := ethclient.DialContext(ctx, dao.RPCURL)
	if err != nil {
		return nil, simulationUpstreamError(ctx, err)
	}
	defer client.Close()

	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		return nil, simulationUpstreamError(ctx, err)
	}
	cacheKey := simulationCacheKey(daoCode, proposalID, validated, blockNumber)
	if cached, found := s.cache.Get(cacheKey); found {
		if result, ok := cached.(*ProposalSimulationResult); ok {
			return result, nil
		}
	}

	result := &ProposalSimulationResult{
		Status:      "success",
		Fidelity:    "basic",
		Provider:    "native",
		ChainID:     dao.ChainID,
		BlockNumber: strconv.FormatUint(blockNumber, 10),
		SimulatedAt: s.now().UTC(),
		Caller:      validated.Caller.Hex(),
		Governor:    dao.Governor.Hex(),
		Warnings:    []string{snapshotWarning},
	}
	if validated.HasXAccount {
		result.Warnings = append(result.Warnings, xAccountWarning)
	}

	call := ethereum.CallMsg{From: validated.Caller, To: &dao.Governor, Value: new(big.Int), Data: validated.ExecuteData}
	if _, err := client.CallContract(ctx, call, new(big.Int).SetUint64(blockNumber)); err != nil {
		revert, reverted := decodeSimulationRevert(err)
		if !reverted {
			return nil, simulationUpstreamError(ctx, err)
		}
		result.Status = "reverted"
		result.Revert = revert
		result.Warnings = append(result.Warnings, "Native RPC simulation does not include traces or state changes.")
		s.cache.SetDefault(cacheKey, result)
		return result, nil
	}

	if s.tenderlySupports(dao.ChainID) {
		rich, err := s.simulateTenderly(ctx, dao, validated, blockNumber, result)
		if err == nil {
			s.cache.SetDefault(cacheKey, rich)
			return rich, nil
		}
		if !s.config.NativeFallback {
			return nil, err
		}
		result.Warnings = append(result.Warnings,
			"Rich simulation unavailable; returned native RPC result.",
			"Native RPC simulation does not include traces or state changes.",
		)
	} else {
		result.Warnings = append(result.Warnings, "Native RPC simulation does not include traces or state changes.")
	}

	s.cache.SetDefault(cacheKey, result)
	return result, nil
}

func (s *ProposalSimulationService) resolveDAO(_ context.Context, daoCode string) (*proposalSimulationDAO, string, error) {
	daoCode = strings.TrimSpace(daoCode)
	enabled, err := s.daoService.HasFeature(daoCode, proposalSimulationFeature)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "dao_not_found", nil
	}
	if err != nil {
		return nil, "", err
	}
	if !enabled {
		return nil, "dao_feature_disabled", nil
	}

	daoConfig, err := s.daoConfigService.StandardConfig(daoCode)
	if err != nil {
		return nil, "dao_config_unavailable", nil
	}
	rpcURL := firstSimulationRPC(daoConfig)
	if daoConfig.Chain.ID <= 0 || !common.IsHexAddress(daoConfig.Contracts.Governor) || rpcURL == "" {
		return nil, "dao_config_incomplete", nil
	}
	if !s.tenderlySupports(daoConfig.Chain.ID) && !s.config.NativeFallback {
		return nil, "provider_unavailable", nil
	}

	return &proposalSimulationDAO{
		ChainID:  daoConfig.Chain.ID,
		RPCURL:   rpcURL,
		Governor: common.HexToAddress(daoConfig.Contracts.Governor),
	}, "", nil
}

func firstSimulationRPC(daoConfig *types.DaoConfig) string {
	for _, candidate := range daoConfig.Chain.RPCs {
		parsed, err := url.Parse(strings.TrimSpace(candidate))
		if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" {
			return parsed.String()
		}
	}
	return ""
}

func (s *ProposalSimulationService) tenderlySupports(chainID int) bool {
	if s.config.TenderlyAccount == "" || s.config.TenderlyProject == "" || s.config.TenderlyKey == "" {
		return false
	}
	_, ok := s.config.TenderlyChains[chainID]
	return ok
}

func validateSimulationRequest(proposalID string, request ProposalSimulationRequest) (*validatedSimulationRequest, error) {
	if !common.IsHexAddress(request.Caller) {
		return nil, simulationError("invalid_request", errors.New("caller must be an EVM address"))
	}
	if len(request.Targets) == 0 || len(request.Targets) > maxSimulationActions {
		return nil, simulationError("invalid_request", fmt.Errorf("targets must contain 1-%d actions", maxSimulationActions))
	}
	if len(request.Targets) != len(request.Values) || len(request.Targets) != len(request.Calldatas) {
		return nil, simulationError("invalid_request", errors.New("targets, values, and calldatas must have equal lengths"))
	}

	validated := &validatedSimulationRequest{
		Caller:    common.HexToAddress(request.Caller),
		Targets:   make([]common.Address, len(request.Targets)),
		Values:    make([]*big.Int, len(request.Values)),
		Calldatas: make([][]byte, len(request.Calldatas)),
	}
	totalCalldata := 0
	for index := range request.Targets {
		if !common.IsHexAddress(request.Targets[index]) {
			return nil, simulationError("invalid_request", fmt.Errorf("targets[%d] must be an EVM address", index))
		}
		validated.Targets[index] = common.HexToAddress(request.Targets[index])

		value, ok := parseUint256(request.Values[index])
		if !ok {
			return nil, simulationError("invalid_request", fmt.Errorf("values[%d] must be an unsigned 256-bit integer", index))
		}
		validated.Values[index] = value

		calldata, err := hexutil.Decode(request.Calldatas[index])
		if err != nil {
			return nil, simulationError("invalid_request", fmt.Errorf("calldatas[%d] must be 0x-prefixed bytes", index))
		}
		totalCalldata += len(calldata)
		if totalCalldata > maxSimulationCalldata {
			return nil, simulationError("invalid_request", errors.New("calldata exceeds size limit"))
		}
		validated.Calldatas[index] = calldata
		validated.HasXAccount = validated.HasXAccount || bytes.HasPrefix(calldata, xAccountSelector)
	}

	descriptionHash, err := hexutil.Decode(request.DescriptionHash)
	if err != nil || len(descriptionHash) != common.HashLength {
		return nil, simulationError("invalid_request", errors.New("descriptionHash must be 32 bytes"))
	}
	copy(validated.DescriptionHash[:], descriptionHash)

	proposalArguments, err := governorSimulationABI.Methods["hashProposal"].Inputs.Pack(
		validated.Targets, validated.Values, validated.Calldatas, validated.DescriptionHash,
	)
	if err != nil {
		return nil, simulationError("invalid_request", err)
	}
	computedID := new(big.Int).SetBytes(crypto.Keccak256(proposalArguments))
	wantedID, ok := parseProposalID(proposalID)
	if !ok {
		return nil, simulationError("invalid_request", errors.New("proposalId must be an unsigned 256-bit integer"))
	}
	if computedID.Cmp(wantedID) != 0 {
		return nil, simulationError("proposal_mismatch", errors.New("payload does not match proposalId"))
	}

	validated.ExecuteData, err = governorSimulationABI.Pack(
		"execute", validated.Targets, validated.Values, validated.Calldatas, validated.DescriptionHash,
	)
	if err != nil {
		return nil, simulationError("invalid_request", err)
	}
	return validated, nil
}

func parseUint256(value string) (*big.Int, bool) {
	value = strings.TrimSpace(value)
	base := 10
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		base = 16
		value = value[2:]
		if len(value) == 0 || len(value) > 64 {
			return nil, false
		}
	} else if len(value) == 0 || len(value) > 78 {
		return nil, false
	}
	parsed, ok := new(big.Int).SetString(value, base)
	return parsed, ok && parsed.Sign() >= 0 && parsed.BitLen() <= 256
}

func parseProposalID(value string) (*big.Int, bool) {
	return parseUint256(value)
}

func simulationCacheKey(daoCode, proposalID string, request *validatedSimulationRequest, blockNumber uint64) string {
	payloadHash := sha256.Sum256(request.ExecuteData)
	return strings.Join([]string{
		daoCode,
		proposalID,
		strings.ToLower(request.Caller.Hex()),
		strconv.FormatUint(blockNumber, 10),
		hex.EncodeToString(payloadHash[:]),
	}, ":")
}

func decodeSimulationRevert(err error) (*ProposalSimulationRevert, bool) {
	revert := &ProposalSimulationRevert{}
	var dataError rpc.DataError
	if errors.As(err, &dataError) {
		if data := simulationRevertData(dataError.ErrorData()); data != "" {
			revert.Data = data
			if decoded, decodeErr := hexutil.Decode(data); decodeErr == nil {
				if reason, reasonErr := abi.UnpackRevert(decoded); reasonErr == nil {
					revert.Reason = reason
				}
			}
		}
	}
	lowerMessage := strings.ToLower(err.Error())
	if revert.Data == "" && !strings.Contains(lowerMessage, "revert") {
		return nil, false
	}
	if revert.Reason == "" {
		revert.Reason = strings.TrimSpace(strings.TrimPrefix(err.Error(), "execution reverted:"))
		if len(revert.Reason) > 500 {
			revert.Reason = revert.Reason[:500]
		}
	}
	return revert, true
}

func simulationRevertData(value any) string {
	if data, ok := value.(string); ok && strings.HasPrefix(data, "0x") {
		return data
	}
	if object, ok := value.(map[string]any); ok {
		for _, key := range []string{"data", "result"} {
			if data := simulationRevertData(object[key]); data != "" {
				return data
			}
		}
	}
	return ""
}

func (s *ProposalSimulationService) simulateTenderly(
	ctx context.Context,
	dao *proposalSimulationDAO,
	request *validatedSimulationRequest,
	blockNumber uint64,
	basic *ProposalSimulationResult,
) (*ProposalSimulationResult, error) {
	body, err := json.Marshal(map[string]any{
		"network_id":      strconv.Itoa(dao.ChainID),
		"from":            request.Caller.Hex(),
		"to":              dao.Governor.Hex(),
		"input":           hexutil.Encode(request.ExecuteData),
		"value":           "0",
		"gas":             30_000_000,
		"block_number":    blockNumber,
		"save":            false,
		"save_if_fails":   false,
		"simulation_type": "full",
	})
	if err != nil {
		return nil, simulationError("provider_unavailable", err)
	}

	endpoint := fmt.Sprintf(s.config.TenderlyURL, url.PathEscape(s.config.TenderlyAccount), url.PathEscape(s.config.TenderlyProject))
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, simulationError("provider_unavailable", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-Access-Key", s.config.TenderlyKey)

	response, err := s.httpClient.Do(httpRequest)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, simulationError("provider_timeout", err)
		}
		return nil, simulationError("provider_unavailable", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, simulationError("provider_rate_limited", errors.New("tenderly rate limited"))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, simulationError("provider_unavailable", fmt.Errorf("tenderly status %d", response.StatusCode))
	}

	var payload struct {
		Transaction *struct {
			Status    *bool           `json:"status"`
			GasUsed   json.RawMessage `json:"gas_used"`
			ErrorInfo *struct {
				Message string `json:"error_message"`
			} `json:"error_info"`
			CallTrace *struct {
				Calls json.RawMessage `json:"calls"`
			} `json:"call_trace"`
			TransactionInfo *struct {
				Logs         json.RawMessage `json:"logs"`
				AssetChanges json.RawMessage `json:"asset_changes"`
				StateChanges json.RawMessage `json:"state_diff"`
			} `json:"transaction_info"`
		} `json:"transaction"`
		Simulation *struct {
			ID string `json:"id"`
		} `json:"simulation"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil || payload.Transaction == nil || payload.Transaction.Status == nil {
		return nil, simulationError("provider_unavailable", errors.New("invalid tenderly response"))
	}

	result := *basic
	result.Fidelity = "rich"
	result.Provider = "tenderly"
	result.GasUsed = rawJSONNumber(payload.Transaction.GasUsed)
	if payload.Transaction.CallTrace != nil {
		result.Calls = payload.Transaction.CallTrace.Calls
	}
	if payload.Transaction.TransactionInfo != nil {
		result.Logs = payload.Transaction.TransactionInfo.Logs
		result.AssetChanges = payload.Transaction.TransactionInfo.AssetChanges
		result.StateChanges = payload.Transaction.TransactionInfo.StateChanges
	}
	if payload.Simulation != nil {
		result.ProviderReference = payload.Simulation.ID
	}
	if !*payload.Transaction.Status {
		result.Status = "reverted"
		result.Revert = &ProposalSimulationRevert{}
		if payload.Transaction.ErrorInfo != nil {
			result.Revert.Reason = payload.Transaction.ErrorInfo.Message
		}
	}
	return &result, nil
}

func rawJSONNumber(value json.RawMessage) string {
	if len(value) == 0 || bytes.Equal(value, []byte("null")) {
		return ""
	}
	var text string
	if value[0] == '"' && json.Unmarshal(value, &text) == nil {
		return text
	}
	return string(value)
}

func simulationError(code string, err error) error {
	return &ProposalSimulationError{Code: code, Err: err}
}

func simulationUpstreamError(ctx context.Context, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return simulationError("provider_timeout", err)
	}
	return simulationError("rpc_unavailable", err)
}
