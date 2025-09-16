package internal

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
)

// GovernorContract handles governor contract interactions
type GovernorContract struct {
	client *ethclient.Client
}

// NewGovernorContract creates a new Governor contract client
func NewGovernorContract(rpcURL string) (*GovernorContract, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}

	return &GovernorContract{
		client: client,
	}, nil
}

// Close closes the client connection
func (g *GovernorContract) Close() {
	g.client.Close()
}

// Governor contract ABI for the state function
const governorStateABI = `[{
	"inputs": [{"internalType": "uint256", "name": "proposalId", "type": "uint256"}],
	"name": "state",
	"outputs": [{"internalType": "enum IGovernor.ProposalState", "name": "", "type": "uint8"}],
	"stateMutability": "view",
	"type": "function"
}]`

// GetProposalState queries the governor contract for proposal state
func (g *GovernorContract) GetProposalState(ctx context.Context, contractAddress, proposalID string) (dbmodels.ProposalState, error) {
	// Parse the ABI
	contractABI, err := abi.JSON(strings.NewReader(governorStateABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse governor contract ABI: %w", err)
	}

	// Convert proposal ID to big.Int (hex format only)
	// Remove 0x or 0X prefix if present
	cleanProposalID := proposalID
	if len(proposalID) >= 2 && (proposalID[:2] == "0x" || proposalID[:2] == "0X") {
		cleanProposalID = proposalID[2:]
	}

	// Parse as hex
	proposalBigInt, ok := new(big.Int).SetString(cleanProposalID, 16)
	if !ok {
		return "", fmt.Errorf("invalid hex proposal ID: %s", proposalID)
	}

	// Pack the function call data using ABI
	callData, err := contractABI.Pack("state", proposalBigInt)
	if err != nil {
		return "", fmt.Errorf("failed to pack function call data: %w", err)
	}

	// Create call message
	contractAddr := common.HexToAddress(contractAddress)
	callMsg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: callData,
	}

	// Call the contract
	result, err := g.client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return "", fmt.Errorf("failed to call contract: %w", err)
	}

	// Unpack the result using ABI
	var stateResult uint8
	err = contractABI.UnpackIntoInterface(&stateResult, "state", result)
	if err != nil {
		return "", fmt.Errorf("failed to unpack contract result: %w", err)
	}

	// Convert to ProposalState
	return convertToProposalState(uint64(stateResult)), nil
}

// convertToProposalState converts numeric state to ProposalState enum
func convertToProposalState(state uint64) dbmodels.ProposalState {
	switch state {
	case 0:
		return dbmodels.ProposalStatePending
	case 1:
		return dbmodels.ProposalStateActive
	case 2:
		return dbmodels.ProposalStateCanceled
	case 3:
		return dbmodels.ProposalStateDefeated
	case 4:
		return dbmodels.ProposalStateSucceeded
	case 5:
		return dbmodels.ProposalStateQueued
	case 6:
		return dbmodels.ProposalStateExpired
	case 7:
		return dbmodels.ProposalStateExecuted
	default:
		return dbmodels.ProposalStateUnknown // Default fallback
	}
}

// GetRPCURL tries to get RPC URL from config or fallback to default
func GetRPCURL(chainRPCs []string, chainID int) string {
	// Use the first RPC from config if available
	if len(chainRPCs) > 0 && strings.TrimSpace(chainRPCs[0]) != "" {
		return chainRPCs[0]
	}

	// Fallback to common public RPCs based on chain ID
	switch chainID {
	case 1:
		return "https://eth.llamarpc.com"
	case 46:
		return "https://rpc.darwinia.network"
	case 56:
		return "https://bsc-dataseed.binance.org"
	case 137:
		return "https://polygon-rpc.com"
	case 42161:
		return "https://arb1.arbitrum.io/rpc"
	case 10:
		return "https://mainnet.optimism.io"
	case 43114:
		return "https://api.avax.network/ext/bc/C/rpc"
	default:
		return "" // No fallback available
	}
}
