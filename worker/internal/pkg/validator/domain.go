package validator

import (
	"strings"

	"github.com/asaskevich/govalidator"
	iputil "github.com/projectdiscovery/utils/ip"
	"golang.org/x/net/idna"
)

// ValidateDomain checks if the input is a valid domain name
// Returns error if invalid
// Note: This function only validates, does not normalize
func ValidateDomain(domain string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ErrEmptyDomain
	}

	if !govalidator.IsDNSName(domain) {
		return ErrInvalidDomain
	}

	return nil
}

// NormalizeDomain normalizes a domain name:
// - Converts to lowercase
// - Handles IDN (punycode) conversion
// - Removes trailing dots
// - Trims whitespace
func NormalizeDomain(domain string) (string, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return "", ErrEmptyDomain
	}

	// Convert to lowercase
	domain = strings.ToLower(domain)

	// Remove trailing dot (FQDN format)
	domain = strings.TrimSuffix(domain, ".")

	// Handle IDN (internationalized domain names)
	// Convert to ASCII (punycode) if needed
	ascii, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return "", ErrInvalidDomain
	}

	return ascii, nil
}

// IsValidSubdomainFormat performs fast basic validation for subdomain format
// This is optimized for parsing tool outputs and may allow some edge cases
// For strict validation, use ValidateDomain instead
func IsValidSubdomainFormat(s string) bool {
	if s == "" {
		return false
	}
	// Skip comment lines (common in tool outputs)
	if strings.HasPrefix(s, "#") {
		return false
	}
	// Skip lines with spaces (likely error messages or headers)
	if strings.Contains(s, " ") {
		return false
	}
	// Skip IP addresses first (faster than DNS validation)
	if iputil.IsIP(s) {
		return false
	}
	// Use standard DNS name validation
	return govalidator.IsDNSName(s)
}