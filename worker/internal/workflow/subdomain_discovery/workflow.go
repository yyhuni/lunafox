package subdomain_discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/yyhuni/lunafox/worker/internal/pkg/validator"
	"github.com/yyhuni/lunafox/worker/internal/results"
	"github.com/yyhuni/lunafox/worker/internal/server"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	"go.uber.org/zap"
)

const Name = "subdomain_discovery"

func init() {
	workflow.Register(Name, func(workDir string) workflow.Workflow {
		return New(workDir)
	})
}

// Workflow implements the subdomain discovery scan workflow
type Workflow struct {
	runner        *activity.Runner
	workDir       string
	stageMetadata map[string]workflow.StageMetadata
}

// New creates a new subdomain discovery workflow
func New(workDir string) *Workflow {
	// Load metadata from templates.yaml
	metadata, err := loader.GetMetadata()
	if err != nil {
		pkg.Logger.Error("Failed to load workflow metadata", zap.Error(err))
		// Continue with empty metadata map - will use default parallel behavior
		return &Workflow{
			runner:        activity.NewRunner(workDir),
			workDir:       workDir,
			stageMetadata: make(map[string]workflow.StageMetadata),
		}
	}

	// Build stage metadata map for quick lookup
	stageMap := make(map[string]workflow.StageMetadata)
	for _, stage := range metadata.Stages {
		stageMap[stage.ID] = stage
	}

	return &Workflow{
		runner:        activity.NewRunner(workDir),
		workDir:       workDir,
		stageMetadata: stageMap,
	}
}

func (w *Workflow) Name() string {
	return Name
}

// Execute runs the subdomain discovery workflow
func (w *Workflow) Execute(params *workflow.Params) (*workflow.Output, error) {
	// Initialize and validate
	ctx, err := w.initialize(params)
	if err != nil {
		return nil, err
	}

	// Run all stages
	allResults := w.runAllStages(ctx)

	// Store result files for streaming in SaveResults
	output := &workflow.Output{
		Data: allResults.files, // Pass file paths instead of parsed data
		Metrics: &workflow.Metrics{
			ProcessedCount: 0, // Will be updated after streaming
			FailedCount:    len(allResults.failed),
			FailedTools:    allResults.failed,
		},
	}

	// Check for complete failure
	if len(allResults.failed) > 0 && len(allResults.success) == 0 {
		return output, fmt.Errorf("all tools failed")
	}

	return output, nil
}

// SaveResults streams subdomain results to the server in batches
func (w *Workflow) SaveResults(ctx context.Context, client server.ServerClient, params *workflow.Params, output *workflow.Output) error {
	files, ok := output.Data.([]string)
	if !ok || len(files) == 0 {
		return nil
	}

	// Stream and deduplicate from files
	parseCtx, parseCancel := context.WithCancel(ctx)
	defer parseCancel()
	subdomainCh, errCh := results.ParseSubdomains(parseCtx, files)

	// Send subdomains in batches
	items, batches, err := results.WriteSubdomains(ctx, client, params.ScanID, params.TargetID, subdomainCh)
	if err != nil {
		parseCancel()
		return err
	}

	// Check for streaming errors
	if err, ok := <-errCh; ok && err != nil {
		return fmt.Errorf("error streaming results: %w", err)
	}

	// Update metrics
	output.Metrics.ProcessedCount = items
	pkg.Logger.Info("Results saved",
		zap.Int("subdomains", items),
		zap.Int("batches", batches))

	return nil
}

// initialize validates params and prepares the workflow context
func (w *Workflow) initialize(params *workflow.Params) (*workflowContext, error) {
	// Config can be either nested under workflow name or flat
	// Try nested first: { "subdomain_discovery": { "recon-tools": ... } }
	// Then flat: { "recon-tools": ... }
	flowConfig := getConfigPath(params.ScanConfig, Name)
	if flowConfig == nil {
		// Use flat config directly
		flowConfig = params.ScanConfig
	}
	if flowConfig == nil {
		return nil, fmt.Errorf("missing %s config", Name)
	}
	if err := validateExplicitConfig(flowConfig); err != nil {
		return nil, err
	}

	workDir := filepath.Join(params.WorkDir, Name)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, err
	}

	// Subdomain discovery only works for domain type targets
	if params.TargetType != "domain" {
		return nil, fmt.Errorf("subdomain discovery requires domain target, got %s", params.TargetType)
	}

	// Normalize domain first
	normalizedDomain, err := validator.NormalizeDomain(params.TargetName)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize domain: %w", err)
	}

	// Validate normalized domain
	if err := validator.ValidateDomain(normalizedDomain); err != nil {
		return nil, fmt.Errorf("invalid target domain: %w", err)
	}

	// Wrap in slice for compatibility with multi-domain processing
	domains := []string{normalizedDomain}

	pkg.Logger.Info("Workflow initialized",
		zap.Int("scanId", params.ScanID),
		zap.String("targetName", params.TargetName),
		zap.String("targetType", params.TargetType))

	ctx := context.Background()
	providerConfigPath, err := w.setupProviderConfig(ctx, params, workDir)
	if err != nil {
		// Log warning but continue - provider config is optional (enhances results but not required)
		pkg.Logger.Warn("Failed to setup provider config, subfinder will run without API keys",
			zap.Error(err))
	}

	return &workflowContext{
		ctx:                ctx,
		domains:            domains,
		config:             flowConfig,
		workDir:            workDir,
		providerConfigPath: providerConfigPath,
		serverClient:       params.ServerClient,
	}, nil
}

// setupProviderConfig fetches and writes the subfinder provider config
// Returns empty string if no config available, error if fetch/write failed
func (w *Workflow) setupProviderConfig(ctx context.Context, params *workflow.Params, workDir string) (string, error) {
	providerConfig, err := params.ServerClient.GetProviderConfig(ctx, params.ScanID, toolSubfinder)
	if err != nil {
		return "", fmt.Errorf("failed to get provider config: %w", err)
	}
	if providerConfig == nil || providerConfig.Content == "" {
		return "", nil // No config available, not an error
	}

	configPath := filepath.Join(workDir, "provider-config.yaml")
	if err := os.WriteFile(configPath, []byte(providerConfig.Content), 0600); err != nil {
		return "", fmt.Errorf("failed to write provider config: %w", err)
	}
	pkg.Logger.Info("Provider config written", zap.String("path", configPath))
	return configPath, nil
}
