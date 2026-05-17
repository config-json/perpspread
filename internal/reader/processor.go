package reader

import (
	"context"
	"sync"
	"time"

	"github.com/config-json/perpspread/internal/core"
	"github.com/shopspring/decimal"
)

type orderSide string

const (
	bidSide orderSide = "bid"
	askSide orderSide = "ask"
)

type sideMetric struct {
	Bid decimal.Decimal
	Ask decimal.Decimal
}

type slippageLevel struct {
	Size     decimal.Decimal
	Slippage *sideMetric
}

type ProcessedOrderbook struct {
	Exchange  core.ExchangeName
	Symbol    string
	Timestamp time.Time
	Spread    decimal.Decimal
	Depth     *sideMetric
	Slippage  []slippageLevel
}

type processor struct {
	inputCh  chan *orderbook
	outputCh chan *ProcessedOrderbook
	workers  int
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc

	slippageLevels []decimal.Decimal
}

const midPriceBuffer = 0.5

func newProcessor(ctx context.Context, slippageLevels []decimal.Decimal) *processor {
	ctx, cancel := context.WithCancel(ctx)

	return &processor{
		inputCh:  make(chan *orderbook, 1000),
		outputCh: make(chan *ProcessedOrderbook, 1000),
		workers:  10,
		ctx:      ctx,
		cancel:   cancel,

		slippageLevels: slippageLevels,
	}
}

func (p *processor) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *processor) worker() {
	defer p.wg.Done()

	for ob := range p.inputCh {
		select {
		case <-p.ctx.Done():
			return
		case p.outputCh <- p.process(ob):
		}
	}
}

func (p *processor) Input() chan<- *orderbook {
	return p.inputCh
}

func (p *processor) Output() <-chan *ProcessedOrderbook {
	return p.outputCh
}

func (p *processor) Close() {
	p.cancel()
	close(p.inputCh)

	p.wg.Wait()
	close(p.outputCh)
}

func (p *processor) process(ob *orderbook) *ProcessedOrderbook {
	return &ProcessedOrderbook{
		Exchange:  ob.Exchange,
		Symbol:    ob.Symbol,
		Timestamp: ob.Timestamp,
		Spread:    calcSpread(ob.Bids, ob.Asks),
		Depth:     calcDepth(ob.Bids, ob.Asks),
		Slippage:  calcSlippageLevels(ob.Bids, ob.Asks, p.slippageLevels),
	}
}

func calcSpread(bids, asks []priceLevel) decimal.Decimal {
	if len(bids) == 0 || len(asks) == 0 {
		return decimal.Zero
	}

	return asks[0].Price.Sub(bids[0].Price)
}

func calcMidPrice(bids, asks []priceLevel) decimal.Decimal {
	if len(bids) == 0 || len(asks) == 0 {
		return decimal.Zero
	}

	return bids[0].Price.Add(asks[0].Price).Div(decimal.NewFromInt(2))
}

func calcDepth(bids, asks []priceLevel) *sideMetric {
	midPrice := calcMidPrice(bids, asks)

	bidDepth := decimal.Zero
	for _, bid := range bids {
		if bid.Price.LessThan(calcMaxDepthPrice(midPrice, bidSide)) {
			break
		}
		bidDepth = bidDepth.Add(bid.Quantity)
	}

	askDepth := decimal.Zero
	for _, ask := range asks {
		if ask.Price.GreaterThan(calcMaxDepthPrice(midPrice, askSide)) {
			break
		}
		askDepth = askDepth.Add(ask.Quantity)
	}

	return &sideMetric{
		Bid: bidDepth,
		Ask: askDepth,
	}
}

func calcSlippageLevels(bids, asks []priceLevel, levels []decimal.Decimal) []slippageLevel {
	midPrice := calcMidPrice(bids, asks)
	result := make([]slippageLevel, 0, len(levels))

	for _, size := range levels {
		bidPrice := calcExecutionPrice(asks, size)
		askPrice := calcExecutionPrice(bids, size)

		result = append(result, slippageLevel{
			Size: size,
			Slippage: &sideMetric{
				Bid: calcSlippage(bidPrice, midPrice),
				Ask: calcSlippage(askPrice, midPrice),
			},
		})
	}

	return result
}

func calcExecutionPrice(orders []priceLevel, size decimal.Decimal) decimal.Decimal {
	filled := decimal.Zero

	for _, order := range orders {
		levelSize := order.Quantity.Mul(order.Price)
		filled = filled.Add(levelSize)

		if filled.GreaterThanOrEqual(size) {
			return order.Price
		}
	}

	return decimal.Zero
}

func calcSlippage(expectedPrice, midPrice decimal.Decimal) decimal.Decimal {
	return expectedPrice.Sub(midPrice).Div(midPrice).Abs()
}

func calcMaxDepthPrice(price decimal.Decimal, mode orderSide) decimal.Decimal {
	if mode == bidSide {
		return price.Mul(decimal.NewFromFloat(1 - midPriceBuffer))
	}

	return price.Mul(decimal.NewFromFloat(1 + midPriceBuffer))
}
