package validator

import (
	"net"
	"strings"

	"golang.org/x/net/idna"
)

// NormalizeSubdomainLine normalizes a subdomain emitted by external tools.
// Invalid tool output lines return ok=false so stream parsers can skip them.
func NormalizeSubdomainLine(s string) (normalized string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	// Skip comment lines (common in tool outputs)
	if strings.HasPrefix(s, "#") {
		return "", false
	}
	// Skip lines with spaces (likely error messages or headers)
	if strings.Contains(s, " ") {
		return "", false
	}
	s = strings.ToLower(strings.TrimSuffix(s, "."))
	ascii, err := idna.Lookup.ToASCII(s)
	if err != nil {
		return "", false
	}
	canonical := strings.ToLower(ascii)
	if len(canonical) > 253 || net.ParseIP(canonical) != nil {
		return "", false
	}
	labels := strings.Split(canonical, ".")
	if len(labels) < 2 {
		return "", false
	}
	for _, label := range labels {
		if !validDNSLabel(label) {
			return "", false
		}
	}
	return canonical, true
}

// IsValidSubdomainFormat performs fast basic validation for subdomain format.
// For normalized output, use NormalizeSubdomainLine.
func IsValidSubdomainFormat(s string) bool {
	_, ok := NormalizeSubdomainLine(s)
	return ok
}

func validDNSLabel(label string) bool {
	if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, value := range label {
		if (value >= 'a' && value <= 'z') || (value >= '0' && value <= '9') || value == '-' {
			continue
		}
		return false
	}
	return true
}
