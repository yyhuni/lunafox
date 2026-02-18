package websocket

import "time"

// Backoff implements exponential backoff with a maximum cap.
type Backoff struct {
	base    time.Duration
	max     time.Duration
	current time.Duration
}

// NewBackoff creates a backoff with the given base and max delay.
func NewBackoff(base, max time.Duration) Backoff {
	return Backoff{
		base: base,
		max:  max,
	}
}

// Next returns the next backoff duration.
func (b *Backoff) Next() time.Duration {
	if b.current <= 0 {
		b.current = b.base
		return b.current
	}
	next := b.current * 2
	if next > b.max {
		next = b.max
	}
	b.current = next
	return b.current
}

// Reset clears the backoff to start over.
func (b *Backoff) Reset() {
	b.current = 0
}
