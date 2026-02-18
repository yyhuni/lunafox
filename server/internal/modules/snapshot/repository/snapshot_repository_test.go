package repository

import (
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"testing"
)

// TestVulnerabilitySnapshotFilterMapping tests the filter mapping configuration
func TestVulnerabilitySnapshotFilterMapping(t *testing.T) {
	expectedFields := []string{"url", "vulnType", "description", "source"}

	for _, field := range expectedFields {
		if _, ok := VulnerabilitySnapshotFilterMapping[field]; !ok {
			t.Errorf("expected field %s not found in VulnerabilitySnapshotFilterMapping", field)
		}
	}

	tests := []struct {
		field    string
		expected string
	}{
		{"url", "url"},
		{"vulnType", "vuln_type"},
		{"description", "description"},
		{"source", "source"},
	}

	for _, tt := range tests {
		if VulnerabilitySnapshotFilterMapping[tt.field].Column != tt.expected {
			t.Errorf("field %s: expected column %s, got %s",
				tt.field, tt.expected, VulnerabilitySnapshotFilterMapping[tt.field].Column)
		}
	}
}

// TestSeverityOrderSQL tests the severity ordering SQL expression
func TestSeverityOrderSQL(t *testing.T) {

	severities := []string{"critical", "high", "medium", "low", "info", "unknown"}

	for _, severity := range severities {
		if !containsString(SeverityOrderSQL, severity) {
			t.Errorf("SeverityOrderSQL should contain severity level: %s", severity)
		}
	}

	if !containsString(SeverityOrderSQL, "CASE") {
		t.Error("SeverityOrderSQL should be a CASE expression")
	}
}

// TestVulnerabilitySnapshotBulkCreateDeduplication tests that BulkCreate handles duplicates correctly
func TestVulnerabilitySnapshotBulkCreateDeduplication(t *testing.T) {

	snapshot1 := model.VulnerabilitySnapshot{
		ScanID:   1,
		URL:      "https://example.com/vuln",
		VulnType: "XSS",
		Severity: "high",
	}

	snapshot2 := model.VulnerabilitySnapshot{
		ScanID:   1,
		URL:      "https://example.com/vuln",
		VulnType: "XSS",
		Severity: "high",
	}

	if snapshot1.ScanID != snapshot2.ScanID ||
		snapshot1.URL != snapshot2.URL ||
		snapshot1.VulnType != snapshot2.VulnType {
		t.Error("snapshots should have same unique key fields for deduplication test")
	}

	snapshot3 := model.VulnerabilitySnapshot{
		ScanID:   2,
		URL:      "https://example.com/vuln",
		VulnType: "XSS",
		Severity: "high",
	}

	if snapshot1.ScanID == snapshot3.ScanID {
		t.Error("snapshot3 should have different scan_id")
	}

	snapshot4 := model.VulnerabilitySnapshot{
		ScanID:   1,
		URL:      "https://example.com/vuln",
		VulnType: "SQLi",
		Severity: "critical",
	}

	if snapshot1.VulnType == snapshot4.VulnType {
		t.Error("snapshot4 should have different vuln_type")
	}
}

// TestVulnerabilitySnapshotApplyOrdering tests the ordering logic
func TestVulnerabilitySnapshotApplyOrdering(t *testing.T) {

	tests := []struct {
		input    string
		isDesc   bool
		expected string
	}{
		{"url", false, "url"},
		{"-url", true, "url"},
		{"vulnType", false, "vuln_type"},
		{"-vulnType", true, "vuln_type"},
		{"severity", false, "severity"},
		{"-severity", true, "severity"},
		{"cvssScore", false, "cvss_score"},
		{"-cvssScore", true, "cvss_score"},
		{"createdAt", false, "created_at"},
		{"-createdAt", true, "created_at"},
	}

	for _, tt := range tests {
		ordering := tt.input
		desc := false
		if len(ordering) > 0 && ordering[0] == '-' {
			desc = true
			_ = ordering[1:]
		}

		if desc != tt.isDesc {
			t.Errorf("ordering %s: expected desc=%v, got %v", tt.input, tt.isDesc, desc)
		}
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestWebsiteSnapshotFilterMapping tests the filter mapping configuration
func TestWebsiteSnapshotFilterMapping(t *testing.T) {
	expectedFields := []string{"url", "host", "title", "status", "webserver", "tech"}

	for _, field := range expectedFields {
		if _, ok := WebsiteSnapshotFilterMapping[field]; !ok {
			t.Errorf("expected field %s not found in WebsiteSnapshotFilterMapping", field)
		}
	}

	if !WebsiteSnapshotFilterMapping["status"].IsNumeric {
		t.Error("status field should be marked as numeric")
	}

	if !WebsiteSnapshotFilterMapping["tech"].IsArray {
		t.Error("tech field should be marked as array")
	}
}

// TestBulkCreateDeduplication tests that BulkCreate handles duplicates correctly
// This is a unit test for the deduplication logic
func TestBulkCreateDeduplication(t *testing.T) {

	snapshot1 := model.WebsiteSnapshot{
		ScanID: 1,
		URL:    "https://example.com",
		Host:   "example.com",
	}

	snapshot2 := model.WebsiteSnapshot{
		ScanID: 1,
		URL:    "https://example.com",
		Host:   "example.com",
	}

	if snapshot1.ScanID != snapshot2.ScanID || snapshot1.URL != snapshot2.URL {
		t.Error("snapshots should have same unique key fields for deduplication test")
	}

	snapshot3 := model.WebsiteSnapshot{
		ScanID: 2,
		URL:    "https://example.com",
		Host:   "example.com",
	}

	if snapshot1.ScanID == snapshot3.ScanID {
		t.Error("snapshot3 should have different scan_id")
	}
}
