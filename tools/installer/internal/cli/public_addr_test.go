package cli

import "testing"

func TestNormalizePublicURL(t *testing.T) {
	url, port, err := NormalizePublicURL("https://example.com:18443", DefaultPublicPort)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if url != "https://example.com:18443" {
		t.Fatalf("unexpected url: %s", url)
	}
	if port != "18443" {
		t.Fatalf("unexpected port: %s", port)
	}
}

func TestNormalizePublicURLAddsDefaultPortByScheme(t *testing.T) {
	url, port, err := NormalizePublicURL("http://example.com", DefaultPublicPort)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if url != "http://example.com:80" {
		t.Fatalf("unexpected url: %s", url)
	}
	if port != "80" {
		t.Fatalf("unexpected port: %s", port)
	}
}

func TestNormalizePublicHostPort(t *testing.T) {
	url, port, err := NormalizePublicHostPort("2001:db8::1", "8443")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if url != "https://[2001:db8::1]:8443" {
		t.Fatalf("unexpected url: %s", url)
	}
	if port != "8443" {
		t.Fatalf("unexpected port: %s", port)
	}
}

func TestParsePublicHostInputRejectsHostWithScheme(t *testing.T) {
	_, err := ParsePublicHostInput("https://example.com")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParsePublicHostInputRejectsHostWithPort(t *testing.T) {
	_, err := ParsePublicHostInput("example.com:8443")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParsePublicHostInputAcceptsDomainAndIPv4(t *testing.T) {
	if host, err := ParsePublicHostInput("api.example.com"); err != nil || host != "api.example.com" {
		t.Fatalf("unexpected result: host=%q err=%v", host, err)
	}
	if host, err := ParsePublicHostInput("1.2.3.4"); err != nil || host != "1.2.3.4" {
		t.Fatalf("unexpected result: host=%q err=%v", host, err)
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
		{host: "example.com", want: false},
	}

	for _, tc := range cases {
		if got := IsLoopbackHost(tc.host); got != tc.want {
			t.Fatalf("host=%s got=%v want=%v", tc.host, got, tc.want)
		}
	}
}
