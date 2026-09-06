package subdomaindiscoveryruntime

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	subdomainspec "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

type Runtime struct{}

func New() *Runtime {
	return &Runtime{}
}

func (w *Runtime) Name() string { return subdomainspec.Name }

type resultArtifacts struct {
	resultArtifactPath string
	workspaceDir       string
}

// Execute consumes only the Target and file paths projected by the generated
// Engine API v2 facade. Every scanner process runs inside this Engine image.
func (w *Runtime) Execute(
	execCtx context.Context,
	target subdomainspec.Target,
	config subdomainspec.Config,
	workspace string,
	progress ProgressReporter,
	providerConfigPath string,
	wordlistPath string,
	bruteforceResolversPath string,
	resolveResolversPath string,
) (*resultArtifacts, error) {
	if execCtx == nil {
		return nil, fmt.Errorf("execution context is required")
	}
	executionProgress, err := newExecutionProgress(execCtx, progress)
	if err != nil {
		return nil, err
	}
	run, err := initializeDiscoveryRun(target, config, workspace, executionProgress, containerToolExecutor{}, providerConfigPath, wordlistPath, bruteforceResolversPath, resolveResolversPath)
	if err != nil {
		return nil, err
	}
	combinedOutcome := w.runAllStages(execCtx, run)
	artifacts := &resultArtifacts{
		resultArtifactPath: combinedOutcome.resultArtifactPath,
		workspaceDir:       workspace,
	}
	if err := stageOutcomeError(execCtx, combinedOutcome); err != nil {
		return artifacts, err
	}
	return artifacts, nil
}

func stageOutcomeError(execCtx context.Context, outcome stageOutcome) error {
	if len(outcome.failed) == 0 {
		return nil
	}
	message := "one or more enabled stages failed: " + strings.Join(outcome.failed, "; ")
	if len(outcome.success) == 0 {
		message = "no stage command completed successfully"
	}
	if execCtx != nil {
		if contextErr := execCtx.Err(); errors.Is(contextErr, context.Canceled) || errors.Is(contextErr, context.DeadlineExceeded) {
			// Signal cancellation owns the final handler result even if a
			// concurrent stage happened to publish another failure first.
			return fmt.Errorf("%s: %w", message, contextErr)
		}
	}
	if outcome.failureCause != nil && (errors.Is(outcome.failureCause, context.Canceled) || errors.Is(outcome.failureCause, context.DeadlineExceeded)) {
		return fmt.Errorf("%s: %w", message, outcome.failureCause)
	}
	return fmt.Errorf("%s", message)
}

func initializeDiscoveryRun(
	target subdomainspec.Target,
	engineConfig subdomainspec.Config,
	workDir string,
	progress executionProgress,
	executor toolExecutor,
	providerConfigPath string,
	wordlistPath string,
	bruteforceResolversPath string,
	resolveResolversPath string,
) (*discoveryRun, error) {
	if executor == nil {
		return nil, fmt.Errorf("tool executor is required")
	}
	log.Printf("discovery run prepared target.name=%s target.type=%s", target.Value, target.Type)
	return &discoveryRun{
		domain:                  target.Value,
		typedConfig:             engineConfig,
		workspaceDir:            workDir,
		progress:                progress,
		executor:                executor,
		providerConfigPath:      providerConfigPath,
		wordlistPath:            wordlistPath,
		bruteforceResolversPath: bruteforceResolversPath,
		resolveResolversPath:    resolveResolversPath,
	}, nil
}
