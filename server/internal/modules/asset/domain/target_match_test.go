package domain

import "testing"

func TestAssetIsURLMatchTarget(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		target TargetRef
		match  bool
	}{
		{
			name:   "domain match",
			rawURL: "https://a.example.com/path",
			target: TargetRef{Name: "example.com", Type: "domain"},
			match:  true,
		},
		{
			name:   "domain mismatch",
			rawURL: "https://evil.com/path",
			target: TargetRef{Name: "example.com", Type: "domain"},
			match:  false,
		},
		{
			name:   "ip match",
			rawURL: "http://1.1.1.1/api",
			target: TargetRef{Name: "1.1.1.1", Type: "ip"},
			match:  true,
		},
		{
			name:   "cidr match",
			rawURL: "http://10.0.0.5/path",
			target: TargetRef{Name: "10.0.0.0/24", Type: "cidr"},
			match:  true,
		},
		{
			name:   "invalid url",
			rawURL: "://bad-url",
			target: TargetRef{Name: "example.com", Type: "domain"},
			match:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual := IsURLMatchTarget(testCase.rawURL, testCase.target)
			if actual != testCase.match {
				t.Fatalf("unexpected match result want=%v got=%v", testCase.match, actual)
			}
		})
	}
}

func TestAssetIsSubdomainMatchTarget(t *testing.T) {
	tests := []struct {
		name      string
		subdomain string
		target    TargetRef
		match     bool
	}{
		{
			name:      "subdomain match",
			subdomain: "a.example.com",
			target:    TargetRef{Name: "example.com", Type: "domain"},
			match:     true,
		},
		{
			name:      "domain itself",
			subdomain: "example.com",
			target:    TargetRef{Name: "example.com", Type: "domain"},
			match:     true,
		},
		{
			name:      "invalid dns",
			subdomain: "bad_host",
			target:    TargetRef{Name: "example.com", Type: "domain"},
			match:     false,
		},
		{
			name:      "target not domain",
			subdomain: "a.example.com",
			target:    TargetRef{Name: "1.1.1.1", Type: "ip"},
			match:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual := IsSubdomainMatchTarget(testCase.subdomain, testCase.target)
			if actual != testCase.match {
				t.Fatalf("unexpected subdomain match want=%v got=%v", testCase.match, actual)
			}
		})
	}
}

func TestAssetExtractHostFromURL(t *testing.T) {
	host := ExtractHostFromURL("https://example.com:8443/path")
	if host != "example.com:8443" {
		t.Fatalf("unexpected host: %s", host)
	}

	empty := ExtractHostFromURL("::bad")
	if empty != "" {
		t.Fatalf("expected empty host for invalid URL, got %s", empty)
	}
}
