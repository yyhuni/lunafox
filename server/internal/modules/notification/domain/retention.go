package domain

import "time"

const NotificationRetention = 90 * 24 * time.Hour

// RetentionBatch bounds physical cleanup work and makes cleanup safe to retry.
type RetentionBatch struct {
	Limit  int
	Before time.Time
}
