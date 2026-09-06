package portscanruntime

import (
	"context"
	"fmt"
	"path/filepath"

	portscancontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

func (r *Runtime) runNaabuActiveStage(ctx context.Context, run *portScanRun) (string, error) {
	if !run.config.NaabuActive.Enabled {
		return "", run.progress.Report(ctx, fmt.Sprintf("skip stage %s %s reason=%s", "1/2", portscancontract.SectionNaabuActive, "disabled"))
	}

	activeConfig := run.config.NaabuActive
	outputPath := filepath.Join(run.workDir, "naabu_active.jsonl")
	command, err := buildNaabuActiveCommand(run.candidatePath, outputPath, activeConfig)
	if err != nil {
		return "", err
	}
	if err := executeNaabuStage(ctx, run, "1/2", portscancontract.SectionNaabuActive, "active port scan", command); err != nil {
		return "", err
	}
	return outputPath, nil
}
