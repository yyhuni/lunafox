package portscanruntime

import (
	"context"
	"fmt"
)

func executeNaabuStage(ctx context.Context, run *portScanRun, stageLabel, stageName, description string, command naabuCommand) error {
	if run == nil || run.progress == nil || run.executor == nil {
		return fmt.Errorf("port scan runtime dependencies are required")
	}
	if err := run.progress.Report(ctx, fmt.Sprintf("start stage %s %s: %s hosts=%d", stageLabel, stageName, description, run.hostCount)); err != nil {
		return err
	}
	if err := run.progress.Report(ctx, fmt.Sprintf(
		"start command stage=%s label=%s command=%s",
		stageName,
		commandLabelForProgress(command),
		renderNaabuCommand(command),
	)); err != nil {
		return err
	}
	if err := run.executor.run(ctx, command); err != nil {
		reportStageFailure(ctx, run, stageLabel, stageName, "command: "+err.Error())
		return err
	}
	if err := reportStageComplete(ctx, run, stageLabel, stageName, 1, 0, 1); err != nil {
		return err
	}
	return nil
}
