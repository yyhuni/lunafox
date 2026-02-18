package subdomain_discovery

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

// runReconStage executes all enabled reconnaissance tools
func (w *Workflow) runReconStage(ctx *workflowContext) stageResult {
	if !isStageEnabled(ctx.config, stageRecon) {
		pkg.Logger.Debug("Reconnaissance stage disabled")
		return stageResult{}
	}
	stageConfig, ok := ctx.config[stageRecon].(map[string]any)
	if !ok {
		pkg.Logger.Debug("Reconnaissance stage not configured")
		return stageResult{}
	}

	var commands []activity.Command

	// Get tools configuration
	toolsConfig, ok := stageConfig["tools"].(map[string]any)
	if !ok {
		pkg.Logger.Debug("No tools configured in reconnaissance stage")
		return stageResult{}
	}

	enabledTools := make([]string, 0, len(toolsConfig))
	enabledConfigs := make(map[string]map[string]any, len(toolsConfig))
	for toolName, rawConfig := range toolsConfig {
		toolConfig, ok := rawConfig.(map[string]any)
		if !ok {
			pkg.Logger.Warn("Invalid tool config type", zap.String("tool", toolName))
			continue
		}
		enabled, _ := toolConfig["enabled"].(bool)
		if !enabled {
			continue
		}
		enabledTools = append(enabledTools, toolName)
		enabledConfigs[toolName] = toolConfig
	}
	sort.Strings(enabledTools)

	if len(enabledTools) == 0 {
		pkg.Logger.Debug("No reconnaissance tools enabled")
		return stageResult{}
	}

	for _, domain := range ctx.domains {
		for _, toolName := range enabledTools {
			toolConfig := enabledConfigs[toolName]
			cmd := w.createReconCommand(ctx, domain, toolName, toolConfig)
			if cmd != nil {
				commands = append(commands, *cmd)
			}
		}
	}

	if len(commands) == 0 {
		pkg.Logger.Debug("No reconnaissance commands created")
		return stageResult{}
	}

	pkg.Logger.Info("Running reconnaissance stage", zap.Int("tools", len(commands)))

	return processResults(w.runStageCommands(ctx, stageRecon, commands))
}

// createReconCommand creates a command for a reconnaissance tool
func (w *Workflow) createReconCommand(ctx *workflowContext, domain, toolName string, toolConfig map[string]any) *activity.Command {
	outputFile := filepath.Join(ctx.workDir, fmt.Sprintf("%s_%s.txt", toolName, sanitizeFilename(domain)))
	logFile := filepath.Join(ctx.workDir, fmt.Sprintf("%s_%s.log", toolName, sanitizeFilename(domain)))

	normalizedConfig, err := normalizeToolConfig(toolName, toolConfig)
	if err != nil {
		pkg.Logger.Error("Failed to normalize tool config",
			zap.String("tool", toolName),
			zap.Error(err))
		return nil
	}

	params := map[string]any{
		"Domain":     domain,
		"OutputFile": outputFile,
	}

	// Add provider config for subfinder
	if toolName == toolSubfinder {
		params["ProviderConfig"] = ""
		if ctx.providerConfigPath != "" {
			params["ProviderConfig"] = ctx.providerConfigPath
		}
	}

	cmdStr, err := buildCommand(toolName, params, normalizedConfig)
	if err != nil {
		pkg.Logger.Error("Failed to build command",
			zap.String("tool", toolName),
			zap.Error(err))
		return nil
	}
	timeout, err := getTimeout(normalizedConfig)
	if err != nil {
		pkg.Logger.Error("Failed to get timeout",
			zap.String("tool", toolName),
			zap.Error(err))
		return nil
	}

	return &activity.Command{
		Name:       fmt.Sprintf("%s_%s", toolName, sanitizeFilename(domain)),
		Command:    cmdStr,
		OutputFile: outputFile,
		LogFile:    logFile,
		Timeout:    timeout,
	}
}
