package app

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/config"
	"github.com/yyhuni/lunafox/agent/internal/docker"
	"github.com/yyhuni/lunafox/agent/internal/domain"
	"github.com/yyhuni/lunafox/agent/internal/health"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	"github.com/yyhuni/lunafox/agent/internal/metrics"
	agentruntime "github.com/yyhuni/lunafox/agent/internal/runtime"
	"github.com/yyhuni/lunafox/agent/internal/task"
	"github.com/yyhuni/lunafox/agent/internal/update"
	"go.uber.org/zap"
)

func Run(ctx context.Context, cfg config.Config) error {
	configUpdater := config.NewUpdater(cfg)

	agentVersion := cfg.AgentVersion
	workerVersion := cfg.WorkerVersion
	hostname := os.Getenv("AGENT_HOSTNAME")
	if hostname == "" {
		var err error
		hostname, err = os.Hostname()
		if err != nil || hostname == "" {
			hostname = "unknown"
		}
	}

	logger.Log.Info("agent starting",
		zap.String("agentVersion", agentVersion),
		zap.String("workerVersion", workerVersion),
		zap.String("hostname", hostname),
		zap.String("runtimeGrpc", cfg.RuntimeGRPCURL),
		zap.Int("maxTasks", cfg.MaxTasks),
		zap.Int("cpuThreshold", cfg.CPUThreshold),
		zap.Int("memThreshold", cfg.MemThreshold),
		zap.Int("diskThreshold", cfg.DiskThreshold),
	)

	runtimeClient := agentruntime.NewClient(cfg.RuntimeGRPCURL, cfg.APIKey)
	workerRuntimeSocket := resolveWorkerRuntimeSocketPath()
	collector := metrics.NewCollector()
	healthManager := health.NewManager()
	taskCounter := &task.Counter{}
	heartbeat := agentruntime.NewHeartbeatSender(
		runtimeClient,
		collector,
		healthManager,
		agentVersion,
		workerVersion,
		hostname,
		taskCounter.Count,
	)

	puller := task.NewPuller(runtimeClient, collector, taskCounter, cfg.MaxTasks, cfg.CPUThreshold, cfg.MemThreshold, cfg.DiskThreshold)

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

	workerToken := strings.TrimSpace(os.Getenv("WORKER_TOKEN"))
	if workerToken == "" {
		logger.Log.Info("WORKER_TOKEN not configured; worker HTTP runtime token is deprecated")
	}

	executor := task.NewExecutor(dockerClient, runtimeClient, taskCounter, workerRuntimeSocket)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := executor.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			logger.Log.Error("executor shutdown error", zap.Error(err))
		}
	}()

	updater := update.NewUpdater(dockerClient, healthManager, puller, executor, configUpdater, cfg.APIKey, workerToken)

	runtimeClient.OnTaskCancel(func(taskID int) {
		logger.Log.Info("task cancel requested", zap.Int("taskId", taskID))
		executor.MarkCancelled(taskID)
		executor.CancelTask(taskID)
	})
	runtimeClient.OnConfigUpdate(func(payload domain.ConfigUpdate) {
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
	runtimeClient.OnUpdateRequired(updater.HandleUpdateRequired)

	logger.Log.Info("starting worker runtime UDS server", zap.String("socket", workerRuntimeSocket))
	workerRuntimeErrCh := make(chan error, 1)
	go func() {
		workerRuntimeErrCh <- agentruntime.RunWorkerRuntimeServer(ctx, workerRuntimeSocket, runtimeClient)
	}()

	logger.Log.Info("starting heartbeat sender")
	go heartbeat.Start(ctx)
	logger.Log.Info("starting task puller")
	go func() {
		_ = puller.Run(ctx)
	}()
	logger.Log.Info("starting task executor")
	go executor.Start(ctx, taskQueue)

	logger.Log.Info("connecting to runtime gRPC stream")
	runtimeErrCh := make(chan error, 1)
	go func() {
		runtimeErrCh <- runtimeClient.Run(ctx)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-runtimeErrCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			runtimeErrCh = nil
		case err := <-workerRuntimeErrCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			workerRuntimeErrCh = nil
		}
		if runtimeErrCh == nil && workerRuntimeErrCh == nil {
			return nil
		}
	}
}

func formatOptionalInt(value *int) string {
	if value == nil {
		return "nil"
	}
	return strconv.Itoa(*value)
}

func resolveWorkerRuntimeSocketPath() string {
	path := strings.TrimSpace(os.Getenv("LUNAFOX_RUNTIME_SOCKET"))
	if path == "" {
		return agentruntime.DefaultWorkerRuntimeSocketPath
	}
	return path
}
