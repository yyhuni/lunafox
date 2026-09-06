package admission

import (
	"testing"
	"time"
)

func TestLimiterEnforcesBurstAndRateBudget(t *testing.T) {
	limiter := NewLimiter(60, 10, 4)
	for call := 0; call < 10; call++ {
		if allowed, _ := limiter.Allow(9); !allowed {
			t.Fatalf("burst request %d was rejected", call+1)
		}
	}
	allowed, retryAfter := limiter.Allow(9)
	if allowed {
		t.Fatal("request beyond burst was allowed")
	}
	if retryAfter < time.Second {
		t.Fatalf("retry delay = %s, want at least one second", retryAfter)
	}
	if got := RetryAfterSeconds(retryAfter); got < 1 {
		t.Fatalf("Retry-After seconds = %d", got)
	}

	if allowed, _ := limiter.Allow(10); !allowed {
		t.Fatal("different key should have an independent budget")
	}
}

func TestLimiterConcurrencyReleaseIsIdempotent(t *testing.T) {
	limiter := NewLimiter(60, 10, 2)
	releaseOne, ok := limiter.Acquire(12)
	if !ok {
		t.Fatal("first permit was rejected")
	}
	releaseTwo, ok := limiter.Acquire(12)
	if !ok {
		t.Fatal("second permit was rejected")
	}
	if _, ok := limiter.Acquire(12); ok {
		t.Fatal("permit beyond concurrency limit was allowed")
	}

	releaseOne()
	releaseOne()
	releaseTwo()
	release, ok := limiter.Acquire(12)
	if !ok {
		t.Fatal("permit was not released")
	}
	release()
}

func TestLimiterRejectsInvalidKeyIDs(t *testing.T) {
	limiter := NewLimiter(60, 10, 4)
	if allowed, _ := limiter.Allow(0); allowed {
		t.Fatal("zero key ID was allowed")
	}
	if _, allowed := limiter.Acquire(-1); allowed {
		t.Fatal("negative key ID was allowed")
	}
}
