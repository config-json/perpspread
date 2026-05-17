package reader

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/config-json/perpspread/internal/core"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

const extendedBaseUrl = "wss://app.extended.exchange/stream.extended.exchange/v1/orderbooks/"

type extendedPriceLevel struct {
	Price    string `json:"p"`
	Quantity string `json:"q"`
}

type extendedOrderbook struct {
	Timestamp int64  `json:"ts"`
	Type      string `json:"type"`
	Data      struct {
		Market string               `json:"m"`
		Bids   []extendedPriceLevel `json:"b"`
		Asks   []extendedPriceLevel `json:"a"`
	} `json:"data"`
	Seq int `json:"seq"`
}

type extendedReader struct {
	*baseReader
}

func newExtendedReader() *extendedReader {
	return &extendedReader{
		baseReader: newBaseReader(),
	}
}

func (r *extendedReader) Connect(ctx context.Context, symbols []string) error {
	for _, market := range symbols {
		err := r.connectMarket(ctx, market)

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *extendedReader) connectMarket(ctx context.Context, symbol string) error {
	extendedSymbol := getExtendedSymbol(symbol)
	url := extendedBaseUrl + extendedSymbol

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)

	if err != nil {
		return err
	}

	r.mu.Lock()
	r.markets[symbol] = &marketState{
		conn:           conn,
		reconnectDelay: reconnectBaseDelay,
		snapshot:       nil,
	}
	r.mu.Unlock()

	go r.manageConnection(ctx, symbol)
	return nil
}

func (r *extendedReader) reconnectMarket(ctx context.Context, symbol string) error {
	url := extendedBaseUrl + getExtendedSymbol(symbol)
	newConn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	market := r.markets[symbol]
	if market.conn != nil {
		market.conn.Close()
	}

	market.conn = newConn
	market.snapshot = nil
	market.seq = 0

	return nil
}

func (r *extendedReader) manageConnection(ctx context.Context, symbol string) {
	connMgr := newConnectionManager(
		r.errorCh,
		&r.mu,
		&r.markets[symbol].reconnectDelay,
	)

	connMgr.manageConnection(
		ctx,
		core.ExtendedName,
		func(ctx context.Context) error {
			r.mu.RLock()
			conn := r.markets[symbol].conn
			r.mu.RUnlock()
			return r.readLoop(ctx, conn)
		},
		func(ctx context.Context) error {
			return r.reconnectMarket(ctx, symbol)
		},
		nil,
	)
}

func (r *extendedReader) readLoop(ctx context.Context, conn *websocket.Conn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return err
			}

			ob, err := r.processEvent(msg)
			if err != nil {
				if err == errSequenceMismatch {
					return err
				}
				r.errorCh <- err
				continue
			}

			if ob != nil {
				r.orderbookCh <- ob
			}
		}
	}
}

func (r *extendedReader) processEvent(msg []byte) (*orderbook, error) {
	var ob extendedOrderbook
	err := json.Unmarshal(msg, &ob)

	if err != nil {
		return nil, err
	}

	symbol := parseExtendedSymbol(ob.Data.Market)

	newOb := &orderbook{
		Exchange:  core.ExtendedName,
		Symbol:    symbol,
		Timestamp: time.UnixMilli(ob.Timestamp),
		Bids:      parseExtendedPriceLevels(ob.Data.Bids),
		Asks:      parseExtendedPriceLevels(ob.Data.Asks),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	market := r.markets[symbol]

	if ob.Seq != market.seq+1 {
		return nil, errSequenceMismatch
	}

	market.seq = ob.Seq

	if ob.Type == "DELTA" {
		if market == nil {
			return nil, nil
		}

		newOb = applyDelta(market.snapshot, newOb, false)
	}

	market.snapshot = newOb
	return newOb, nil
}

func parseExtendedPriceLevels(data []extendedPriceLevel) []priceLevel {
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

func getExtendedSymbol(symbol string) string {
	return fmt.Sprintf("%s-USD", symbol)
}

func parseExtendedSymbol(symbol string) string {
	parts := strings.Split(symbol, "-")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}
