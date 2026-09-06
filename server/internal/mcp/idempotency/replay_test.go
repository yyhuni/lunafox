package idempotency

import (
	"strings"
	"testing"
)

func TestNormalizeRequestIDUsesOneBoundedBusinessKey(t *testing.T) {
	got, err := NormalizeRequestID("  caller-retry-1  ")
	if err != nil || got != "caller-retry-1" {
		t.Fatalf("NormalizeRequestID() = %q, %v", got, err)
	}
	if got, err := NormalizeRequestID(" \t "); err != nil || got != "" {
		t.Fatalf("omitted request ID = %q, %v", got, err)
	}
	for _, value := range []string{"line\nbreak", strings.Repeat("x", MaxRequestIDLength+1)} {
		if _, err := NormalizeRequestID(value); err == nil {
			t.Fatalf("NormalizeRequestID(%q) unexpectedly succeeded", value)
		}
	}
}

func TestFingerprintCanonicalizesObjectKeysButPreservesArrayOrder(t *testing.T) {
	first, err := Fingerprint("review_vulnerability", map[string]any{
		"configuration": map[string]any{"threads": 10, "timeout": 30},
		"names":         []string{"vulnerabilities/1", "vulnerabilities/2"},
	})
	if err != nil {
		t.Fatalf("first fingerprint: %v", err)
	}
	second, err := Fingerprint("review_vulnerability", map[string]any{
		"names":         []string{"vulnerabilities/1", "vulnerabilities/2"},
		"configuration": map[string]any{"timeout": 30, "threads": 10},
	})
	if err != nil || first != second {
		t.Fatalf("object-key canonicalization = %q, %q, %v", first, second, err)
	}
	reordered, err := Fingerprint("review_vulnerability", map[string]any{
		"names":         []string{"vulnerabilities/2", "vulnerabilities/1"},
		"configuration": map[string]any{"timeout": 30, "threads": 10},
	})
	if err != nil {
		t.Fatalf("reordered fingerprint: %v", err)
	}
	if reordered == first {
		t.Fatal("array order was lost from fingerprint")
	}
}
