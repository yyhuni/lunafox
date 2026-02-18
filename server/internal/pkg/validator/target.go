package validator

import (
	"net"
	"net/url"
	"strings"

	"github.com/asaskevich/govalidator"
)

// Target type constants
const (
	TargetTypeDomain = "domain"
	TargetTypeIP     = "ip"
	TargetTypeCIDR   = "cidr"
)

// IsURLMatchTarget checks if URL hostname matches target
// Returns true if the URL's hostname belongs to the target
//
// Matching rules by target type:
//   - domain: hostname equals target or ends with .target
//   - ip: hostname must exactly equal target
//   - cidr: hostname must be an IP within the CIDR range
func IsURLMatchTarget(urlStr, targetName, targetType string) bool {
	if urlStr == "" || targetName == "" {
		return false
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return false
	}

	targetName = strings.ToLower(targetName)

	switch targetType {
	case TargetTypeDomain:
		// hostname equals target or ends with .target
		return hostname == targetName || strings.HasSuffix(hostname, "."+targetName)

	case TargetTypeIP:
		// hostname must exactly equal target
		return hostname == targetName

	case TargetTypeCIDR:
		// hostname must be an IP within the CIDR range
		ip := net.ParseIP(hostname)
		if ip == nil {
			return false
		}
		_, network, err := net.ParseCIDR(targetName)
		if err != nil {
			return false
		}
		return network.Contains(ip)

	default:
		return false
	}
}

// IsSubdomainOfTarget checks if subdomain belongs to target domain
// Returns true if subdomain is a valid DNS name and equals target or ends with .target
func IsSubdomainOfTarget(subdomain, targetDomain string) bool {
	subdomain = strings.ToLower(strings.TrimSpace(subdomain))
	targetDomain = strings.ToLower(strings.TrimSpace(targetDomain))

	if subdomain == "" || targetDomain == "" {
		return false
	}

	// Validate DNS name format
	if !govalidator.IsDNSName(subdomain) {
		return false
	}

	return subdomain == targetDomain || strings.HasSuffix(subdomain, "."+targetDomain)
}

// DetectTargetType auto-detects target type from input string.
// Returns empty string if the input is not a valid target format.
func DetectTargetType(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// Check CIDR first (must be before IP check since "10.0.0.0" is valid for both)
	if _, _, err := net.ParseCIDR(name); err == nil {
		return TargetTypeCIDR
	}

	// Check IP
	if net.ParseIP(name) != nil {
		return TargetTypeIP
	}

	// Check if it looks like an IP but is invalid (e.g., 999.999.999.999)
	// This prevents invalid IPs from being classified as domains
	if looksLikeIP(name) {
		return "" // Invalid IP format
	}

	// Check domain
	if govalidator.IsDNSName(name) {
		return TargetTypeDomain
	}

	return ""
}

// looksLikeIP checks if a string looks like an IP address format
// (e.g., "999.999.999.999" or "::gggg")
func looksLikeIP(s string) bool {
	// Check for IPv4-like format (digits and dots only)
	if strings.Count(s, ".") == 3 {
		parts := strings.Split(s, ".")
		allNumeric := true
		for _, part := range parts {
			if part == "" {
				allNumeric = false
				break
			}
			for _, c := range part {
				if c < '0' || c > '9' {
					allNumeric = false
					break
				}
			}
			if !allNumeric {
				break
			}
		}
		if allNumeric {
			return true
		}
	}

	// Check for IPv6-like format (contains colons)
	if strings.Contains(s, ":") && !strings.Contains(s, "://") {
		return true
	}

	return false
}
