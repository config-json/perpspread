package api

import (
	"net/http"

	"github.com/config-json/perpspread/internal/config"
)

func (api *API) handleSymbols(w http.ResponseWriter, r *http.Request) {
	symbols := make([]string, len(config.Reader.Markets))

	for i, market := range config.Reader.Markets {
		symbols[i] = market.Symbol
	}

	api.ok(w, r, symbols)
}

func (api *API) handleExchanges(w http.ResponseWriter, r *http.Request) {
	exchanges := config.Reader.Exchanges
	api.ok(w, r, exchanges)
}

func (api *API) handleSlippage(w http.ResponseWriter, r *http.Request) {
	slippageLevels := config.Reader.SlippageLevels
	api.ok(w, r, slippageLevels)
}
