package services

import (
	"strconv"
	"time"

	"github.com/ringecosystem/degov-square/database"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/internal/utils"
	"gorm.io/gorm"
)

type TreasuryService struct {
	db  *gorm.DB
	okx *internal.OkxAPI
}

func NewTreasuryService() *TreasuryService {
	cfg := config.GetConfig()
	okx := internal.NewOkxAPI(internal.OkxOptions{
		Project:    cfg.GetStringRequired("OKX_PROJECT"),
		AccessKey:  cfg.GetStringRequired("OKX_ACCESS_KEY"),
		SecretKey:  cfg.GetStringRequired("OKX_SECRET_KEY"),
		Passphrase: cfg.GetStringRequired("OKX_PASSPHRASE"),
	})
	return &TreasuryService{
		db:  database.GetDB(),
		okx: okx,
	}
}

// Load all assets of treasury
func (s *TreasuryService) LoadTreasuryAssets(input *gqlmodels.TreasuryAssetsInput) ([]*gqlmodels.TreasuryAsset, error) {
	var assets []*gqlmodels.TreasuryAsset

	balances, err := s.okx.Balances(internal.OkxBalanceOptions{
		Chains:  []string{input.Chain},
		Address: input.Address,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for _, balance := range balances {
		platforms := balance.Platforms
		if len(platforms) == 0 {
			continue
		}
		firstPlatform := platforms[0]
		nativeToken := 0
		if firstPlatform.Native {
			nativeToken = 1
		}

		var treasuryHistoricalPrices []*gqlmodels.TreasuryHistoricalPrice
		histories, err := s.okx.HistoricalPrice(internal.OkxHistoricalPriceOptions{
			Chain:   input.Chain,
			Address: firstPlatform.Address,
			Limit:   5,
			Begin:   utils.Int64Ptr(now.Add(-5 * 24 * time.Hour).UnixMilli()), // 5 days ago
			End:     utils.Int64Ptr(now.UnixMilli()),
			Period:  internal.OkxPeriod1d,
		})
		if err == nil && len(histories) > 0 {
			history := histories[0]
			for _, hp := range history.Prices {
				timestampInt64, err := strconv.ParseInt(hp.Time, 10, 64)
				if err != nil {
					continue
				}
				treasuryHistoricalPrices = append(treasuryHistoricalPrices, &gqlmodels.TreasuryHistoricalPrice{
					Timestamp: timestampInt64,
					Price:     hp.Price,
				})
			}
		}

		assets = append(assets, &gqlmodels.TreasuryAsset{
			Chain:            input.Chain,
			Address:          firstPlatform.Address,
			Name:             balance.Name,
			Symbol:           balance.Symbol,
			Logo:             &balance.LogoURI,
			Native:           int32(nativeToken),
			Price:            firstPlatform.Price,
			Balance:          firstPlatform.Balance,
			BalanceRaw:       firstPlatform.BalanceRaw,
			BalanceUsd:       firstPlatform.BalanceUSD,
			DisplayDecimals:  int32(firstPlatform.DisplayDecimals),
			HistoricalPrices: treasuryHistoricalPrices,
		})
	}

	return assets, nil
}
