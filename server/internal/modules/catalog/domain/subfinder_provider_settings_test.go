package domain

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

type subfinderConfigurableSourcesFixture struct {
	Version  string   `json:"version"`
	Required []string `json:"required"`
	Optional []string `json:"optional"`
}

func TestSubfinderProviderRegistryMatchesSubfinderFixture(t *testing.T) {
	fixture := loadSubfinderConfigurableSourcesFixture(t)
	if fixture.Version != SubfinderProviderRegistryVersion {
		t.Fatalf("expected registry version %q, got %q", fixture.Version, SubfinderProviderRegistryVersion)
	}

	definitions := SubfinderProviderDefinitions()
	if len(definitions) != len(fixture.Required)+len(fixture.Optional) {
		t.Fatalf("expected %d registry entries, got %d", len(fixture.Required)+len(fixture.Optional), len(definitions))
	}

	assertFixtureSourcesRegistered(t, fixture.Required, SubfinderProviderKeyRequirementRequired)
	assertFixtureSourcesRegistered(t, fixture.Optional, SubfinderProviderKeyRequirementOptional)
}

func TestBuildSubfinderProviderCredentialValueUsesRegistryFormatters(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		config       SubfinderProviderConfig
		want         string
	}{
		{name: "github single key", providerName: "github", config: enabledProvider("apiKey", "github-token"), want: "github-token"},
		{name: "unregistered gitlab omitted", providerName: "gitlab", config: enabledProvider("apiKey", "gitlab-token"), want: ""},
		{name: "virustotal single key", providerName: "virustotal", config: enabledProvider("apiKey", "vt-token"), want: "vt-token"},
		{name: "shodan single key", providerName: "shodan", config: enabledProvider("apiKey", "shodan-token"), want: "shodan-token"},
		{name: "securitytrails single key", providerName: "securitytrails", config: enabledProvider("apiKey", "st-token"), want: "st-token"},
		{name: "fofa composite", providerName: "fofa", config: enabledProvider("email", "ops@example.com", "apiKey", "fofa-token"), want: "ops@example.com:fofa-token"},
		{name: "censys pat only", providerName: "censys", config: enabledProvider("pat", "censys-pat"), want: "censys-pat"},
		{name: "censys pat org", providerName: "censys", config: enabledProvider("pat", "censys-pat", "orgId", "org-1"), want: "censys-pat:org-1"},
		{name: "dnsrepo composite", providerName: "dnsrepo", config: enabledProvider("token", "dns-token", "apiKey", "dns-key"), want: "dns-token:dns-key"},
		{name: "domainsproject composite", providerName: "domainsproject", config: enabledProvider("username", "user", "password", "pass"), want: "user:pass"},
		{name: "facebook composite", providerName: "facebook", config: enabledProvider("appId", "app", "appSecret", "secret"), want: "app:secret"},
		{name: "intelx composite", providerName: "intelx", config: enabledProvider("host", "2.intelx.io", "apiKey", "intelx-key"), want: "2.intelx.io:intelx-key"},
		{name: "redhuntlabs composite", providerName: "redhuntlabs", config: enabledProvider("baseUrl", "https://api.redhuntlabs.com/community/v1/domains/subdomains", "blobrKey", "blobr"), want: "https://api.redhuntlabs.com/community/v1/domains/subdomains:blobr"},
		{name: "zoomeyeapi composite", providerName: "zoomeyeapi", config: enabledProvider("host", "zoomeye.org", "apiKey", "zoom-key"), want: "zoomeye.org:zoom-key"},
		{name: "disabled provider omitted", providerName: "github", config: SubfinderProviderConfig{Enabled: false, Values: map[string]string{"apiKey": "github-token"}}, want: ""},
		{name: "missing required composite part omitted", providerName: "fofa", config: enabledProvider("email", "ops@example.com"), want: ""},
		{name: "unknown provider omitted", providerName: "unknown", config: enabledProvider("apiKey", "unknown-token"), want: ""},
		{name: "legacy hunter omitted", providerName: "hunter", config: enabledProvider("apiKey", "hunter-token"), want: ""},
		{name: "legacy zoomeye omitted", providerName: "zoomeye", config: enabledProvider("apiKey", "zoomeye-token"), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSubfinderProviderCredentialValue(tt.providerName, tt.config)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestSubfinderProviderRegistryRejectsUnsupportedSources(t *testing.T) {
	for _, providerName := range []string{"gitlab", "hunter", "zoomeye"} {
		if _, ok := LookupSubfinderProviderDefinition(providerName); ok {
			t.Fatalf("unsupported provider %q must not be an active registry entry", providerName)
		}
	}
}

func loadSubfinderConfigurableSourcesFixture(t *testing.T) subfinderConfigurableSourcesFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/subfinder_v2_12_configurable_sources.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture subfinderConfigurableSourcesFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return fixture
}

func assertFixtureSourcesRegistered(t *testing.T, sourceNames []string, requirement string) {
	t.Helper()
	for _, sourceName := range sourceNames {
		definition, ok := LookupSubfinderProviderDefinition(sourceName)
		if !ok {
			t.Fatalf("expected registry entry for %q", sourceName)
		}
		if definition.SourceName != sourceName {
			t.Fatalf("expected source name %q, got %q", sourceName, definition.SourceName)
		}
		if definition.KeyRequirement != requirement {
			t.Fatalf("expected %q requirement for %q, got %q", requirement, sourceName, definition.KeyRequirement)
		}
		if !slices.Contains(SubfinderProviderKeys(), sourceName) {
			t.Fatalf("expected provider keys to include %q", sourceName)
		}
	}
}

func enabledProvider(pairs ...string) SubfinderProviderConfig {
	values := make(map[string]string, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		values[pairs[i]] = pairs[i+1]
	}
	return SubfinderProviderConfig{Enabled: true, Values: values}
}
