package reader

import (
	"context"
	"sync"
	"time"

	"github.com/config-json/perpspread/internal/core"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

type priceLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

type orderbook struct {
	Exchange  core.ExchangeName
	Symbol    string
	Timestamp time.Time
	Bids      []priceLevel
	Asks      []priceLevel
}

type reader interface {
	Connect(ctx context.Context, markets []string) error
	Stream() <-chan *orderbook
	Error() <-chan error
	Close() error
}

type marketState struct {
	// Optional field for exchanges that only support one market / connection
	conn           *websocket.Conn
	reconnectDelay time.Duration
	snapshot       *orderbook
	seq            int
}

type baseReader struct {
	// Optional field for exchanges that support multiple markets / connection
	conn           *websocket.Conn
	reconnectDelay time.Duration
	orderbookCh    chan *orderbook
	errorCh        chan error
	mu             sync.RWMutex

	markets map[string]*marketState
}

func newBaseReader() *baseReader {
	return &baseReader{
		reconnectDelay: reconnectBaseDelay,
		orderbookCh:    make(chan *orderbook, 100),
		errorCh:        make(chan error, 100),
		mu:             sync.RWMutex{},

		markets: make(map[string]*marketState),
	}
}

func (r *baseReader) Stream() <-chan *orderbook {
	return r.orderbookCh
}

func (r *baseReader) Error() <-chan error {
	return r.errorCh
}

func (r *baseReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		err := r.conn.Close()
		r.conn = nil
		r.markets = nil
		return err
	}

	for _, market := range r.markets {
		if market.conn != nil {
			err := market.conn.Close()
			if err != nil {
				return err
			}
		}
	}

	close(r.orderbookCh)
	close(r.errorCh)

	return nil
}

func (r *baseReader) reconnect(ctx context.Context, url string) error {
	newConn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)

	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		r.conn.Close()
	}

	r.conn = newConn
	return nil
}
