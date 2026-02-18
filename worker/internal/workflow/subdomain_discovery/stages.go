package subdomain_discovery

import (
	"context"

	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/server"
)

// workflowContext holds shared context for stage execution
type workflowContext struct {
	ctx                context.Context
	domains            []string
	config             map[string]any
	workDir            string
	providerConfigPath string
	serverClient       server.ServerClient // for downloading wordlists etc.
}

// stageResult holds the output of a stage execution
type stageResult struct {
	files   []string
	failed  []string
	success []string
}

// merge combines another stageResult into this one
func (sr *stageResult) merge(other stageResult) {
	sr.files = append(sr.files, other.files...)
	sr.failed = append(sr.failed, other.failed...)
	sr.success = append(sr.success, other.success...)
}

// runAllStages executes all discovery stages and collects results
func (w *Workflow) runAllStages(ctx *workflowContext) stageResult {
	var allResults stageResult

	// Stage 1: Reconnaissance (always runs if configured)
	allResults.merge(w.runReconStage(ctx))

	// Stage 2: Bruteforce (optional)
	if isStageEnabled(ctx.config, stageBruteforce) {
		allResults.merge(w.runBruteforceStage(ctx))
	}

	// Stage 3: Permutation (optional, requires previous output)
	if isStageEnabled(ctx.config, stagePermutation) && len(allResults.files) > 0 {
		allResults.merge(w.runMergeStage(ctx, allResults.files, stagePermutation, toolSubdomainPermutationResolve))
	}

	// Stage 4: Resolve (optional, requires previous output)
	if isStageEnabled(ctx.config, stageResolve) && len(allResults.files) > 0 {
		allResults.merge(w.runMergeStage(ctx, allResults.files, stageResolve, toolSubdomainResolve))
	}

	return allResults
}

// processResults converts activity results to stageResult
func processResults(results []*activity.Result) stageResult {
	var sr stageResult
	for _, r := range results {
		if r.Error != nil {
			sr.failed = append(sr.failed, r.Name)
		} else {
			sr.success = append(sr.success, r.Name)
			if r.OutputFile != "" {
				sr.files = append(sr.files, r.OutputFile)
			}
		}
	}
	return sr
}
