package portscanruntime

import (
	"context"
	"fmt"
	"path/filepath"

	portscancontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

func (r *Runtime) runNaabuPassiveStage(ctx context.Context, run *portScanRun) (string, error) {
	if !run.config.NaabuPassive.Enabled {
		return "", run.progress.Report(ctx, fmt.Sprintf("skip stage %s %s reason=%s", "2/2", portscancontract.SectionNaabuPassive, "disabled"))
	}

	passiveConfig := run.config.NaabuPassive
	outputPath := filepath.Join(run.workDir, "naabu_passive.jsonl")
	command, err := buildNaabuPassiveCommand(run.candidatePath, outputPath, passiveConfig)
	if err != nil {
		return "", err
	}
	if err := executeNaabuStage(ctx, run, "2/2", portscancontract.SectionNaabuPassive, "passive port scan", command); err != nil {
		return "", err
	}
	return outputPath, nil
}
