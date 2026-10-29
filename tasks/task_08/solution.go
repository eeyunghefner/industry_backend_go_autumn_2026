package main

import (
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	var last time.Time
	if clock != nil {
		last = clock.Now()
	}
	return &Limiter{clock: clock, burst: burst, rate: ratePerSec, tokens: float64(burst), last: last}
}
func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.burst <= 0 || l.clock == nil {
		return false
	}

	now := l.clock.Now()
	passed := now.Sub(l.last).Seconds()
	if passed > 0 && l.rate > 0 {
		l.tokens += passed * l.rate
		l.last = now
	}
	if l.tokens > float64(l.burst) {
		l.tokens = float64(l.burst)
	}

	if n <= 0 || n > l.burst || float64(n) > l.tokens {
		return false
	}

	l.tokens -= float64(n)
	return true
}
