package reader

import (
	"context"

	"github.com/config-json/perpspread/internal/config"
)

type Reader struct {
	multiplexer *multiplexer
	processor   *processor
	outputCh    chan *ProcessedOrderbook
	errorCh     chan error
	ctx         context.Context
	cancel      context.CancelFunc
}

func New(ctx context.Context) *Reader {
	ctx, cancel := context.WithCancel(ctx)

	return &Reader{
		multiplexer: newMultiplexer(ctx),
		processor:   newProcessor(ctx, config.Reader.SlippageLevels),
		outputCh:    make(chan *ProcessedOrderbook, 1000),
		errorCh:     make(chan error, 1000),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (r *Reader) Start() {
	symbols := make([]string, len(config.Reader.Markets))
	for i, market := range config.Reader.Markets {
		symbols[i] = market.Symbol
	}

	for _, exchange := range config.Reader.Exchanges {
		r.multiplexer.AddReader(exchange, symbols)
	}

	r.processor.Start()
	go r.run()
}

func (r *Reader) Output() <-chan *ProcessedOrderbook {
	return r.outputCh
}

func (r *Reader) Error() <-chan error {
	return r.errorCh
}

func (r *Reader) Close() {
	r.cancel()
	r.processor.Close()
	r.multiplexer.Close()
	close(r.outputCh)
	close(r.errorCh)
}

func (r *Reader) run() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case processedOb := <-r.processor.Output():
			r.outputCh <- processedOb
		case ob := <-r.multiplexer.Output():
			r.processor.Input() <- ob
		case err := <-r.multiplexer.Error():
			r.errorCh <- err
		}
	}
}
