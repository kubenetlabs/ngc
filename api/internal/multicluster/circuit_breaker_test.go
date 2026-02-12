package multicluster

import (
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedByDefault(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)
	if cb.State() != StateClosed {
		t.Fatalf("expected StateClosed, got %d", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Fatal("expected StateClosed after 2 failures")
	}

	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatal("expected StateOpen after 3 failures")
	}
}

func TestCircuitBreaker_SuccessResetsClosed(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()
	cb.RecordFailure()

	if cb.State() != StateClosed {
		t.Fatal("expected StateClosed after success reset")
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)

	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatal("expected StateOpen")
	}

	time.Sleep(15 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Fatal("expected StateHalfOpen after reset timeout")
	}
}

func TestCircuitBreaker_HalfOpenSuccessCloses(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)

	cb.RecordFailure()
	time.Sleep(15 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Fatal("expected StateHalfOpen")
	}

	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Fatal("expected StateClosed after success in half-open")
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)

	cb.RecordFailure()
	time.Sleep(15 * time.Millisecond)

	// Should be half-open
	_ = cb.State()

	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatal("expected StateOpen after failure in half-open")
	}
}
