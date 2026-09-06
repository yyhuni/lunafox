package portscanruntime

import (
	"context"
	"fmt"

	portscancontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

type Runtime struct{}

func New() *Runtime {
	return &Runtime{}
}

func (r *Runtime) Name() string { return portscancontract.Name }

type resultArtifacts struct {
	resultArtifactPaths []string
	workspaceDir        string
}

// Execute consumes only values already projected by the generated Engine API
// v2 author facade. The Engine owns naabu's process lifecycle inside its image.
func (r *Runtime) Execute(
	ctx context.Context,
	config portscancontract.Config,
	target portscancontract.Target,
	subdomainsPath string,
	workDir string,
	progress ProgressReporter,
) (*resultArtifacts, error) {
	if ctx == nil {
		return nil, fmt.Errorf("execution context is required")
	}
	if progress == nil {
		return nil, fmt.Errorf("engine progress port is required")
	}
	run, err := initializePortScan(ctx, config, target, subdomainsPath, workDir, progress, containerNaabuExecutor{})
	if err != nil {
		return nil, err
	}
	resultPaths, err := r.runAllStages(ctx, run)
	if err != nil {
		return &resultArtifacts{resultArtifactPaths: resultPaths, workspaceDir: workDir}, err
	}
	return &resultArtifacts{resultArtifactPaths: resultPaths, workspaceDir: workDir}, nil
}

func initializePortScan(
	ctx context.Context,
	config portscancontract.Config,
	target portscancontract.Target,
	subdomainsPath string,
	workDir string,
	progress ProgressReporter,
	executor naabuExecutor,
) (*portScanRun, error) {
	if executor == nil {
		return nil, fmt.Errorf("naabu executor is required")
	}
	candidatePath, hostCount, err := prepareTargetCandidates(ctx, target, subdomainsPath, workDir)
	if err != nil {
		return nil, err
	}
	return &portScanRun{config: config, candidatePath: candidatePath, hostCount: hostCount, workDir: workDir, progress: progress, executor: executor}, nil
}
