package reader

import (
	"context"
	"errors"
	"sync"

	"github.com/config-json/perpspread/internal/core"
)

type multiplexer struct {
	readers  []reader
	outputCh chan *orderbook
	errorCh  chan error

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func newMultiplexer(ctx context.Context) *multiplexer {
	ctx, cancel := context.WithCancel(ctx)

	return &multiplexer{
		readers:  make([]reader, 0),
		outputCh: make(chan *orderbook, 1000),
		errorCh:  make(chan error, 1000),

		wg:     sync.WaitGroup{},
		ctx:    ctx,
		cancel: cancel,
	}
}

func (m *multiplexer) AddReader(exchange core.ExchangeName, symbols []string) {
	var r reader

	switch exchange {
	case core.ExtendedName:
		r = newExtendedReader()
	case core.HyperliquidName:
		r = newHyperliquidReader()
	case core.LighterName:
		r = newLighterReader()
	}

	r.Connect(m.ctx, symbols)
	m.readers = append(m.readers, r)

	m.wg.Go(func() {
		for {
			select {
			case <-m.ctx.Done():
				return
			case ob := <-r.Stream():
				m.outputCh <- ob
			case err := <-r.Error():
				m.errorCh <- err
			}
		}
	})
}

func (m *multiplexer) Connect(ctx context.Context) error {
	for _, r := range m.readers {
		err := r.Connect(ctx, []string{})

		if err != nil {
			return err
		}
	}
	return nil
}

func (m *multiplexer) Output() <-chan *orderbook {
	return m.outputCh
}

func (m *multiplexer) Error() <-chan error {
	return m.errorCh
}

func (m *multiplexer) Close() error {
	m.cancel()
	var errs []error

	for _, r := range m.readers {
		err := r.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	m.wg.Wait()
	close(m.outputCh)
	close(m.errorCh)

	return errors.Join(errs...)
}
