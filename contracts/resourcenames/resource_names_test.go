package resourcenames

import "testing"

func TestScanResourceIdentity(t *testing.T) {
	if got := Scan(7); got != "scans/7" {
		t.Fatalf("Scan(7) = %q", got)
	}
	if got, err := ParseScan("scans/7"); err != nil || got != 7 {
		t.Fatalf("ParseScan(scans/7) = %d, %v", got, err)
	}
	for _, value := range []string{"", "scan/7", "scans/0", "scans/-1", "scans/7/tasks/1"} {
		if _, err := ParseScan(value); err == nil {
			t.Fatalf("ParseScan(%q) succeeded", value)
		}
	}
	if got, err := ParseScan(" scans / 7 "); err != nil || Scan(got) != "scans/7" {
		t.Fatalf("ParseScan must normalize parseable input for a caller-owned round-trip check: %d, %v", got, err)
	}
}

func TestWordlistResourceIdentityUsesPositiveCatalogID(t *testing.T) {
	if got := Wordlist(7); got != "wordlists/7" {
		t.Fatalf("Wordlist(7) = %q", got)
	}
	if got, err := ParseWordlist("wordlists/7"); err != nil || got != 7 {
		t.Fatalf("ParseWordlist(wordlists/7) = %d, %v", got, err)
	}
	for _, value := range []string{"", "resolvers.txt", "wordlists/default", "wordlists/0", "wordlists/-1", "wordlists/7/extra", "../wordlists/7"} {
		if _, err := ParseWordlist(value); err == nil {
			t.Fatalf("ParseWordlist(%q) succeeded", value)
		}
	}
}

func TestEnginePackageResourceIdentity(t *testing.T) {
	name := EnginePackage("engine.lunafox.subdomain_discovery", "sha256-abcdef")
	if name != "engines/engine.lunafox.subdomain_discovery/packages/sha256-abcdef" {
		t.Fatalf("unexpected engine package resource name: %q", name)
	}

	engineID, packageID, err := ParseEnginePackage(name)
	if err != nil {
		t.Fatalf("ParseEnginePackage failed: %v", err)
	}
	if engineID != "engine.lunafox.subdomain_discovery" {
		t.Fatalf("expected engine id, got %q", engineID)
	}
	if packageID != "sha256-abcdef" {
		t.Fatalf("expected package id, got %q", packageID)
	}
}

func TestParseEnginePackageRejectsNonCanonicalNames(t *testing.T) {
	tests := []string{
		"",
		"enginePackages/pkg",
		"engines/engine.lunafox.subdomain_discovery",
		"engines/engine.lunafox.subdomain_discovery/packages",
		"engines/engine.lunafox.subdomain_discovery/packages/runtime.subdomain_discovery",
		"engines/engine.lunafox.subdomain_discovery/packages/pkg/extra",
		"engines//packages/pkg",
		"engines/engine.lunafox.subdomain_discovery/packages/../escape",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			if _, _, err := ParseEnginePackage(value); err == nil {
				t.Fatalf("ParseEnginePackage(%q) succeeded, want error", value)
			}
		})
	}
}

func TestParseEngineRejectsNestedSubresources(t *testing.T) {
	tests := []string{
		"engines/engine.lunafox.subdomain_discovery/packages/sha256-pkg",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			if _, err := ParseEngine(value); err == nil {
				t.Fatalf("ParseEngine(%q) succeeded, want error", value)
			}
		})
	}
}

func TestParseScanWorkflowRejectsManifestTypedRefsAsResourceIDs(t *testing.T) {
	tests := []string{
		"scanWorkflows/engine.lunafox.subdomain_discovery",
		"scanWorkflows/runtime.subdomain_discovery",
		"scanWorkflows/schema.engine.subdomain_discovery",
		"scanWorkflows/workflow.subdomain_discovery",
	}
	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			if _, err := ParseScanWorkflow(value); err == nil {
				t.Fatalf("expected manifest typed ref workflow resource to fail")
			}
		})
	}
}

func TestParseScanWorkflowAcceptsCanonicalScanWorkflowID(t *testing.T) {
	workflowID, err := ParseScanWorkflow("scanWorkflows/wf-3f2504e0-4f89-41d3-9a0c-0305e82c3301")
	if err != nil {
		t.Fatalf("ParseScanWorkflow failed: %v", err)
	}
	if workflowID != "wf-3f2504e0-4f89-41d3-9a0c-0305e82c3301" {
		t.Fatalf("unexpected workflow ID: %q", workflowID)
	}
}

func TestParseScanWorkflowRejectsDefinitionInvalidIDs(t *testing.T) {
	tests := []string{
		"scanWorkflows/1bad",
		"scanWorkflows/_bad",
		"scanWorkflows/all",
		"scanWorkflows/bad-name",
		"scanWorkflows/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			if _, err := ParseScanWorkflow(value); err == nil {
				t.Fatalf("ParseScanWorkflow(%q) succeeded, want error", value)
			}
		})
	}
}

func TestBlacklistPolicyResourceIdentity(t *testing.T) {
	if got := BlacklistPolicy(); got != "blacklistPolicy" {
		t.Fatalf("BlacklistPolicy() = %q", got)
	}
	if err := ParseBlacklistPolicy("blacklistPolicy"); err != nil {
		t.Fatalf("ParseBlacklistPolicy() error = %v", err)
	}
	for _, value := range []string{"", "blacklistPolicies/1", "blacklistPolicy/extra"} {
		if err := ParseBlacklistPolicy(value); err == nil {
			t.Fatalf("ParseBlacklistPolicy(%q) succeeded", value)
		}
	}

	name := TargetBlacklistPolicy(42)
	if name != "targets/42/blacklistPolicy" {
		t.Fatalf("TargetBlacklistPolicy(42) = %q", name)
	}
	if targetID, err := ParseTargetBlacklistPolicy(name); err != nil || targetID != 42 {
		t.Fatalf("ParseTargetBlacklistPolicy(%q) = %d, %v", name, targetID, err)
	}
	for _, value := range []string{"targets/0/blacklistPolicy", "targets/42/blacklistPolicies", "blacklistPolicy", "targets/42/blacklistPolicy/extra"} {
		if _, err := ParseTargetBlacklistPolicy(value); err == nil {
			t.Fatalf("ParseTargetBlacklistPolicy(%q) succeeded", value)
		}
	}
}
