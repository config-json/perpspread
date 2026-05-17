package api

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/config-json/perpspread/internal/config"
	"github.com/config-json/perpspread/internal/core"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type requestParams struct {
	Symbol    string
	Interval  pgtype.Interval
	Timeframe pgtype.Interval
	Exchanges []core.ExchangeName
	Sizes     []int32
}

func parseTimeString(timeStr string) (pgtype.Interval, error) {
	d, err := time.ParseDuration(timeStr)

	if err == nil {
		return pgtype.Interval{
			Valid:        true,
			Microseconds: d.Microseconds(),
		}, nil
	}

	if timeStr[len(timeStr)-1:] == "D" {
		var days int32
		_, err := fmt.Sscanf(timeStr, "%dD", &days)
		if err == nil {
			return pgtype.Interval{
				Valid:  true,
				Days:   days,
				Months: 0,
			}, nil
		}
	}

	if timeStr[len(timeStr)-1:] == "W" {
		var weeks int32
		_, err := fmt.Sscanf(timeStr, "%dW", &weeks)
		if err == nil {
			return pgtype.Interval{
				Valid:  true,
				Days:   weeks * 7,
				Months: 0,
			}, nil
		}
	}

	if timeStr[len(timeStr)-1:] == "M" {
		var months int32
		_, err := fmt.Sscanf(timeStr, "%dM", &months)
		if err == nil {
			return pgtype.Interval{
				Valid:  true,
				Days:   0,
				Months: months,
			}, nil
		}
	}

	return pgtype.Interval{}, errInvalidPeriod
}

func intervalToDuration(interval pgtype.Interval) time.Duration {
	microseconds := interval.Microseconds
	microseconds += int64(interval.Days) * 24 * 60 * 60 * 1000000
	microseconds += int64(interval.Months) * 30 * 24 * 60 * 60 * 1000000
	return time.Duration(microseconds) * time.Microsecond
}

func roundToNiceInterval(d time.Duration) time.Duration {
	niceIntervals := []time.Duration{
		5 * time.Second,
		10 * time.Second,
		30 * time.Second,
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
		2 * time.Hour,
		6 * time.Hour,
		12 * time.Hour,
		24 * time.Hour,
	}

	for _, interval := range niceIntervals {
		if d <= interval {
			return interval
		}
	}

	return 24 * time.Hour
}

func calcInterval(period pgtype.Interval) pgtype.Interval {
	d := intervalToDuration(period)

	targetPoints := 200
	intervalDuration := d / time.Duration(targetPoints)

	intervalDuration = roundToNiceInterval(intervalDuration)

	return pgtype.Interval{
		Valid:        true,
		Microseconds: intervalDuration.Microseconds(),
	}
}

func parsePeriod(periodStr string) (pgtype.Interval, pgtype.Interval, error) {

	period, err := parseTimeString(periodStr)
	if err != nil {
		return pgtype.Interval{}, pgtype.Interval{}, err
	}

	interval := calcInterval(period)

	return period, interval, nil
}

func parseExchangeNames(exchangeNames []string) []core.ExchangeName {
	var exchanges []core.ExchangeName

	if len(exchangeNames) == 0 {
		return core.AllExchangeNames
	}

	for _, name := range exchangeNames {
		exchangeName := core.ExchangeName(name)
		if !slices.Contains(core.AllExchangeNames, exchangeName) {
			continue
		}
		exchanges = append(exchanges, exchangeName)
	}

	return exchanges
}

func parseSizes(sizesStr []string) []int32 {
	sizes := make([]int32, 0, len(sizesStr))

	if len(sizesStr) == 0 {
		for _, sizeDecimal := range config.Reader.SlippageLevels {
			sizes = append(sizes, int32(sizeDecimal.IntPart()))
		}
	} else {
		for _, sizeStr := range sizesStr {
			size, err := strconv.Atoi(sizeStr)

			if err == nil {
				sizes = append(sizes, int32(size))
			}
		}
	}

	return sizes
}

func getParams(r *http.Request) (*requestParams, error) {
	market := chi.URLParam(r, "market")
	period := r.URL.Query().Get("period")
	exchange := r.URL.Query()["exchange"]
	sizesStr := r.URL.Query()["size"]

	exchangeNames := parseExchangeNames(exchange)

	if market == "" {
		return nil, errMissingSymbol
	}

	timeframe, interval, err := parsePeriod(period)
	if err != nil {
		return nil, err
	}

	sizes := parseSizes(sizesStr)

	return &requestParams{
		Symbol:    market,
		Interval:  interval,
		Timeframe: timeframe,
		Exchanges: exchangeNames,
		Sizes:     sizes,
	}, nil
}
