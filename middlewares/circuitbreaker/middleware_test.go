package circuitbreaker

import (
	"testing"
	"time"

	"github.com/ink-yht-code/gint/logger"
)

func init() {
	_ = logger.Init(logger.DefaultConfig())
}

func TestHalfOpenRequestLimit(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          5 * time.Millisecond,
		HalfOpenRequests: 1,
	})

	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatalf("state = %v, want %v", cb.State(), StateOpen)
	}

	time.Sleep(10 * time.Millisecond)

	if !cb.Allow() {
		t.Fatal("first half-open request should be allowed")
	}
	if cb.Allow() {
		t.Fatal("second half-open request should be rejected by limit")
	}
}

func TestHalfOpenSlotReleasedAfterRecordSuccess(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          5 * time.Millisecond,
		HalfOpenRequests: 1,
	})

	cb.RecordFailure()
	time.Sleep(10 * time.Millisecond)

	if !cb.Allow() {
		t.Fatal("first half-open request should be allowed")
	}
	cb.RecordSuccess()

	if !cb.Allow() {
		t.Fatal("half-open slot should be released after RecordSuccess")
	}
}
