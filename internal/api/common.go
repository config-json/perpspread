package api

import (
	"math"
	"net/http"
	"time"

	"github.com/config-json/perpspread/internal/config"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgtype"
)

type apiStatus string

const (
	apiStatusOk    apiStatus = "OK"
	apiStatusError apiStatus = "ERROR"
)

type apiResponse struct {
	Status apiStatus `json:"status"`
	Data   any       `json:"data,omitempty"`
	Error  string    `json:"error,omitempty"`
}

func roundPct(val float64) float64 {
	factor := math.Pow(10, float64(6))
	return math.Round(val*factor) / factor
}

func roundAsset(val float64, symbol string) float64 {
	var market *config.MarketConfig
	for _, m := range config.Reader.Markets {
		if m.Symbol == symbol {
			market = &m
			break
		}
	}

	if market == nil {
		return val
	}

	factor := math.Pow(10, float64(market.Precision))
	return math.Round(val*factor) / factor
}

func numericToInt(n pgtype.Numeric) int {
	if n.Valid {
		i, _ := n.Int64Value()
		return int(i.Int64)
	}
	return 0
}

type chartPoint struct {
	Time time.Time `json:"time"`
	Bid  float64   `json:"bid"`
	Ask  float64   `json:"ask"`
}

type ApiError int

const (
	errInvalidPeriod ApiError = iota
	errMissingSymbol
	errMissingPeriod
)

func (e ApiError) Error() string {
	switch e {
	case errInvalidPeriod:
		return "invalid period specified"
	case errMissingSymbol:
		return "symbol parameter is required"
	case errMissingPeriod:
		return "period parameter is required"
	default:
		return "unknown API error"
	}
}

func (api *API) ok(w http.ResponseWriter, r *http.Request, data any) {
	render.JSON(w, r, apiResponse{
		Status: apiStatusOk,
		Data:   data,
	})
}

func (api *API) error(w http.ResponseWriter, r *http.Request, err error, status int) {
	render.Status(r, status)
	render.JSON(w, r, apiResponse{
		Status: apiStatusError,
		Error:  err.Error(),
	})
}
