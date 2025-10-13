package internal

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const OKX_API_ENDPOINT = "https://www.okx.com"

type OkxPeriod string

const (
	OkxPeriod1m  OkxPeriod = "1m"
	OkxPeriod5m  OkxPeriod = "5m"
	OkxPeriod30m OkxPeriod = "30m"
	OkxPeriod1h  OkxPeriod = "1h"
	OkxPeriod1d  OkxPeriod = "1d"
)

// OkxBalanceOptions represents options for balance query
type OkxBalanceOptions struct {
	Chains  []string `json:"chains"`
	Address string   `json:"address"`
}

// OkxPriceOptions represents options for price query
type OkxPriceOptions struct {
	Chain   string `json:"chain"`
	Address string `json:"address"`
}

// PriceOutput represents price output
type PriceOutput struct {
	ChainID         string `json:"chainId"`
	TokenAddress    string `json:"tokenAddress"`
	Price           string `json:"price"`
	Time            string `json:"time"`
	DisplayDecimals int    `json:"displayDecimals"`
}

// OkxSignOptions represents signing options
type OkxSignOptions struct {
	API    string      `json:"api"`
	Method string      `json:"method"`
	Body   interface{} `json:"body,omitempty"`
}

type OkxHistoricalPriceOptions struct {
	Chain   string    `json:"chain"`
	Address string    `json:"address"`
	Limit   int       `json:"limit"`
	Cursor  *string   `json:"cursor,omitempty"`
	Begin   *int64    `json:"begin"`
	End     *int64    `json:"end"`
	Period  OkxPeriod `json:"period"`
}

// OkxResp represents OKX API response
type OkxResp[T any] struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

// OkxBalance represents balance response
type OkxBalance struct {
	TokenAssets []OkxTokenAssets `json:"tokenAssets"`
}

// OkxPrice represents price response
type OkxPrice struct {
	ChainIndex   string `json:"chainIndex"`
	TokenAddress string `json:"tokenAddress"`
	Time         string `json:"time"`
	Price        string `json:"price"`
}

// OkxTransactionsByAddress represents transaction history response
type OkxTransactionsByAddress struct {
	Cursor          string           `json:"cursor"`
	TransactionList []OkxTransaction `json:"transactionList"`
}

// OkxTransaction represents a single transaction
type OkxTransaction struct {
	ChainIndex   string                 `json:"chainIndex"`
	TxHash       string                 `json:"txHash"`
	MethodID     string                 `json:"methodId"`
	Nonce        string                 `json:"nonce"`
	TxTime       string                 `json:"txTime"`
	TokenAddress string                 `json:"tokenAddress"`
	Amount       string                 `json:"amount"`
	Symbol       string                 `json:"symbol"`
	TxFee        string                 `json:"txFee"`
	TxStatus     string                 `json:"txStatus"`
	HitBlacklist bool                   `json:"hitBlacklist"`
	Tag          string                 `json:"tag"`
	IType        string                 `json:"itype"`
	From         []OkxTransactionAmount `json:"from"`
	To           []OkxTransactionAmount `json:"to"`
}

// OkxTransactionAmount represents transaction amount
type OkxTransactionAmount struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
}

// OkxTokenAssets represents token assets
type OkxTokenAssets struct {
	ChainIndex      string `json:"chainIndex"`
	TokenAddress    string `json:"tokenAddress"`
	Symbol          string `json:"symbol"`
	Balance         string `json:"balance"`
	TokenPrice      string `json:"tokenPrice"`
	TokenType       string `json:"tokenType"`
	IsRiskToken     bool   `json:"isRiskToken"`
	TransferAmount  string `json:"transferAmount"`
	AvailableAmount string `json:"availableAmount"`
	RawBalance      string `json:"rawBalance"`
	Address         string `json:"address"`
}

// OkxHistoryOptions represents options for history query
type OkxHistoryOptions struct {
	Address      string   `json:"address"`
	Chains       []string `json:"chains"`
	TokenAddress string   `json:"tokenAddress,omitempty"`
	Begin        int64    `json:"begin,omitempty"`
	End          int64    `json:"end,omitempty"`
	Cursor       string   `json:"cursor,omitempty"`
	Limit        int      `json:"limit,omitempty"`
}

// WalletTokenBalance represents wallet token balance
type WalletTokenBalance struct {
	Platforms []WalletTokenPlatform `json:"platforms"`
	ID        string                `json:"id"`
	Symbol    string                `json:"symbol"`
	Name      string                `json:"name"`
	LogoURI   string                `json:"logoURI"`
}

// WalletTokenPlatform represents wallet token platform
type WalletTokenPlatform struct {
	Address         string `json:"address"`
	ID              int    `json:"id"`
	Name            string `json:"name"`
	LogoURI         string `json:"logoURI"`
	Decimals        int    `json:"decimals"`
	Native          bool   `json:"native"`
	Price           string `json:"price"`
	Balance         string `json:"balance"`
	BalanceRaw      string `json:"balanceRaw"`
	BalanceUSD      string `json:"balanceUSD"`
	DisplayDecimals int    `json:"displayDecimals"`
}

// WalletHistory represents wallet history
type WalletHistory struct {
	Cursor       string              `json:"cursor"`
	Transactions []WalletTransaction `json:"transactions"`
}

// WalletTransaction represents wallet transaction
type WalletTransaction struct {
	Chain        interface{}            `json:"chain"`
	TxHash       string                 `json:"txHash"`
	MethodID     string                 `json:"methodId"`
	Nonce        string                 `json:"nonce"`
	TxTime       string                 `json:"txTime"`
	From         []OkxTransactionAmount `json:"from"`
	To           []OkxTransactionAmount `json:"to"`
	TokenAddress string                 `json:"tokenAddress"`
	Amount       string                 `json:"amount"`
	Symbol       string                 `json:"symbol"`
	TxFee        string                 `json:"txFee"`
	TxStatus     string                 `json:"txStatus"`
	HitBlacklist bool                   `json:"hitBlacklist"`
	IType        string                 `json:"itype"`
}

type WalletHistoricalPrice struct {
	Cursor string                `json:"cursor"`
	Prices []WalletTimeWithPrice `json:"prices"`
}

type WalletTimeWithPrice struct {
	Time  string `json:"time"`
	Price string `json:"price"`
}

// OkxAPI represents OKX API client
type OkxAPI struct {
	OKXProject    string
	OKXAccessKey  string
	OKXSecretKey  string
	OKXPassphrase string
}

type OkxOptions struct {
	Project    string
	AccessKey  string
	SecretKey  string
	Passphrase string
}

// NewOkxAPI creates a new OKX API client
func NewOkxAPI(options OkxOptions) *OkxAPI {
	return &OkxAPI{
		OKXProject:    options.Project,
		OKXAccessKey:  options.AccessKey,
		OKXSecretKey:  options.SecretKey,
		OKXPassphrase: options.Passphrase,
	}
}

// generateSignature generates OKX API signature
func (api *OkxAPI) generateSignature(options OkxSignOptions) map[string]string {
	now := time.Now()
	isoString := now.UTC().Format("2006-01-02T15:04:05.000Z")
	method := strings.ToUpper(options.Method)

	var body string
	if options.Body != nil {
		bodyBytes, _ := json.Marshal(options.Body)
		body = string(bodyBytes)
	}

	seed := isoString + method + options.API + body

	h := hmac.New(sha256.New, []byte(api.OKXSecretKey))
	h.Write([]byte(seed))
	signatureBase64 := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return map[string]string{
		"OK-ACCESS-PROJECT":    api.OKXProject,
		"OK-ACCESS-KEY":        api.OKXAccessKey,
		"OK-ACCESS-PASSPHRASE": api.OKXPassphrase,
		"OK-ACCESS-TIMESTAMP":  isoString,
		"OK-ACCESS-SIGN":       signatureBase64,
		"Content-Type":         "application/json",
	}
}

// calculateDisplayDecimals calculates display decimals for a number
func calculateDisplayDecimals(num string) int {
	// Simple implementation - in real scenario you might want more sophisticated logic
	if strings.Contains(num, ".") {
		parts := strings.Split(num, ".")
		if len(parts) > 1 {
			return len(parts[1])
		}
	}
	return 0
}

// calculateTokenDecimals calculates token decimals from balance and balanceRaw
func calculateTokenDecimals(balance, balanceRaw string) int {
	// Parse balance as big.Float for precision
	balanceFloat, _, err := big.ParseFloat(balance, 10, 256, big.ToNearestEven)
	if err != nil || balanceFloat.Cmp(big.NewFloat(0)) == 0 {
		return 18 // Default to 18 decimals if parsing fails or balance is 0
	}

	// Parse balanceRaw as big.Int
	balanceRawInt, ok := new(big.Int).SetString(balanceRaw, 10)
	if !ok || balanceRawInt.Cmp(big.NewInt(0)) == 0 {
		return 18 // Default to 18 decimals if parsing fails or balanceRaw is 0
	}

	// Convert balanceRaw to big.Float for division
	balanceRawFloat := new(big.Float).SetInt(balanceRawInt)

	// Calculate the ratio: balanceRaw / balance = 10^decimals
	ratio := new(big.Float).Quo(balanceRawFloat, balanceFloat)

	// Convert ratio to float64 for log calculation
	ratioFloat64, _ := ratio.Float64()

	// Handle edge cases
	if ratioFloat64 <= 0 {
		return 18
	}

	// Calculate log10 to find decimals
	// decimals = log10(ratio)
	decimals := int(math.Round(math.Log10(ratioFloat64)))

	// Validate the result - should be between 0 and 30
	if decimals < 0 {
		return 0
	}
	if decimals > 30 {
		return 18 // Default to 18 if calculated decimals seems too high
	}

	return decimals
}

// Price gets single token price
func (api *OkxAPI) Price(options OkxPriceOptions) (*PriceOutput, error) {
	if options.Chain == "" || options.Address == "" {
		return nil, fmt.Errorf("chain and address are required")
	}

	prices, err := api.Prices([]OkxPriceOptions{options})
	if err != nil {
		return nil, err
	}

	if len(prices) > 0 {
		return &prices[0], nil
	}

	return nil, nil
}

// Prices gets multiple token prices
func (api *OkxAPI) Prices(options []OkxPriceOptions) ([]PriceOutput, error) {
	apiPath := "/api/v5/wallet/token/real-time-price"

	body := make([]map[string]string, len(options))
	for i, item := range options {
		body[i] = map[string]string{
			"chainIndex":   item.Chain,
			"tokenAddress": item.Address,
		}
	}

	headers := api.generateSignature(OkxSignOptions{
		API:    apiPath,
		Method: "POST",
		Body:   body,
	})

	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", OKX_API_ENDPOINT+apiPath, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var okxResp OkxResp[[]OkxPrice]
	if err := json.Unmarshal(responseBody, &okxResp); err != nil {
		return nil, err
	}

	outputs := make([]PriceOutput, len(okxResp.Data))
	for i, item := range okxResp.Data {
		outputs[i] = PriceOutput{
			ChainID:         item.ChainIndex,
			TokenAddress:    item.TokenAddress,
			Price:           item.Price,
			Time:            item.Time,
			DisplayDecimals: calculateDisplayDecimals(item.Price),
		}
	}

	return outputs, nil
}

// Balances gets wallet token balances
func (api *OkxAPI) Balances(options OkxBalanceOptions) ([]WalletTokenBalance, error) {
	if len(options.Chains) == 0 {
		return []WalletTokenBalance{}, nil
	}

	apiPath := fmt.Sprintf("/api/v5/wallet/asset/all-token-balances-by-address?address=%s&chains=%s",
		options.Address, strings.Join(options.Chains, ","))

	headers := api.generateSignature(OkxSignOptions{
		API:    apiPath,
		Method: "GET",
	})

	req, err := http.NewRequest("GET", OKX_API_ENDPOINT+apiPath, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var okxResp OkxResp[[]OkxBalance]
	if err := json.Unmarshal(responseBody, &okxResp); err != nil {
		return nil, err
	}

	if okxResp.Code != "0" || len(okxResp.Data) == 0 {
		return []WalletTokenBalance{}, nil
	}

	okxTokenAssets := okxResp.Data[0].TokenAssets
	walletTokens := []WalletTokenBalance{}

	for _, ota := range okxTokenAssets {
		// Note: In the original TypeScript code, there are calls to HelixboxToken.chain()
		// and HelixboxToken.find() which would need to be implemented in Go
		// For now, we'll create a simplified version

		balance, _ := strconv.ParseFloat(ota.Balance, 64)
		tokenPrice, _ := strconv.ParseFloat(ota.TokenPrice, 64)
		balanceUSD := balance * tokenPrice

		// Get logo URI for the token
		logoURI := api.getLogoURI(ota.ChainIndex, ota.TokenAddress)

		// Convert ChainIndex from string to int for ID field
		chainIDInt, _ := strconv.Atoi(ota.ChainIndex)

		// Calculate token decimals from balance and balanceRaw
		tokenDecimals := calculateTokenDecimals(ota.Balance, ota.RawBalance)

		// This is a simplified version - you'll need to implement the HelixboxToken logic
		walletToken := WalletTokenBalance{
			ID:      ota.TokenAddress, // Simplified - should use proper token ID
			Symbol:  ota.Symbol,
			Name:    ota.Symbol, // Simplified - should get proper name
			LogoURI: logoURI,    // Use generated logo URI from TrustWallet
			Platforms: []WalletTokenPlatform{
				{
					Address:         ota.TokenAddress,
					ID:              chainIDInt,
					LogoURI:         logoURI,       // Use generated logo URI from TrustWallet
					Decimals:        tokenDecimals, // Calculate decimals from balance and balanceRaw
					Native:          ota.TokenAddress == "0x0000000000000000000000000000000000000000",
					Price:           ota.TokenPrice,
					Balance:         ota.Balance,
					BalanceRaw:      ota.RawBalance,
					BalanceUSD:      fmt.Sprintf("%.2f", balanceUSD),
					DisplayDecimals: calculateDisplayDecimals(ota.TokenPrice),
				},
			},
		}
		walletTokens = append(walletTokens, walletToken)
	}

	return walletTokens, nil
}

// History gets wallet transaction history
func (api *OkxAPI) History(options OkxHistoryOptions) ([]WalletHistory, error) {
	apiPath := fmt.Sprintf("/api/v5/wallet/post-transaction/transactions-by-address?address=%s&chains=%s",
		options.Address, strings.Join(options.Chains, ","))

	if options.Begin > 0 {
		apiPath += fmt.Sprintf("&begin=%d", options.Begin)
	}
	if options.End > 0 {
		apiPath += fmt.Sprintf("&end=%d", options.End)
	}
	if options.Cursor != "" {
		apiPath += fmt.Sprintf("&cursor=%s", options.Cursor)
	}
	if options.TokenAddress != "" {
		apiPath += fmt.Sprintf("&tokenAddress=%s", options.TokenAddress)
	}
	if options.Limit > 0 {
		apiPath += fmt.Sprintf("&limit=%d", options.Limit)
	}

	headers := api.generateSignature(OkxSignOptions{
		API:    apiPath,
		Method: "GET",
	})

	req, err := http.NewRequest("GET", OKX_API_ENDPOINT+apiPath, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var okxResp OkxResp[[]OkxTransactionsByAddress]
	if err := json.Unmarshal(responseBody, &okxResp); err != nil {
		return nil, err
	}

	if len(okxResp.Data) == 0 {
		return []WalletHistory{}, nil
	}

	histories := make([]WalletHistory, len(okxResp.Data))
	for i, data := range okxResp.Data {
		transactions := make([]WalletTransaction, len(data.TransactionList))
		for j, tl := range data.TransactionList {
			// Note: In the original TypeScript code, there's a call to HelixboxChain.get()
			// which would need to be implemented in Go
			transaction := WalletTransaction{
				Chain:        nil, // Would need to implement HelixboxChain.get()
				TxHash:       tl.TxHash,
				MethodID:     tl.MethodID,
				Nonce:        tl.Nonce,
				TxTime:       tl.TxTime,
				From:         tl.From,
				To:           tl.To,
				TokenAddress: tl.TokenAddress,
				Amount:       tl.Amount,
				Symbol:       tl.Symbol,
				TxFee:        tl.TxFee,
				TxStatus:     tl.TxStatus,
				HitBlacklist: tl.HitBlacklist,
				IType:        tl.IType,
			}
			transactions[j] = transaction
		}

		histories[i] = WalletHistory{
			Cursor:       data.Cursor,
			Transactions: transactions,
		}
	}

	return histories, nil
}

func (api *OkxAPI) HistoricalPrice(options OkxHistoricalPriceOptions) ([]WalletHistoricalPrice, error) {
	if options.Chain == "" {
		return nil, fmt.Errorf("chain is required")
	}

	apiPath := fmt.Sprintf("/api/v5/wallet/token/historical-price?chainIndex=%s", options.Chain)

	// Add optional parameters
	if options.Address != "" {
		apiPath += fmt.Sprintf("&tokenAddress=%s", options.Address)
	}
	if options.Limit > 0 {
		apiPath += fmt.Sprintf("&limit=%d", options.Limit)
	} else {
		// Set default limit as per API documentation
		apiPath += "&limit=50"
	}
	if options.Cursor != nil && *options.Cursor != "" {
		apiPath += fmt.Sprintf("&cursor=%s", *options.Cursor)
	}
	if options.Begin != nil && *options.Begin > 0 {
		apiPath += fmt.Sprintf("&begin=%d", *options.Begin)
	}
	if options.End != nil && *options.End > 0 {
		apiPath += fmt.Sprintf("&end=%d", *options.End)
	}
	if options.Period != "" {
		apiPath += fmt.Sprintf("&period=%s", options.Period)
	} else {
		// Set default period as per API documentation
		apiPath += "&period=1d"
	}

	headers := api.generateSignature(OkxSignOptions{
		API:    apiPath,
		Method: "GET",
	})

	req, err := http.NewRequest("GET", OKX_API_ENDPOINT+apiPath, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var okxResp OkxResp[[]WalletHistoricalPrice]
	if err := json.Unmarshal(responseBody, &okxResp); err != nil {
		return nil, err
	}

	if okxResp.Code != "0" {
		return nil, fmt.Errorf("OKX API error: %s - %s", okxResp.Code, okxResp.Msg)
	}

	return okxResp.Data, nil
}

// chainIDToTrustWalletName maps chain IDs to TrustWallet blockchain names
var chainIDToTrustWalletName = map[string]string{
	"1":     "ethereum", // Ethereum Mainnet
	"10":    "optimism", // Optimism
	"56":    "binance",  // BNB Smart Chain (Binance)
	"8453":  "base",     // Base
	"42161": "arbitrum", // Arbitrum One
	"81457": "blast",    // Blast
	"1284":  "moonbeam", // Moonbeam
}

func (api *OkxAPI) getLogoURI(chain string, address string) string {
	checkedEvmAddress := common.HexToAddress(address)

	// Map chainID to TrustWallet blockchain name
	trustWalletChainName := chain
	if mappedName, exists := chainIDToTrustWalletName[chain]; exists {
		trustWalletChainName = mappedName
	}

	return fmt.Sprintf("https://raw.githubusercontent.com/trustwallet/assets/refs/heads/master/blockchains/%s/assets/%s/logo.png", trustWalletChainName, checkedEvmAddress.Hex())
}
