package subdomain_discovery

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/yyhuni/lunafox/worker/internal/pkg/validator"
	"go.uber.org/zap"
)

// wildcardCheckResult holds the result of wildcard detection
type wildcardCheckResult struct {
	isWildcard     bool
	originalCount  int
	sampleCount    int
	expansionRatio float64
	reason         string
}

type wildcardSettings struct {
	sampleTimeout      time.Duration
	tests              int
	batch              int
	sampleMultiplier   int
	expansionThreshold int
	resolversPath      string
}

func buildWildcardSettings(toolName string, config map[string]any, resolversPath string) (wildcardSettings, error) {
	sampleTimeoutSeconds, err := getIntValue(config, "wildcard-sample-timeout-runtime")
	if err != nil {
		return wildcardSettings{}, err
	}
	if sampleTimeoutSeconds <= 0 {
		return wildcardSettings{}, fmt.Errorf("wildcard-sample-timeout must be > 0")
	}
	tests, err := getIntValue(config, "wildcard-tests-cli")
	if err != nil {
		return wildcardSettings{}, err
	}
	if tests <= 0 {
		return wildcardSettings{}, fmt.Errorf("wildcard-tests must be > 0")
	}
	batch, err := getIntValue(config, "wildcard-batch-cli")
	if err != nil {
		return wildcardSettings{}, err
	}
	if batch <= 0 {
		return wildcardSettings{}, fmt.Errorf("wildcard-batch must be > 0")
	}
	sampleMultiplier, err := getIntValue(config, "wildcard-sample-multiplier-runtime")
	if err != nil {
		return wildcardSettings{}, err
	}
	if sampleMultiplier <= 0 {
		return wildcardSettings{}, fmt.Errorf("wildcard-sample-multiplier must be > 0")
	}
	expansionThreshold, err := getIntValue(config, "wildcard-expansion-threshold-runtime")
	if err != nil {
		return wildcardSettings{}, err
	}
	if expansionThreshold <= 0 {
		return wildcardSettings{}, fmt.Errorf("wildcard-expansion-threshold must be > 0")
	}
	if resolversPath == "" {
		return wildcardSettings{}, fmt.Errorf("resolvers-path is required")
	}

	return wildcardSettings{
		sampleTimeout:      time.Duration(sampleTimeoutSeconds) * time.Second,
		tests:              tests,
		batch:              batch,
		sampleMultiplier:   sampleMultiplier,
		expansionThreshold: expansionThreshold,
		resolversPath:      resolversPath,
	}, nil
}

// runMergeStage merges input files and runs a processing tool (resolve or permutation)
func (w *Workflow) runMergeStage(ctx *workflowContext, inputFiles []string, stageName, toolName string) stageResult {
	stageConfig, ok := ctx.config[stageName].(map[string]any)
	if !ok {
		pkg.Logger.Debug("Stage not configured", zap.String("stage", stageName))
		return stageResult{}
	}

	// Get tools configuration
	toolsConfig, ok := stageConfig["tools"].(map[string]any)
	if !ok {
		pkg.Logger.Debug("No tools configured in stage", zap.String("stage", stageName))
		return stageResult{}
	}

	// Get tool-specific config
	toolConfig, _ := toolsConfig[toolName].(map[string]any)
	normalizedConfig, err := normalizeToolConfig(toolName, toolConfig)
	if err != nil {
		pkg.Logger.Error("Failed to normalize tool config",
			zap.String("tool", toolName),
			zap.Error(err))
		return stageResult{failed: []string{stageName}}
	}
	resolversPath := getStringValue(normalizedConfig, "resolvers-path-cli", "")
	if resolversPath == "" {
		pkg.Logger.Error("Resolvers path is required in config",
			zap.String("stage", stageName))
		return stageResult{failed: []string{stageName}}
	}

	// Merge all input files into one
	mergedFile := filepath.Join(ctx.workDir, fmt.Sprintf("%s_input.txt", stageName))
	if err := w.mergeFiles(inputFiles, mergedFile); err != nil {
		pkg.Logger.Error("Failed to merge files",
			zap.String("stage", stageName),
			zap.Error(err))
		return stageResult{failed: []string{stageName}}
	}

	// For permutation stage, check for wildcard domains first
	if stageName == stagePermutation {
		settings, err := buildWildcardSettings(toolName, normalizedConfig, resolversPath)
		if err != nil {
			pkg.Logger.Error("Failed to load wildcard settings",
				zap.Error(err))
			return stageResult{failed: []string{stageName}}
		}
		checkResult := w.checkWildcard(ctx.ctx, mergedFile, ctx.workDir, settings)
		if checkResult.isWildcard {
			pkg.Logger.Warn("Skipping permutation stage due to wildcard detection",
				zap.String("reason", checkResult.reason),
				zap.Float64("expansionRatio", checkResult.expansionRatio))
			return stageResult{
				failed: []string{fmt.Sprintf("%s (wildcard: %s)", stageName, checkResult.reason)},
			}
		}
		pkg.Logger.Info("Wildcard check passed, proceeding with permutation",
			zap.Int("originalCount", checkResult.originalCount),
			zap.Int("sampleCount", checkResult.sampleCount))
	}

	outputFile := filepath.Join(ctx.workDir, fmt.Sprintf("%s_output.txt", stageName))
	logFile := filepath.Join(ctx.workDir, fmt.Sprintf("%s.log", stageName))

	params := map[string]any{
		"InputFile":  mergedFile,
		"OutputFile": outputFile,
		"Resolvers":  resolversPath,
	}

	cmdStr, err := buildCommand(toolName, params, normalizedConfig)
	if err != nil {
		pkg.Logger.Error("Failed to build command",
			zap.String("stage", stageName),
			zap.String("tool", toolName),
			zap.Error(err))
		return stageResult{failed: []string{stageName}}
	}

	timeout, err := getTimeout(normalizedConfig)
	if err != nil {
		pkg.Logger.Error("Failed to get timeout",
			zap.String("stage", stageName),
			zap.Error(err))
		return stageResult{failed: []string{stageName}}
	}

	cmd := activity.Command{
		Name:       stageName,
		Command:    cmdStr,
		OutputFile: outputFile,
		LogFile:    logFile,
		Timeout:    timeout,
	}

	pkg.Logger.Info("Running merge stage",
		zap.String("stage", stageName),
		zap.Int("inputFiles", len(inputFiles)),
		zap.Duration("timeout", timeout))

	return processResults(w.runStageCommands(ctx, stageName, []activity.Command{cmd}))
}

// mergeFiles reads all input files, deduplicates entries, and writes to outputFile
// Uses streaming to minimize memory usage
func (w *Workflow) mergeFiles(inputFiles []string, outputFile string) error {
	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	seen := make(map[string]struct{}, 100000) // pre-allocate for better performance
	writer := bufio.NewWriter(out)
	defer func() { _ = writer.Flush() }()

	for _, f := range inputFiles {
		if err := w.streamMergeFile(f, seen, writer); err != nil {
			pkg.Logger.Debug("Failed to process file during merge",
				zap.String("file", f),
				zap.Error(err))
			continue
		}
	}

	return writer.Flush()
}

// streamMergeFile streams a single file and writes unique subdomains to the writer
func (w *Workflow) streamMergeFile(filePath string, seen map[string]struct{}, writer *bufio.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !validator.IsValidSubdomainFormat(line) {
			continue
		}

		lower := strings.ToLower(line)
		if _, exists := seen[lower]; !exists {
			seen[lower] = struct{}{}
			if _, err := fmt.Fprintln(writer, line); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

// checkWildcard performs wildcard detection before permutation stage
// Returns true if wildcard is detected (should skip permutation)
func (w *Workflow) checkWildcard(ctx context.Context, inputFile, workDir string, settings wildcardSettings) wildcardCheckResult {
	originalCount := countFileLines(inputFile)
	if originalCount == 0 {
		return wildcardCheckResult{
			isWildcard: false,
			reason:     "empty input file",
		}
	}
	sampleSize := originalCount * settings.sampleMultiplier
	maxAllowed := originalCount * settings.expansionThreshold
	sampleOutput := filepath.Join(workDir, "wildcard_sample.txt")
	logFile := filepath.Join(workDir, "wildcard_detection.log")

	// Build sampling command: dnsgen | head | puredns resolve
	sampleCmd := fmt.Sprintf(
		"cat '%s' | dnsgen - | head -n %d | puredns resolve -r '%s' --write '%s' --wildcard-tests %d --wildcard-batch %d --quiet",
		inputFile, sampleSize, settings.resolversPath, sampleOutput, settings.tests, settings.batch,
	)

	pkg.Logger.Info("Wildcard detection: sampling",
		zap.Int("originalCount", originalCount),
		zap.Int("sampleSize", sampleSize),
		zap.Int("threshold", maxAllowed))

	// Run sampling with runner
	result := w.runner.Run(ctx, activity.Command{
		Name:       "wildcard_detection",
		Command:    sampleCmd,
		OutputFile: sampleOutput,
		LogFile:    logFile,
		Timeout:    settings.sampleTimeout,
	})

	// Handle execution errors
	if result.Error != nil {
		if result.ExitCode == activity.ExitCodeTimeout {
			pkg.Logger.Warn("Wildcard detection timeout")
			return wildcardCheckResult{
				isWildcard:    true,
				originalCount: originalCount,
				reason:        "sampling timeout",
			}
		}
		// Non-timeout error, continue anyway
		pkg.Logger.Debug("Wildcard sampling command error (continuing)", zap.Error(result.Error))
	}

	// Count sample results
	sampleCount := countFileLines(sampleOutput)

	pkg.Logger.Info("Wildcard detection: sample result",
		zap.Int("sampleCount", sampleCount),
		zap.Int("originalCount", originalCount),
		zap.Int("threshold", maxAllowed))

	// Check if expansion ratio exceeds threshold
	if sampleCount > maxAllowed {
		ratio := float64(sampleCount) / float64(originalCount)
		pkg.Logger.Warn("Wildcard detected: expansion ratio too high",
			zap.Int("sampleCount", sampleCount),
			zap.Int("threshold", maxAllowed),
			zap.Float64("ratio", ratio))

		return wildcardCheckResult{
			isWildcard:     true,
			originalCount:  originalCount,
			sampleCount:    sampleCount,
			expansionRatio: ratio,
			reason:         fmt.Sprintf("expansion ratio %.1fx exceeds threshold %dx", ratio, settings.expansionThreshold),
		}
	}

	return wildcardCheckResult{
		isWildcard:     false,
		originalCount:  originalCount,
		sampleCount:    sampleCount,
		expansionRatio: float64(sampleCount) / float64(originalCount),
	}
}
