package storage

import (
	"sync"
	"time"
)

type rateLimiter struct {
	lastWrite     map[string]time.Time
	writeInterval time.Duration
	mu            sync.RWMutex
}

func newRateLimiter(writeInterval time.Duration) *rateLimiter {
	return &rateLimiter{
		lastWrite:     make(map[string]time.Time),
		writeInterval: writeInterval,
		mu:            sync.RWMutex{},
	}
}

func (rl *rateLimiter) ShouldWrite(key string) bool {
	rl.mu.RLock()
	lastTime, exists := rl.lastWrite[key]
	rl.mu.RUnlock()

	if !exists {
		return true
	}

	return time.Since(lastTime) >= rl.writeInterval
}

func (rl *rateLimiter) MarkWritten(key string) {
	rl.mu.Lock()
	rl.lastWrite[key] = time.Now()
	rl.mu.Unlock()
}
