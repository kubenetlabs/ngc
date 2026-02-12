package multicluster

import (
	"sync"
	"time"
)

// BreakerState represents the circuit breaker state.
type BreakerState int

const (
	StateClosed   BreakerState = iota // Normal — requests pass through
	StateOpen                          // Tripped — requests fail fast
	StateHalfOpen                      // Testing — one request allowed
)

// CircuitBreaker implements a simple per-cluster circuit breaker.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            BreakerState
	failures         int
	maxFailures      int
	resetTimeout     time.Duration
	lastFailureTime  time.Time
}

// NewCircuitBreaker creates a circuit breaker that opens after maxFailures
// consecutive failures and resets after resetTimeout.
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        StateClosed,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
	}
}

// State returns the current circuit breaker state, transitioning from Open
// to HalfOpen if the reset timeout has elapsed.
func (cb *CircuitBreaker) State() BreakerState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && time.Since(cb.lastFailureTime) >= cb.resetTimeout {
		cb.state = StateHalfOpen
	}
	return cb.state
}

// AllowRequest returns true if the circuit breaker allows a request.
// In HalfOpen state, only one probe request is allowed.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			return true // allow the probe request
		}
		return false
	case StateHalfOpen:
		return false // only allow one probe (already allowed on transition)
	default:
		return false
	}
}

// RecordSuccess resets the circuit breaker to Closed.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = StateClosed
}

// RecordFailure increments the failure count and opens the circuit if
// maxFailures is reached.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = StateOpen
	}
}
