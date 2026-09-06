package subdomain

import (
	"encoding/json"
	"strings"
	"testing"

	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

func TestSubdomainOutputUsesResourceNameAndDNSName(t *testing.T) {
	output := toSubdomainOutput(&service.Subdomain{
		ID:       7,
		TargetID: 3,
		DNSName:  "api.example.com",
	})

	if output.Name != "targets/3/subdomains/7" {
		t.Fatalf("expected resource name, got %q", output.Name)
	}
	if output.DNSName != "api.example.com" {
		t.Fatalf("expected DNS name business field, got %q", output.DNSName)
	}

	payload, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	if !strings.Contains(string(payload), `"dnsName":"api.example.com"`) {
		t.Fatalf("expected dnsName JSON field, got %s", payload)
	}
	if strings.Contains(string(payload), `"hostname"`) {
		t.Fatalf("hostname JSON field must not be emitted, got %s", payload)
	}
}
