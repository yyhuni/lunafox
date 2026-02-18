package steps

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/ui"
)

func TestParseBuildxDriver(t *testing.T) {
	driver := parseBuildxDriver("Name: default\nDriver: docker-container\n")
	if driver != "docker-container" {
		t.Fatalf("unexpected driver: %s", driver)
	}
}

func TestBuildBakeContentIncludesCaches(t *testing.T) {
	content := buildBakeContent("/root/lunafox", "on", "https://proxy.golang.org,direct", true, "/tmp/a", "/tmp/b", "docker.io", "yyhuni")
	if !strings.Contains(content, "cache-from") {
		t.Fatalf("expected cache-from in bake content")
	}
}

func TestResolveVersionProdRequiresVersion(t *testing.T) {
	dir := t.TempDir()
	installer := NewInstaller(cli.Options{
		Mode:           cli.ModeProd,
		Version:        "",
		DockerDir:      filepath.Join(dir, "docker"),
		ComposeFile:    filepath.Join(dir, "docker", "docker-compose.yml"),
		ImageRegistry:  "docker.io",
		ImageNamespace: "yyhuni",
	}, nil, ui.NewPrinter(io.Discard, io.Discard))

	err := installer.resolveVersion()
	if err == nil {
		t.Fatalf("expected resolveVersion to fail without explicit version")
	}
}

func TestResolveVersionProdUsesExplicitVersion(t *testing.T) {
	installer := NewInstaller(cli.Options{
		Mode:           cli.ModeProd,
		Version:        "v1.2.3",
		ImageRegistry:  "docker.io",
		ImageNamespace: "yyhuni",
	}, nil, ui.NewPrinter(io.Discard, io.Discard))

	if err := installer.resolveVersion(); err != nil {
		t.Fatalf("resolveVersion failed: %v", err)
	}
	if installer.version != "v1.2.3" {
		t.Fatalf("unexpected version: %s", installer.version)
	}
}

func TestCheckURLReadyAndWarm(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/ok":
			writer.WriteHeader(http.StatusOK)
		case "/not-found":
			writer.WriteHeader(http.StatusNotFound)
		default:
			writer.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	pool := x509.NewCertPool()
	pool.AddCert(server.Certificate())
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    pool,
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	if !checkURLReady(client, server.URL+"/ok") {
		t.Fatalf("expected /ok to be ready")
	}
	if checkURLReady(client, server.URL+"/not-found") {
		t.Fatalf("expected /not-found not to be ready")
	}
	if !checkURLWarm(client, server.URL+"/not-found") {
		t.Fatalf("expected /not-found to be warm")
	}
}

func TestResolvePublicPort(t *testing.T) {
	got, err := resolvePublicPort("18083")
	if err != nil {
		t.Fatalf("resolve port failed: %v", err)
	}
	if got != "18083" {
		t.Fatalf("unexpected preferred port: %s", got)
	}

	if _, err := resolvePublicPort(""); err == nil {
		t.Fatalf("expected empty port to fail")
	}
	if _, err := resolvePublicPort("abc"); err == nil {
		t.Fatalf("expected non-numeric port to fail")
	}
	if _, err := resolvePublicPort("70000"); err == nil {
		t.Fatalf("expected out-of-range port to fail")
	}
}

func TestParseHostPortForProbe(t *testing.T) {
	host, port, err := parseHostPortForProbe("https://example.com:8443")
	if err != nil {
		t.Fatalf("parse host port: %v", err)
	}
	if host != "example.com" || port != "8443" {
		t.Fatalf("unexpected parse result: host=%s port=%s", host, port)
	}

	host, port, err = parseHostPortForProbe("https://example.com")
	if err != nil {
		t.Fatalf("parse host port: %v", err)
	}
	if host != "example.com" || port != "443" {
		t.Fatalf("unexpected parse result: host=%s port=%s", host, port)
	}
}

func TestParseHostPortForProbeRejectInvalidURL(t *testing.T) {
	_, _, err := parseHostPortForProbe("://bad")
	if err == nil {
		t.Fatalf("expected parse failure")
	}
}
