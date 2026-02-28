package runtime

import (
	"context"
	"time"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	"github.com/yyhuni/lunafox/agent/internal/health"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	"github.com/yyhuni/lunafox/agent/internal/metrics"
	"go.uber.org/zap"
)

type HeartbeatSender struct {
	client      *Client
	collector   *metrics.Collector
	health      *health.Manager
	version     string
	hostname    string
	startedAt   time.Time
	taskCount   func() int
	interval    time.Duration
	sendTimeout time.Duration
}

func NewHeartbeatSender(client *Client, collector *metrics.Collector, healthManager *health.Manager, version, hostname string, taskCount func() int) *HeartbeatSender {
	return &HeartbeatSender{
		client:      client,
		collector:   collector,
		health:      healthManager,
		version:     version,
		hostname:    hostname,
		startedAt:   time.Now(),
		taskCount:   taskCount,
		interval:    5 * time.Second,
		sendTimeout: 3 * time.Second,
	}
}

func (sender *HeartbeatSender) Start(ctx context.Context) {
	ticker := time.NewTicker(sender.interval)
	defer ticker.Stop()

	sender.sendOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sender.sendOnce(ctx)
		}
	}
}

func (sender *HeartbeatSender) sendOnce(ctx context.Context) {
	if sender == nil || sender.client == nil || sender.collector == nil || sender.health == nil {
		return
	}

	cpu, mem, disk := sender.collector.Sample()
	uptime := int64(time.Since(sender.startedAt).Seconds())

	runningTasks := 0
	if sender.taskCount != nil {
		runningTasks = sender.taskCount()
	}

	healthStatus := sender.health.Get()
	payload := &runtimev1.Heartbeat{
		CpuUsage:      cpu,
		MemUsage:      mem,
		DiskUsage:     disk,
		RunningTasks:  int32(runningTasks),
		Version:       sender.version,
		Hostname:      sender.hostname,
		UptimeSeconds: uptime,
		Health: &runtimev1.HealthStatus{
			State:   healthStatus.State,
			Reason:  healthStatus.Reason,
			Message: healthStatus.Message,
		},
	}

	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), sender.sendTimeout)
	defer cancel()
	if err := sender.client.SendHeartbeat(sendCtx, payload); err != nil {
		logger.Log.Debug("failed to send runtime heartbeat", zap.Error(err))
	}
}
