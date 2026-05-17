package api

import (
	"net/http"

	"github.com/config-json/perpspread/internal/storage"
	"github.com/go-chi/chi/v5"
)

type API struct {
	s *storage.Storage
}

func New(s *storage.Storage) *chi.Mux {
	r := chi.NewRouter()
	api := &API{s: s}

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/info", func(r chi.Router) {
		r.Get("/markets", api.handleSymbols)
		r.Get("/exchanges", api.handleExchanges)
		r.Get("/slippage", api.handleSlippage)
	})

	r.Get("/stats/{market}", api.handleStats)

	r.Route("/chart", func(r chi.Router) {
		r.Get("/slippage/{market}", api.handleSlippageChart)
		r.Get("/depth/{market}", api.handleDepthChart)
	})

	return r
}
