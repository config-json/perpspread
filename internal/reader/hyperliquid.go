package reader

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/config-json/perpspread/internal/core"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

const hyperliquidOrderbookChannel = "l2Book"
const hyperliquidBaseUrl = "wss://api.hyperliquid.xyz/ws"

type hyperliquidPriceLevel struct {
	Price    string `json:"px"`
	Quantity string `json:"sz"`
}

type hyperliquidOrderbook struct {
	Market    string                     `json:"coin"`
	Timestamp int64                      `json:"time"`
	Levels    [2][]hyperliquidPriceLevel `json:"levels"`
}

type hyperliquidMessage struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

type hyperliquidReader struct {
	*baseReader
}

func newHyperliquidReader() *hyperliquidReader {
	return &hyperliquidReader{
		baseReader: newBaseReader(),
	}
}

func (r *hyperliquidReader) Connect(ctx context.Context, symbols []string) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, hyperliquidBaseUrl, nil)

	if err != nil {
		return err
	}

	r.mu.Lock()
	r.conn = conn
	r.mu.Unlock()

	go r.manageConnection(ctx)

	err = r.subscribeToMarkets(symbols)
	if err != nil {
		return err
	}

	return nil
}

func (r *hyperliquidReader) subscribe(symbol string) error {
	msg := map[string]any{
		"method": "subscribe",
		"subscription": map[string]any{
			"type": hyperliquidOrderbookChannel,
			"coin": symbol,
		},
	}

	return r.conn.WriteJSON(msg)
}

func (r *hyperliquidReader) subscribeToMarkets(symbols []string) error {
	for _, symbol := range symbols {
		if err := r.subscribe(symbol); err != nil {
			return err
		}
		r.mu.Lock()
		r.markets[symbol] = &marketState{
			snapshot: nil,
		}
		r.mu.Unlock()
	}
	return nil
}

func (r *hyperliquidReader) resubscribe() error {
	r.mu.RLock()
	symbols := make([]string, 0, len(r.markets))
	for symbol := range r.markets {
		symbols = append(symbols, symbol)
	}
	r.mu.RUnlock()

	for _, symbol := range symbols {
		if err := r.subscribe(symbol); err != nil {
			return err
		}
	}
	return nil
}

func (r *hyperliquidReader) manageConnection(ctx context.Context) {
	connMgr := newConnectionManager(
		r.errorCh,
		&r.mu,
		&r.reconnectDelay,
	)

	connMgr.manageConnection(
		ctx,
		core.HyperliquidName,
		r.readLoop,
		func(ctx context.Context) error {
			return r.reconnect(ctx, hyperliquidBaseUrl)
		},
		r.resubscribe,
	)
}

func (r *hyperliquidReader) readLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			r.mu.RLock()
			conn := r.conn
			r.mu.RUnlock()

			_, msg, err := conn.ReadMessage()

			if err != nil {
				return err
			}

			ob, err := processEvent(msg)

			if err != nil {
				r.errorCh <- err
				continue
			}

			if ob != nil {
				r.orderbookCh <- ob
			}
		}
	}
}

func processEvent(msg []byte) (*orderbook, error) {
	var parsedMsg hyperliquidMessage
	err := json.Unmarshal(msg, &parsedMsg)

	if err != nil {
		return nil, err
	}

	if parsedMsg.Channel == "error" {
		return nil, fmt.Errorf(string(parsedMsg.Data))
	}

	if parsedMsg.Channel != hyperliquidOrderbookChannel {
		return nil, nil
	}

	var ob hyperliquidOrderbook
	err = json.Unmarshal(parsedMsg.Data, &ob)

	if err != nil {
		return nil, err
	}

	newOb := &orderbook{
		Exchange:  core.HyperliquidName,
		Symbol:    ob.Market,
		Timestamp: time.UnixMilli(ob.Timestamp),
		Bids:      parseHyperliquidPriceLevels(ob.Levels[0]),
		Asks:      parseHyperliquidPriceLevels(ob.Levels[1]),
	}

	return newOb, nil
}

func parseHyperliquidPriceLevels(data []hyperliquidPriceLevel) []priceLevel {
	result := make([]priceLevel, len(data))
	for i, level := range data {
		price := decimal.RequireFromString(level.Price)
		quantity := decimal.RequireFromString(level.Quantity)

		result[i] = priceLevel{
			Price:    price,
			Quantity: quantity,
		}
	}
	return result
}
