package handler

import (
	"encoding/json"
	"strings"
	"testing"

	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

func TestSnapshotOutputsUseResourceNames(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "website", got: toWebsiteSnapshotOutput(&service.WebsiteSnapshot{ID: 7, ScanID: 3, URL: "https://example.com"}).Name, want: "scans/3/websiteSnapshots/7"},
		{name: "subdomain", got: toSubdomainSnapshotOutput(&service.SubdomainSnapshot{ID: 7, ScanID: 3, DNSName: "api.example.com"}).Name, want: "scans/3/subdomainSnapshots/7"},
		{name: "endpoint", got: toEndpointSnapshotOutput(&service.EndpointSnapshot{ID: 7, ScanID: 3, URL: "https://example.com/api"}).Name, want: "scans/3/endpointSnapshots/7"},
		{name: "directory", got: toDirectorySnapshotOutput(&service.DirectorySnapshot{ID: 7, ScanID: 3, URL: "https://example.com/admin"}).Name, want: "scans/3/directorySnapshots/7"},
		{name: "host port", got: toHostPortSnapshotOutput(&service.HostPortSnapshot{ID: 7, ScanID: 3, IP: "192.0.2.10"}).Name, want: "scans/3/hostPortSnapshots/7"},
		{name: "screenshot", got: toScreenshotSnapshotOutput(&service.ScreenshotSnapshot{ID: 7, ScanID: 3, URL: "https://example.com"}).Name, want: "scans/3/screenshotSnapshots/7"},
		{name: "vulnerability", got: toVulnerabilitySnapshotOutput(&service.VulnerabilitySnapshot{ID: 7, ScanID: 3, URL: "https://example.com"}).Name, want: "scans/3/vulnerabilitySnapshots/7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("expected resource name %q, got %q", tt.want, tt.got)
			}
		})
	}
}

func TestSubdomainSnapshotOutputUsesDNSNameBusinessField(t *testing.T) {
	output := toSubdomainSnapshotOutput(&service.SubdomainSnapshot{
		ID:      7,
		ScanID:  3,
		DNSName: "api.example.com",
	})

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
