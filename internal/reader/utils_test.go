package reader

import (
	"testing"

	"github.com/config-json/perpspread/internal/core"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestApplyDelta(t *testing.T) {
	snapshot := &orderbook{
		Exchange: core.ExtendedName,
		Symbol:   "ETH-USD",
		Bids: []priceLevel{
			{Price: decimal.NewFromFloat(2000.1), Quantity: decimal.NewFromFloat(1)},
			{Price: decimal.NewFromFloat(2000), Quantity: decimal.NewFromFloat(2)},
			{Price: decimal.NewFromFloat(1999.9), Quantity: decimal.NewFromFloat(3)},
		},
		Asks: []priceLevel{
			{Price: decimal.NewFromFloat(2000.2), Quantity: decimal.NewFromFloat(1.5)},
			{Price: decimal.NewFromFloat(2000.3), Quantity: decimal.NewFromFloat(2.5)},
			{Price: decimal.NewFromFloat(2000.4), Quantity: decimal.NewFromFloat(3.5)},
		},
	}

	delta := &orderbook{
		Exchange: core.ExtendedName,
		Symbol:   "ETH-USD",
		Bids: []priceLevel{
			{Price: decimal.NewFromFloat(2000.1), Quantity: decimal.NewFromFloat(-1)},
			{Price: decimal.NewFromFloat(2000), Quantity: decimal.NewFromFloat(-2)},
			{Price: decimal.NewFromFloat(1999.9), Quantity: decimal.NewFromFloat(-2)},
			{Price: decimal.NewFromFloat(1999.8), Quantity: decimal.NewFromFloat(2)},
			{Price: decimal.NewFromFloat(1999.7), Quantity: decimal.NewFromFloat(3)},
		},
		Asks: []priceLevel{
			{Price: decimal.NewFromFloat(2000), Quantity: decimal.NewFromFloat(1)},
			{Price: decimal.NewFromFloat(2000.1), Quantity: decimal.NewFromFloat(3)},
			{Price: decimal.NewFromFloat(2000.2), Quantity: decimal.NewFromFloat(-1.5)},
			{Price: decimal.NewFromFloat(2000.3), Quantity: decimal.NewFromFloat(-2.5)},
			{Price: decimal.NewFromFloat(2000.4), Quantity: decimal.NewFromFloat(-3.5)},
		},
	}

	// when
	ob := applyDelta(snapshot, delta, false)

	// then
	expected := &orderbook{
		Exchange: core.ExtendedName,
		Symbol:   "ETH-USD",
		Bids: []priceLevel{
			{Price: decimal.NewFromFloat(1999.9), Quantity: decimal.NewFromFloat(1)},
			{Price: decimal.NewFromFloat(1999.8), Quantity: decimal.NewFromFloat(2)},
			{Price: decimal.NewFromFloat(1999.7), Quantity: decimal.NewFromFloat(3)},
		},
		Asks: []priceLevel{
			{Price: decimal.NewFromFloat(2000), Quantity: decimal.NewFromFloat(1)},
			{Price: decimal.NewFromFloat(2000.1), Quantity: decimal.NewFromFloat(3)},
		},
	}

	require.Equal(t, expected, ob)
}
