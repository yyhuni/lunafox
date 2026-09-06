package domain

import (
	"net"
	"strings"

	"github.com/asaskevich/govalidator"
	contractresults "github.com/yyhuni/lunafox/contracts/results"
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

func IsURLMatchTarget(urlString string, target ScanTargetRef) bool {
	normalizedTargetName := normalizeTargetName(target.Name)
	if normalizedTargetName == "" {
		return false
	}

	hostname, err := contractresults.DeriveObservedAssetURLHost(urlString)
	if err != nil {
		return false
	}

	switch NormalizeTargetType(target.Type) {
	case TargetTypeDomain:
		return hostname == normalizedTargetName || strings.HasSuffix(hostname, "."+normalizedTargetName)
	case TargetTypeIP:
		return hostname == normalizedTargetName
	case TargetTypeCIDR:
		hostIP := net.ParseIP(hostname)
		if hostIP == nil || hostIP.To4() == nil {
			return false
		}
		_, network, err := net.ParseCIDR(normalizedTargetName)
		if err != nil || network.IP.To4() == nil {
			return false
		}
		return network.Contains(hostIP.To4())
	default:
		return false
	}
}

func IsSubdomainMatchTarget(subdomain string, target ScanTargetRef) bool {
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

func normalizeTargetName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
