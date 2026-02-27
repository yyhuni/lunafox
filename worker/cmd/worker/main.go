package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/yyhuni/lunafox/worker/internal/config"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/yyhuni/lunafox/worker/internal/server"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	"go.uber.org/zap"

	// Import workflows to trigger init() registration
	_ "github.com/yyhuni/lunafox/worker/internal/workflow/subdomain_discovery"
)

type runtimeClient interface {
	server.ServerClient
	Close() error
}

var (
	loadConfig = config.Load
	initLogger = pkg.InitLogger
	syncLogger = pkg.Sync
	newClient  = func(socketPath, taskToken string, taskID int) (runtimeClient, error) {
		return server.NewRuntimeClient(socketPath, taskToken, taskID)
	}
	getWorkflow = workflow.Get
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Printf("Worker failed: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// Load configuration from environment variables
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize logger
	if err := initLogger(cfg.LogLevel); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer syncLogger()

	pkg.Logger.Info("Worker starting",
		zap.Int("taskId", cfg.TaskID),
		zap.Int("scanId", cfg.ScanID),
		zap.Int("targetId", cfg.TargetID),
		zap.String("targetName", cfg.TargetName),
		zap.String("targetType", cfg.TargetType),
		zap.String("workflow", cfg.WorkflowName))

	// Create runtime client (worker -> local agent UDS gRPC).
	agentRuntimeClient, err := newClient(cfg.AgentSocket, cfg.TaskToken, cfg.TaskID)
	if err != nil {
		return fmt.Errorf("create runtime client: %w", err)
	}
	defer func() {
		_ = agentRuntimeClient.Close()
	}()

	// Create workflow params
	params := &workflow.Params{
		ScanID:       cfg.ScanID,
		TargetID:     cfg.TargetID,
		TargetName:   cfg.TargetName,
		TargetType:   cfg.TargetType,
		WorkDir:      cfg.WorkspaceDir,
		ScanConfig:   cfg.Config,
		ServerClient: agentRuntimeClient,
	}

	// Get and execute the workflow
	w := getWorkflow(cfg.WorkflowName, cfg.WorkspaceDir)
	if w == nil {
		return fmt.Errorf("unknown workflow name: %s", cfg.WorkflowName)
	}

	// Execute workflow
	// Status is managed by Agent based on container exit code:
	// - exit 0 = completed
	// - exit 1 = failed
	output, execErr := w.Execute(params)
	if execErr != nil {
		return fmt.Errorf("execute workflow %s: %w", cfg.WorkflowName, execErr)
	}

	// Save results
	if err := w.SaveResults(ctx, agentRuntimeClient, params, output); err != nil {
		return fmt.Errorf("save workflow results %s: %w", cfg.WorkflowName, err)
	}

	pkg.Logger.Info("Worker completed successfully",
		zap.Int("scanId", cfg.ScanID))
	// exit 0 - Agent will detect and set status to "completed"
	return nil
}
