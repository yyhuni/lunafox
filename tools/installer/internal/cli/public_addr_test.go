package cli

import "testing"

func TestNormalizePublicURL(t *testing.T) {
	url, port, err := NormalizePublicURL("https://10.8.0.25:18443", DefaultPublicPort)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if url != "https://10.8.0.25:18443" {
		t.Fatalf("unexpected url: %s", url)
	}
	if port != "18443" {
		t.Fatalf("unexpected port: %s", port)
	}
}

func TestNormalizePublicURLAddsDefaultPortByScheme(t *testing.T) {
	url, port, err := NormalizePublicURL("http://10.8.0.25", DefaultPublicPort)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if url != "http://10.8.0.25:80" {
		t.Fatalf("unexpected url: %s", url)
	}
	if port != "80" {
		t.Fatalf("unexpected port: %s", port)
	}
}

func TestNormalizePublicHostPort(t *testing.T) {
	url, port, err := NormalizePublicHostPort("10.8.0.25", "8443")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if url != "https://10.8.0.25:8443" {
		t.Fatalf("unexpected url: %s", url)
	}
	if port != "8443" {
		t.Fatalf("unexpected port: %s", port)
	}
}

func TestParsePublicHostInputRejectsHostWithScheme(t *testing.T) {
	_, err := ParsePublicHostInput("https://127.0.0.1")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParsePublicHostInputRejectsHostWithPort(t *testing.T) {
	_, err := ParsePublicHostInput("127.0.0.1:8443")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParsePublicHostInputAcceptsIPAndLocalhost(t *testing.T) {
	if host, err := ParsePublicHostInput("1.2.3.4"); err != nil || host != "1.2.3.4" {
		t.Fatalf("unexpected result: host=%q err=%v", host, err)
	}
	if host, err := ParsePublicHostInput("LOCALHOST"); err != nil || host != "localhost" {
		t.Fatalf("unexpected localhost normalize: host=%q err=%v", host, err)
	}
}

func TestParsePublicHostInputRejectsDomain(t *testing.T) {
	_, err := ParsePublicHostInput("api.example.com")
	if err == nil {
		t.Fatalf("expected domain to be rejected")
	}
}

func TestNormalizePublicURLRejectsDomainHost(t *testing.T) {
	_, _, err := NormalizePublicURL("https://example.com:443", DefaultPublicPort)
	if err == nil {
		t.Fatalf("expected domain host to be rejected")
	}
}

func TestParsePublicHostInputRejectsIPv6(t *testing.T) {
	for _, raw := range []string{"2001:db8::1", "::1", "[::1]"} {
		if _, err := ParsePublicHostInput(raw); err == nil {
			t.Fatalf("expected ipv6 to be rejected: %s", raw)
		}
	}
}

func TestNormalizePublicURLRejectsIPv6Host(t *testing.T) {
	_, _, err := NormalizePublicURL("https://[2001:db8::1]:443", DefaultPublicPort)
	if err == nil {
		t.Fatalf("expected ipv6 url host to be rejected")
	}
}

func TestIsLoopbackHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{host: "localhost", want: true},
		{host: "127.0.0.1", want: true},
		{host: "::1", want: true},
		{host: "8.8.8.8", want: false},
	}

	for _, tc := range cases {
		if got := IsLoopbackHost(tc.host); got != tc.want {
			t.Fatalf("host=%s got=%v want=%v", tc.host, got, tc.want)
		}
	}
}
