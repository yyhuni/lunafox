package application

import (
	"context"

	"github.com/yyhuni/lunafox/server/internal/cache"
)

// HeartbeatCachePort defines heartbeat cache operations required by runtime service.
type HeartbeatCachePort interface {
	Set(ctx context.Context, agentID int, data *cache.HeartbeatData) error
	Get(ctx context.Context, agentID int) (*cache.HeartbeatData, error)
	Delete(ctx context.Context, agentID int) error
}
