package gateway

import (
	"sync"
	"time"
)

type circuitState uint8
const (
	closed circuitState = iota
	open
	halfOpen
)

// CircuitBreaker is a compact three-state breaker. A single trial is admitted
// in half-open state, preventing a recovering provider from being flooded.
type CircuitBreaker struct {
	mu sync.Mutex
	state circuitState
	failures, threshold int
	openedAt time.Time
	cooldown time.Duration
	trialActive bool
}

func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	if threshold < 1 { threshold = 3 }
	if cooldown <= 0 { cooldown = 10 * time.Second }
	return &CircuitBreaker{threshold: threshold, cooldown: cooldown}
}

func (c *CircuitBreaker) Allow(now time.Time) bool {
	c.mu.Lock(); defer c.mu.Unlock()
	if c.state == open && now.Sub(c.openedAt) >= c.cooldown { c.state = halfOpen }
	if c.state == open || (c.state == halfOpen && c.trialActive) { return false }
	if c.state == halfOpen { c.trialActive = true }
	return true
}

func (c *CircuitBreaker) Success() {
	c.mu.Lock(); defer c.mu.Unlock()
	c.state, c.failures, c.trialActive = closed, 0, false
}

func (c *CircuitBreaker) Failure(now time.Time) {
	c.mu.Lock(); defer c.mu.Unlock()
	c.trialActive = false
	c.failures++
	if c.state == halfOpen || c.failures >= c.threshold { c.state, c.openedAt = open, now }
}

// CancelTrial releases a half-open reservation when the caller, rather than the
// provider, canceled an attempt. Client cancellation must not poison health.
func (c *CircuitBreaker) CancelTrial() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == halfOpen {
		c.trialActive = false
	}
}

func (c *CircuitBreaker) State(now time.Time) string {
	c.mu.Lock(); defer c.mu.Unlock()
	if c.state == open && now.Sub(c.openedAt) >= c.cooldown { return "half_open" }
	return [...]string{"closed", "open", "half_open"}[c.state]
}
