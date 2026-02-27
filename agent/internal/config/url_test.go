package config

import "testing"

func TestValidateRuntimeGRPCURL(t *testing.T) {
	validURLs := []string{
		"https://example.com",
		"http://example.com",
		"https://example.com/api",
		"https://example.com/base",
		"wss://example.com",
		"ws://example.com",
	}

	for _, input := range validURLs {
		if err := validateRuntimeGRPCURL(input); err != nil {
			t.Fatalf("unexpected error for %s: %v", input, err)
		}
	}
}

func TestValidateRuntimeGRPCURLInvalid(t *testing.T) {
	if err := validateRuntimeGRPCURL("example.com"); err == nil {
		t.Fatalf("expected error for missing scheme")
	}
	if err := validateRuntimeGRPCURL(" "); err == nil {
		t.Fatalf("expected error for empty url")
	}
	if err := validateRuntimeGRPCURL("ftp://example.com"); err == nil {
		t.Fatalf("expected error for unsupported scheme")
	}
	if err := validateRuntimeGRPCURL("https://"); err == nil {
		t.Fatalf("expected error for missing host")
	}
}
