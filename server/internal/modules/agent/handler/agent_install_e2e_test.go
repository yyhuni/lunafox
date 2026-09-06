package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const agentInstallE2EEnvironment = "LUNAFOX_RUN_AGENT_INSTALL_E2E"

// TestAgentInstallScriptRegistersAndConnects runs the released installation
// path against a real Docker daemon. It is opt-in because it builds an Agent
// image, manages the fixed production container/volume names, and installs the
// Loki Docker logging plugin when it is absent.
func TestAgentInstallScriptRegistersAndConnects(t *testing.T) {
	if testing.Short() {
		t.Skip("skip Docker Agent installation end-to-end test in short mode")
	}
	if os.Getenv(agentInstallE2EEnvironment) != "1" {
		t.Skipf("set %s=1 to run Docker Agent installation end-to-end test", agentInstallE2EEnvironment)
	}
	if runtime.GOOS != "linux" {
		t.Skip("the install E2E uses the Linux Docker bridge gateway exposed to launched Agent containers")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	dockerBin := requireDockerForAgentInstallE2E(t, ctx)
	ensureAgentInstallE2ECleanStart(t, ctx, dockerBin)

	imageRef := "lunafox-agent-install-e2e:" + strings.ToLower(fmt.Sprintf("%x-dev", time.Now().UnixNano()))
	defer removeAgentInstallE2EResources(dockerBin, imageRef)
	buildAgentInstallE2EImage(t, ctx, dockerBin, imageRef)

	control := &agentInstallE2EControlServer{
		expectedAuthenticationToken: "agent-install-e2e-auth-token",
		registered:                  make(chan string, 1),
		heartbeats:                  make(chan string, 1),
	}
	grpcServer := grpc.NewServer()
	agentcontrolv1.RegisterControlPlaneServiceServer(grpcServer, control)
	defer grpcServer.Stop()

	gin.SetMode(gin.TestMode)
	agentStore := &handlerAgentStoreStub{}
	tokenStore := &agentInstallE2ETokenStore{}
	facade := agentapp.NewAgentFacade(
		agentapp.NewAgentQueryService(agentStore),
		agentapp.NewAgentCommandService(agentStore),
		agentapp.NewAgentRegistrationService(
			agentStore,
			tokenStore,
			handlerClockStub{now: time.Now().UTC()},
			&handlerTokenGenStub{values: []string{"a1b2c3d4", "agent-install-e2e-auth-token"}},
		),
	)
	handler := NewAgentHandler(
		facade,
		runtimeConfigPublisherStub{},
		"0.0.0-e2e",
		"",
		"http://server:9090",
		imageRef,
		"lunafox_data:/opt/lunafox",
		nil,
	)

	api := gin.New()
	api.POST("/v1/admin/agentRegistrationTokens", handler.CreateRegistrationToken)
	api.GET("/v1/agents:downloadInstallScript", handler.DownloadInstallScript)
	api.POST("/v1/agents:register", handler.Register)
	api.Any("/loki/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	server := startAgentInstallE2EServer(t, api, grpcServer)
	defer server.Close()
	publicURL := fmt.Sprintf("https://%s", agentInstallE2EBridgeURL(t, ctx, dockerBin, server.Listener.Addr().String()))
	handler.publicURL = publicURL

	registrationToken := issueAgentInstallE2ERegistrationToken(t, server)
	installAgentFromGeneratedScript(t, ctx, publicURL, registrationToken)
	if len(agentStore.created) != 1 {
		t.Fatalf("registered agents = %d, want 1", len(agentStore.created))
	}
	if got := agentStore.created[0].AuthenticationToken; got != control.expectedAuthenticationToken {
		t.Fatalf("registered agent authentication token = %q, want %q", got, control.expectedAuthenticationToken)
	}

	waitForAgentInstallE2ESignal(t, ctx, control.registered, "control-plane session registration")
	waitForAgentInstallE2ESignal(t, ctx, control.heartbeats, "first Agent heartbeat")
}

type agentInstallE2ETokenStore struct {
	token *agentdomain.RegistrationToken
}

func (store *agentInstallE2ETokenStore) Create(_ context.Context, token *agentdomain.RegistrationToken) error {
	token.ID = 1
	store.token = token
	return nil
}

func (store *agentInstallE2ETokenStore) FindValid(_ context.Context, token string, now time.Time) (*agentdomain.RegistrationToken, error) {
	if store.token == nil || store.token.Token != token || !store.token.ExpiresAt.After(now) {
		return nil, nil
	}
	return store.token, nil
}

func (store *agentInstallE2ETokenStore) GetResourceByID(_ context.Context, _ int) (*agentdomain.RegistrationTokenResource, error) {
	return nil, nil
}

func (store *agentInstallE2ETokenStore) DeleteNeverAttributedBefore(_ context.Context, now time.Time) error {
	if store.token != nil && !store.token.ExpiresAt.After(now) {
		store.token = nil
	}
	return nil
}

type agentInstallE2EControlServer struct {
	agentcontrolv1.UnimplementedControlPlaneServiceServer

	expectedAuthenticationToken string
	registered                  chan string
	heartbeats                  chan string
}

func (server *agentInstallE2EControlServer) Connect(stream grpc.BidiStreamingServer[agentcontrolv1.ConnectRequest, agentcontrolv1.ConnectResponse]) error {
	values := metadata.ValueFromIncomingContext(stream.Context(), grpcauth.AgentAuthenticationTokenMetadataKey)
	if len(values) != 1 || values[0] != server.expectedAuthenticationToken {
		return status.Error(codes.Unauthenticated, "unexpected Agent authentication token")
	}

	first, err := stream.Recv()
	if err != nil {
		return err
	}
	registration := first.GetRegisterSession()
	if registration == nil || registration.GetAgent() == "" || registration.GetSession() == "" {
		return status.Error(codes.InvalidArgument, "Agent did not register a control-plane session")
	}
	if err := stream.Send(&agentcontrolv1.ConnectResponse{Payload: &agentcontrolv1.ConnectResponse_SessionReady{
		SessionReady: &agentcontrolv1.SessionReady{SessionEpoch: 1},
	}}); err != nil {
		return err
	}
	notifyAgentInstallE2E(server.registered, registration.GetAgent())

	for {
		request, err := stream.Recv()
		if err != nil {
			return err
		}
		if heartbeat := request.GetHeartbeat(); heartbeat != nil {
			notifyAgentInstallE2E(server.heartbeats, heartbeat.GetAgent())
		}
	}
}

func notifyAgentInstallE2E(ch chan<- string, value string) {
	select {
	case ch <- value:
	default:
	}
}

func requireDockerForAgentInstallE2E(t *testing.T, ctx context.Context) string {
	t.Helper()
	dockerBin, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker CLI is required for Agent installation end-to-end test")
	}
	if _, err := runAgentInstallE2EDocker(ctx, dockerBin, "info", "--format", "{{.ServerVersion}}"); err != nil {
		t.Skipf("Docker daemon is not available: %v", err)
	}
	return dockerBin
}

func ensureAgentInstallE2ECleanStart(t *testing.T, ctx context.Context, dockerBin string) {
	t.Helper()
	for _, container := range []string{"lunafox-agent"} {
		if _, err := runAgentInstallE2EDocker(ctx, dockerBin, "container", "inspect", container); err == nil {
			t.Fatalf("refusing to run install E2E while production-named container %q exists", container)
		}
	}
	for _, volume := range []string{"lunafox_data", "lunafox_engine_execution", "lunafox_agent_state"} {
		if _, err := runAgentInstallE2EDocker(ctx, dockerBin, "volume", "inspect", volume); err == nil {
			t.Fatalf("refusing to run install E2E while production-named volume %q exists", volume)
		}
	}
}

func buildAgentInstallE2EImage(t *testing.T, ctx context.Context, dockerBin, imageRef string) {
	t.Helper()
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve package directory: %v", err)
	}
	rootDir := filepath.Clean(filepath.Join(packageDir, "../../../../.."))
	if _, err := runAgentInstallE2EDocker(
		ctx,
		dockerBin,
		"build",
		"--build-context", "contracts="+filepath.Join(rootDir, "contracts"),
		"--file", filepath.Join(rootDir, "agent", "Dockerfile"),
		"--tag", imageRef,
		filepath.Join(rootDir, "agent"),
	); err != nil {
		t.Fatalf("build Agent image: %v", err)
	}
}

func startAgentInstallE2EServer(t *testing.T, api http.Handler, grpcServer *grpc.Server) *httptest.Server {
	t.Helper()
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.ProtoMajor == 2 && strings.HasPrefix(request.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(writer, request)
			return
		}
		api.ServeHTTP(writer, request)
	})
	server := httptest.NewUnstartedServer(handler)
	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("listen for Agent install E2E server: %v", err)
	}
	server.Listener = listener
	server.EnableHTTP2 = true
	server.StartTLS()
	return server
}

func agentInstallE2EBridgeURL(t *testing.T, ctx context.Context, dockerBin, address string) string {
	t.Helper()
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		t.Fatalf("parse E2E listener address %q: %v", address, err)
	}
	gateway, err := runAgentInstallE2EDocker(ctx, dockerBin, "network", "inspect", "bridge", "--format", "{{(index .IPAM.Config 0).Gateway}}")
	if err != nil {
		t.Fatalf("resolve Docker bridge gateway: %v", err)
	}
	gateway = strings.TrimSpace(gateway)
	if net.ParseIP(gateway) == nil {
		t.Fatalf("Docker bridge gateway %q is not an IP address", gateway)
	}
	return net.JoinHostPort(gateway, port)
}

func issueAgentInstallE2ERegistrationToken(t *testing.T, server *httptest.Server) string {
	t.Helper()
	port := server.Listener.Addr().(*net.TCPAddr).Port
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://127.0.0.1:%d/v1/admin/agentRegistrationTokens", port), nil)
	if err != nil {
		t.Fatalf("build registration token request: %v", err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("request registration token: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("registration token status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode registration token response: %v", err)
	}
	if payload.Token == "" {
		t.Fatal("registration token response was empty")
	}
	return payload.Token
}

func installAgentFromGeneratedScript(t *testing.T, ctx context.Context, publicURL, registrationToken string) {
	t.Helper()
	command := `set -euo pipefail; curl -kfsSL "$1/v1/agents:downloadInstallScript?registrationToken=$2&profile=external" | bash`
	process := exec.CommandContext(ctx, "bash", "-c", command, "agent-install-e2e", publicURL, registrationToken)
	output, err := process.CombinedOutput()
	if err != nil {
		t.Fatalf("download and execute Agent install script: %v\n%s", err, output)
	}
}

func waitForAgentInstallE2ESignal(t *testing.T, ctx context.Context, signal <-chan string, name string) {
	t.Helper()
	select {
	case value := <-signal:
		if value == "" {
			t.Fatalf("%s was empty", name)
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for %s: %v", name, ctx.Err())
	}
}

func runAgentInstallE2EDocker(ctx context.Context, dockerBin string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, dockerBin, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("docker %s: %w\n%s", strings.Join(args, " "), err, output)
	}
	return string(output), nil
}

func removeAgentInstallE2EResources(dockerBin, imageRef string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, _ = runAgentInstallE2EDocker(ctx, dockerBin, "container", "rm", "-f", "lunafox-agent")
	for _, volume := range []string{"lunafox_agent_state", "lunafox_engine_execution", "lunafox_data"} {
		_, _ = runAgentInstallE2EDocker(ctx, dockerBin, "volume", "rm", "-f", volume)
	}
	_, _ = runAgentInstallE2EDocker(ctx, dockerBin, "image", "rm", "-f", imageRef)
}
