package api

import (
	"net/http"

	"github.com/config-json/perpspread/internal/core"
	db "github.com/config-json/perpspread/internal/storage/db/generated"
)

type depthChartResponse = map[core.ExchangeName][]chartPoint

func (api *API) handleDepthChart(w http.ResponseWriter, r *http.Request) {
	params, err := getParams(r)
	if err != nil {
		api.error(w, r, err, http.StatusBadRequest)
		return
	}

	res, err := api.s.Queries.GetDepthChart(r.Context(), db.GetDepthChartParams{
		Symbol:    params.Symbol,
		Timeframe: params.Timeframe,
		Interval:  params.Interval,
		Exchanges: core.ExchangeNamesToStrings(params.Exchanges),
	})

	if err != nil {
		api.error(w, r, nil, http.StatusInternalServerError)
		return
	}

	api.ok(w, r, parseDepthChart(res))
}

func parseDepthChart(res []db.GetDepthChartRow) depthChartResponse {
	out := make(depthChartResponse)

	for _, row := range res {
		key := core.ExchangeName(row.Exchange)

		point := chartPoint{
			Time: row.Bucket.Time,
			Bid:  roundAsset(row.AvgDepthBid, row.Symbol),
			Ask:  roundAsset(row.AvgDepthAsk, row.Symbol),
		}

		out[key] = append(out[key], point)
	}

	return out
}
