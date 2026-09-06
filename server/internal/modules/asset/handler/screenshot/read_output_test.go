package screenshot

import (
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

func TestScreenshotOutputUsesResourceName(t *testing.T) {
	output := toScreenshotOutput(&assetdomain.Screenshot{
		ID:       7,
		TargetID: 3,
		URL:      "https://example.com",
	})

	if output.Name != "targets/3/screenshots/7" {
		t.Fatalf("expected resource name, got %q", output.Name)
	}
	if output.URL != "https://example.com" {
		t.Fatalf("expected URL business field, got %q", output.URL)
	}
}
