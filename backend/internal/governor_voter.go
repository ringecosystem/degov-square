package internal

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ringecosystem/degov-square/internal/config"
)

// GovernorVoter handles governor contract voting interactions
type GovernorVoter struct {
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	chainID    *big.Int
}

// Governor contract ABI for castVoteWithReason
const castVoteWithReasonABI = `[{
	"inputs": [
		{"internalType": "uint256", "name": "proposalId", "type": "uint256"},
		{"internalType": "uint8", "name": "support", "type": "uint8"},
		{"internalType": "string", "name": "reason", "type": "string"}
	],
	"name": "castVoteWithReason",
	"outputs": [{"internalType": "uint256", "name": "", "type": "uint256"}],
	"stateMutability": "nonpayable",
	"type": "function"
}]`

// NewGovernorVoter creates a new GovernorVoter client
func NewGovernorVoter(rpcURL string, chainID int) (*GovernorVoter, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}

	// Load private key from config
	cfg := config.GetConfig()
	privateKeyHex := cfg.GetString("DEGOV_AGENT_PRIVATE_KEY")
	if privateKeyHex == "" {
		client.Close()
		return nil, fmt.Errorf("DEGOV_AGENT_PRIVATE_KEY is not set")
	}

	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &GovernorVoter{
		client:     client,
		privateKey: privateKey,
		chainID:    big.NewInt(int64(chainID)),
	}, nil
}

// Close closes the client connection
func (g *GovernorVoter) Close() {
	g.client.Close()
}

// GetVoterAddress returns the address of the voter account
func (g *GovernorVoter) GetVoterAddress() common.Address {
	publicKey := g.privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	return crypto.PubkeyToAddress(*publicKeyECDSA)
}

// CastVoteWithReason casts a vote on a proposal with a reason
func (g *GovernorVoter) CastVoteWithReason(ctx context.Context, contractAddress string, proposalID string, support int, reason string) (string, error) {
	// Parse the ABI
	contractABI, err := abi.JSON(strings.NewReader(castVoteWithReasonABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse cast vote ABI: %w", err)
	}

	// Convert proposal ID to big.Int (hex format)
	cleanProposalID := proposalID
	if len(proposalID) >= 2 && (proposalID[:2] == "0x" || proposalID[:2] == "0X") {
		cleanProposalID = proposalID[2:]
	}

	proposalBigInt, ok := new(big.Int).SetString(cleanProposalID, 16)
	if !ok {
		return "", fmt.Errorf("invalid hex proposal ID: %s", proposalID)
	}

	// Pack the function call data
	callData, err := contractABI.Pack("castVoteWithReason", proposalBigInt, uint8(support), reason)
	if err != nil {
		return "", fmt.Errorf("failed to pack function call data: %w", err)
	}

	// Get voter address
	voterAddress := g.GetVoterAddress()

	// Get nonce
	nonce, err := g.client.PendingNonceAt(ctx, voterAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	// Estimate gas
	contractAddr := common.HexToAddress(contractAddress)
	gasLimit, err := g.client.EstimateGas(ctx, ethereum.CallMsg{
		From: voterAddress,
		To:   &contractAddr,
		Data: callData,
	})
	if err != nil {
		return "", fmt.Errorf("failed to estimate gas: %w", err)
	}

	// Add 20% buffer to gas limit
	gasLimit = gasLimit * 120 / 100

	// Get gas price
	gasPrice, err := g.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create transaction
	tx := types.NewTransaction(nonce, contractAddr, big.NewInt(0), gasLimit, gasPrice, callData)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(g.chainID), g.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = g.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction receipt
	receipt, err := bind.WaitMined(ctx, g.client, signedTx)
	if err != nil {
		return signedTx.Hash().Hex(), fmt.Errorf("transaction sent but failed to wait for receipt: %w", err)
	}

	if receipt.Status == types.ReceiptStatusFailed {
		return signedTx.Hash().Hex(), fmt.Errorf("transaction failed: status %d", receipt.Status)
	}

	return signedTx.Hash().Hex(), nil
}
