package subdomaindiscoveryruntime

import (
	"context"
	"log"

	subdomainspec "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

// runBruteforceStage executes subdomain bruteforce for the target domain.
func (w *Runtime) runBruteforceStage(execCtx context.Context, run *discoveryRun) stageOutcome {
	if run.progress != nil {
		if err := run.progress.report("start stage 2/3 bruteforce: enumerate subdomains from wordlist"); err != nil {
			return stageOutcome{failed: []string{subdomainspec.StageBruteforce}}
		}
	}

	toolConfig := run.typedConfig.Bruteforce
	wordlistPath := run.wordlistPath
	resolversPath := run.bruteforceResolversPath

	outputPath, err := run.allocateWorkspaceFilePath("bruteforce_puredns")
	if err != nil {
		log.Printf("failed to allocate bruteforce output path domain=%s error=%v", run.domain, err)
		reportStageFailure(run, "2/3 bruteforce", "workspace: "+err.Error())
		return stageFailureOutcome(subdomainspec.StageBruteforce, "workspace: "+err.Error())
	}
	cmd := createBruteforceCommand(run.domain, toolConfig, wordlistPath, resolversPath, outputPath)

	log.Printf("running bruteforce stage domain=%s wordlist=%s", run.domain, wordlistPath)
	outcome := collectCommandOutcomes([]commandResult{executeStageCommand(execCtx, run, subdomainspec.StageBruteforce, cmd, outputPath)})
	reportStageComplete(run, "2/3 bruteforce", outcome)
	return outcome
}

// createBruteforceCommand creates a bruteforce command for a domain
func createBruteforceCommand(
	domain string,
	toolConfig subdomainspec.BruteforceConfig,
	wordlistPath string,
	resolversPath string,
	outputPath string,
) toolInvocation {
	tool, args := buildPurednsBruteforceInvocation(domain, outputPath, wordlistPath, resolversPath, toolConfig)
	return buildToolInvocation(tool, args, toolConfig.Timeout)
}
