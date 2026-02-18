package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTLSClientForServer(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(server.Certificate())
	return NewClient(ClientOptions{
		TLSConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
		Timeout:   5 * time.Second,
	})
}

func TestIssueRegistrationTokenLoginFailure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/auth/login" {
			writer.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(writer).Encode(map[string]string{"message": "bad creds"})
			return
		}
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTLSClientForServer(t, server)
	token, err := client.IssueRegistrationToken(context.Background(), server.URL, "admin", "admin")
	if token != "" {
		t.Fatalf("expected empty token, got %s", token)
	}
	if err == nil {
		t.Fatalf("expected error")
	}

	stageErr, ok := err.(*StageError)
	if !ok {
		t.Fatalf("expected StageError, got %T", err)
	}
	if stageErr.Stage != "login" {
		t.Fatalf("expected login stage, got %s", stageErr.Stage)
	}
}

func TestIssueRegistrationTokenSuccess(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/auth/login":
			writer.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(writer).Encode(map[string]string{"accessToken": "abc"})
		case "/api/agents/registration-tokens":
			if !strings.HasPrefix(request.Header.Get("Authorization"), "Bearer ") {
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			writer.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(writer).Encode(map[string]string{"token": "reg"})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTLSClientForServer(t, server)
	token, err := client.IssueRegistrationToken(context.Background(), server.URL, "admin", "admin")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if token != "reg" {
		t.Fatalf("unexpected token: %s", token)
	}
}

func TestDownloadInstallScriptUsesModeQuery(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/agent/install-script" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if got := request.URL.Query().Get("token"); got != "reg-token" {
			t.Fatalf("unexpected token query: %s", got)
		}
		if got := request.URL.Query().Get("mode"); got != "local" {
			t.Fatalf("unexpected mode query: %s", got)
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("#!/usr/bin/env bash\necho ok\n"))
	}))
	defer server.Close()

	client := newTLSClientForServer(t, server)
	script, installURL, err := client.DownloadInstallScript(context.Background(), server.URL, "reg-token", "local")
	if err != nil {
		t.Fatalf("DownloadInstallScript error: %v", err)
	}
	if !strings.Contains(script, "echo ok") {
		t.Fatalf("unexpected script body: %s", script)
	}
	if !strings.Contains(installURL, "mode=local") {
		t.Fatalf("install url should include mode=local: %s", installURL)
	}
}

func TestDownloadInstallScriptInvalidModeFallsBackToRemote(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/agent/install-script" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if got := request.URL.Query().Get("mode"); got != "remote" {
			t.Fatalf("unexpected mode query: %s", got)
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("#!/usr/bin/env bash\necho ok\n"))
	}))
	defer server.Close()

	client := newTLSClientForServer(t, server)
	_, _, err := client.DownloadInstallScript(context.Background(), server.URL, "reg-token", "bad-mode")
	if err != nil {
		t.Fatalf("DownloadInstallScript error: %v", err)
	}
}

func TestBuildInstallEnv(t *testing.T) {
	env, err := BuildInstallEnv(Config{Mode: "dev", RegisterURL: "https://a", AgentServerURL: "http://b", NetworkName: "luna", WorkerToken: "w"})
	if err != nil {
		t.Fatalf("BuildInstallEnv error: %v", err)
	}
	flatten := map[string]string{}
	for _, item := range env {
		flatten[item.Key] = item.Value
	}
	if flatten["LUNAFOX_AGENT_DOCKER_NETWORK"] != "luna" {
		t.Fatalf("expected docker network")
	}
	if flatten["WORKER_TOKEN"] != "w" {
		t.Fatalf("expected worker token")
	}
}

func TestBuildInstallEnvProd(t *testing.T) {
	env, err := BuildInstallEnv(Config{Mode: "prod", RegisterURL: "https://a", AgentServerURL: "http://b", NetworkName: "luna"})
	if err != nil {
		t.Fatalf("BuildInstallEnv error: %v", err)
	}
	flatten := map[string]string{}
	for _, item := range env {
		flatten[item.Key] = item.Value
	}
	if _, exists := flatten["WORKER_TOKEN"]; exists {
		t.Fatalf("worker token should be empty when not provided")
	}
}

func TestBuildInstallEnvRequiresURLs(t *testing.T) {
	if _, err := BuildInstallEnv(Config{RegisterURL: "", AgentServerURL: "http://b"}); err == nil {
		t.Fatalf("expected missing register url error")
	}
	if _, err := BuildInstallEnv(Config{RegisterURL: "https://a", AgentServerURL: ""}); err == nil {
		t.Fatalf("expected missing agent server url error")
	}
}
