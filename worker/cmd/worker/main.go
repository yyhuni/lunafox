package main

import (
	"context"
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

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	if err := pkg.InitLogger(cfg.LogLevel); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer pkg.Sync()

	pkg.Logger.Info("Worker starting",
		zap.Int("scanId", cfg.ScanID),
		zap.Int("targetId", cfg.TargetID),
		zap.String("targetName", cfg.TargetName),
		zap.String("targetType", cfg.TargetType),
		zap.String("workflow", cfg.WorkflowName))

	// Create server client (implements ServerClient, ResultSaver)
	serverClient := server.NewClient(cfg.ServerURL, cfg.ServerToken)

	// Create workflow params
	params := &workflow.Params{
		ScanID:       cfg.ScanID,
		TargetID:     cfg.TargetID,
		TargetName:   cfg.TargetName,
		TargetType:   cfg.TargetType,
		WorkDir:      cfg.WorkspaceDir,
		ScanConfig:   cfg.Config,
		ServerClient: serverClient,
	}

	// Get and execute the workflow
	w := workflow.Get(cfg.WorkflowName, cfg.WorkspaceDir)
	if w == nil {
		pkg.Logger.Error("Unknown workflow name", zap.String("workflow", cfg.WorkflowName))
		os.Exit(1)
	}

	// Execute workflow
	// Status is managed by Agent based on container exit code:
	// - exit 0 = completed
	// - exit 1 = failed
	output, execErr := w.Execute(params)
	if execErr != nil {
		pkg.Logger.Error("Workflow execution failed",
			zap.Int("scanId", cfg.ScanID),
			zap.String("workflow", cfg.WorkflowName),
			zap.String("targetName", cfg.TargetName),
			zap.String("targetType", cfg.TargetType),
			zap.Error(execErr))
		os.Exit(1) // Agent will detect exit code and set status to "failed"
	}

	// Save results
	ctx := context.Background()
	if err := w.SaveResults(ctx, serverClient, params, output); err != nil {
		fileCount := 0
		if output != nil {
			if files, ok := output.Data.([]string); ok {
				fileCount = len(files)
			}
		}
		pkg.Logger.Error("Failed to save results",
			zap.Int("scanId", cfg.ScanID),
			zap.String("workflow", cfg.WorkflowName),
			zap.Int("files", fileCount),
			zap.Error(err))
		os.Exit(1)
	}

	pkg.Logger.Info("Worker completed successfully",
		zap.Int("scanId", cfg.ScanID))
	// exit 0 - Agent will detect and set status to "completed"
}
