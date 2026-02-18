package activity

import (
	"embed"
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

//go:embed testdata/*.yaml
var testFS embed.FS

func init() {
	// Initialize test logger
	logger, _ := zap.NewDevelopment()
	pkg.Logger = logger
}

func TestTemplateLoader_Load_ValidTemplate(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/valid_template.yaml")

	templates, err := loader.Load()
	require.NoError(t, err)
	assert.NotEmpty(t, templates)

	// Validate template content
	subfinder, ok := templates["subfinder"]
	require.True(t, ok)
	assert.Equal(t, "subfinder -d {{.Domain}} -o {{quote .OutputFile}}", subfinder.BaseCommand)
	assert.Equal(t, "Subfinder", subfinder.Metadata.DisplayName)
	assert.Equal(t, "recon", subfinder.Metadata.Stage)
}

func TestTemplateLoader_Load_WithMetadata(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/valid_template.yaml")

	metadata, err := loader.GetMetadata()
	require.NoError(t, err)
	assert.Equal(t, "test_workflow", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Stages)
}

func TestTemplateLoader_Load_InvalidYAML(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/invalid_yaml.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestTemplateLoader_Load_MissingBaseCommand(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_base_command.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_command is required")
}

func TestTemplateLoader_Load_InvalidGoTemplate(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/invalid_go_template.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base_command syntax")
}

func TestTemplateLoader_Load_InvalidParameterType(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/invalid_param_type.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config_schema.type")
}

func TestTemplateLoader_Load_MissingStageReference(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_stage_ref.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in workflow metadata")
}

func TestTemplateLoader_Get_ExistingTemplate(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/valid_template.yaml")

	tmpl, err := loader.Get("subfinder")
	require.NoError(t, err)
	assert.Equal(t, "Subfinder", tmpl.Metadata.DisplayName)
}

func TestTemplateLoader_Get_NonExistingTemplate(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/valid_template.yaml")

	_, err := loader.Get("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestTemplateLoader_Load_CachedWithSyncOnce(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/valid_template.yaml")

	// First load
	templates1, err1 := loader.Load()
	require.NoError(t, err1)

	// Second load (should use cache)
	templates2, err2 := loader.Load()
	require.NoError(t, err2)

	// Should return the same map (same pointer)
	assert.Equal(t, templates1, templates2)
}

func TestTemplateLoader_Validate_DefaultValueTypeMismatch(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/type_mismatch.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected integer")
}

func TestTemplateLoader_Validate_MissingMetadata(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_metadata.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestTemplateLoader_Validate_MissingToolMetadata(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_tool_metadata.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "display_name is required")
}

func TestTemplateLoader_Validate_InvalidSemanticID(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/invalid_semantic_id.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid semantic_id")
}

func TestTemplateLoader_Validate_InvalidConfigKey(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/invalid_config_key.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config_schema.key")
}

func TestTemplateLoader_Validate_DuplicateConfigKey(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/duplicate_config_key.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate config_schema.key")
}

func TestTemplateLoader_Validate_InvalidShowAs(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/invalid_show_as.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config_example.show_as")
}

func TestTemplateLoader_Validate_DuplicateSemanticID(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/duplicate_semantic_id.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate semantic_id")
}

func TestTemplateLoader_Validate_InvalidVar(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/invalid_var.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid var")
}

func TestTemplateLoader_Validate_MissingShowAs(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_show_as.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config_example.show_as is required")
}

func TestTemplateLoader_Validate_RequiredWithDefault(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/required_with_default.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required cannot be true when default is set")
}

func TestTemplateLoader_Validate_MissingStageID(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_stage_id.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stage metadata: id is required")
}

func TestTemplateLoader_Validate_MissingVersion(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_version.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestTemplateLoader_Validate_MissingParamVar(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_param_var.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "param var is required")
}

func TestTemplateLoader_Validate_DuplicateParamVar(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/duplicate_param_var.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate var")
}

func TestTemplateLoader_Validate_MissingSemanticID(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_semantic_id.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "semantic_id is required")
}

func TestTemplateLoader_Validate_MissingConfigKey(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_config_key.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config_schema.key is required")
}

func TestTemplateLoader_Validate_MissingConfigType(t *testing.T) {
	loader := NewTemplateLoader(testFS, "testdata/missing_config_type.yaml")

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config_schema.type is required")
}
