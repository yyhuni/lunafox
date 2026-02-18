package subdomain_discovery

import (
	"fmt"
	"path/filepath"

	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

// runBruteforceStage executes subdomain bruteforce for all domains
func (w *Workflow) runBruteforceStage(ctx *workflowContext) stageResult {
	stageConfig, ok := ctx.config[stageBruteforce].(map[string]any)
	if !ok {
		pkg.Logger.Debug("Bruteforce stage not configured")
		return stageResult{}
	}

	// Get tools configuration
	toolsConfig, ok := stageConfig["tools"].(map[string]any)
	if !ok {
		pkg.Logger.Debug("No tools configured in bruteforce stage")
		return stageResult{}
	}

	// Get tool-specific config
	toolConfig, _ := toolsConfig[toolSubdomainBruteforce].(map[string]any)
	normalizedConfig, err := normalizeToolConfig(toolSubdomainBruteforce, toolConfig)
	if err != nil {
		pkg.Logger.Error("Failed to normalize bruteforce config", zap.Error(err))
		return stageResult{failed: []string{stageBruteforce + " (invalid config)"}}
	}

	// Get wordlist name from config (required, no default)
	wordlistName := getStringValue(normalizedConfig, "subdomain-wordlist-name-runtime", "")
	if wordlistName == "" {
		pkg.Logger.Error("Bruteforce stage requires subdomain-wordlist-name in config")
		return stageResult{failed: []string{stageBruteforce + " (missing subdomain-wordlist-name in config)"}}
	}

	wordlistBasePath := getStringValue(normalizedConfig, "subdomain-wordlist-base-path-runtime", "")
	if wordlistBasePath == "" {
		pkg.Logger.Error("Bruteforce stage requires subdomain-wordlist-base-path in config")
		return stageResult{failed: []string{stageBruteforce + " (missing subdomain-wordlist-base-path in config)"}}
	}

	resolversPath := getStringValue(normalizedConfig, "resolvers-path-cli", "")
	if resolversPath == "" {
		pkg.Logger.Error("Bruteforce stage requires resolvers-path in config")
		return stageResult{failed: []string{stageBruteforce + " (missing resolvers-path in config)"}}
	}

	// Ensure wordlist exists locally (download from server if needed)
	wordlistPath, err := ctx.serverClient.EnsureWordlistLocal(ctx.ctx, wordlistName, wordlistBasePath)
	if err != nil {
		pkg.Logger.Error("Failed to get wordlist",
			zap.String("wordlist", wordlistName),
			zap.Error(err))
		return stageResult{failed: []string{stageBruteforce + " (wordlist: " + err.Error() + ")"}}
	}

	var commands []activity.Command

	for _, domain := range ctx.domains {
		cmd := w.createBruteforceCommand(ctx, domain, normalizedConfig, wordlistPath, resolversPath)
		if cmd != nil {
			commands = append(commands, *cmd)
		}
	}

	if len(commands) == 0 {
		pkg.Logger.Debug("No bruteforce commands created")
		return stageResult{}
	}

	pkg.Logger.Info("Running bruteforce stage",
		zap.Int("domains", len(commands)),
		zap.String("wordlist", wordlistPath))

	return processResults(w.runStageCommands(ctx, stageBruteforce, commands))
}

// createBruteforceCommand creates a bruteforce command for a domain
func (w *Workflow) createBruteforceCommand(ctx *workflowContext, domain string, toolConfig map[string]any, wordlistPath, resolversPath string) *activity.Command {
	outputFile := filepath.Join(ctx.workDir, fmt.Sprintf("bruteforce_%s.txt", sanitizeFilename(domain)))
	logFile := filepath.Join(ctx.workDir, fmt.Sprintf("bruteforce_%s.log", sanitizeFilename(domain)))

	params := map[string]any{
		"Domain":     domain,
		"OutputFile": outputFile,
		"Wordlist":   wordlistPath,
		"Resolvers":  resolversPath,
	}

	cmdStr, err := buildCommand(toolSubdomainBruteforce, params, toolConfig)
	if err != nil {
		pkg.Logger.Error("Failed to build bruteforce command",
			zap.String("domain", domain),
			zap.Error(err))
		return nil
	}
	timeout, err := getTimeout(toolConfig)
	if err != nil {
		pkg.Logger.Error("Failed to get timeout",
			zap.String("domain", domain),
			zap.Error(err))
		return nil
	}

	return &activity.Command{
		Name:       fmt.Sprintf("bruteforce_%s", sanitizeFilename(domain)),
		Command:    cmdStr,
		OutputFile: outputFile,
		LogFile:    logFile,
		Timeout:    timeout,
	}
}
