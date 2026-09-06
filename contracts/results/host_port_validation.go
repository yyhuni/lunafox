package results

import (
	"net"
	"strings"
)

// normalizeHostPortHost returns a temporary comparison value for Server URL
// authority checks. Callers must never write it back into an Engine result.
func normalizeHostPortHost(host string) (string, bool) {
	host = strings.TrimSpace(host)
	if host == "" || strings.ContainsAny(host, "\r\n") {
		return "", false
	}
	if ip := net.ParseIP(host); ip != nil {
		if strings.Contains(host, ":") || ip.To4() == nil {
			return "", false
		}
		return ip.To4().String(), true
	}
	return NormalizeSubdomainDNSName(host)
}
