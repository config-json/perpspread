package config

import (
	"github.com/config-json/perpspread/internal/core"
	"github.com/shopspring/decimal"
)

type readerConfig struct {
	Markets        []MarketConfig
	Exchanges      []core.ExchangeName
	SlippageLevels []decimal.Decimal
}

type MarketConfig struct {
	Symbol              string `json:"symbol"`
	Precision           int    `json:"precision"`
	PrecisionCollateral int    `json:"precisionCollateral"`
}

var Reader = &readerConfig{
	Exchanges: []core.ExchangeName{
		core.ExtendedName,
		// Disabled until in-house reader node is created
		// core.HyperliquidName,
		core.LighterName,
	},
	Markets: []MarketConfig{
		{Symbol: "BTC", Precision: 1, PrecisionCollateral: 1},
		{Symbol: "ETH", Precision: 2, PrecisionCollateral: 2},
	},
	SlippageLevels: []decimal.Decimal{
		decimal.NewFromInt(1_000),
		decimal.NewFromInt(10_000),
		decimal.NewFromInt(100_000),
		decimal.NewFromInt(500_000),
		decimal.NewFromInt(1_000_000),
	},
}
