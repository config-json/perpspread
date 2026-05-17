package reader

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/config-json/perpspread/internal/core"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

const lighterBaseUrl = "wss://mainnet.zklighter.elliot.ai/stream"

type lighterPriceLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"size"`
}

type lighterOrderbook struct {
	Offset int                 `json:"offset"`
	Asks   []lighterPriceLevel `json:"asks"`
	Bids   []lighterPriceLevel `json:"bids"`
}

type lighterMessageType struct {
	Channel string
	Type    string
}

type lighterMessage struct {
	Type      string          `json:"type"`
	Channel   string          `json:"channel"`
	Offset    int             `json:"offset"`
	Orderbook json.RawMessage `json:"order_book"`
}

type lighterReader struct {
	*baseReader
}

func newLighterReader() *lighterReader {
	return &lighterReader{
		baseReader: newBaseReader(),
	}
}

func (r *lighterReader) Connect(ctx context.Context, symbols []string) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, lighterBaseUrl, nil)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.conn = conn
	r.mu.Unlock()

	err = r.subscribeToMarkets(symbols)
	if err != nil {
		return err
	}

	go r.manageConnection(ctx)
	return nil
}

func (r *lighterReader) subscribe(symbol string) error {
	msg := map[string]any{
		"type":    "subscribe",
		"channel": "order_book/" + getLighterSymbol(symbol),
	}

	r.mu.RLock()
	conn := r.conn
	defer r.mu.RUnlock()

	return conn.WriteJSON(msg)
}

func (r *lighterReader) subscribeToMarkets(symbols []string) error {
	for _, symbol := range symbols {
		if err := r.subscribe(symbol); err != nil {
			return err
		}
		r.mu.Lock()
		r.baseReader.markets[symbol] = &marketState{
			snapshot: nil,
		}
		r.mu.Unlock()
	}
	return nil
}

func (r *lighterReader) resubscribe() error {
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

func (r *lighterReader) manageConnection(ctx context.Context) {
	connMgr := newConnectionManager(
		r.errorCh,
		&r.mu,
		&r.reconnectDelay,
	)

	connMgr.manageConnection(
		ctx,
		core.LighterName,
		r.readLoop,
		func(ctx context.Context) error {
			return r.reconnect(ctx, lighterBaseUrl)
		},
		r.resubscribe,
	)
}

func (r *lighterReader) readLoop(ctx context.Context) error {
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

			ob, err := r.processEvent(msg)
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

func (r *lighterReader) processEvent(msg []byte) (*orderbook, error) {
	var parsedMsg lighterMessage
	err := json.Unmarshal(msg, &parsedMsg)
	if err != nil {
		return nil, err
	}

	if parsedMsg.Type == "ping" {
		err := r.conn.WriteJSON(map[string]any{
			"type": "pong",
		})
		return nil, err
	}

	parsedMsgType := parseLighterMessageType(parsedMsg.Type)
	symbol := parseChannelSymbol(parsedMsg.Channel)

	if parsedMsgType == nil || !strings.Contains(parsedMsgType.Channel, "order_book") {
		return nil, nil
	}

	var ob lighterOrderbook
	err = json.Unmarshal(parsedMsg.Orderbook, &ob)
	if err != nil {
		return nil, err
	}

	newOb := &orderbook{
		Exchange:  core.LighterName,
		Symbol:    symbol,
		Timestamp: time.Now(),
		Asks:      parseLighterPriceLevels(ob.Asks),
		Bids:      parseLighterPriceLevels(ob.Bids),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	market := r.baseReader.markets[symbol]
	if market == nil {
		return nil, nil
	}

	if parsedMsgType.Type == "update" {
		if market.snapshot == nil {
			return nil, nil
		}
		newOb = applyDelta(market.snapshot, newOb, true)
	}

	market.snapshot = newOb
	return newOb, nil
}

func parseLighterMessageType(msgType string) *lighterMessageType {
	parts := strings.Split(msgType, "/")
	if len(parts) != 2 {
		return nil
	}
	return &lighterMessageType{
		Type:    parts[0],
		Channel: parts[1],
	}
}

func parseLighterPriceLevels(data []lighterPriceLevel) []priceLevel {
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

func parseChannelSymbol(channel string) string {
	parts := strings.Split(channel, ":")
	if len(parts) != 2 {
		return ""
	}
	return parseLighterSymbol(parts[1])
}

func parseLighterSymbol(symbol string) string {
	switch symbol {
	case "0":
		return "ETH"
	case "1":
		return "BTC"
	default:
		panic("unknown lighter symbol")
	}
}

func getLighterSymbol(symbol string) string {
	switch symbol {
	case "ETH":
		return "0"
	case "BTC":
		return "1"
	default:
		panic("unknown lighter symbol")
	}
}
