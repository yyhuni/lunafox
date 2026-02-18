package validator

import "testing"

func TestIsURLMatchTarget(t *testing.T) {
	tests := []struct {
		name       string
		urlStr     string
		targetName string
		targetType string
		expected   bool
	}{
		// Domain type - exact match
		{"domain exact match", "https://example.com/path", "example.com", "domain", true},
		{"domain exact match with port", "https://example.com:8080/path", "example.com", "domain", true},

		// Domain type - suffix match
		{"domain suffix match", "https://api.example.com/path", "example.com", "domain", true},
		{"domain deep suffix match", "https://v1.api.example.com/path", "example.com", "domain", true},

		// Domain type - no match
		{"domain no match", "https://other.com/path", "example.com", "domain", false},
		{"domain partial no match", "https://notexample.com/path", "example.com", "domain", false},
		{"domain suffix partial no match", "https://fakeexample.com/path", "example.com", "domain", false},

		// IP type - exact match
		{"ip exact match", "https://192.168.1.1/path", "192.168.1.1", "ip", true},
		{"ip exact match with port", "https://192.168.1.1:8080/path", "192.168.1.1", "ip", true},

		// IP type - no match
		{"ip no match", "https://192.168.1.2/path", "192.168.1.1", "ip", false},
		{"ip hostname no match", "https://example.com/path", "192.168.1.1", "ip", false},

		// CIDR type - in range
		{"cidr in range", "https://10.1.2.3/path", "10.0.0.0/8", "cidr", true},
		{"cidr in range exact", "https://10.0.0.1/path", "10.0.0.0/8", "cidr", true},
		{"cidr /24 in range", "https://192.168.1.100/path", "192.168.1.0/24", "cidr", true},

		// CIDR type - out of range
		{"cidr out of range", "https://192.168.1.1/path", "10.0.0.0/8", "cidr", false},
		{"cidr hostname not ip", "https://example.com/path", "10.0.0.0/8", "cidr", false},

		// Edge cases
		{"empty url", "", "example.com", "domain", false},
		{"empty target", "https://example.com/path", "", "domain", false},
		{"invalid url", "not-a-url", "example.com", "domain", false},
		{"unknown target type", "https://example.com/path", "example.com", "unknown", false},

		// Case insensitive
		{"domain case insensitive", "https://EXAMPLE.COM/path", "example.com", "domain", true},
		{"domain case insensitive target", "https://example.com/path", "EXAMPLE.COM", "domain", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsURLMatchTarget(tt.urlStr, tt.targetName, tt.targetType)
			if result != tt.expected {
				t.Errorf("IsURLMatchTarget(%q, %q, %q) = %v, want %v",
					tt.urlStr, tt.targetName, tt.targetType, result, tt.expected)
			}
		})
	}
}

func TestIsSubdomainOfTarget(t *testing.T) {
	tests := []struct {
		name         string
		subdomain    string
		targetDomain string
		expected     bool
	}{
		// Exact match
		{"exact match", "example.com", "example.com", true},

		// Suffix match
		{"suffix match", "api.example.com", "example.com", true},
		{"deep suffix match", "v1.api.example.com", "example.com", true},

		// No match
		{"no match", "other.com", "example.com", false},
		{"partial no match", "notexample.com", "example.com", false},
		{"suffix partial no match", "fakeexample.com", "example.com", false},

		// Edge cases
		{"empty subdomain", "", "example.com", false},
		{"empty target", "api.example.com", "", false},
		{"both empty", "", "", false},
		{"whitespace subdomain", "  ", "example.com", false},
		{"whitespace target", "api.example.com", "  ", false},

		// Case insensitive
		{"case insensitive subdomain", "API.EXAMPLE.COM", "example.com", true},
		{"case insensitive target", "api.example.com", "EXAMPLE.COM", true},

		// With whitespace (should be trimmed)
		{"subdomain with whitespace", "  api.example.com  ", "example.com", true},
		{"target with whitespace", "api.example.com", "  example.com  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSubdomainOfTarget(tt.subdomain, tt.targetDomain)
			if result != tt.expected {
				t.Errorf("IsSubdomainOfTarget(%q, %q) = %v, want %v",
					tt.subdomain, tt.targetDomain, result, tt.expected)
			}
		})
	}
}


func TestDetectTargetType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Valid domains
		{"domain simple", "example.com", "domain"},
		{"domain with subdomain", "api.example.com", "domain"},
		{"domain with whitespace", "  example.com  ", "domain"},

		// Valid IPs
		{"ipv4", "192.168.1.1", "ip"},
		{"ipv4 with whitespace", "  192.168.1.1  ", "ip"},
		{"ipv6", "::1", "ip"},
		{"ipv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", "ip"},

		// Valid CIDRs
		{"cidr /8", "10.0.0.0/8", "cidr"},
		{"cidr /24", "192.168.1.0/24", "cidr"},
		{"cidr /32", "192.168.1.1/32", "cidr"},

		// Invalid IPs (should NOT be classified as domain)
		{"invalid ip 999", "999.999.999.999", ""},
		{"invalid ip 256", "256.256.256.256", ""},
		{"invalid ip partial", "192.168.1.999", ""},
		// Note: "1.2.3.4.5" is technically a valid DNS name, so it's classified as domain

		// Invalid inputs
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"invalid format", "not-valid!", ""},

		// Edge cases
		{"ip-like but valid domain", "1-2-3-4.example.com", "domain"},
		{"numeric domain", "1.2.3.4.5", "domain"}, // Valid DNS name with 5 parts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectTargetType(tt.input)
			if result != tt.expected {
				t.Errorf("DetectTargetType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLooksLikeIP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// IPv4-like
		{"valid ipv4 format", "192.168.1.1", true},
		{"invalid ipv4 values", "999.999.999.999", true},
		{"ipv4 with extra octet", "1.2.3.4.5", false},

		// IPv6-like
		{"ipv6 short", "::1", true},
		{"ipv6 full", "2001:db8::1", true},

		// Not IP-like
		{"domain", "example.com", false},
		{"domain with numbers", "api123.example.com", false},
		{"url", "https://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := looksLikeIP(tt.input)
			if result != tt.expected {
				t.Errorf("looksLikeIP(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
