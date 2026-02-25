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
	if !strings.Contains(body, `LOKI_PUSH_URL="https://public.example.com:8083/loki/api/v1/push"`) {
		t.Fatalf("expected LOKI_PUSH_URL to be rendered as full backend-provided URL, body=%s", body)
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
	if !strings.Contains(body, `LOKI_PUSH_URL="https://public.example.com:8083/loki/api/v1/push"`) {
		t.Fatalf("expected LOKI_PUSH_URL to use PUBLIC_URL host and /loki/api/v1/push path, body=%s", body)
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
	if !strings.Contains(body, `LOKI_PUSH_URL="https://public.example.com:8083/loki/api/v1/push"`) {
		t.Fatalf("expected default mode to include full LOKI_PUSH_URL, body=%s", body)
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

func TestInstallScriptUsesPortableSedPatternForAgentID(t *testing.T) {
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
	if strings.Contains(body, `[0-9]\+`) {
		t.Fatalf("agent id parser must avoid GNU sed extension \\\\+, body=%s", body)
	}
	if !strings.Contains(body, `[0-9][0-9]*`) {
		t.Fatalf("expected portable POSIX sed agent id pattern, body=%s", body)
	}
}

func TestInstallScriptSupportsTaggedLokiPluginName(t *testing.T) {
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
	if !strings.Contains(body, `grep -Eq '^loki(:|$)'`) {
		t.Fatalf("expected script to detect tagged loki plugin names, body=%s", body)
	}
	if !strings.Contains(body, `grafana/loki-docker-driver:3.6.7-`) {
		t.Fatalf("expected script to pin loki plugin to 3.6.7 per arch, body=%s", body)
	}
	if strings.Contains(body, `grafana/loki-docker-driver:3.6.6-`) {
		t.Fatalf("expected script to avoid legacy 3.6.6 fallback, body=%s", body)
	}
	if strings.Contains(body, `grafana/loki-docker-driver:latest`) {
		t.Fatalf("expected script to avoid floating latest fallback, body=%s", body)
	}
}

func TestInstallScriptUsesExpectedLokiDriverOptions(t *testing.T) {
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
	if strings.Contains(body, "loki-no-file") {
		t.Fatalf("expected canonical no-file option name, body=%s", body)
	}
	if !strings.Contains(body, `--log-opt "no-file=true"`) {
		t.Fatalf("expected script to enable no-file for loki logging, body=%s", body)
	}
	if !strings.Contains(body, `--log-opt "loki-batch-size=1048576"`) {
		t.Fatalf("expected script to set explicit loki-batch-size, body=%s", body)
	}
}

func TestInstallScriptSetsContainerNameLabelForLokiQueryCompatibility(t *testing.T) {
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
	if !strings.Contains(body, `loki-external-labels=agent_id=$AGENT_ID,container_name=lunafox-agent`) {
		t.Fatalf("expected script to pin container_name label for query compatibility, body=%s", body)
	}
}
