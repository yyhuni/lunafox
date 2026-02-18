package repository

import (
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
)

func TestSubfinderProviderSettingsModelToDomain(t *testing.T) {
	settings := &model.SubfinderProviderSettings{
		ID: 1,
		Providers: model.SubfinderProviderConfigs{
			"censys": {
				Enabled:   true,
				APIId:     "censys-id",
				APISecret: "censys-secret",
			},
			"fofa": {
				Enabled: true,
				Email:   "test@example.com",
				APIKey:  "fofa-key",
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
	if actual.Providers["censys"].APIID != "censys-id" {
		t.Fatalf("expected censys api id mapped, got %q", actual.Providers["censys"].APIID)
	}
	if actual.Providers["censys"].APISecret != "censys-secret" {
		t.Fatalf("expected censys api secret mapped, got %q", actual.Providers["censys"].APISecret)
	}
	if actual.Providers["fofa"].Email != "test@example.com" {
		t.Fatalf("expected fofa email mapped, got %q", actual.Providers["fofa"].Email)
	}
}

func TestSubfinderProviderSettingsDomainToModel(t *testing.T) {
	settings := &catalogdomain.SubfinderProviderSettings{
		ID: 1,
		Providers: catalogdomain.SubfinderProviderConfigs{
			"censys": {
				Enabled:   true,
				APIID:     "censys-id",
				APISecret: "censys-secret",
			},
			"fofa": {
				Enabled: true,
				Email:   "test@example.com",
				APIKey:  "fofa-key",
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
	if actual.Providers["censys"].APIId != "censys-id" {
		t.Fatalf("expected censys api id mapped, got %q", actual.Providers["censys"].APIId)
	}
	if actual.Providers["censys"].APISecret != "censys-secret" {
		t.Fatalf("expected censys api secret mapped, got %q", actual.Providers["censys"].APISecret)
	}
	if actual.Providers["fofa"].Email != "test@example.com" {
		t.Fatalf("expected fofa email mapped, got %q", actual.Providers["fofa"].Email)
	}
}
