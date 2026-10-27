package subdomaindiscoveryruntime

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	subdomainspec "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

const (
	minRecursiveDepth int64 = 1
	maxRecursiveDepth int64 = 5
)

// runBruteforceStage executes subdomain bruteforce for the target domain.
func (w *Runtime) runBruteforceStage(execCtx context.Context, run *discoveryRun) (outcome stageOutcome) {
	if run.progress != nil {
		if err := run.progress.report("start stage 2/3 bruteforce: enumerate subdomains from wordlist"); err != nil {
			return appendBruteforceFailure(outcome, err)
		}
	}
	defer func() {
		if outcome.failureCause != nil {
			reportStageFailure(run, "2/3 bruteforce", outcome.failureCause.Error())
			return
		}
		reportStageComplete(run, "2/3 bruteforce", outcome)
	}()

	toolConfig := run.typedConfig.Bruteforce
	if err := validateBruteforceConfig(toolConfig); err != nil {
		return appendBruteforceFailure(outcome, fmt.Errorf("config: %w", err))
	}

	log.Printf("running bruteforce stage domain=%s wordlist=%s recursive=%t depth=%d", run.domain, run.wordlistPath, toolConfig.Recursive, toolConfig.RecursiveDepth)
	rootResult, err := executeBruteforceCommand(execCtx, run, run.domain)
	if err != nil {
		return appendBruteforceFailure(outcome, fmt.Errorf("workspace: %w", err))
	}
	outcome = collectCommandOutcomes([]commandResult{rootResult})
	if len(outcome.failed) > 0 || !toolConfig.Recursive {
		return outcome
	}

	frontierArtifacts := []string{rootResult.OutputArtifactPath}
	for layer := int64(1); layer <= toolConfig.RecursiveDepth; layer++ {
		if err := execCtx.Err(); err != nil {
			return appendBruteforceFailure(outcome, err)
		}

		frontierPath, err := deduplicateSubdomainArtifacts(execCtx, run.workspaceDir, frontierArtifacts)
		if err != nil {
			return appendBruteforceFailure(outcome, fmt.Errorf("recursive layer %d frontier: %w", layer, err))
		}

		// Parent processes stay serial and fail closed so each PureDNS process keeps
		// its configured rate limit and a partial frontier never expands further.
		nextLayerArtifacts := make([]string, 0)
		frontierErr := streamSubdomainArtifact(execCtx, frontierPath, func(parentDomain string) error {
			result, err := executeBruteforceCommand(execCtx, run, parentDomain)
			if err != nil {
				return err
			}
			parentOutcome := collectCommandOutcomes([]commandResult{result})
			outcome.appendOutcome(parentOutcome)
			if len(parentOutcome.failed) > 0 {
				return parentOutcome.failureCause
			}
			nextLayerArtifacts = append(nextLayerArtifacts, result.OutputArtifactPath)
			return nil
		})
		if frontierErr != nil {
			return appendBruteforceFailure(outcome, fmt.Errorf("recursive layer %d: %w", layer, frontierErr))
		}
		if len(nextLayerArtifacts) == 0 {
			break
		}
		frontierArtifacts = nextLayerArtifacts
	}

	return outcome
}

func validateBruteforceConfig(config subdomainspec.BruteforceConfig) error {
	if config.RecursiveDepth < minRecursiveDepth || config.RecursiveDepth > maxRecursiveDepth {
		return fmt.Errorf("recursive-depth must be between %d and %d", minRecursiveDepth, maxRecursiveDepth)
	}
	return nil
}

func executeBruteforceCommand(
	execCtx context.Context,
	run *discoveryRun,
	domain string,
) (commandResult, error) {
	outputPath, err := run.allocateWorkspaceFilePath("bruteforce_puredns")
	if err != nil {
		log.Printf("failed to allocate bruteforce output path domain=%s error=%v", domain, err)
		return commandResult{}, err
	}
	cmd := createBruteforceCommand(domain, run.typedConfig.Bruteforce, run.wordlistPath, run.bruteforceResolversPath, outputPath)
	return executeStageCommand(execCtx, run, subdomainspec.StageBruteforce, cmd, outputPath), nil
}

func appendBruteforceFailure(outcome stageOutcome, err error) stageOutcome {
	if len(outcome.failed) == 0 {
		outcome.failed = append(outcome.failed, subdomainspec.StageBruteforce)
	}
	if outcome.failureCause == nil {
		outcome.failureCause = err
	}
	return outcome
}

func streamSubdomainArtifact(ctx context.Context, path string, visit func(string) error) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open recursive frontier %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		parent, ok := normalizeSubdomainArtifactLine(scanner.Text())
		if !ok {
			continue
		}
		if err := visit(parent); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan recursive frontier %s: %w", path, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return file.Close()
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
