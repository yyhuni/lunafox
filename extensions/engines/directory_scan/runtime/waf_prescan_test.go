package directoryscanruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestLooksWAFBlocked(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		header  http.Header
		body    string
		blocked bool
	}{
		{name: "403", status: 403, blocked: true},
		{name: "429", status: 429, blocked: true},
		{name: "cloudflare body", status: 200, body: "<html>Just a moment...</html>", blocked: true},
		{name: "challenge header", status: 200, header: http.Header{"Cf-Mitigated": {"challenge"}}, blocked: true},
		{name: "plain 200", status: 200, body: "ok", blocked: false},
		{name: "generic access denied is not enough", status: 200, body: "access denied", blocked: false},
		{name: "404 is not a block", status: 404, blocked: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			blocked, _ := looksWAFBlocked(test.status, test.header, []byte(test.body))
			if blocked != test.blocked {
				t.Fatalf("looksWAFBlocked() = %t, want %t", blocked, test.blocked)
			}
		})
	}
}

func TestWAFPrescanDecisionTable(t *testing.T) {
	tests := []struct {
		name          string
		probeStatus   int
		probeBody     string
		controlStatus int
		wantBlocked   bool
		wantRequests  int32
	}{
		{name: "blocked with healthy control", probeStatus: 403, probeBody: "blocked", controlStatus: 200, wantBlocked: true, wantRequests: 2},
		{name: "rate limited", probeStatus: 429, controlStatus: 200, wantBlocked: true, wantRequests: 2},
		{name: "challenge signature", probeStatus: 200, probeBody: "<html>Just a moment...</html>", controlStatus: 200, wantBlocked: true, wantRequests: 2},
		{name: "control also blocked", probeStatus: 403, controlStatus: 403, wantBlocked: false, wantRequests: 2},
		{name: "control unhealthy", probeStatus: 403, controlStatus: 500, wantBlocked: false, wantRequests: 2},
		{name: "healthy site", probeStatus: 200, probeBody: "hello", controlStatus: 200, wantBlocked: false, wantRequests: 1},
		{name: "redirect is not a block", probeStatus: 302, controlStatus: 200, wantBlocked: false, wantRequests: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				if strings.HasPrefix(request.URL.Path, wafProbePathPrefix) {
					response.WriteHeader(test.probeStatus)
					if test.probeBody != "" {
						_, _ = response.Write([]byte(test.probeBody))
					}
					return
				}
				response.WriteHeader(test.controlStatus)
			}))
			defer server.Close()

			config := defaultFFUFConfig()
			config.RequestTimeout = 5
			prescan, err := newWAFPrescan(config)
			if err != nil {
				t.Fatal(err)
			}
			blocked, _, err := prescan.Blocked(context.Background(), server.URL)
			if err != nil {
				t.Fatalf("Blocked() error = %v", err)
			}
			if blocked != test.wantBlocked {
				t.Fatalf("Blocked() = %t, want %t", blocked, test.wantBlocked)
			}
			if requests.Load() != test.wantRequests {
				t.Fatalf("probe requests = %d, want %d", requests.Load(), test.wantRequests)
			}
			if _, _, _ = prescan.Blocked(context.Background(), server.URL); requests.Load() != test.wantRequests {
				t.Fatalf("cached second call issued extra requests: %d", requests.Load())
			}
		})
	}
}

func TestWAFPrescanProbeErrorIsBestEffort(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()

	config := defaultFFUFConfig()
	config.RequestTimeout = 2
	prescan, err := newWAFPrescan(config)
	if err != nil {
		t.Fatal(err)
	}
	blocked, _, err := prescan.Blocked(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Blocked() error = %v", err)
	}
	if blocked {
		t.Fatal("closed server must not be treated as WAF-blocked")
	}
}

func TestWAFPrescanRejectsNonHTTPCandidate(t *testing.T) {
	config := defaultFFUFConfig()
	prescan, err := newWAFPrescan(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := prescan.Blocked(context.Background(), "not-a-url"); err == nil {
		t.Fatal("non-HTTP candidate must return an error")
	}
}
