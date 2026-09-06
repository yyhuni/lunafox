package handler

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newInstallScriptHandlerForTest(publicURL string) *AgentHandler {
	return &AgentHandler{
		agentVersion:            "v1.2.3",
		publicURL:               publicURL,
		agentControlInternalURL: "http://server:9090",
		agentImageRef:           "docker.io/example/lunafox-agent:v1.2.3",
		sharedDataVolumeBind:    "lunafox_data:/opt/lunafox",
	}
}

func registerInstallScriptRoutes(router *gin.Engine, handler *AgentHandler) {
	router.GET("/v1/agents:downloadInstallScript", handler.DownloadInstallScript)
}

func TestInstallScriptExternalUsesPublicURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=external", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `REGISTER_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected REGISTER_URL to use PUBLIC_URL, body=%s", body)
	}
	if !strings.Contains(body, `AGENT_CONTROL_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected AGENT_CONTROL_URL to use PUBLIC_URL in external profile, body=%s", body)
	}
	if strings.Contains(body, `AGENT_SERVER_URL=`) {
		t.Fatalf("expected AGENT_SERVER_URL removed from script, body=%s", body)
	}
	if strings.Contains(body, `-e SERVER_URL=`) {
		t.Fatalf("expected SERVER_URL container env removed from script, body=%s", body)
	}
	if !strings.Contains(body, `NETWORK_NAME="${LUNAFOX_AGENT_DOCKER_NETWORK:-off}"`) {
		t.Fatalf("expected external profile to default docker network to off, body=%s", body)
	}
	if !strings.Contains(body, `REQUIRE_DOCKER_NETWORK="0"`) {
		t.Fatalf("expected external profile to keep docker network optional, body=%s", body)
	}
	if strings.Contains(body, "using default bridge") {
		t.Fatalf("expected external profile to avoid implicit default bridge fallback, body=%s", body)
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
	for _, removed := range []string{"WORKER_VERSION", "WORKER_IMAGE_REF", "workerVersion"} {
		if strings.Contains(body, removed) {
			t.Fatalf("install script retained removed Worker field %q, body=%s", removed, body)
		}
	}
	if strings.Contains(body, `"displayName":"%s"`) {
		t.Fatalf("expected registration payload template to omit displayName input, body=%s", body)
	}
	if !strings.Contains(body, `"observedHostname":"%s"`) {
		t.Fatalf("expected registration payload template to include observedHostname, body=%s", body)
	}
	if !strings.Contains(body, `AGENT_INSTANCE_ID="$(printf '%s' "$response" | sed -n 's/.*"instanceId"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"`) {
		t.Fatalf("expected script to capture instanceId from registration response, body=%s", body)
	}
	if !strings.Contains(body, `AGENT_DISPLAY_NAME="$(printf '%s' "$response" | sed -n 's/.*"displayName"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"`) {
		t.Fatalf("expected script to capture displayName from registration response, body=%s", body)
	}
	if strings.Contains(body, `DISPLAY_NAME="${LUNAFOX_AGENT_DISPLAY_NAME:-}"`) {
		t.Fatalf("expected script to stop reading display name from install env, body=%s", body)
	}
	if !strings.Contains(body, `-e AGENT_INSTANCE_ID="$AGENT_INSTANCE_ID"`) {
		t.Fatalf("expected script to pass AGENT_INSTANCE_ID into container env, body=%s", body)
	}
	if !strings.Contains(body, `-e AGENT_DISPLAY_NAME="$AGENT_DISPLAY_NAME"`) {
		t.Fatalf("expected script to pass AGENT_DISPLAY_NAME into container env, body=%s", body)
	}
	if !strings.Contains(body, `STATE_VOLUME="lunafox_agent_state"`) {
		t.Fatalf("expected script to define dedicated state volume, body=%s", body)
	}
	if !strings.Contains(body, `-v "${STATE_VOLUME}:/var/lib/lunafox-agent"`) {
		t.Fatalf("expected script to mount state volume at /var/lib/lunafox-agent, body=%s", body)
	}
	for _, required := range []string{
		`ENGINE_EXECUTION_VOLUME="lunafox_engine_execution"`,
		`ENGINE_EXECUTION_ROOT="/var/lib/lunafox/engine-execution"`,
		`-e LUNAFOX_ENGINE_EXECUTION_VOLUME="$ENGINE_EXECUTION_VOLUME"`,
		`-e LUNAFOX_ENGINE_EXECUTION_ROOT="$ENGINE_EXECUTION_ROOT"`,
		`-v "${ENGINE_EXECUTION_VOLUME}:${ENGINE_EXECUTION_ROOT}"`,
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("expected Engine execution named-volume contract %q, body=%s", required, body)
		}
	}
	for _, forbidden := range []string{
		"lunafox_worker_execution",
		"LUNAFOX_WORKER_EXECUTION_SOCKET",
		"worker-execution.sock",
		"Pulling worker image",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected legacy Worker execution bootstrap %q to be removed, body=%s", forbidden, body)
		}
	}
}

func TestInstallScriptPreflightsExactNamedVolumesBeforeRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")
	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=external", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, required := range []string{
		"SHARED_DATA_VOLUME=\"lunafox_data\"",
		"ensure_named_volume_identity \"$SHARED_DATA_VOLUME\"",
		"ensure_named_volume_identity \"$ENGINE_EXECUTION_VOLUME\"",
		"volume inspect --format",
		"run_engine_mount_preflight",
		"engine-mount-preflight",
		"--profile capability",
		"--execution-volume \"$ENGINE_EXECUTION_VOLUME\"",
		"--shared-volume \"$SHARED_DATA_VOLUME\"",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("install script is missing preflight contract %q", required)
		}
	}
	imageIndex := strings.Index(body, "\nensure_images\n")
	volumeIndex := strings.Index(body, "\nensure_named_volumes\n")
	preflightIndex := strings.Index(body, "\nrun_engine_mount_preflight\n")
	registrationIndex := strings.Index(body, "\nregister_agent\n")
	startIndex := strings.Index(body, "\n$DOCKER_CMD run -d --restart")
	if imageIndex < 0 || volumeIndex <= imageIndex || preflightIndex <= volumeIndex ||
		registrationIndex <= preflightIndex || startIndex <= registrationIndex {
		t.Fatalf("unexpected installer order: image=%d volumes=%d preflight=%d registration=%d start=%d", imageIndex, volumeIndex, preflightIndex, registrationIndex, startIndex)
	}
}

func TestRenderedInstallScriptHasValidBashSyntax(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")
	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=external", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	command := exec.Command("bash", "-n")
	command.Stdin = strings.NewReader(recorder.Body.String())
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("rendered install script failed bash -n: %v\n%s", err, output)
	}
}

func TestInstallScriptFailsWhenAgentVersionMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")
	handler.agentVersion = ""

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=external", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Agent version is not configured") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptFailsWhenSharedVolumeIdentityDrifts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")
	handler.sharedDataVolumeBind = "custom_data:/opt/lunafox:rw"
	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=external", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Shared data volume bind is invalid") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptDevSemVerStillForcesLocalImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")
	handler.agentVersion = "0.0.0-dev"
	handler.agentImageRef = "docker.io/example/lunafox-agent:dev"

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `AGENT_VERSION="0.0.0-dev"`) {
		t.Fatalf("expected SemVer dev agent version, body=%s", body)
	}
	if strings.Contains(body, `[ "$AGENT_VERSION" = "dev" ]`) {
		t.Fatalf("expected local dev image detection to avoid bare dev version checks, body=%s", body)
	}
	if !strings.Contains(body, `case "$AGENT_IMAGE_REF" in`) || !strings.Contains(body, `*:dev)`) {
		t.Fatalf("expected local dev image detection to use dev image refs, body=%s", body)
	}
}

func TestInstallScriptInternalUsesInternalRuntimeURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `REGISTER_URL="https://public.example.com:8083"`) {
		t.Fatalf("expected REGISTER_URL to use PUBLIC_URL, body=%s", body)
	}
	if !strings.Contains(body, `AGENT_CONTROL_URL="http://server:9090"`) {
		t.Fatalf("expected AGENT_CONTROL_URL to use internal agent control URL in internal profile, body=%s", body)
	}
	if strings.Contains(body, `AGENT_SERVER_URL=`) {
		t.Fatalf("expected AGENT_SERVER_URL removed from script, body=%s", body)
	}
	if strings.Contains(body, `-e SERVER_URL=`) {
		t.Fatalf("expected SERVER_URL container env removed from script, body=%s", body)
	}
	if !strings.Contains(body, `NETWORK_NAME="${LUNAFOX_AGENT_DOCKER_NETWORK:-lunafox_network}"`) {
		t.Fatalf("expected internal profile to default docker network to lunafox_network, body=%s", body)
	}
	if !strings.Contains(body, `REQUIRE_DOCKER_NETWORK="1"`) {
		t.Fatalf("expected internal profile to require docker network, body=%s", body)
	}
	if !strings.Contains(body, `LOKI_PUSH_URL="https://public.example.com:8083/loki/api/v1/push"`) {
		t.Fatalf("expected LOKI_PUSH_URL to use PUBLIC_URL host and /loki/api/v1/push path, body=%s", body)
	}
	for _, removed := range []string{"WORKER_VERSION", "WORKER_IMAGE_REF", "workerVersion"} {
		if strings.Contains(body, removed) {
			t.Fatalf("internal install script retained removed Worker field %q, body=%s", removed, body)
		}
	}
}

func TestInstallScriptInternalFailsWhenRuntimeInternalURLMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")
	handler.agentControlInternalURL = ""

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Agent control internal URL is not configured") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptModeQueryRejectedOnInternalProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal&mode=internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "mode query parameter is no longer supported") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptModeQueryRejectedOnExternalProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=external&mode=external", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "mode query parameter is no longer supported") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptTokenQueryRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?token=test-token&profile=internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "token query parameter is no longer supported") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptRequiresRegistrationToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?profile=internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Missing registration token") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptRequiresExplicitProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Install script profile is required") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptRejectsUnknownProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=sidecar", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Invalid install script profile") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptRejectsOldProfileNames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	for _, profile := range []string{"local", "remote"} {
		request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile="+profile, nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for profile %s, got %d, body=%s", profile, recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), "Invalid install script profile") {
			t.Fatalf("unexpected response for profile %s: %s", profile, recorder.Body.String())
		}
	}
}

func TestInstallScriptRequiresHTTPSPublicURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("http://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=external", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "PUBLIC_URL must use https scheme") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestInstallScriptInternalProfileAddsNetworkFailFastChecks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "LUNAFOX_AGENT_DOCKER_NETWORK is required for internal profile.") {
		t.Fatalf("expected internal profile to fail fast on missing docker network env, body=%s", body)
	}
	if !strings.Contains(body, "Create it or set LUNAFOX_AGENT_DOCKER_NETWORK to an existing network") {
		t.Fatalf("expected internal profile to emit actionable docker network message, body=%s", body)
	}
}

func TestInstallScriptUsesPortableSedPatternForAgentID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal", nil)
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

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal", nil)
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

func TestInstallScriptUsesDockerPluginCommandsForLokiDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `$DOCKER_CMD plugin ls --format '{{.Name}} {{.Enabled}}'`) {
		t.Fatalf("expected script to inspect Loki with docker plugin ls, body=%s", body)
	}
	if !strings.Contains(body, `$DOCKER_CMD plugin install "$ref" --alias loki --grant-all-permissions`) {
		t.Fatalf("expected script to install Loki with docker plugin install, body=%s", body)
	}
	if strings.Contains(body, `$DOCKER_CMD engine `) {
		t.Fatalf("expected script to avoid obsolete docker engine subcommands, body=%s", body)
	}
}

func TestInstallScriptUsesExpectedLokiDriverOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	if strings.Contains(body, "loki-no-file") {
		t.Fatalf("expected canonical no-file option name, body=%s", body)
	}
	if !strings.Contains(body, `--log-opt "no-file=false"`) {
		t.Fatalf("expected script to keep loki no-file disabled, body=%s", body)
	}
	if !strings.Contains(body, `--log-opt "loki-batch-size=1048576"`) {
		t.Fatalf("expected script to set explicit loki-batch-size, body=%s", body)
	}
}

func TestInstallScriptFailsFastWhenLokiPluginUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	if strings.Contains(body, "agent container will start without Loki log driver") {
		t.Fatalf("expected script to avoid Loki silent downgrade path, body=%s", body)
	}
	if strings.Contains(body, "USE_LOKI_DRIVER=0") {
		t.Fatalf("expected script to avoid disabling Loki driver after plugin install failure, body=%s", body)
	}
	if !strings.Contains(body, "Failed to install Loki Docker plugin, exiting.") {
		t.Fatalf("expected script to fail fast on Loki plugin install failure, body=%s", body)
	}
}

func TestInstallScriptSetsContainerNameLabelForLokiQueryCompatibility(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newInstallScriptHandlerForTest("https://public.example.com:8083")

	router := gin.New()
	registerInstallScriptRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/v1/agents:downloadInstallScript?registrationToken=test-token&profile=internal", nil)
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
