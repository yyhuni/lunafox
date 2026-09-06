package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	agentIngressDockerE2EEnv           = "LUNAFOX_RUN_AGENT_INGRESS_DOCKER_E2E"
	agentIngressDockerE2ENginxImageEnv = "LUNAFOX_AGENT_INGRESS_DOCKER_E2E_NGINX_IMAGE"

	// The development image is built by the normal local install before this
	// opt-in test runs, so the test does not make an unrelated registry pull a
	// prerequisite for verifying DNS recovery.
	agentIngressDockerE2EDevelopmentNginxImage = "yyhuni/lunafox-nginx:0.0.0-dev"
	agentIngressDockerE2EBaseNginxImage        = "nginx:1.28.2-alpine"
)

func TestAgentIngressDockerDNSRecoversAfterServerAddressChange(t *testing.T) {
	if testing.Short() || os.Getenv(agentIngressDockerE2EEnv) != "1" {
		t.Skip("set LUNAFOX_RUN_AGENT_INGRESS_DOCKER_E2E=1 to run the isolated Docker ingress scenario")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	repositoryRoot := bootstrapRepositoryRoot(t)
	fixture := newAgentIngressDockerFixture(t, ctx, repositoryRoot)
	fixture.resolveNginxImage()
	fixture.buildHarnessImage()
	fixture.writeComposeFiles()
	fixture.compose("config", "-q")
	fixture.compose("up", "-d", "server")
	fixture.compose("up", "-d", "nginx")

	initialServerID := fixture.composeServiceID("server")
	initialAddress := fixture.containerAddress(initialServerID)
	fixture.network = fixture.containerNetwork(initialServerID)

	fixture.runClient("e2e-token", "--required-ready", "1", "--timeout", "20s")
	fixture.runClient("wrong-token", "--expect-unauthenticated", "--timeout", "20s")
	fixture.recreateNginx(fixture.composeServiceID("nginx"))
	nginxID := fixture.composeServiceID("nginx")
	fixture.runClient("e2e-token", "--required-ready", "1", "--timeout", "20s")

	fixture.startLongClient()
	fixture.waitForLog(fixture.longClientName, "READY 1", 20*time.Second)

	fixture.compose("rm", "-s", "-f", "server")
	fixture.waitForLog(fixture.longClientName, "DISCONNECTED 1", 20*time.Second)
	changedAddress := fixture.recreateServerWithDifferentAddress(initialAddress)
	if changedAddress == initialAddress {
		t.Fatalf("Server address did not change: %s", changedAddress)
	}
	if currentNginxID := fixture.composeServiceID("nginx"); currentNginxID != nginxID {
		t.Fatalf("Nginx was recreated during Server recovery: before=%s after=%s", nginxID, currentNginxID)
	}

	// The deployed Nginx configuration uses a ten-second DNS validity window.
	time.Sleep(12 * time.Second)
	fixture.runClient("e2e-token", "--required-ready", "1", "--timeout", "20s")
	fixture.waitForLog(fixture.longClientName, "READY 2", 30*time.Second)
	fixture.waitForContainerExit(fixture.longClientName, 20*time.Second)
}

type agentIngressDockerFixture struct {
	t              *testing.T
	ctx            context.Context
	repositoryRoot string
	serverRoot     string
	tempDir        string
	project        string
	image          string
	nginxImage     string
	composeFile    string
	network        string
	longClientName string
	holderNames    []string
}

func (fixture *agentIngressDockerFixture) resolveNginxImage() {
	fixture.t.Helper()
	if configuredImage := strings.TrimSpace(os.Getenv(agentIngressDockerE2ENginxImageEnv)); configuredImage != "" {
		if !fixture.dockerImageExists(configuredImage) {
			fixture.t.Fatalf("configured Nginx image %q is not available locally", configuredImage)
		}
		fixture.nginxImage = configuredImage
		return
	}

	for _, image := range []string{
		agentIngressDockerE2EDevelopmentNginxImage,
		agentIngressDockerE2EBaseNginxImage,
	} {
		if fixture.dockerImageExists(image) {
			fixture.nginxImage = image
			return
		}
	}

	fixture.t.Fatalf(
		"no local Nginx image is available; build %q or set %s to an existing image",
		agentIngressDockerE2EDevelopmentNginxImage,
		agentIngressDockerE2ENginxImageEnv,
	)
}

func newAgentIngressDockerFixture(t *testing.T, ctx context.Context, repositoryRoot string) *agentIngressDockerFixture {
	t.Helper()
	stamp := time.Now().UnixNano()
	fixture := &agentIngressDockerFixture{
		t:              t,
		ctx:            ctx,
		repositoryRoot: repositoryRoot,
		serverRoot:     filepath.Join(repositoryRoot, "server"),
		tempDir:        t.TempDir(),
		project:        fmt.Sprintf("agentingresse2e%d", stamp),
		image:          fmt.Sprintf("lunafox-agent-ingress-e2e:%d", stamp),
		longClientName: fmt.Sprintf("agentingresse2eclient%d", stamp),
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		fixture.runDockerQuietly(cleanupCtx, "rm", "-f", fixture.longClientName)
		for _, holderName := range fixture.holderNames {
			fixture.runDockerQuietly(cleanupCtx, "rm", "-f", holderName)
		}
		fixture.runComposeQuietly(cleanupCtx, "down", "--volumes", "--remove-orphans")
		fixture.runDockerQuietly(cleanupCtx, "image", "rm", "-f", fixture.image)
	})
	return fixture
}

func (fixture *agentIngressDockerFixture) buildHarnessImage() {
	fixture.t.Helper()
	binaryPath := filepath.Join(fixture.tempDir, "agent-ingress-dns-e2e")
	architecture := fixture.dockerArchitecture()
	command := exec.CommandContext(fixture.ctx, "go", "build", "-trimpath", "-o", binaryPath, "./cmd/testsupport/agent-ingress-dns-e2e")
	command.Dir = fixture.serverRoot
	command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+architecture)
	if output, err := command.CombinedOutput(); err != nil {
		fixture.t.Fatalf("build Agent ingress Docker harness: %v\n%s", err, output)
	}
	if err := os.WriteFile(filepath.Join(fixture.tempDir, "Dockerfile"), []byte("FROM scratch\nCOPY agent-ingress-dns-e2e /agent-ingress-dns-e2e\nENTRYPOINT [\"/agent-ingress-dns-e2e\"]\n"), 0o600); err != nil {
		fixture.t.Fatalf("write harness Dockerfile: %v", err)
	}
	fixture.runDocker("build", "--tag", fixture.image, fixture.tempDir)
}

func (fixture *agentIngressDockerFixture) writeComposeFiles() {
	fixture.t.Helper()
	certificatePath, privateKeyPath := writeAgentIngressE2ECertificate(fixture.t, fixture.tempDir)
	nginxPath := filepath.Join(fixture.tempDir, "nginx.conf")
	nginxConfig := `worker_processes 1;
events { worker_connections 128; }
http {
  resolver 127.0.0.11 valid=10s ipv6=off;
  upstream backend_grpc {
    zone backend_grpc 64k;
    server server:9090 resolve;
  }
  server {
    listen 443 ssl;
    http2 on;
    ssl_certificate /etc/nginx/tls/tls.crt;
    ssl_certificate_key /etc/nginx/tls/tls.key;
    location ~ ^/lunafox\.agent\.control\.v1\.ControlPlaneService/ {
      grpc_set_header X-Forwarded-For $remote_addr;
      grpc_pass grpc://backend_grpc;
    }
  }
}
`
	if err := os.WriteFile(nginxPath, []byte(nginxConfig), 0o600); err != nil {
		fixture.t.Fatalf("write Nginx config: %v", err)
	}
	fixture.composeFile = filepath.Join(fixture.tempDir, "compose.yaml")
	compose := fmt.Sprintf(`services:
  server:
    image: %s
    command: ["server", "--listen", ":9090", "--token", "e2e-token"]
    expose:
      - "9090"
  nginx:
    image: %s
    depends_on:
      server:
        condition: service_started
    volumes:
      - %s:/etc/nginx/nginx.conf:ro
      - %s:/etc/nginx/tls/tls.crt:ro
      - %s:/etc/nginx/tls/tls.key:ro
`, fixture.image, fixture.nginxImage, filepath.ToSlash(nginxPath), filepath.ToSlash(certificatePath), filepath.ToSlash(privateKeyPath))
	if err := os.WriteFile(fixture.composeFile, []byte(compose), 0o600); err != nil {
		fixture.t.Fatalf("write Compose fixture: %v", err)
	}
}

func (fixture *agentIngressDockerFixture) recreateNginx(previousID string) {
	fixture.t.Helper()
	fixture.compose("rm", "-s", "-f", "nginx")
	fixture.compose("up", "-d", "--no-deps", "nginx")
	if currentID := fixture.composeServiceID("nginx"); currentID == previousID {
		fixture.t.Fatalf("Nginx was not recreated: %s", currentID)
	}
}

func (fixture *agentIngressDockerFixture) recreateServerWithDifferentAddress(previousAddress string) string {
	fixture.t.Helper()
	for attempt := 1; attempt <= 4; attempt++ {
		holderName := fmt.Sprintf("%sholder%d", fixture.project, attempt)
		fixture.holderNames = append(fixture.holderNames, holderName)
		fixture.runDocker("run", "-d", "--name", holderName, "--network", fixture.network, "--entrypoint", "/bin/sh", fixture.nginxImage, "-c", "sleep 90")
		fixture.compose("up", "-d", "--no-deps", "server")
		serverID := fixture.composeServiceID("server")
		address := fixture.containerAddress(serverID)
		if address != previousAddress {
			return address
		}
		fixture.compose("rm", "-s", "-f", "server")
	}
	fixture.t.Fatalf("could not force a Docker-assigned Server address change from %s", previousAddress)
	return ""
}

func (fixture *agentIngressDockerFixture) startLongClient() {
	fixture.t.Helper()
	fixture.runDocker("run", "-d", "--name", fixture.longClientName, "--network", fixture.network,
		fixture.image, "client", "--address", "nginx:443", "--token", "e2e-token", "--required-ready", "2", "--timeout", "75s")
}

func (fixture *agentIngressDockerFixture) runClient(token string, arguments ...string) {
	fixture.t.Helper()
	args := []string{"run", "--rm", "--network", fixture.network, fixture.image, "client", "--address", "nginx:443", "--token", token}
	args = append(args, arguments...)
	fixture.runDocker(args...)
}

func (fixture *agentIngressDockerFixture) compose(arguments ...string) string {
	fixture.t.Helper()
	args := []string{"compose", "--project-name", fixture.project, "--file", fixture.composeFile}
	args = append(args, arguments...)
	return fixture.runDocker(args...)
}

func (fixture *agentIngressDockerFixture) composeServiceID(service string) string {
	fixture.t.Helper()
	id := strings.TrimSpace(fixture.compose("ps", "-q", service))
	if id == "" {
		fixture.t.Fatalf("Compose service %s has no container", service)
	}
	return id
}

func (fixture *agentIngressDockerFixture) containerAddress(containerID string) string {
	fixture.t.Helper()
	address := strings.TrimSpace(fixture.runDocker("inspect", "--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", containerID))
	if address == "" {
		fixture.t.Fatalf("container %s has no Docker network address", containerID)
	}
	return address
}

func (fixture *agentIngressDockerFixture) containerNetwork(containerID string) string {
	fixture.t.Helper()
	network := strings.TrimSpace(fixture.runDocker("inspect", "--format", "{{range $name, $_ := .NetworkSettings.Networks}}{{$name}}{{end}}", containerID))
	if network == "" {
		fixture.t.Fatalf("container %s has no Docker network", containerID)
	}
	return network
}

func (fixture *agentIngressDockerFixture) dockerArchitecture() string {
	fixture.t.Helper()
	architecture := strings.TrimSpace(fixture.runDocker("info", "--format", "{{.Architecture}}"))
	switch architecture {
	case "amd64", "x86_64":
		return "amd64"
	case "arm64", "aarch64":
		return "arm64"
	default:
		fixture.t.Fatalf("unsupported Docker architecture %q", architecture)
		return ""
	}
}

func (fixture *agentIngressDockerFixture) waitForLog(containerName, want string, timeout time.Duration) {
	fixture.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		logs := fixture.runDocker("logs", containerName)
		if strings.Contains(logs, want) {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	fixture.t.Fatalf("container %s did not emit %q\n%s", containerName, want, fixture.runDocker("logs", containerName))
}

func (fixture *agentIngressDockerFixture) waitForContainerExit(containerName string, timeout time.Duration) {
	fixture.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state := strings.TrimSpace(fixture.runDocker("inspect", "--format", "{{.State.Running}}:{{.State.ExitCode}}", containerName))
		if state == "false:0" {
			return
		}
		if strings.HasPrefix(state, "false:") {
			fixture.t.Fatalf("container %s exited unsuccessfully: %s\n%s", containerName, state, fixture.runDocker("logs", containerName))
		}
		time.Sleep(250 * time.Millisecond)
	}
	fixture.t.Fatalf("container %s did not exit\n%s", containerName, fixture.runDocker("logs", containerName))
}

func (fixture *agentIngressDockerFixture) runDocker(arguments ...string) string {
	fixture.t.Helper()
	command := fixture.dockerCommand(fixture.ctx, arguments...)
	output, err := command.CombinedOutput()
	if err != nil {
		fixture.t.Fatalf("docker %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func (fixture *agentIngressDockerFixture) dockerImageExists(image string) bool {
	fixture.t.Helper()
	return fixture.dockerCommand(fixture.ctx, "image", "inspect", image).Run() == nil
}

func (fixture *agentIngressDockerFixture) dockerCommand(ctx context.Context, arguments ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, "docker", arguments...)
	// Docker Compose can leave a plugin child holding its output pipe after the
	// command context expires. Bound the wait so a failed opt-in E2E run cannot
	// strand the Go test process or its temporary project.
	command.WaitDelay = 5 * time.Second
	return command
}

func (fixture *agentIngressDockerFixture) runDockerQuietly(ctx context.Context, arguments ...string) {
	_ = fixture.dockerCommand(ctx, arguments...).Run()
}

func (fixture *agentIngressDockerFixture) runComposeQuietly(ctx context.Context, arguments ...string) {
	args := []string{"compose", "--project-name", fixture.project, "--file", fixture.composeFile}
	args = append(args, arguments...)
	fixture.runDockerQuietly(ctx, args...)
}

func writeAgentIngressE2ECertificate(t *testing.T, directory string) (string, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate Nginx test key: %v", err)
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		t.Fatalf("generate Nginx test certificate serial: %v", err)
	}
	now := time.Now().UTC()
	certificate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: "nginx"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"nginx"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &certificate, &certificate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create Nginx test certificate: %v", err)
	}
	certificatePath := filepath.Join(directory, "tls.crt")
	privateKeyPath := filepath.Join(directory, "tls.key")
	if err := os.WriteFile(certificatePath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write Nginx test certificate: %v", err)
	}
	if err := os.WriteFile(privateKeyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}), 0o600); err != nil {
		t.Fatalf("write Nginx test key: %v", err)
	}
	return certificatePath, privateKeyPath
}
