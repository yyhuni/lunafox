package subdomaindiscoveryruntime

import (
	"context"
	"log"

	subdomainspec "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

// runResolveStage resolves the normalized union of the upstream discovery artifacts.
func (w *Runtime) runResolveStage(execCtx context.Context, run *discoveryRun, inputFiles []string) stageOutcome {
	const stageLabel = "3/3 resolve"
	if run.progress != nil {
		if err := run.progress.report("start stage 3/3 resolve: verify discovered subdomains"); err != nil {
			return stageOutcome{failed: []string{subdomainspec.StageResolve}}
		}
	}

	inputFile, err := deduplicateSubdomainArtifacts(execCtx, run.workspaceDir, inputFiles)
	if err != nil {
		log.Printf("failed to deduplicate subdomain input stage=%s error=%v", subdomainspec.StageResolve, err)
		reportStageFailure(run, stageLabel, "input: "+err.Error())
		return stageFailureOutcome(subdomainspec.StageResolve, "input: "+err.Error())
	}
	if countFileLines(inputFile) == 0 {
		reportStageSkip(run, stageLabel, "no previous outputs")
		return stageOutcome{}
	}

	outputPath, err := run.allocateWorkspaceFilePath("resolve_puredns")
	if err != nil {
		log.Printf("failed to allocate resolve output path error=%v", err)
		reportStageFailure(run, stageLabel, "workspace: "+err.Error())
		return stageFailureOutcome(subdomainspec.StageResolve, "workspace: "+err.Error())
	}
	tool, args := buildPurednsResolveInvocation(inputFile, outputPath, run.resolveResolversPath, run.typedConfig.Resolve)
	command := buildToolInvocation(tool, args, run.typedConfig.Resolve.Timeout)

	log.Printf("running DNS resolve stage input.file_count=%d timeout.seconds=%d", len(inputFiles), command.TimeoutSeconds)
	outcome := collectCommandOutcomes([]commandResult{executeStageCommand(execCtx, run, subdomainspec.StageResolve, command, outputPath)})
	if len(outcome.outputArtifactPaths) == 1 {
		outcome.resultArtifactPath = outcome.outputArtifactPaths[0]
	}
	reportStageComplete(run, stageLabel, outcome)
	return outcome
}
