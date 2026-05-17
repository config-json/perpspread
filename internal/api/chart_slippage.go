package api

import (
	"net/http"

	"github.com/config-json/perpspread/internal/core"
	db "github.com/config-json/perpspread/internal/storage/db/generated"
)

type slippageChart struct {
	Size  int          `json:"size"`
	Chart []chartPoint `json:"chart"`
}

type slippageChartResponse = map[core.ExchangeName][]slippageChart

func (api *API) handleSlippageChart(w http.ResponseWriter, r *http.Request) {
	params, err := getParams(r)

	if err != nil {
		api.error(w, r, err, http.StatusBadRequest)
		return
	}

	res, err := api.s.Queries.GetSlippageChart(r.Context(), db.GetSlippageChartParams{
		Symbol:    params.Symbol,
		Timeframe: params.Timeframe,
		Sizes:     params.Sizes,
		Interval:  params.Interval,
		Exchanges: core.ExchangeNamesToStrings(params.Exchanges),
	})

	if err != nil {
		api.error(w, r, err, http.StatusInternalServerError)
		return
	}

	api.ok(w, r, parseSlippageChart(res))
}

func parseSlippageChart(res []db.GetSlippageChartRow) slippageChartResponse {
	out := make(slippageChartResponse)

	type key struct {
		exchange core.ExchangeName
		size     int
	}

	temp := make(map[key][]chartPoint)

	for _, row := range res {
		k := key{
			exchange: core.ExchangeName(row.Exchange),
			size:     numericToInt(row.Size),
		}

		temp[k] = append(temp[k], chartPoint{
			Time: row.Bucket.Time,
			Bid:  roundPct(row.AvgSlippageBid),
			Ask:  roundPct(row.AvgSlippageAsk),
		})
	}

	for k, points := range temp {
		out[k.exchange] = append(out[k.exchange], slippageChart{
			Size:  k.size,
			Chart: points,
		})
	}

	return out
}
