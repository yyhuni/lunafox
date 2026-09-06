package geolocation

import "testing"

func TestNormalizePublicIP(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "public IPv4", raw: " 8.8.8.8 ", want: "8.8.8.8", ok: true},
		{name: "mapped public IPv4", raw: "::ffff:8.8.8.8", want: "8.8.8.8", ok: true},
		{name: "public IPv6", raw: "2606:4700:4700::1111", want: "2606:4700:4700::1111", ok: true},
		{name: "malformed", raw: "not-an-ip"},
		{name: "private IPv4", raw: "10.0.0.1"},
		{name: "loopback IPv4", raw: "127.0.0.1"},
		{name: "link-local IPv4", raw: "169.254.1.1"},
		{name: "CGNAT IPv4", raw: "100.64.0.1"},
		{name: "documentation IPv4", raw: "192.0.2.1"},
		{name: "benchmark IPv4", raw: "198.18.0.1"},
		{name: "reserved IPv4", raw: "240.0.0.1"},
		{name: "multicast IPv4", raw: "224.0.0.1"},
		{name: "unspecified IPv4", raw: "0.0.0.0"},
		{name: "private IPv6", raw: "fd00::1"},
		{name: "loopback IPv6", raw: "::1"},
		{name: "link-local IPv6", raw: "fe80::1%en0"},
		{name: "documentation IPv6", raw: "2001:db8::1"},
		{name: "discard-only IPv6", raw: "100::1"},
		{name: "NAT64 IPv6", raw: "64:ff9b::808:808"},
		{name: "deprecated site-local IPv6", raw: "fec0::1"},
		{name: "multicast IPv6", raw: "ff02::1"},
		{name: "unspecified IPv6", raw: "::"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := NormalizePublicIP(test.raw)
			if ok != test.ok || got != test.want {
				t.Fatalf("NormalizePublicIP(%q) = (%q, %t), want (%q, %t)", test.raw, got, ok, test.want, test.ok)
			}
		})
	}
}
