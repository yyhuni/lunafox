package domain

import "testing"

func TestIsURLMatchTarget(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		target ScanTargetRef
		match  bool
	}{
		{
			name:   "domain match",
			rawURL: "https://a.example.com/path",
			target: ScanTargetRef{Name: "example.com", Type: "domain"},
			match:  true,
		},
		{
			name:   "domain mismatch",
			rawURL: "https://evil.com/path",
			target: ScanTargetRef{Name: "example.com", Type: "domain"},
			match:  false,
		},
		{
			name:   "ip match",
			rawURL: "http://1.1.1.1/api",
			target: ScanTargetRef{Name: "1.1.1.1", Type: "ip"},
			match:  true,
		},
		{
			name:   "cidr match",
			rawURL: "http://10.0.0.5/path",
			target: ScanTargetRef{Name: "10.0.0.0/24", Type: "cidr"},
			match:  true,
		},
		{
			name:   "invalid url",
			rawURL: "://bad-url",
			target: ScanTargetRef{Name: "example.com", Type: "domain"},
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

func TestIsSubdomainMatchTarget(t *testing.T) {
	tests := []struct {
		name      string
		subdomain string
		target    ScanTargetRef
		match     bool
	}{
		{
			name:      "subdomain match",
			subdomain: "a.example.com",
			target:    ScanTargetRef{Name: "example.com", Type: "domain"},
			match:     true,
		},
		{
			name:      "domain itself",
			subdomain: "example.com",
			target:    ScanTargetRef{Name: "example.com", Type: "domain"},
			match:     true,
		},
		{
			name:      "invalid dns",
			subdomain: "bad_host",
			target:    ScanTargetRef{Name: "example.com", Type: "domain"},
			match:     false,
		},
		{
			name:      "target not domain",
			subdomain: "a.example.com",
			target:    ScanTargetRef{Name: "1.1.1.1", Type: "ip"},
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

func TestIsDomainTargetType(t *testing.T) {
	if !IsDomainTargetType("domain") {
		t.Fatalf("domain should be recognized as domain type")
	}
	if IsDomainTargetType("ip") {
		t.Fatalf("ip should not be recognized as domain type")
	}
}
