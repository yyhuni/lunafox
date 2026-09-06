package application

import (
	"context"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

// OutboxStore owns claim and terminal state changes for immutable producer
// envelopes. Calls are intentionally independent of provider I/O.
type OutboxStore interface {
	Claim(context.Context, string, time.Time, time.Duration, int) ([]domain.OutboxEvent, error)
	MarkPublished(context.Context, int64, string, time.Time) error
	MarkUnsupported(context.Context, int64, string, string, time.Time) error
	ReleaseLease(context.Context, int64, string, time.Time) error
	DeletePublishedBefore(context.Context, domain.RetentionBatch) (int64, error)
}
