package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newInstallScriptHandlerForTest(publicURL string) *AgentHandler {
	return &AgentHandler{
		serverVersion:        "v1.2.3",
		publicURL:            publicURL,
		agentInternalURL:     "http://server:8080",
		agentImageRef:        "docker.io/example/lunafox-agent:v1.2.3",
		workerImageRef:       "docker.io/example/lunafox-worker:v1.2.3",
		sharedDataVolumeBind: "lunafox_data:/opt/lunafox",
		workerToken:          "worker-token",
	}
}

func TestInstallScriptRemoteUsesPublicURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	router.GET("/api/agent/install-script", handler.InstallScript)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script?token=test-token&mode=remote", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `REGISTER_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected REGISTER_URL to use PUBLIC_URL, body=%s", body)
	}
	if !strings.Contains(body, `AGENT_SERVER_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected AGENT_SERVER_URL to use PUBLIC_URL in remote mode, body=%s", body)
	}
}

func TestInstallScriptLocalUsesInternalServerURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	router.GET("/api/agent/install-script", handler.InstallScript)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script?token=test-token&mode=local", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `REGISTER_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected REGISTER_URL to use PUBLIC_URL, body=%s", body)
	}
	if !strings.Contains(body, `AGENT_SERVER_URL="http://server:8080"`) {
		t.Fatalf("expected AGENT_SERVER_URL to use internal URL in local mode, body=%s", body)
	}
}

func TestInstallScriptDefaultModeUsesRemote(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	router.GET("/api/agent/install-script", handler.InstallScript)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script?token=test-token", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `AGENT_SERVER_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected default mode to use remote AGENT_SERVER_URL, body=%s", body)
	}
}

func TestInstallScriptInvalidModeFallsBackToRemote(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	router.GET("/api/agent/install-script", handler.InstallScript)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script?token=test-token&mode=invalid", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `AGENT_SERVER_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected invalid mode to fall back to remote AGENT_SERVER_URL, body=%s", body)
	}
}

func TestInstallScriptRequiresHTTPSPublicURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("http://public.example.com:8083")

	router := gin.New()
	router.GET("/api/agent/install-script", handler.InstallScript)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script?token=test-token&mode=remote", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "PUBLIC_URL must use https scheme") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}
