package website

import (
	"testing"
	"time"

	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

func TestWebsiteOutputUsesResourceName(t *testing.T) {
	output := toWebsiteOutput(&service.WebsiteReadModel{Website: service.Website{
		ID:       7,
		TargetID: 3,
		URL:      "https://example.com",
		Host:     "example.com",
	}})
	if output.Name != "targets/3/websites/7" {
		t.Fatalf("expected resource name, got %q", output.Name)
	}
	if output.URL != "https://example.com" || output.Host != "example.com" {
		t.Fatalf("expected business fields to keep URL and host, got %#v", output)
	}
}

func TestWebsiteOutputIncludesScreenshotSummary(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	status := int16(200)
	output := toWebsiteOutput(&service.WebsiteReadModel{
		Website: service.Website{ID: 7, TargetID: 3, URL: "https://example.com", Host: "example.com"},
		Screenshot: &service.WebsiteScreenshotSummary{
			ID:         11,
			URL:        "https://example.com",
			StatusCode: &status,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	})
	if output.Screenshot == nil {
		t.Fatal("expected screenshot summary")
	}
	if output.Screenshot.Name != "targets/3/screenshots/11" || output.Screenshot.URL != "https://example.com" {
		t.Fatalf("unexpected screenshot summary: %+v", output.Screenshot)
	}
}
