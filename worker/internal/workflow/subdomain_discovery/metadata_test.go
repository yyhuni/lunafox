package subdomain_discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

// init initializes the test environment
func init() {
	// Initialize logger for the test environment (if not initialized yet)
	if pkg.Logger == nil {
		logger, _ := zap.NewDevelopment()
		pkg.Logger = logger
	}
}

// TestStagesMatchMetadata verifies stage constants in code match templates.yaml definitions
func TestStagesMatchMetadata(t *testing.T) {
	// Load metadata from templates.yaml
	metadata, err := loader.GetMetadata()
	require.NoError(t, err, "Failed to load metadata from templates.yaml")

	// Stage names defined in code
	codeStages := map[string]bool{
		stageRecon:       true,
		stageBruteforce:  true,
		stagePermutation: true,
		stageResolve:     true,
	}

	// Extract stage IDs from metadata
	metadataStages := make(map[string]bool)
	for _, stage := range metadata.Stages {
		metadataStages[stage.ID] = true
	}

	// Check each stage defined in code exists in metadata
	for stageName := range codeStages {
		assert.True(t, metadataStages[stageName],
			"Stage '%s' is defined in code but missing in templates.yaml metadata", stageName)
	}

	// Check each stage in metadata exists in code
	for stageName := range metadataStages {
		assert.True(t, codeStages[stageName],
			"Stage '%s' is defined in templates.yaml but missing in code constants", stageName)
	}
}

// TestToolsMatchMetadata verifies tool constants in code match templates.yaml definitions
func TestToolsMatchMetadata(t *testing.T) {
	// Load all tool templates from templates.yaml
	templates, err := loader.Load()
	require.NoError(t, err, "Failed to load templates from templates.yaml")

	// Tool names defined in code
	codeTools := map[string]bool{
		toolSubfinder:                   true,
		toolSubdomainBruteforce:         true,
		toolSubdomainPermutationResolve: true,
		toolSubdomainResolve:            true,
	}

	// Extract tool names from templates
	metadataTools := make(map[string]bool)
	for toolName := range templates {
		metadataTools[toolName] = true
	}

	// Check each tool defined in code exists in templates
	for toolName := range codeTools {
		assert.True(t, metadataTools[toolName],
			"Tool '%s' is defined in code but missing in templates.yaml", toolName)
	}

	// Note: reverse check is skipped (templates may contain unused tool definitions)
}

// TestStageToolMapping verifies each stage has mapped tools
func TestStageToolMapping(t *testing.T) {
	// Load all tool templates
	templates, err := loader.Load()
	require.NoError(t, err, "Failed to load templates")

	// Count tools per stage
	stageTools := make(map[string][]string)
	for toolName, tmpl := range templates {
		stageTools[tmpl.Metadata.Stage] = append(stageTools[tmpl.Metadata.Stage], toolName)
	}

	// Verify each stage has at least one tool
	metadata, err := loader.GetMetadata()
	require.NoError(t, err, "Failed to load metadata")

	for _, stage := range metadata.Stages {
		tools := stageTools[stage.ID]
		assert.NotEmpty(t, tools,
			"Stage '%s' should have at least one tool defined", stage.ID)

		t.Logf("Stage '%s' has %d tool(s): %v", stage.ID, len(tools), tools)
	}
}

// TestGeneratedConstantsNotEmpty verifies generated constants are not empty
func TestGeneratedConstantsNotEmpty(t *testing.T) {
	// Verify stage constants
	assert.NotEmpty(t, stageRecon, "stageRecon should not be empty")
	assert.NotEmpty(t, stageBruteforce, "stageBruteforce should not be empty")
	assert.NotEmpty(t, stagePermutation, "stagePermutation should not be empty")
	assert.NotEmpty(t, stageResolve, "stageResolve should not be empty")

	// Verify tool constants
	assert.NotEmpty(t, toolSubfinder, "toolSubfinder should not be empty")
	assert.NotEmpty(t, toolSubdomainBruteforce, "toolSubdomainBruteforce should not be empty")
	assert.NotEmpty(t, toolSubdomainPermutationResolve, "toolSubdomainPermutationResolve should not be empty")
	assert.NotEmpty(t, toolSubdomainResolve, "toolSubdomainResolve should not be empty")
}

// TestTemplateYAMLStructure verifies the basic templates.yaml structure
func TestTemplateYAMLStructure(t *testing.T) {
	metadata, err := loader.GetMetadata()
	require.NoError(t, err, "Failed to load metadata")

	// Verify workflow metadata
	assert.NotEmpty(t, metadata.Name, "Workflow name should not be empty")
	assert.NotEmpty(t, metadata.DisplayName, "Workflow display name should not be empty")
	assert.NotEmpty(t, metadata.Version, "Workflow version should not be empty")
	assert.NotEmpty(t, metadata.Stages, "Workflow should have at least one stage")
}
