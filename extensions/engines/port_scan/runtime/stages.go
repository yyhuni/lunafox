package portscanruntime

import (
	"context"

	portscancontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

type portScanRun struct {
	config        portscancontract.Config
	candidatePath string
	hostCount     uint64
	workDir       string
	progress      ProgressReporter
	executor      naabuExecutor
}

func (r *Runtime) runAllStages(ctx context.Context, run *portScanRun) ([]string, error) {
	var resultPaths []string

	activeOutputPath, err := r.runNaabuActiveStage(ctx, run)
	if err != nil {
		return resultPaths, err
	}
	if activeOutputPath != "" {
		resultPaths = append(resultPaths, activeOutputPath)
	}

	passiveOutputPath, err := r.runNaabuPassiveStage(ctx, run)
	if err != nil {
		return resultPaths, err
	}
	if passiveOutputPath != "" {
		resultPaths = append(resultPaths, passiveOutputPath)
	}

	return resultPaths, nil
}
