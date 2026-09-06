package repository

import (
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
)

func TestSubfinderProviderSettingsModelToDomainMigratesLegacyProviders(t *testing.T) {
	settings := &model.SubfinderProviderSettings{
		ID: 1,
		Providers: model.SubfinderProviderConfigs{
			"fofa": {
				Enabled: true,
				Email:   "test@example.com",
				APIKey:  "fofa-key",
			},
			"shodan": {
				Enabled: true,
				APIKey:  "shodan-key",
			},
			"censys": {
				Enabled:   true,
				APIID:     "old-id",
				APISecret: "old-secret",
			},
			"zoomeye": {
				Enabled: true,
				APIKey:  "old-zoom",
			},
			"hunter": {
				Enabled: true,
				APIKey:  "hunter-key",
			},
			"gitlab": {
				Enabled: true,
				Values:  map[string]string{"apiKey": "gitlab-key"},
			},
		},
	}

	actual := subfinderProviderSettingsModelToDomain(settings)
	if actual == nil {
		t.Fatalf("expected non-nil domain settings")
	}
	if actual.ID != 1 {
		t.Fatalf("expected id=1, got %d", actual.ID)
	}
	if got := actual.Providers["fofa"].Values["email"]; got != "test@example.com" {
		t.Fatalf("expected fofa email migrated, got %q", got)
	}
	if got := actual.Providers["fofa"].Values["apiKey"]; got != "fofa-key" {
		t.Fatalf("expected fofa api key migrated, got %q", got)
	}
	if got := actual.Providers["shodan"].Values["apiKey"]; got != "shodan-key" {
		t.Fatalf("expected shodan api key migrated, got %q", got)
	}
	if actual.Providers["censys"].Status != catalogdomain.SubfinderProviderStatusRequiresReconfiguration {
		t.Fatalf("expected legacy censys requires reconfiguration, got %+v", actual.Providers["censys"])
	}
	if actual.Providers["zoomeyeapi"].Status != catalogdomain.SubfinderProviderStatusRequiresReconfiguration {
		t.Fatalf("expected legacy zoomeye requires reconfiguration, got %+v", actual.Providers["zoomeyeapi"])
	}
	if _, ok := actual.Providers["hunter"]; ok {
		t.Fatalf("legacy hunter must not be active provider entry")
	}
	if len(actual.Providers["hunter_legacy"].Values) == 0 || actual.Providers["hunter_legacy"].Status != catalogdomain.SubfinderProviderStatusUnsupported {
		t.Fatalf("expected hunter legacy status preserved, got %+v", actual.Providers["hunter_legacy"])
	}
	if _, ok := actual.Providers["gitlab"]; ok {
		t.Fatalf("unregistered gitlab settings must be ignored, got %+v", actual.Providers["gitlab"])
	}
}

func TestSubfinderProviderSettingsModelToDomainKeepsEmptyLegacyCredentialsUnconfigured(t *testing.T) {
	settings := &model.SubfinderProviderSettings{
		ID: 1,
		Providers: model.SubfinderProviderConfigs{
			"censys": {
				Enabled:   false,
				APIID:     "",
				APISecret: "",
			},
			"zoomeye": {
				Enabled: false,
				APIKey:  "",
			},
		},
	}

	actual := subfinderProviderSettingsModelToDomain(settings)

	for _, providerName := range []string{"censys", "zoomeyeapi"} {
		provider, ok := actual.Providers[providerName]
		if !ok {
			t.Fatalf("expected migrated provider %q", providerName)
		}
		if provider.Status != catalogdomain.SubfinderProviderStatusUnconfigured {
			t.Fatalf("expected empty legacy %s to remain unconfigured, got %+v", providerName, provider)
		}
		if provider.Enabled {
			t.Fatalf("expected empty legacy %s to remain disabled, got %+v", providerName, provider)
		}
		if len(provider.Values) != 0 {
			t.Fatalf("expected empty legacy %s values to be discarded, got %+v", providerName, provider.Values)
		}
	}
}

func TestSubfinderProviderSettingsDomainToModelPersistsRegistryValues(t *testing.T) {
	settings := &catalogdomain.SubfinderProviderSettings{
		ID: 1,
		Providers: catalogdomain.SubfinderProviderConfigs{
			"censys": {
				Enabled: true,
				Status:  catalogdomain.SubfinderProviderStatusConfigured,
				Values:  map[string]string{"pat": "censys-pat", "orgId": "org-1"},
			},
			"fofa": {
				Enabled: true,
				Status:  catalogdomain.SubfinderProviderStatusConfigured,
				Values:  map[string]string{"email": "test@example.com", "apiKey": "fofa-key"},
			},
		},
	}

	actual := subfinderProviderSettingsDomainToModel(settings)
	if actual == nil {
		t.Fatalf("expected non-nil persistence settings")
	}
	if actual.ID != 1 {
		t.Fatalf("expected id=1, got %d", actual.ID)
	}
	if actual.Providers["censys"].Values["pat"] != "censys-pat" || actual.Providers["censys"].Values["orgId"] != "org-1" {
		t.Fatalf("expected censys values persisted, got %+v", actual.Providers["censys"])
	}
	if actual.Providers["fofa"].Values["email"] != "test@example.com" || actual.Providers["fofa"].Values["apiKey"] != "fofa-key" {
		t.Fatalf("expected fofa values persisted, got %+v", actual.Providers["fofa"])
	}
	if actual.Providers["censys"].APIID != "" || actual.Providers["censys"].APISecret != "" {
		t.Fatalf("new registry persistence must not write legacy censys fields: %+v", actual.Providers["censys"])
	}
}
