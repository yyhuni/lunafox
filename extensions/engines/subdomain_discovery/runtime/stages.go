package subdomaindiscoveryruntime

import (
	"context"

	subdomainspec "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

// discoveryRun holds shared inputs for one subdomain discovery execution.
type discoveryRun struct {
	// The Engine contract supplies exactly one canonical domain target per execution.
	domain                      string
	typedConfig                 subdomainspec.Config
	workspaceDir                string
	progress                    executionProgress
	executor                    toolExecutor
	providerConfigPath          string
	wordlistPath                string
	bruteforceResolversPath     string
	resolveResolversPath        string
	allocatedWorkspaceFilePaths map[string]struct{}
}

// stageOutcome holds output artifact paths and tool status from a discovery stage.
type stageOutcome struct {
	outputArtifactPaths []string
	// Only resolve produces a reportable final artifact; earlier stage outputs
	// remain inputs to later discovery stages.
	resultArtifactPath string
	failed             []string
	success            []string
	failureCause       error
}

func (outcome *stageOutcome) appendOutcome(other stageOutcome) {
	outcome.outputArtifactPaths = append(outcome.outputArtifactPaths, other.outputArtifactPaths...)
	if other.resultArtifactPath != "" {
		outcome.resultArtifactPath = other.resultArtifactPath
	}
	outcome.failed = append(outcome.failed, other.failed...)
	outcome.success = append(outcome.success, other.success...)
	if outcome.failureCause == nil {
		outcome.failureCause = other.failureCause
	}
}

// runAllStages executes all discovery stages and collects results
func (w *Runtime) runAllStages(execCtx context.Context, run *discoveryRun) stageOutcome {
	var combinedOutcome stageOutcome

	if run.typedConfig.Recon.Enabled {
		combinedOutcome.appendOutcome(w.runReconStage(execCtx, run))
	}

	if run.typedConfig.Bruteforce.Enabled {
		combinedOutcome.appendOutcome(w.runBruteforceStage(execCtx, run))
	}

	if len(combinedOutcome.outputArtifactPaths) > 0 {
		combinedOutcome.appendOutcome(w.runResolveStage(execCtx, run, combinedOutcome.outputArtifactPaths))
	} else {
		reportStageSkip(run, "3/3 resolve", "no previous outputs")
	}

	return combinedOutcome
}
