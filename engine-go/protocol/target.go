package protocol

import (
	"net"
	"strings"

	"golang.org/x/net/idna"
)

// NormalizeDNSName validates and canonicalizes a DNS target/result value.
// It is a protocol value helper, not a result-type registry or authorization
// rule.
func NormalizeDNSName(name string) (string, bool) {
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
		if !validDNSLabel(label) {
			return "", false
		}
	}
	return name, true
}

func validDNSLabel(label string) bool {
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
