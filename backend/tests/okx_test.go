package tests

import (
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/internal/utils"
)

func init() {
	err := godotenv.Load("../.env")
	if err != nil {
		slog.Warn("No .env file found, using default environment variables")
		panic(err)
	}

	err = config.InitConfig()
	if err != nil {
		slog.Error("Failed to initialize configuration", "error", err)
		panic(err)
	}
}

func okx() *internal.OkxAPI {
	cfg := config.GetConfig()
	okx := internal.NewOkxAPI(internal.OkxOptions{
		Project:    cfg.GetStringRequired("OKX_PROJECT"),
		AccessKey:  cfg.GetStringRequired("OKX_ACCESS_KEY"),
		SecretKey:  cfg.GetStringRequired("OKX_SECRET_KEY"),
		Passphrase: cfg.GetStringRequired("OKX_PASSPHRASE"),
	})
	return okx
}

func TestBalances(t *testing.T) {
	okx := okx()

	balances, err := okx.Balances(internal.OkxBalanceOptions{
		Chains:  []string{"1"},
		Address: "0xc18360217d8f7ab5e7c516566761ea12ce7f9d72",
	})
	if err != nil {
		t.Errorf("Failed to get OKX balances: %v", err)
		return
	}

	// Log results using t.Logf so they're visible even without -v flag
	t.Logf("Successfully retrieved %d token balances", len(balances))

	for i, balance := range balances {
		// if i < 3 { // Show first 3 tokens for brevity
		t.Logf(
			"Token %d: %s (%s) - { Balance: %s, balanceRaw: %s, balanceUSD: %s } - Decimals: %d",
			i+1, balance.Symbol, balance.ID,
			balance.Platforms[0].Balance,
			balance.Platforms[0].BalanceRaw,
			balance.Platforms[0].BalanceUSD,
			balance.Platforms[0].Decimals,
		)
		// }
	}

	// Verify we got some balances
	if len(balances) == 0 {
		t.Error("Expected some balances, but got empty result")
	}
}

func TestPrices(t *testing.T) {
	okx := okx()

	// Test getting price for USDT on Ethereum
	price, err := okx.Price(internal.OkxPriceOptions{
		Chain:   "1",
		Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", // USDT
	})
	if err != nil {
		t.Errorf("Failed to get OKX price: %v", err)
		return
	}

	if price != nil {
		t.Logf("USDT Price: %s (Chain: %s, Token: %s)",
			price.Price, price.ChainID, price.TokenAddress)
	} else {
		t.Error("Expected price data, but got nil")
	}

	// Test getting multiple prices
	prices, err := okx.Prices([]internal.OkxPriceOptions{
		{Chain: "1", Address: "0xdac17f958d2ee523a2206206994597c13d831ec7"}, // USDT
		{Chain: "1", Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"}, // USDC
	})
	if err != nil {
		t.Errorf("Failed to get OKX prices: %v", err)
		return
	}

	t.Logf("Retrieved %d token prices", len(prices))
	for i, p := range prices {
		t.Logf("Price %d: %s (Token: %s)", i+1, p.Price, p.TokenAddress)
	}
}

func TestHistory(t *testing.T) {
	okx := okx()

	history, err := okx.History(internal.OkxHistoryOptions{
		Address: "0xc18360217d8f7ab5e7c516566761ea12ce7f9d72",
		Chains:  []string{"1"},
		Limit:   5, // Limit to 5 transactions for testing
	})
	if err != nil {
		t.Errorf("Failed to get OKX history: %v", err)
		return
	}

	t.Logf("Retrieved %d history entries", len(history))

	for i, h := range history {
		if i < 2 { // Show first 2 history entries
			t.Logf("History %d: %d transactions (Cursor: %s)",
				i+1, len(h.Transactions), h.Cursor)

			for j, tx := range h.Transactions {
				if j < 2 { // Show first 2 transactions
					t.Logf("  Transaction %d: %s (%s %s)",
						j+1, tx.TxHash, tx.Amount, tx.Symbol)
				}
			}
		}
	}
}

func TestHistoricalPrice(t *testing.T) {
	okx := okx()

	now := time.Now()
	histories, err := okx.HistoricalPrice(internal.OkxHistoricalPriceOptions{
		Chain:   "1",
		Address: "0xc18360217d8f7ab5e7c516566761ea12ce7f9d72", // USDT
		Limit:   10,
		Begin:   utils.Int64Ptr(now.Add(-5 * 24 * time.Hour).UnixMilli()), // 5 days ago
		End:     utils.Int64Ptr(now.UnixMilli()),
		Period:  internal.OkxPeriod1d,
	})
	if err != nil {
		t.Errorf("Failed to get OKX historical prices: %v", err)
		return
	}

	t.Logf("Retrieved %d historical price entries", len(histories))

	for i, h := range histories {
		t.Logf("History %d: %d prices (Cursor: %s)",
			i+1, len(h.Prices), h.Cursor)

		for j, p := range h.Prices {
			timestampMillis, err := strconv.ParseInt(p.Time, 10, 64)
			if err != nil {
				t.Fatalf("Failed to parse timestamp string '%s': %v", p.Time, err)
			}

			goTime := time.Unix(timestampMillis/1000, (timestampMillis%1000)*1000000)
			formattedTime := goTime.Format("2006-01-02 15:04:05")
			t.Logf("  Price %d: %s (%s)",
				j+1, p.Price, formattedTime)
		}
	}
}
