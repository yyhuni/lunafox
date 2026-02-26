package config

import "testing"

func TestValidateServerURL(t *testing.T) {
	validURLs := []string{
		"https://example.com",
		"http://example.com",
		"https://example.com/api",
		"https://example.com/base",
		"wss://example.com",
		"ws://example.com",
	}

	for _, input := range validURLs {
		if err := validateServerURL(input); err != nil {
			t.Fatalf("unexpected error for %s: %v", input, err)
		}
	}
}

func TestValidateServerURLInvalid(t *testing.T) {
	if err := validateServerURL("example.com"); err == nil {
		t.Fatalf("expected error for missing scheme")
	}
	if err := validateServerURL(" "); err == nil {
		t.Fatalf("expected error for empty url")
	}
	if err := validateServerURL("ftp://example.com"); err == nil {
		t.Fatalf("expected error for unsupported scheme")
	}
	if err := validateServerURL("https://"); err == nil {
		t.Fatalf("expected error for missing host")
	}
}
