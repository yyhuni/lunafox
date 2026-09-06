package scanworkflow

import (
	"strings"
	"testing"
)

func TestScanWorkflowResourceIdentityNamespacesAreDisjoint(t *testing.T) {
	for _, value := range []string{"default", "release_2026", "wf-123e4567-e89b-12d3-a456-426614174000"} {
		if err := ValidateScanWorkflowID(value); err != nil {
			t.Fatalf("ValidateScanWorkflowID(%q) error = %v", value, err)
		}
	}
	for _, value := range []string{"wf-123E4567-e89b-12d3-a456-426614174000", "wf-not-a-uuid", "wf-123e4567e89b12d3a456426614174000", " all"} {
		if err := ValidateScanWorkflowID(value); err == nil {
			t.Fatalf("ValidateScanWorkflowID(%q) succeeded", value)
		}
	}
}

func TestValidateWorkflowMetadataAllowsEmptyDescription(t *testing.T) {
	if err := ValidateWorkflowMetadata("Discovery", ""); err != nil {
		t.Fatalf("empty optional description rejected: %v", err)
	}
	if err := ValidateWorkflowMetadata("", "optional"); err == nil {
		t.Fatal("missing displayName succeeded")
	}
}

func TestCanonicalWorkflowDigestAndETagAreStable(t *testing.T) {
	stages := []Stage{{StageID: "discovery", Steps: []Step{{StepID: "discover", EngineID: "engine.lunafox.discovery", ProfileDefaultEnabled: true}}}}
	digest, err := CanonicalWorkflowDigest("wf-123e4567-e89b-12d3-a456-426614174000", "Discovery", "Run discovery.", stages)
	if err != nil {
		t.Fatal(err)
	}
	again, err := CanonicalWorkflowDigest("wf-123e4567-e89b-12d3-a456-426614174000", "Discovery", "Run discovery.", stages)
	if err != nil || digest != again {
		t.Fatalf("digest stability = %q, %q, %v", digest, again, err)
	}
	etag, err := NewETag("wf-123e4567-e89b-12d3-a456-426614174000", 4, digest)
	if err != nil {
		t.Fatal(err)
	}
	version, parsedDigest, err := ParseETag("wf-123e4567-e89b-12d3-a456-426614174000", etag)
	if err != nil || version != 4 || parsedDigest != digest {
		t.Fatalf("ParseETag() = %d, %q, %v", version, parsedDigest, err)
	}
	if _, _, err := ParseETag("default", etag); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected resource mismatch, got %v", err)
	}
}

func TestCanonicalWorkflowDigestIncludesProfileDefaultEnabled(t *testing.T) {
	base := []Stage{{StageID: "discovery", Steps: []Step{{
		StepID: "discover", EngineID: "engine.lunafox.discovery", ProfileDefaultEnabled: true,
	}}}}
	changed := []Stage{{StageID: "discovery", Steps: []Step{{
		StepID: "discover", EngineID: "engine.lunafox.discovery", ProfileDefaultEnabled: false,
	}}}}
	first, err := CanonicalWorkflowDigest("wf-123e4567-e89b-12d3-a456-426614174000", "Discovery", "Run discovery.", base)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CanonicalWorkflowDigest("wf-123e4567-e89b-12d3-a456-426614174000", "Discovery", "Run discovery.", changed)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("Profile default metadata did not affect canonical digest: %s", first)
	}
}
