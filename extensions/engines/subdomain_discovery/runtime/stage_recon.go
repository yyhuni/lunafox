package subdomaindiscoveryruntime

import (
	"context"
	"log"
	"strconv"

	subdomainspec "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

// runReconStage executes all enabled reconnaissance tools
func (w *Runtime) runReconStage(execCtx context.Context, run *discoveryRun) stageOutcome {
	if run.progress != nil {
		if err := run.progress.report("start stage 1/3 recon: collect subdomains from passive sources"); err != nil {
			return stageOutcome{failed: []string{subdomainspec.StageRecon}}
		}
	}

	toolConfig := run.typedConfig.Recon
	providerConfigPath := run.providerConfigPath

	outputPath, err := run.allocateWorkspaceFilePath("recon_subfinder")
	if err != nil {
		log.Printf("failed to allocate recon output path domain=%s error=%v", run.domain, err)
		reportStageFailure(run, "1/3 recon", "workspace: "+err.Error())
		return stageFailureOutcome(subdomainspec.StageRecon, "workspace: "+err.Error())
	}
	cmd := createReconCommand(run.domain, toolConfig, providerConfigPath, outputPath)

	log.Printf("running reconnaissance stage domain=%s", run.domain)

	outcome := collectCommandOutcomes([]commandResult{executeStageCommand(execCtx, run, subdomainspec.StageRecon, cmd, outputPath)})
	reportStageComplete(run, "1/3 recon", outcome)
	return outcome
}

// createReconCommand creates a command for the subfinder reconnaissance tool.
func createReconCommand(
	domain string,
	toolConfig subdomainspec.ReconConfig,
	providerConfigPath string,
	outputPath string,
) toolInvocation {
	tool, args := buildSubfinderReconInvocation(domain, outputPath, providerConfigPath, toolConfig)
	return buildToolInvocation(tool, args, toolConfig.Timeout)
}

func buildSubfinderReconInvocation(
	domain string,
	outputFile string,
	providerConfigPath string,
	toolConfig subdomainspec.ReconConfig,
) (string, []string) {
	args := []string{
		"-d", domain,
		"-all",
		"-o", outputFile,
		"-v",
		"-t", strconv.FormatInt(toolConfig.Threads, 10),
		"-pc", providerConfigPath,
	}
	return subdomainspec.ToolSubfinder, args
}
