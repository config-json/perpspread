package reader

import (
	"sort"

	"github.com/shopspring/decimal"
)

type extendedError int

const (
	errSequenceMismatch extendedError = iota
)

func (e extendedError) Error() string {
	switch e {
	case errSequenceMismatch:
		return "sequence mismatch"
	default:
		return "unknown error"
	}
}

func applyDelta(snapshot, delta *orderbook, replace bool) *orderbook {
	return &orderbook{
		Exchange:  snapshot.Exchange,
		Symbol:    snapshot.Symbol,
		Timestamp: delta.Timestamp,
		Bids:      applySide(snapshot.Bids, delta.Bids, replace, true),
		Asks:      applySide(snapshot.Asks, delta.Asks, replace, false),
	}
}

func applySide(snapshot, delta []priceLevel, replace, isBid bool) []priceLevel {
	result := make([]priceLevel, len(snapshot))
	copy(result, snapshot)

	cache := make(map[string]int)

	for i, lvl := range result {
		cache[lvl.Price.String()] = i
	}

	for _, d := range delta {
		priceKey := d.Price.String()
		idx, ok := cache[priceKey]

		if ok {
			if replace {
				result[idx].Quantity = d.Quantity
			} else {
				result[idx].Quantity = result[idx].Quantity.Add(d.Quantity)
			}
			continue
		}

		result = append(result, d)
		cache[priceKey] = len(result) - 1
	}

	filtered := make([]priceLevel, 0, len(result))
	for _, lvl := range result {
		if lvl.Quantity.GreaterThan(decimal.Zero) {
			filtered = append(filtered, lvl)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if isBid {
			return filtered[i].Price.GreaterThan(filtered[j].Price)
		}
		return filtered[i].Price.LessThan(filtered[j].Price)
	})

	return filtered
}
