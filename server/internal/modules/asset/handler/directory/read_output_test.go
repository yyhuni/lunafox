package directory

import (
	"testing"

	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

func TestDirectoryOutputUsesResourceName(t *testing.T) {
	output := toDirectoryOutput(&service.Directory{
		ID:       7,
		TargetID: 3,
		URL:      "https://example.com/admin",
	})

	if output.Name != "targets/3/directories/7" {
		t.Fatalf("expected resource name, got %q", output.Name)
	}
	if output.URL != "https://example.com/admin" {
		t.Fatalf("expected URL business field, got %q", output.URL)
	}
}
