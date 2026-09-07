package portscanruntime

import (
	"context"
	"fmt"
	"strings"
)

func reportStageFailure(ctx context.Context, run *portScanRun, stageLabel string, stageName string, reason string) {
	if run == nil || run.progress == nil {
		return
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "unknown"
	}
	_ = run.progress.Report(ctx, fmt.Sprintf("fail stage %s %s reason=%s", stageLabel, stageName, reason))
}

func reportStageComplete(ctx context.Context, run *portScanRun, stageLabel string, stageName string, success int, failed int, outputs int) error {
	if run == nil || run.progress == nil {
		return fmt.Errorf("engine progress port is required")
	}
	return run.progress.Report(ctx, fmt.Sprintf("complete stage %s %s success=%d failed=%d outputs=%d", stageLabel, stageName, success, failed, outputs))
}
