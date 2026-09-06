package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"gopkg.in/yaml.v3"
)

type executionSubfinderProviderSettingsStoreStub struct {
	settings *catalogdomain.SubfinderProviderSettings
	err      error
	ctx      context.Context
}

func (stub *executionSubfinderProviderSettingsStoreStub) GetInstanceContext(ctx context.Context) (*catalogdomain.SubfinderProviderSettings, error) {
	stub.ctx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if stub.err != nil {
		return nil, stub.err
	}
	return stub.settings, nil
}

func TestExecutionProviderConfigSourceIncludesCompleteEmptyRegistryAndForwardsContext(t *testing.T) {
	store := &executionSubfinderProviderSettingsStoreStub{settings: &catalogdomain.SubfinderProviderSettings{
		Providers: catalogdomain.SubfinderProviderConfigs{},
	}}
	source := NewExecutionProviderConfigSource(store)
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("request"), "artifact-1")

	content, err := source.GetExecutionSubfinderProviderConfig(ctx)
	if err != nil {
		t.Fatalf("GetExecutionSubfinderProviderConfig returned error: %v", err)
	}
	if strings.TrimSpace(content) == "" || store.ctx != ctx {
		t.Fatalf("execution provider source did not preserve context: content=%q ctx=%v", content, store.ctx)
	}
	var decoded map[string][]string
	if err := yaml.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("decode execution provider config: %v", err)
	}
	definitions := catalogdomain.SubfinderProviderDefinitions()
	if len(decoded) != len(definitions) {
		t.Fatalf("provider key count = %d, want %d", len(decoded), len(definitions))
	}
	for _, definition := range definitions {
		values, ok := decoded[definition.SourceName]
		if !ok || values == nil || len(values) != 0 {
			t.Fatalf("provider %q = %#v, %v; want explicit []", definition.SourceName, values, ok)
		}
	}
}

func TestExecutionProviderConfigSourceUsesCanonicalRegistryProjection(t *testing.T) {
	store := &executionSubfinderProviderSettingsStoreStub{settings: &catalogdomain.SubfinderProviderSettings{
		Providers: catalogdomain.SubfinderProviderConfigs{
			"fofa":       {Enabled: true, Values: map[string]string{"email": "test@example.com", "apiKey": "fofa-secret"}},
			"zoomeyeapi": {Enabled: true, Values: map[string]string{"host": "zoomeye.org", "apiKey": "zoom-secret"}},
			"hunter":     {Enabled: true, Values: map[string]string{"apiKey": "retired-secret"}},
			"gitlab":     {Enabled: true, Values: map[string]string{"apiKey": "legacy-secret"}},
		},
	}}

	content, err := NewExecutionProviderConfigSource(store).GetExecutionSubfinderProviderConfig(context.Background())
	if err != nil {
		t.Fatalf("GetExecutionSubfinderProviderConfig returned error: %v", err)
	}
	var decoded map[string][]string
	if err := yaml.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("decode execution provider config: %v", err)
	}
	if got := decoded["fofa"]; len(got) != 1 || got[0] != "test@example.com:fofa-secret" {
		t.Fatalf("fofa credentials = %#v", got)
	}
	if got := decoded["zoomeyeapi"]; len(got) != 1 || got[0] != "zoomeye.org:zoom-secret" {
		t.Fatalf("zoomeyeapi credentials = %#v", got)
	}
	if _, exists := decoded["hunter"]; exists {
		t.Fatal("retired provider must not enter the execution provider projection")
	}
	if _, exists := decoded["gitlab"]; exists {
		t.Fatal("unregistered GitLab provider must not enter the execution provider projection")
	}
}

func TestValidateCompleteSubfinderProviderConfigMatchesCanonicalRegistryFixture(t *testing.T) {
	fixturePath := filepath.Join(
		"..", "..", "..", "..", "..",
		"extensions", "engines", "subdomain_discovery", "tests", "container", "subfinder-provider-config-"+catalogdomain.SubfinderProviderRegistryVersion+".yaml",
	)
	canonical, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read canonical Subfinder provider fixture: %v", err)
	}
	canonicalText := string(canonical)

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{name: "canonical full mapping", content: canonicalText},
		{name: "empty", content: "", wantErr: true},
		{name: "empty mapping", content: "{}\n", wantErr: true},
		{name: "invalid YAML", content: "github: [\n", wantErr: true},
		{name: "missing key", content: strings.Replace(canonicalText, "github: []\n", "", 1), wantErr: true},
		{name: "extra key", content: canonicalText + "unexpected: []\n", wantErr: true},
		{name: "null list", content: strings.Replace(canonicalText, "github: []", "github:", 1), wantErr: true},
		{name: "wrong value type", content: strings.Replace(canonicalText, "github: []", "github: token", 1), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCompleteSubfinderProviderConfig(test.content)
			if test.wantErr && err == nil {
				t.Fatal("validateCompleteSubfinderProviderConfig() succeeded, want rejection")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("validateCompleteSubfinderProviderConfig() error = %v", err)
			}
		})
	}
}

func TestExecutionProviderConfigSourceDoesNotTreatMissingSingletonAsNoCredentials(t *testing.T) {
	source := NewExecutionProviderConfigSource(&executionSubfinderProviderSettingsStoreStub{err: ErrSubfinderProviderSettingsNotFound})

	if _, err := source.GetExecutionSubfinderProviderConfig(context.Background()); !errors.Is(err, ErrExecutionProviderConfigInvalid) {
		t.Fatalf("missing settings error = %v, want ErrExecutionProviderConfigInvalid", err)
	}
}

func TestExecutionProviderConfigSourcePropagatesCancellation(t *testing.T) {
	store := &executionSubfinderProviderSettingsStoreStub{settings: &catalogdomain.SubfinderProviderSettings{
		Providers: catalogdomain.SubfinderProviderConfigs{},
	}}
	source := NewExecutionProviderConfigSource(store)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := source.GetExecutionSubfinderProviderConfig(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetExecutionSubfinderProviderConfig error = %v, want context.Canceled", err)
	}
}

func TestExecutionProviderConfigSourceRejectsNilContext(t *testing.T) {
	source := NewExecutionProviderConfigSource(&executionSubfinderProviderSettingsStoreStub{settings: &catalogdomain.SubfinderProviderSettings{}})

	if _, err := source.GetExecutionSubfinderProviderConfig(nil); !errors.Is(err, ErrExecutionProviderConfigInvalid) {
		t.Fatalf("nil context error = %v, want ErrExecutionProviderConfigInvalid", err)
	}
}
