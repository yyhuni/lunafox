package application

import (
	"errors"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type subfinderAPIKeySettingsStoreStub struct {
	settings *catalogdomain.SubfinderProviderSettings
	updated  *catalogdomain.SubfinderProviderSettings
	err      error
}

func (stub *subfinderAPIKeySettingsStoreStub) GetInstance() (*catalogdomain.SubfinderProviderSettings, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return stub.settings, nil
}

func (stub *subfinderAPIKeySettingsStoreStub) Update(settings *catalogdomain.SubfinderProviderSettings) error {
	copySettings := cloneSettingsForTest(settings)
	stub.updated = copySettings
	stub.settings = copySettings
	return nil
}

func TestSubfinderAPIKeySettingsServiceGetSettingsReturnsRegistryProviders(t *testing.T) {
	service := NewSubfinderAPIKeySettingsService(&subfinderAPIKeySettingsStoreStub{
		settings: &catalogdomain.SubfinderProviderSettings{Providers: catalogdomain.SubfinderProviderConfigs{
			"fofa": {Enabled: true, Values: map[string]string{"email": "ops@example.com", "apiKey": "fofa-secret"}},
		}},
	})

	settings, err := service.GetSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !settings.Providers["fofa"].Enabled || settings.Providers["fofa"].Values["email"] != "ops@example.com" || settings.Providers["fofa"].Values["apiKey"] != "fofa-secret" {
		t.Fatalf("unexpected fofa settings: %+v", settings.Providers["fofa"])
	}
	for _, definition := range catalogdomain.SubfinderProviderDefinitions() {
		if _, ok := settings.Providers[definition.Key]; !ok {
			t.Fatalf("expected provider %q to be present in settings", definition.Key)
		}
	}
	if _, ok := settings.Providers["hunter"]; ok {
		t.Fatalf("legacy hunter must not be active provider settings: %+v", settings.Providers["hunter"])
	}
}

func TestSubfinderAPIKeySettingsServiceUpdateSettingsPreservesOmittedProviders(t *testing.T) {
	store := &subfinderAPIKeySettingsStoreStub{
		settings: &catalogdomain.SubfinderProviderSettings{Providers: catalogdomain.SubfinderProviderConfigs{
			"fofa":   {Enabled: true, Values: map[string]string{"email": "ops@example.com", "apiKey": "fofa-secret"}},
			"shodan": {Enabled: true, Values: map[string]string{"apiKey": "old-shodan"}},
		}},
	}
	service := NewSubfinderAPIKeySettingsService(store)

	settings, err := service.UpdateSettings(catalogdomain.SubfinderProviderConfigs{
		"shodan": {Enabled: true, Values: map[string]string{"apiKey": "new-shodan"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.Providers["shodan"].Values["apiKey"] != "new-shodan" {
		t.Fatalf("expected shodan to update, got %+v", settings.Providers["shodan"])
	}
	if settings.Providers["fofa"].Values["apiKey"] != "fofa-secret" || settings.Providers["fofa"].Values["email"] != "ops@example.com" {
		t.Fatalf("expected omitted fofa settings to be preserved, got %+v", settings.Providers["fofa"])
	}
	if store.updated == nil || store.updated.ID != 1 {
		t.Fatalf("expected singleton update with id=1, got %+v", store.updated)
	}
}

func TestSubfinderAPIKeySettingsServiceUpdateSettingsRejectsUnsupportedProvider(t *testing.T) {
	service := NewSubfinderAPIKeySettingsService(&subfinderAPIKeySettingsStoreStub{})

	_, err := service.UpdateSettings(catalogdomain.SubfinderProviderConfigs{
		"unknown": {Enabled: true, Values: map[string]string{"apiKey": "secret"}},
	})
	if !errors.Is(err, ErrUnsupportedSubfinderProvider) {
		t.Fatalf("expected ErrUnsupportedSubfinderProvider, got %v", err)
	}
}

func TestSubfinderAPIKeySettingsServiceUpdateSettingsRejectsMissingRequiredFields(t *testing.T) {
	service := NewSubfinderAPIKeySettingsService(&subfinderAPIKeySettingsStoreStub{})

	_, err := service.UpdateSettings(catalogdomain.SubfinderProviderConfigs{
		"fofa": {Enabled: true, Values: map[string]string{"email": "ops@example.com"}},
	})
	if !errors.Is(err, ErrInvalidSubfinderProviderSettings) {
		t.Fatalf("expected ErrInvalidSubfinderProviderSettings, got %v", err)
	}
}

func cloneSettingsForTest(settings *catalogdomain.SubfinderProviderSettings) *catalogdomain.SubfinderProviderSettings {
	copySettings := *settings
	copySettings.Providers = make(catalogdomain.SubfinderProviderConfigs, len(settings.Providers))
	for providerName, providerConfig := range settings.Providers {
		values := make(map[string]string, len(providerConfig.Values))
		for fieldName, value := range providerConfig.Values {
			values[fieldName] = value
		}
		providerConfig.Values = values
		copySettings.Providers[providerName] = providerConfig
	}
	return &copySettings
}
