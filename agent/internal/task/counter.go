package task

import "sync/atomic"

// Counter tracks running task count.
type Counter struct {
	value int64
}

// Inc increments the counter.
func (c *Counter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

// Dec decrements the counter.
func (c *Counter) Dec() {
	atomic.AddInt64(&c.value, -1)
}

// Count returns current count.
func (c *Counter) Count() int {
	return int(atomic.LoadInt64(&c.value))
}
