package app

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/config"
	"github.com/yyhuni/lunafox/agent/internal/docker"
	"github.com/yyhuni/lunafox/agent/internal/domain"
	"github.com/yyhuni/lunafox/agent/internal/health"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	"github.com/yyhuni/lunafox/agent/internal/metrics"
	"github.com/yyhuni/lunafox/agent/internal/protocol"
	"github.com/yyhuni/lunafox/agent/internal/task"
	"github.com/yyhuni/lunafox/agent/internal/update"
	agentws "github.com/yyhuni/lunafox/agent/internal/websocket"
	"go.uber.org/zap"
)

func Run(ctx context.Context, cfg config.Config, wsURL string) error {
	configUpdater := config.NewUpdater(cfg)

	version := cfg.AgentVersion
	hostname := os.Getenv("AGENT_HOSTNAME")
	if hostname == "" {
		var err error
		hostname, err = os.Hostname()
		if err != nil || hostname == "" {
			hostname = "unknown"
		}
	}

	logger.Log.Info("agent starting",
		zap.String("version", version),
		zap.String("hostname", hostname),
		zap.String("server", cfg.ServerURL),
		zap.String("ws", wsURL),
		zap.Int("maxTasks", cfg.MaxTasks),
		zap.Int("cpuThreshold", cfg.CPUThreshold),
		zap.Int("memThreshold", cfg.MemThreshold),
		zap.Int("diskThreshold", cfg.DiskThreshold),
	)

	client := agentws.NewClient(wsURL, cfg.APIKey)
	collector := metrics.NewCollector()
	healthManager := health.NewManager()
	taskCounter := &task.Counter{}
	heartbeat := agentws.NewHeartbeatSender(client, collector, healthManager, version, hostname, taskCounter.Count)

	taskClient := task.NewClient(cfg.ServerURL, cfg.APIKey)
	puller := task.NewPuller(taskClient, collector, taskCounter, cfg.MaxTasks, cfg.CPUThreshold, cfg.MemThreshold, cfg.DiskThreshold)

	taskQueue := make(chan *domain.Task, cfg.MaxTasks)
	puller.SetOnTask(func(t *domain.Task) {
		logger.Log.Info("task received",
			zap.Int("taskId", t.ID),
			zap.Int("scanId", t.ScanID),
			zap.String("workflow", t.WorkflowName),
			zap.Int("stage", t.Stage),
			zap.String("target", t.TargetName),
		)
		taskQueue <- t
	})

	dockerClient, err := docker.NewClient()
	if err != nil {
		logger.Log.Warn("docker client unavailable", zap.Error(err))
	} else {
		logger.Log.Info("docker client ready")
	}

	workerToken := os.Getenv("WORKER_TOKEN")
	if workerToken == "" {
		return errors.New("WORKER_TOKEN environment variable is required")
	}
	logger.Log.Info("worker token loaded")

	executor := task.NewExecutor(dockerClient, taskClient, taskCounter, cfg.ServerURL, workerToken)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := executor.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			logger.Log.Error("executor shutdown error", zap.Error(err))
		}
	}()

	updater := update.NewUpdater(dockerClient, healthManager, puller, executor, configUpdater, cfg.APIKey, workerToken)

	handler := agentws.NewHandler()
	handler.OnTaskAvailable(puller.NotifyTaskAvailable)
	handler.OnTaskCancel(func(taskID int) {
		logger.Log.Info("task cancel requested", zap.Int("taskId", taskID))
		executor.MarkCancelled(taskID)
		executor.CancelTask(taskID)
	})
	handler.OnConfigUpdate(func(payload protocol.ConfigUpdatePayload) {
		logger.Log.Info("config update received",
			zap.String("maxTasks", formatOptionalInt(payload.MaxTasks)),
			zap.String("cpuThreshold", formatOptionalInt(payload.CPUThreshold)),
			zap.String("memThreshold", formatOptionalInt(payload.MemThreshold)),
			zap.String("diskThreshold", formatOptionalInt(payload.DiskThreshold)),
		)
		cfgUpdate := config.Update{
			MaxTasks:      payload.MaxTasks,
			CPUThreshold:  payload.CPUThreshold,
			MemThreshold:  payload.MemThreshold,
			DiskThreshold: payload.DiskThreshold,
		}
		configUpdater.Apply(cfgUpdate)
		puller.UpdateConfig(cfgUpdate.MaxTasks, cfgUpdate.CPUThreshold, cfgUpdate.MemThreshold, cfgUpdate.DiskThreshold)
	})
	handler.OnUpdateRequired(updater.HandleUpdateRequired)
	client.SetOnMessage(handler.Handle)

	logger.Log.Info("starting heartbeat sender")
	go heartbeat.Start(ctx)
	logger.Log.Info("starting task puller")
	go func() {
		_ = puller.Run(ctx)
	}()
	logger.Log.Info("starting task executor")
	go executor.Start(ctx, taskQueue)

	logger.Log.Info("connecting to server websocket")
	if err := client.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func formatOptionalInt(value *int) string {
	if value == nil {
		return "nil"
	}
	return strconv.Itoa(*value)
}
