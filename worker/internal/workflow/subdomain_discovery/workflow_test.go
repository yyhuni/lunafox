package subdomain_discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/server"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
)

type providerClient struct {
	config *server.ProviderConfig
	err    error
}

func (p providerClient) GetProviderConfig(ctx context.Context, scanID int, toolName string) (*server.ProviderConfig, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.config, nil
}

func (p providerClient) EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error) {
	return "", nil
}

func (p providerClient) PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error {
	return nil
}

type capturePostClient struct {
	calls int
	items []any
	err   error
}

func (c *capturePostClient) GetProviderConfig(ctx context.Context, scanID int, toolName string) (*server.ProviderConfig, error) {
	return nil, nil
}

func (c *capturePostClient) EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error) {
	return "", nil
}

func (c *capturePostClient) PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error {
	c.calls++
	c.items = append(c.items, items...)
	return c.err
}

func countGoroutinesWithStackFragment(fragment string) int {
	size := 1 << 20
	for {
		buf := make([]byte, size)
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return strings.Count(string(buf[:n]), fragment)
		}
		size *= 2
		if size > 1<<26 {
			return strings.Count(string(buf[:n]), fragment)
		}
	}
}

func validWorkflowConfig() map[string]any {
	return map[string]any{
		"apiVersion":    "v1",
		"schemaVersion": "1.0.0",
		stageRecon: map[string]any{
			"enabled": true,
			"tools": map[string]any{
				toolSubfinder: map[string]any{
					"enabled":         true,
					"timeout-runtime": 3600,
					"threads-cli":     10,
				},
			},
		},
		stageBruteforce: map[string]any{
			"enabled": false,
			"tools": map[string]any{
				toolSubdomainBruteforce: map[string]any{"enabled": false},
			},
		},
		stagePermutation: map[string]any{
			"enabled": false,
			"tools": map[string]any{
				toolSubdomainPermutationResolve: map[string]any{"enabled": false},
			},
		},
		stageResolve: map[string]any{
			"enabled": false,
			"tools": map[string]any{
				toolSubdomainResolve: map[string]any{"enabled": false},
			},
		},
	}
}

func validScanConfig() map[string]any {
	return map[string]any{
		Name: validWorkflowConfig(),
	}
}

func validTypedWorkflowConfig(t *testing.T) WorkflowConfig {
	t.Helper()
	cfg, err := decodeTypedWorkflowConfig(validScanConfig())
	require.NoError(t, err)
	return cfg
}

func TestInitializeMissingConfig(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	_, err := w.initialize(&workflow.Params{
		ScanConfig:   nil,
		TargetType:   "domain",
		TargetName:   "example.com",
		WorkDir:      t.TempDir(),
		ServerClient: providerClient{},
	})
	require.Error(t, err)
}

func TestInitializeInvalidTargetType(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	_, err := w.initialize(&workflow.Params{
		ScanConfig:   validScanConfig(),
		TargetType:   "ip",
		TargetName:   "1.1.1.1",
		WorkDir:      t.TempDir(),
		ServerClient: providerClient{},
	})
	require.Error(t, err)
}

func TestInitializeInvalidDomain(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	_, err := w.initialize(&workflow.Params{
		ScanConfig:   validScanConfig(),
		TargetType:   "domain",
		TargetName:   "bad domain",
		WorkDir:      t.TempDir(),
		ServerClient: providerClient{},
	})
	require.Error(t, err)
}

func TestInitializeNormalizesDomain(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	ctx, err := w.initialize(&workflow.Params{
		ScanConfig:   validScanConfig(),
		TargetType:   "domain",
		TargetName:   "Example.COM.",
		WorkDir:      t.TempDir(),
		ServerClient: providerClient{},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"example.com"}, ctx.domains)
}

func TestInitializeStoresTypedConfigForExecution(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	ctx, err := w.initialize(&workflow.Params{
		ScanConfig:   validScanConfig(),
		TargetType:   "domain",
		TargetName:   "example.com",
		WorkDir:      t.TempDir(),
		ServerClient: providerClient{},
	})
	require.NoError(t, err)
	require.Equal(t, ContractAPIVersion, ctx.typedConfig.APIVersion)
	require.Equal(t, ContractSchemaVer, ctx.typedConfig.SchemaVersion)
	require.True(t, ctx.typedConfig.Recon.Enabled)
	require.True(t, ctx.typedConfig.Recon.Tools.Subfinder.Enabled)
}

func TestInitializeUsesPreDecodedWorkflowConfig(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	typedCfg := validTypedWorkflowConfig(t)

	ctx, err := w.initialize(&workflow.Params{
		WorkflowConfig: typedCfg,
		TargetType:     "domain",
		TargetName:     "example.com",
		WorkDir:        t.TempDir(),
		ServerClient:   providerClient{},
	})
	require.NoError(t, err)
	require.Equal(t, typedCfg, ctx.typedConfig)
}

func TestInitializeRejectsInvalidTypedWorkflowConfigType(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())

	_, err := w.initialize(&workflow.Params{
		WorkflowConfig: map[string]any{"invalid": true},
		TargetType:     "domain",
		TargetName:     "example.com",
		WorkDir:        t.TempDir(),
		ServerClient:   providerClient{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid typed config")
}

func TestInitializeRejectsInvalidTypedWorkflowConfigValue(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	invalidTypedCfg := validTypedWorkflowConfig(t)
	invalidTypedCfg.APIVersion = ""

	_, err := w.initialize(&workflow.Params{
		WorkflowConfig: invalidTypedCfg,
		TargetType:     "domain",
		TargetName:     "example.com",
		WorkDir:        t.TempDir(),
		ServerClient:   providerClient{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid typed workflow config")
}

func TestInitializeFlatConfigRejected(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())

	_, err := w.initialize(&workflow.Params{
		ScanConfig:   validWorkflowConfig(),
		TargetType:   "domain",
		TargetName:   "example.com",
		WorkDir:      t.TempDir(),
		ServerClient: providerClient{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing "+Name+" config")
}

func TestInitializeProviderConfigWritten(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	workDir := t.TempDir()

	ctx, err := w.initialize(&workflow.Params{
		ScanConfig:   validScanConfig(),
		TargetType:   "domain",
		TargetName:   "example.com",
		WorkDir:      workDir,
		ServerClient: providerClient{config: &server.ProviderConfig{Content: "api: key"}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, ctx.providerConfigPath)

	data, err := os.ReadFile(ctx.providerConfigPath)
	require.NoError(t, err)
	assert.Equal(t, "api: key", string(data))
}

func TestSetupProviderConfigNoContent(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	workDir := t.TempDir()

	path, err := w.setupProviderConfig(context.Background(), &workflow.Params{
		ScanID:       1,
		ServerClient: providerClient{config: nil},
	}, workDir)
	require.NoError(t, err)
	assert.Equal(t, "", path)

	path, err = w.setupProviderConfig(context.Background(), &workflow.Params{
		ScanID:       1,
		ServerClient: providerClient{config: &server.ProviderConfig{Content: ""}},
	}, workDir)
	require.NoError(t, err)
	assert.Equal(t, "", path)
}

func TestSetupProviderConfigWriteError(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	workDir := t.TempDir()

	require.NoError(t, os.Chmod(workDir, 0500))
	t.Cleanup(func() {
		_ = os.Chmod(workDir, 0700)
	})

	_, err := w.setupProviderConfig(context.Background(), &workflow.Params{
		ScanID:       1,
		ServerClient: providerClient{config: &server.ProviderConfig{Content: "api: key"}},
	}, workDir)
	require.Error(t, err)
}

func TestInitializeProviderConfigErrorIgnored(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	_, err := w.initialize(&workflow.Params{
		ScanConfig:   validScanConfig(),
		TargetType:   "domain",
		TargetName:   "example.com",
		WorkDir:      t.TempDir(),
		ServerClient: providerClient{err: errors.New("boom")},
	})
	require.NoError(t, err)
}

func TestSaveResultsNoFiles(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	output := &workflow.Output{
		Data:    "not-files",
		Metrics: &workflow.Metrics{},
	}

	err := w.SaveResults(context.Background(), providerClient{}, &workflow.Params{}, output)
	require.NoError(t, err)
}

func TestSaveResultsSuccessUpdatesMetrics(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	dir := t.TempDir()
	file := filepath.Join(dir, "out.txt")
	require.NoError(t, os.WriteFile(file, []byte("a.example.com\nb.example.com\n"), 0644))

	client := &capturePostClient{}
	output := &workflow.Output{
		Data:    []string{file},
		Metrics: &workflow.Metrics{},
	}
	params := &workflow.Params{ScanID: 1, TargetID: 2}

	err := w.SaveResults(context.Background(), client, params, output)
	require.NoError(t, err)
	assert.Equal(t, 1, client.calls)
	assert.Equal(t, 2, output.Metrics.ProcessedCount)
}

func TestSaveResultsWriteSubdomainsError(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	dir := t.TempDir()
	file := filepath.Join(dir, "out.txt")
	require.NoError(t, os.WriteFile(file, []byte("a.example.com\n"), 0644))

	client := &capturePostClient{err: errors.New("post failed")}
	output := &workflow.Output{
		Data:    []string{file},
		Metrics: &workflow.Metrics{},
	}
	params := &workflow.Params{ScanID: 1, TargetID: 2}

	err := w.SaveResults(context.Background(), client, params, output)
	require.Error(t, err)
}

func TestSaveResultsWriteSubdomainsErrorCancelsParser(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())

	// Clean baseline: no leftover parser goroutine from other tests.
	require.Eventually(t, func() bool {
		return countGoroutinesWithStackFragment("worker/internal/results.ParseSubdomains.func1") == 0
	}, time.Second, 20*time.Millisecond)

	dir := t.TempDir()
	file := filepath.Join(dir, "many.txt")
	var b strings.Builder
	for i := 0; i < 20000; i++ {
		b.WriteString("sub")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(".example.com\n")
	}
	require.NoError(t, os.WriteFile(file, []byte(b.String()), 0644))

	client := &capturePostClient{err: errors.New("post failed")}
	output := &workflow.Output{
		Data:    []string{file},
		Metrics: &workflow.Metrics{},
	}
	params := &workflow.Params{ScanID: 1, TargetID: 2}

	err := w.SaveResults(context.Background(), client, params, output)
	require.Error(t, err)

	require.Eventually(t, func() bool {
		return countGoroutinesWithStackFragment("worker/internal/results.ParseSubdomains.func1") == 0
	}, 2*time.Second, 20*time.Millisecond)
}

func TestSaveResultsParseError(t *testing.T) {
	withNopLogger(t)
	w := New(t.TempDir())
	dir := t.TempDir()
	file := filepath.Join(dir, "long.txt")
	longLine := strings.Repeat("a", 70*1024)
	require.NoError(t, os.WriteFile(file, []byte(longLine+"\n"), 0644))

	client := &capturePostClient{}
	output := &workflow.Output{
		Data:    []string{file},
		Metrics: &workflow.Metrics{},
	}
	params := &workflow.Params{ScanID: 1, TargetID: 2}

	err := w.SaveResults(context.Background(), client, params, output)
	require.Error(t, err)
}
