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
		runtimeInternalURL:   "http://server:9090",
		agentImageRef:        "docker.io/example/lunafox-agent:v1.2.3",
		workerImageRef:       "docker.io/example/lunafox-worker:v1.2.3",
		sharedDataVolumeBind: "lunafox_data:/opt/lunafox",
	}
}

func registerInstallScriptRoutes(router *gin.Engine, handler *AgentHandler) {
	router.GET("/api/agent/install-script", handler.InstallScript)
	router.GET("/api/agent/install-script/local", handler.InstallScriptLocal)
	router.GET("/api/agent/install-script/remote", handler.InstallScriptRemote)
}

func TestInstallScriptRemoteUsesPublicURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script/remote?token=test-token", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `REGISTER_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected REGISTER_URL to use PUBLIC_URL, body=%s", body)
	}
	if !strings.Contains(body, `RUNTIME_GRPC_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected RUNTIME_GRPC_URL to use PUBLIC_URL in remote mode, body=%s", body)
	}
	if strings.Contains(body, `AGENT_SERVER_URL=`) {
		t.Fatalf("expected AGENT_SERVER_URL removed from script, body=%s", body)
	}
	if strings.Contains(body, `-e SERVER_URL=`) {
		t.Fatalf("expected SERVER_URL container env removed from script, body=%s", body)
	}
	if !strings.Contains(body, `NETWORK_NAME="${LUNAFOX_AGENT_DOCKER_NETWORK:-off}"`) {
		t.Fatalf("expected remote profile to default docker network to off, body=%s", body)
	}
	if !strings.Contains(body, `REQUIRE_DOCKER_NETWORK="0"`) {
		t.Fatalf("expected remote profile to keep docker network optional, body=%s", body)
	}
	if strings.Contains(body, "using default bridge") {
		t.Fatalf("expected remote profile to avoid implicit default bridge fallback, body=%s", body)
	}
	if !strings.Contains(body, "resolve_network_args() {") {
		t.Fatalf("expected resolve_network_args helper function in script, body=%s", body)
	}
	if !strings.Contains(body, "register_agent() {") {
		t.Fatalf("expected register_agent helper function in script, body=%s", body)
	}
	if !strings.Contains(body, "ensure_images() {") {
		t.Fatalf("expected ensure_images helper function in script, body=%s", body)
	}
	if !strings.Contains(body, "resolve_logging_args() {") {
		t.Fatalf("expected resolve_logging_args helper function in script, body=%s", body)
	}
	if !strings.Contains(body, "validate_inputs() {") {
		t.Fatalf("expected validate_inputs helper function in script, body=%s", body)
	}
	if !strings.Contains(body, `LOKI_PUSH_URL="https://public.example.com:8083/loki/api/v1/push"`) {
		t.Fatalf("expected LOKI_PUSH_URL to be rendered as full backend-provided URL, body=%s", body)
	}
}

func TestInstallScriptLocalUsesInternalRuntimeURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script/local?token=test-token", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `REGISTER_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected REGISTER_URL to use PUBLIC_URL, body=%s", body)
	}
	if !strings.Contains(body, `RUNTIME_GRPC_URL="http://server:9090"`) {
		t.Fatalf("expected RUNTIME_GRPC_URL to use internal runtime URL in local mode, body=%s", body)
	}
	if strings.Contains(body, `AGENT_SERVER_URL=`) {
		t.Fatalf("expected AGENT_SERVER_URL removed from script, body=%s", body)
	}
	if strings.Contains(body, `-e SERVER_URL=`) {
		t.Fatalf("expected SERVER_URL container env removed from script, body=%s", body)
	}
	if !strings.Contains(body, `NETWORK_NAME="${LUNAFOX_AGENT_DOCKER_NETWORK:-lunafox_network}"`) {
		t.Fatalf("expected local profile to default docker network to lunafox_network, body=%s", body)
	}
	if !strings.Contains(body, `REQUIRE_DOCKER_NETWORK="1"`) {
		t.Fatalf("expected local profile to require docker network, body=%s", body)
	}
	if !strings.Contains(body, `LOKI_PUSH_URL="https://public.example.com:8083/loki/api/v1/push"`) {
		t.Fatalf("expected LOKI_PUSH_URL to use PUBLIC_URL host and /loki/api/v1/push path, body=%s", body)
	}
}

func TestInstallScriptModeQueryRejectedOnLocalProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script/local?token=test-token&mode=local", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "mode query parameter is no longer supported") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptModeQueryRejectedOnRemoteProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script/remote?token=test-token&mode=remote", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "mode query parameter is no longer supported") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptLegacyEndpointRequiresExplicitProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script?token=test-token", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Install script profile is required") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptRequiresHTTPSPublicURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("http://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script/remote?token=test-token", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "PUBLIC_URL must use https scheme") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptLocalProfileAddsNetworkFailFastChecks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script/local?token=test-token", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "LUNAFOX_AGENT_DOCKER_NETWORK is required for local profile.") {
		t.Fatalf("expected local profile to fail fast on missing docker network env, body=%s", body)
	}
	if !strings.Contains(body, "Create it or set LUNAFOX_AGENT_DOCKER_NETWORK to an existing network") {
		t.Fatalf("expected local profile to emit actionable docker network message, body=%s", body)
	}
}

func TestInstallScriptUsesPortableSedPatternForAgentID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script/local?token=test-token", nil)
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
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script/local?token=test-token", nil)
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
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script/local?token=test-token", nil)
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
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/api/agent/install-script/local?token=test-token", nil)
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
