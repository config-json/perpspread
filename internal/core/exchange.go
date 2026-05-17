package core

type ExchangeName string

const (
	ExtendedName    = "extended"
	LighterName     = "lighter"
	HyperliquidName = "hyperliquid"
)

var AllExchangeNames = []ExchangeName{
	ExtendedName,
	LighterName,
	HyperliquidName,
}

func ExchangeNamesToStrings(exchangeNames []ExchangeName) []string {
	out := make([]string, len(exchangeNames))
	for i, ex := range exchangeNames {
		out[i] = string(ex)
	}
	return out
}
