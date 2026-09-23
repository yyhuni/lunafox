package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComposeEdgeHealthProbeUsesFixedComposeEndpointAndPublicHost(t *testing.T) {
	if composeEdgeProbeURL != "https://nginx" {
		t.Fatalf("Compose edge endpoint = %q, want fixed nginx service", composeEdgeProbeURL)
	}

	var gotHost string
	var gotPath string
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotHost = request.Host
		gotPath = request.URL.Path
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	probe := composeEdgeHealthProbeWithClient(server.URL, "public.example:8443", "/healthChecks/readiness", server.Client())
	if err := probe(context.Background()); err != nil {
		t.Fatalf("Compose edge probe failed: %v", err)
	}
	if gotHost != "public.example:8443" {
		t.Fatalf("edge Host = %q, want public.example:8443", gotHost)
	}
	if gotPath != "/healthChecks/readiness" {
		t.Fatalf("edge path = %q, want readiness path", gotPath)
	}
}

func TestComposeEdgeHealthProbeRejectsUnhealthyResponse(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := composeEdgeHealthProbeWithClient(server.URL, "public.example", "/", server.Client())(context.Background())
	if err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("edge probe error = %v, want HTTP 503", err)
	}
}

func TestComposeEdgePublicHostRejectsNonIngressURLShapes(t *testing.T) {
	for _, rawURL := range []string{
		"",
		"http://public.example",
		"https://public.example/path",
		"https://user@public.example",
		"https://public.example?probe=1",
	} {
		t.Run(rawURL, func(t *testing.T) {
			if _, err := composeEdgePublicHost(rawURL); err == nil {
				t.Fatalf("composeEdgePublicHost(%q) succeeded", rawURL)
			}
		})
	}

	host, err := composeEdgePublicHost("https://public.example:8443")
	if err != nil {
		t.Fatalf("composeEdgePublicHost valid URL: %v", err)
	}
	if host != "public.example:8443" {
		t.Fatalf("public host = %q, want public.example:8443", host)
	}
}

func TestNewComposeEdgeHTTPClientIsDirectAndScopedToInternalTLS(t *testing.T) {
	client := newComposeEdgeHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("Compose edge probe must not use an environment proxy")
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("Compose edge probe must accept the generated internal self-signed certificate")
	}
}
