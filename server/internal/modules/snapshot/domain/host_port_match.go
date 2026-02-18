package domain

import (
	"net"
	"strings"
)

func IsHostPortMatchTarget(host, ip string, target ScanTargetRef) bool {
	normalizedHost := normalizeTargetName(host)
	normalizedIP := strings.TrimSpace(ip)
	normalizedTargetName := normalizeTargetName(target.Name)
	if normalizedHost == "" || normalizedIP == "" || normalizedTargetName == "" {
		return false
	}

	switch NormalizeTargetType(target.Type) {
	case TargetTypeDomain:
		return normalizedHost == normalizedTargetName || strings.HasSuffix(normalizedHost, "."+normalizedTargetName)
	case TargetTypeIP:
		return normalizedIP == normalizedTargetName
	case TargetTypeCIDR:
		ipAddress := net.ParseIP(normalizedIP)
		if ipAddress == nil {
			return false
		}
		_, network, err := net.ParseCIDR(normalizedTargetName)
		if err != nil {
			return false
		}
		return network.Contains(ipAddress)
	default:
		return false
	}
}
