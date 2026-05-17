package api

import (
	"net/http"
	"slices"

	"github.com/config-json/perpspread/internal/core"
	db "github.com/config-json/perpspread/internal/storage/db/generated"
)

type bidAskStat struct {
	Bid float64 `json:"bid"`
	Ask float64 `json:"ask"`
}

type slippageStat struct {
	Size int `json:"size"`
	bidAskStat
}

type statsResponse struct {
	Exchange core.ExchangeName `json:"exchange"`
	Spread   float64           `json:"spread"`
	Depth    *bidAskStat       `json:"depth"`
	Slippage []slippageStat    `json:"slippage"`
}

func (api *API) handleStats(w http.ResponseWriter, r *http.Request) {
	params, err := getParams(r)
	if err != nil {
		api.error(w, r, err, http.StatusBadRequest)
		return
	}

	snapshots, err := api.s.Queries.GetSnapshotStats(r.Context(), db.GetSnapshotStatsParams{
		Symbol:    params.Symbol,
		Timeframe: params.Timeframe,
		Exchanges: core.ExchangeNamesToStrings(params.Exchanges),
	})

	if err != nil {
		api.error(w, r, err, http.StatusInternalServerError)
		return
	}

	slippages, err := api.s.Queries.GetSlippageStats(r.Context(), db.GetSlippageStatsParams{
		Symbol:    params.Symbol,
		Timeframe: params.Timeframe,
		Exchanges: core.ExchangeNamesToStrings(params.Exchanges),
	})

	api.ok(w, r, joinStats(snapshots, slippages))
}

func joinStats(snapshots []db.GetSnapshotStatsRow, slippages []db.GetSlippageStatsRow) []statsResponse {
	sorted := make(map[core.ExchangeName]statsResponse)

	for _, snap := range snapshots {
		ex := core.ExchangeName(snap.Exchange)
		sorted[ex] = statsResponse{
			Exchange: ex,
			Spread:   roundAsset(snap.AvgSpread, snap.Symbol),
			Depth: &bidAskStat{
				Bid: roundAsset(snap.AvgDepthBid, snap.Symbol),
				Ask: roundAsset(snap.AvgDepthAsk, snap.Symbol),
			},
			Slippage: []slippageStat{},
		}
	}

	for _, slip := range slippages {
		ex := core.ExchangeName(slip.Exchange)

		res := sorted[ex]
		res.Slippage = append(res.Slippage, slippageStat{
			Size: numericToInt(slip.Size),
			bidAskStat: bidAskStat{
				Bid: roundPct(slip.AvgSlippageBid),
				Ask: roundPct(slip.AvgSlippageAsk),
			},
		})
		sorted[ex] = res
	}

	out := make([]statsResponse, 0, len(sorted))

	for _, res := range sorted {
		out = append(out, res)
	}

	slices.SortFunc(out, func(a, b statsResponse) int {
		if a.Exchange < b.Exchange {
			return -1
		}
		if a.Exchange > b.Exchange {
			return 1
		}
		return 0
	})

	return out
}
