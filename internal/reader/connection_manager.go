package reader

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const maxReconnectDelay = 30 * time.Second
const reconnectBaseDelay = 1 * time.Second

type connectionManager struct {
	reconnectDelay *time.Duration
	errorCh        chan<- error
	mu             *sync.RWMutex
}

func newConnectionManager(errorCh chan<- error, mu *sync.RWMutex, delayPtr *time.Duration) *connectionManager {
	return &connectionManager{
		reconnectDelay: delayPtr,
		errorCh:        errorCh,
		mu:             mu,
	}
}

func (cm *connectionManager) manageConnection(
	ctx context.Context,
	exchangeName string,
	readLoopFn func(ctx context.Context) error,
	reconnectFn func(ctx context.Context) error,
	resubscribeFn func() error,
) {
	for {
		err := readLoopFn(ctx)

		if ctx.Err() != nil {
			return
		}

		cm.errorCh <- fmt.Errorf("reconnecting to %s: %v", exchangeName, err)

		cm.mu.RLock()
		delay := *cm.reconnectDelay
		cm.mu.RUnlock()

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		err = reconnectFn(ctx)
		if err != nil {
			cm.errorCh <- err
			cm.backoff()
			continue
		}

		if resubscribeFn != nil {
			err = resubscribeFn()
			if err != nil {
				cm.errorCh <- err
				cm.backoff()
				continue
			}
		}

		cm.mu.Lock()
		*cm.reconnectDelay = reconnectBaseDelay
		cm.mu.Unlock()
	}
}

func (cm *connectionManager) backoff() {
	cm.mu.Lock()
	if *cm.reconnectDelay < maxReconnectDelay {
		*cm.reconnectDelay *= 2
	}
	cm.mu.Unlock()
}
