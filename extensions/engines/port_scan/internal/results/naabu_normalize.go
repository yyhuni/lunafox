package results

import (
	"net"
	"strings"

	contractresults "github.com/yyhuni/lunafox/engines/port_scan/contract"
	"golang.org/x/net/idna"
)

// normalizeNaabuHostPort turns a scanner observation into the complete
// canonical fact required before this Engine sends it across the result boundary.
func normalizeNaabuHostPort(raw naabuLine) (contractresults.HostPort, bool) {
	ip, ok := normalizeNaabuIPv4(raw.IP)
	if !ok || raw.Port < 1 || raw.Port > 65535 {
		return contractresults.HostPort{}, false
	}

	host := strings.TrimSpace(raw.Host)
	if host == "" {
		// Naabu may omit host for an IP-identity observation. Only a valid IPv4
		// can supply that missing identity; no unrelated field is guessed.
		host = ip
	} else {
		host, ok = normalizeNaabuHost(host)
		if !ok {
			return contractresults.HostPort{}, false
		}
	}

	return contractresults.HostPort{Host: host, IP: ip, Port: raw.Port}, true
}

func normalizeNaabuIPv4(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	ip := net.ParseIP(raw)
	if ip == nil || strings.Contains(raw, ":") || ip.To4() == nil {
		return "", false
	}
	return ip.To4().String(), true
}

func normalizeNaabuHost(raw string) (string, bool) {
	host := strings.TrimSpace(raw)
	if host == "" || strings.ContainsAny(host, "\r\n") {
		return "", false
	}
	if ip := net.ParseIP(host); ip != nil {
		if strings.Contains(host, ":") || ip.To4() == nil {
			return "", false
		}
		return ip.To4().String(), true
	}
	return normalizeNaabuDNSName(host)
}

func normalizeNaabuDNSName(name string) (string, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimSuffix(name, ".")
	if name == "" || strings.ContainsAny(name, " \t\r\n") {
		return "", false
	}
	ascii, err := idna.Lookup.ToASCII(name)
	if err != nil {
		return "", false
	}
	name = strings.ToLower(ascii)
	if len(name) > 253 || net.ParseIP(name) != nil {
		return "", false
	}
	labels := strings.Split(name, ".")
	if len(labels) < 2 {
		return "", false
	}
	for _, label := range labels {
		if !validNaabuDNSLabel(label) {
			return "", false
		}
	}
	return name, true
}

func validNaabuDNSLabel(label string) bool {
	if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}
