package internal

import (
	"strings"
	"testing"
)

func TestBalancesReturnsErrorOnOkxError(t *testing.T) {
	responseBody := []byte(`{
		"code": "50125",
		"msg": "Your API key or regions have no access to current services",
		"data": []
	}`)

	var api OkxAPI
	balances, err := api.parseBalancesResponse(responseBody)
	if err == nil {
		t.Fatalf("expected OKX API error, got nil")
	}
	if balances != nil {
		t.Fatalf("expected nil balances on error, got %#v", balances)
	}
	if !strings.Contains(err.Error(), "50125") {
		t.Fatalf("expected error to include OKX code, got %q", err.Error())
	}
}

func TestBalancesParsesSuccessfulResponse(t *testing.T) {
	responseBody := []byte(`{
		"code": "0",
		"msg": "",
		"data": [{
			"tokenAssets": [{
				"chainIndex": "1",
				"tokenAddress": "",
				"symbol": "ETH",
				"balance": "1.5",
				"tokenPrice": "3000",
				"rawBalance": "1500000000000000000",
				"address": "0xabc"
			}]
		}]
	}`)

	var api OkxAPI
	balances, err := api.parseBalancesResponse(responseBody)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(balances) != 1 {
		t.Fatalf("expected 1 balance, got %d", len(balances))
	}

	balance := balances[0]
	if balance.Symbol != "ETH" {
		t.Fatalf("expected ETH symbol, got %q", balance.Symbol)
	}
	if len(balance.Platforms) != 1 {
		t.Fatalf("expected 1 platform, got %d", len(balance.Platforms))
	}
	if balance.Platforms[0].BalanceUSD != "4500.00" {
		t.Fatalf("expected balance USD 4500.00, got %q", balance.Platforms[0].BalanceUSD)
	}
	if !balance.Platforms[0].Native {
		t.Fatalf("expected native token to be true")
	}
}

func TestPricesReturnsErrorOnOkxError(t *testing.T) {
	responseBody := []byte(`{
		"code": "50125",
		"msg": "Your API key or regions have no access to current services",
		"data": []
	}`)

	var api OkxAPI
	prices, err := api.parsePricesResponse(responseBody)
	if err == nil {
		t.Fatalf("expected OKX API error, got nil")
	}
	if prices != nil {
		t.Fatalf("expected nil prices on error, got %#v", prices)
	}
	if !strings.Contains(err.Error(), "50125") {
		t.Fatalf("expected error to include OKX code, got %q", err.Error())
	}
}

func TestHistoryReturnsErrorOnOkxError(t *testing.T) {
	responseBody := []byte(`{
		"code": "50125",
		"msg": "Your API key or regions have no access to current services",
		"data": []
	}`)

	var api OkxAPI
	history, err := api.parseHistoryResponse(responseBody)
	if err == nil {
		t.Fatalf("expected OKX API error, got nil")
	}
	if history != nil {
		t.Fatalf("expected nil history on error, got %#v", history)
	}
	if !strings.Contains(err.Error(), "50125") {
		t.Fatalf("expected error to include OKX code, got %q", err.Error())
	}
}
