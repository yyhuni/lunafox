package websocket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/health"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	"github.com/yyhuni/lunafox/agent/internal/metrics"
	"github.com/yyhuni/lunafox/agent/internal/protocol"
	"go.uber.org/zap"
)

// HeartbeatSender sends periodic heartbeat messages over WebSocket.
type HeartbeatSender struct {
	client     *Client
	collector  *metrics.Collector
	health     *health.Manager
	version    string
	hostname   string
	startedAt  time.Time
	taskCount  func() int
	interval   time.Duration
	lastSentAt time.Time
}

// NewHeartbeatSender creates a heartbeat sender.
func NewHeartbeatSender(client *Client, collector *metrics.Collector, healthManager *health.Manager, version, hostname string, taskCount func() int) *HeartbeatSender {
	return &HeartbeatSender{
		client:    client,
		collector: collector,
		health:    healthManager,
		version:   version,
		hostname:  hostname,
		startedAt: time.Now(),
		taskCount: taskCount,
		interval:  5 * time.Second,
	}
}

// Start begins sending heartbeats until context is canceled.
func (h *HeartbeatSender) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	h.sendOnce()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.sendOnce()
		}
	}
}

func (h *HeartbeatSender) sendOnce() {
	cpu, mem, disk := h.collector.Sample()
	uptime := int64(time.Since(h.startedAt).Seconds())
	tasks := 0
	if h.taskCount != nil {
		tasks = h.taskCount()
	}

	status := h.health.Get()
	payload := protocol.HeartbeatPayload{
		CPU:      cpu,
		Mem:      mem,
		Disk:     disk,
		Tasks:    tasks,
		Version:  h.version,
		Hostname: h.hostname,
		Uptime:   uptime,
		Health: protocol.HealthStatus{
			State:   status.State,
			Reason:  status.Reason,
			Message: status.Message,
			Since:   status.Since,
		},
	}

	msg := protocol.Message{
		Type:      protocol.MessageTypeHeartbeat,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Log.Warn("failed to marshal heartbeat message", zap.Error(err))
		return
	}
	if !h.client.Send(data) {
		logger.Log.Warn("failed to send heartbeat: client not connected")
	}
}
