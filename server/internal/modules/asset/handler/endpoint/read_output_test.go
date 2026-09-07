package endpoint

import (
	"testing"

	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

func TestEndpointOutputUsesResourceName(t *testing.T) {
	output := toEndpointOutput(&service.Endpoint{
		ID:       7,
		TargetID: 3,
		URL:      "https://example.com/api",
		Host:     "example.com",
	})

	if output.Name != "targets/3/endpoints/7" {
		t.Fatalf("expected resource name, got %q", output.Name)
	}
	if output.URL != "https://example.com/api" || output.Host != "example.com" {
		t.Fatalf("expected business fields to keep URL and host, got %#v", output)
	}
}
