package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/config-json/perpspread/internal/reader"
	db "github.com/config-json/perpspread/internal/storage/db/generated"
)

type Writer struct {
	inputCh     chan *reader.ProcessedOrderbook
	errorCh     chan error
	rateLimiter *rateLimiter
	storage     *Storage
	ctx         context.Context
}

func getKey(symbol, exchange string) string {
	return fmt.Sprintf("%s-%s", symbol, exchange)
}

func NewWriter(ctx context.Context) *Writer {
	storage, err := New(ctx)

	if err != nil {
		panic(err)
	}

	return &Writer{
		inputCh: make(chan *reader.ProcessedOrderbook, 100),
		errorCh: make(chan error, 100),
		// FIXME: this should come from config
		rateLimiter: newRateLimiter(time.Second * 5),
		ctx:         ctx,
		storage:     storage,
	}
}

func (w *Writer) Input() chan<- *reader.ProcessedOrderbook {
	return w.inputCh
}

func (w *Writer) Error() <-chan error {
	return w.errorCh
}

func (w *Writer) Start() {
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case ob := <-w.inputCh:
				w.processOrderbook(ob)
			}
		}
	}()
}

func (w *Writer) Close() {
	close(w.inputCh)
	close(w.errorCh)
	w.storage.Close()
}

func (w *Writer) processOrderbook(ob *reader.ProcessedOrderbook) {
	key := getKey(ob.Symbol, string(ob.Exchange))

	if !w.rateLimiter.ShouldWrite(key) {
		return
	}

	if err := w.writeSnapshot(ob); err != nil {
		w.errorCh <- err
		return
	}

	if err := w.writeSlippages(ob); err != nil {
		w.errorCh <- err
	}

	w.rateLimiter.MarkWritten(key)
}

func (w *Writer) writeSnapshot(ob *reader.ProcessedOrderbook) error {
	snapshot := db.SetSnapshotParams{
		Time:     timeToPgTimestamptz(ob.Timestamp),
		Symbol:   ob.Symbol,
		Exchange: string(ob.Exchange),
		Spread:   decimalToNumeric(ob.Spread),
		DepthBid: decimalToNumeric(ob.Depth.Bid),
		DepthAsk: decimalToNumeric(ob.Depth.Ask),
	}

	return w.storage.Queries.SetSnapshot(w.ctx, snapshot)
}

func (w *Writer) writeSlippages(ob *reader.ProcessedOrderbook) error {
	slippages := make([]db.SetSlippagesParams, 0, len(ob.Slippage))

	for _, slippage := range ob.Slippage {
		slippages = append(slippages, db.SetSlippagesParams{
			Time:        timeToPgTimestamptz(ob.Timestamp),
			Symbol:      ob.Symbol,
			Exchange:    string(ob.Exchange),
			Size:        decimalToNumeric(slippage.Size),
			SlippageBid: decimalToNumeric(slippage.Slippage.Bid),
			SlippageAsk: decimalToNumeric(slippage.Slippage.Ask),
		})
	}

	_, err := w.storage.Queries.SetSlippages(w.ctx, slippages)
	return err
}
