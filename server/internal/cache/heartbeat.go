package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

// HeartbeatData represents agent heartbeat information
type HeartbeatData struct {
	CPU       float64       `json:"cpu"`
	Mem       float64       `json:"mem"`
	Disk      float64       `json:"disk"`
	Tasks     int           `json:"tasks"`
	Version   string        `json:"version"`
	Hostname  string        `json:"hostname"`
	Uptime    int64         `json:"uptime"`
	Health    *HealthStatus `json:"health,omitempty"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// HealthStatus represents agent health in the heartbeat cache.
type HealthStatus = agentdomain.HealthStatus

// HeartbeatCache defines the interface for heartbeat cache operations
type HeartbeatCache interface {
	Set(ctx context.Context, agentID int, data *HeartbeatData) error
	Get(ctx context.Context, agentID int) (*HeartbeatData, error)
	Delete(ctx context.Context, agentID int) error
}

// heartbeatCache implements HeartbeatCache
type heartbeatCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewHeartbeatCache creates a new heartbeat cache
func NewHeartbeatCache(client *redis.Client) HeartbeatCache {
	return &heartbeatCache{
		client: client,
		ttl:    15 * time.Second, // 15 second TTL as per spec
	}
}

// Set stores heartbeat data in Redis
func (c *heartbeatCache) Set(ctx context.Context, agentID int, data *HeartbeatData) error {
	key := fmt.Sprintf("agent:%d:heartbeat", agentID)

	// Set updated_at to current time
	data.UpdatedAt = time.Now().UTC()

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat data: %w", err)
	}

	// Store in Redis with TTL
	return c.client.Set(ctx, key, jsonData, c.ttl).Err()
}

// Get retrieves heartbeat data from Redis
func (c *heartbeatCache) Get(ctx context.Context, agentID int) (*HeartbeatData, error) {
	key := fmt.Sprintf("agent:%d:heartbeat", agentID)

	// Get from Redis
	jsonData, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Key doesn't exist
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get heartbeat data: %w", err)
	}

	// Unmarshal JSON
	var data HeartbeatData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal heartbeat data: %w", err)
	}

	return &data, nil
}

// Delete removes heartbeat data from Redis
func (c *heartbeatCache) Delete(ctx context.Context, agentID int) error {
	key := fmt.Sprintf("agent:%d:heartbeat", agentID)
	return c.client.Del(ctx, key).Err()
}
