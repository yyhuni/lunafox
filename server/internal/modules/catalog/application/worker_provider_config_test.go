package application

import (
	"errors"
	"strings"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type workerScanGuardStub struct {
	called bool
	lastID int
	err    error
}

func (stub *workerScanGuardStub) EnsureActiveByID(id int) error {
	stub.called = true
	stub.lastID = id
	return stub.err
}

type workerSettingsStoreStub struct {
	settings *catalogdomain.SubfinderProviderSettings
	err      error
}

func (stub *workerSettingsStoreStub) GetInstance() (*catalogdomain.SubfinderProviderSettings, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return stub.settings, nil
}

func TestWorkerProviderConfigServiceGetProviderConfigToolRequired(t *testing.T) {
	service := NewWorkerProviderConfigService(&workerScanGuardStub{}, &workerSettingsStoreStub{})

	_, err := service.GetProviderConfig(1, "  ")
	if !errors.Is(err, ErrWorkerToolRequired) {
		t.Fatalf("expected ErrWorkerToolRequired, got %v", err)
	}
}

func TestWorkerProviderConfigServiceGetProviderConfigScanGuardError(t *testing.T) {
	guard := &workerScanGuardStub{err: ErrWorkerScanNotFound}
	service := NewWorkerProviderConfigService(guard, &workerSettingsStoreStub{})

	_, err := service.GetProviderConfig(9, "subfinder")
	if !errors.Is(err, ErrWorkerScanNotFound) {
		t.Fatalf("expected ErrWorkerScanNotFound, got %v", err)
	}
	if !guard.called || guard.lastID != 9 {
		t.Fatalf("expected scan guard called with id=9, got called=%v id=%d", guard.called, guard.lastID)
	}
}

func TestWorkerProviderConfigServiceGetProviderConfigSettingsNotFound(t *testing.T) {
	service := NewWorkerProviderConfigService(&workerScanGuardStub{}, &workerSettingsStoreStub{err: ErrWorkerProviderConfigSettingsNotFound})

	config, err := service.GetProviderConfig(1, "subfinder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config != "" {
		t.Fatalf("expected empty config, got %q", config)
	}
}

func TestWorkerProviderConfigServiceGetProviderConfigSubfinder(t *testing.T) {
	guard := &workerScanGuardStub{}
	settings := &catalogdomain.SubfinderProviderSettings{Providers: catalogdomain.SubfinderProviderConfigs{
		"fofa": {Enabled: true, Email: "test@example.com", APIKey: "secret"},
	}}
	service := NewWorkerProviderConfigService(guard, &workerSettingsStoreStub{settings: settings})

	config, err := service.GetProviderConfig(7, "subfinder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !guard.called || guard.lastID != 7 {
		t.Fatalf("expected scan guard called with id=7, got called=%v id=%d", guard.called, guard.lastID)
	}
	if !strings.Contains(config, "fofa:") {
		t.Fatalf("expected config contains fofa section, got %q", config)
	}
	if !strings.Contains(config, "test@example.com:secret") {
		t.Fatalf("expected fofa credential in config, got %q", config)
	}
}
