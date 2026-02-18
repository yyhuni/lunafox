package domain

import (
	"net"
	"net/url"
	"strings"

	"github.com/asaskevich/govalidator"
)

const (
	TargetTypeDomain = "domain"
	TargetTypeIP     = "ip"
	TargetTypeCIDR   = "cidr"
)

func NormalizeTargetType(targetType string) string {
	return strings.ToLower(strings.TrimSpace(targetType))
}

func IsDomainTargetType(targetType string) bool {
	return NormalizeTargetType(targetType) == TargetTypeDomain
}

func IsURLMatchTarget(urlString string, target TargetRef) bool {
	normalizedTargetName := normalizeTargetName(target.Name)
	if strings.TrimSpace(urlString) == "" || normalizedTargetName == "" {
		return false
	}

	parsed, err := url.Parse(strings.TrimSpace(urlString))
	if err != nil {
		return false
	}

	hostname := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if hostname == "" {
		return false
	}

	switch NormalizeTargetType(target.Type) {
	case TargetTypeDomain:
		return hostname == normalizedTargetName || strings.HasSuffix(hostname, "."+normalizedTargetName)
	case TargetTypeIP:
		return hostname == normalizedTargetName
	case TargetTypeCIDR:
		hostIP := net.ParseIP(hostname)
		if hostIP == nil {
			return false
		}
		_, network, err := net.ParseCIDR(normalizedTargetName)
		if err != nil {
			return false
		}
		return network.Contains(hostIP)
	default:
		return false
	}
}

func IsSubdomainMatchTarget(subdomain string, target TargetRef) bool {
	normalizedSubdomain := normalizeTargetName(subdomain)
	normalizedTargetName := normalizeTargetName(target.Name)
	if normalizedSubdomain == "" || normalizedTargetName == "" {
		return false
	}
	if !IsDomainTargetType(target.Type) {
		return false
	}
	if !govalidator.IsDNSName(normalizedSubdomain) {
		return false
	}
	return normalizedSubdomain == normalizedTargetName || strings.HasSuffix(normalizedSubdomain, "."+normalizedTargetName)
}

func ExtractHostFromURL(urlString string) string {
	parsed, err := url.Parse(strings.TrimSpace(urlString))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Host)
}

func normalizeTargetName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
