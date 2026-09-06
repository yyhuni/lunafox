package admission

import (
	"math"
	"sync"
	"time"
)

const (
	defaultRatePerMinute = 60.0
	defaultBurst         = 10.0
	defaultConcurrency   = 4
)

// Limiter combines a per-key token bucket with a bounded concurrency counter.
type Limiter struct {
	mu             sync.Mutex
	ratePerSecond  float64
	burst          float64
	maxConcurrency int
	clock          func() time.Time
	buckets        map[int]*bucket
	concurrent     map[int]int
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// NewLimiter creates the first-release MCP admission policy.
func NewLimiter(ratePerMinute float64, burst, maxConcurrency int) *Limiter {
	if ratePerMinute <= 0 {
		ratePerMinute = defaultRatePerMinute
	}
	if burst <= 0 {
		burst = int(defaultBurst)
	}
	if maxConcurrency <= 0 {
		maxConcurrency = defaultConcurrency
	}
	return &Limiter{
		ratePerSecond:  ratePerMinute / 60,
		burst:          float64(burst),
		maxConcurrency: maxConcurrency,
		clock:          time.Now,
		buckets:        make(map[int]*bucket),
		concurrent:     make(map[int]int),
	}
}

// Allow consumes one request token and returns the retry delay when rejected.
func (limiter *Limiter) Allow(keyID int) (allowed bool, retryAfter time.Duration) {
	if keyID <= 0 {
		return false, time.Minute
	}
	now := limiter.clock()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	state := limiter.buckets[keyID]
	if state == nil {
		state = &bucket{tokens: limiter.burst, lastSeen: now}
		limiter.buckets[keyID] = state
	}
	if elapsed := now.Sub(state.lastSeen).Seconds(); elapsed > 0 {
		state.tokens = math.Min(limiter.burst, state.tokens+elapsed*limiter.ratePerSecond)
		state.lastSeen = now
	}
	if state.tokens >= 1 {
		state.tokens--
		return true, 0
	}
	missing := 1 - state.tokens
	seconds := missing / limiter.ratePerSecond
	return false, time.Duration(math.Ceil(seconds*1000)) * time.Millisecond
}

// RetryAfterSeconds converts a duration to the HTTP Retry-After delta format.
// A rejected request always gets at least one second so clients do not spin.
func RetryAfterSeconds(delay time.Duration) int {
	if delay <= 0 {
		return 1
	}
	seconds := int(math.Ceil(delay.Seconds()))
	if seconds < 1 {
		return 1
	}
	return seconds
}

// Acquire reserves one tool execution slot. The returned release function is
// idempotent so cancellation and normal completion cannot double-decrement.
func (limiter *Limiter) Acquire(keyID int) (release func(), allowed bool) {
	if keyID <= 0 {
		return func() {}, false
	}
	limiter.mu.Lock()
	if limiter.concurrent[keyID] >= limiter.maxConcurrency {
		limiter.mu.Unlock()
		return func() {}, false
	}
	limiter.concurrent[keyID]++
	limiter.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			limiter.mu.Lock()
			if limiter.concurrent[keyID] > 1 {
				limiter.concurrent[keyID]--
			} else {
				delete(limiter.concurrent, keyID)
			}
			limiter.mu.Unlock()
		})
	}, true
}

// Limits returns the configured policy for documentation and tests.
func (limiter *Limiter) Limits() (ratePerMinute float64, burst, maxConcurrency int) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	return limiter.ratePerSecond * 60, int(limiter.burst), limiter.maxConcurrency
}
