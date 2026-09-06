package conformance

import (
	"context"
	"encoding/base64"
	"errors"
	"go/parser"
	"go/token"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

const rawBindingsImport = "github.com/yyhuni/lunafox/engine-go/protocol"

func TestProtocolDirectEngineConformance(t *testing.T) {
	binary := buildProtocolDirectFixture(t)
	token := canonicalFixtureToken(t)

	cases := []struct {
		name         string
		subdomains   bool
		hostPorts    bool
		disableDNS   bool
		wantProgress string
	}{
		{name: "target only", wantProgress: "protocol-direct-ready inputs=none"},
		{name: "subdomains", subdomains: true, wantProgress: "protocol-direct-ready inputs=none"},
		{name: "hostPorts", hostPorts: true, wantProgress: "protocol-direct-ready inputs=none"},
		{name: "both inputs", subdomains: true, hostPorts: true, wantProgress: "protocol-direct-ready inputs=none"},
		{name: "disabled config section", disableDNS: true, wantProgress: "protocol-direct-ready inputs=none"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newFixture(t, tc.subdomains, tc.hostPorts)
			if tc.disableDNS {
				fixture.context.Config.Sections[0].Enabled = proto.Bool(false)
				fixture.context.Config.Sections[0].Params = nil
				fixture.context.ConfigResources = nil
			}
			writeFixtureContext(t, fixture.root, fixture.context)
			writeFixtureCredential(t, fixture.root, []byte(token))
			endpoint, recorder := startReportingServer(t, fixture.root, token)

			output, err := runFixture(t, binary, fixture.root, nil)
			if err != nil {
				t.Fatalf("protocol-direct fixture failed: %v\n%s", err, output)
			}
			progress, results, authorizations := recorder.snapshot()
			if len(progress) != 1 || progress[0].GetMessage() != tc.wantProgress {
				t.Fatalf("progress requests = %#v, want one message %q", progress, tc.wantProgress)
			}
			if len(results) != 1 {
				t.Fatalf("result requests = %d, want 1", len(results))
			}
			if got, want := results[0].GetResultType(), "asset.subdomain.v1"; got != want {
				t.Fatalf("result type = %q, want %q", got, want)
			}
			if got, want := results[0].GetItems(), [][]byte{
				[]byte(`{"dnsName":"api.example.com"}`),
				[]byte(`{"dnsName":"www.example.com"}`),
			}; !equalByteSlices(got, want) {
				t.Fatalf("result items = %q, want %q", got, want)
			}
			if len(authorizations) != 2 {
				t.Fatalf("authorization observations = %d, want 2", len(authorizations))
			}
			for _, values := range authorizations {
				if len(values) != 1 || values[0] != "Bearer "+token {
					t.Fatal("authorization metadata is not exactly one canonical Bearer value")
				}
			}
			if endpoint != fixturePath(fixture.root, protocol.ExecutionEndpointPath) {
				t.Fatalf("reporting endpoint = %q, want fixed ABI suffix", endpoint)
			}
		})
	}
}

func TestProtocolDirectFixtureRejectsProtoJSONAndUnknownBinaryFields(t *testing.T) {
	binary := buildProtocolDirectFixture(t)
	token := canonicalFixtureToken(t)
	fixture := newFixture(t, false, false)
	writeFixtureCredential(t, fixture.root, []byte(token))

	for _, tc := range []struct {
		name    string
		payload func(*protocol.EngineExecutionContext) []byte
	}{
		{
			name: "ProtoJSON",
			payload: func(*protocol.EngineExecutionContext) []byte {
				return []byte(`{"target":{"type":"domain","value":"example.com"}}`)
			},
		},
		{
			name: "unknown binary field",
			payload: func(context *protocol.EngineExecutionContext) []byte {
				payload, err := proto.Marshal(context)
				if err != nil {
					t.Fatalf("marshal fixture Context: %v", err)
				}
				return protowire.AppendTag(payload, 99, protowire.VarintType)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeFixtureBytes(t, fixturePath(fixture.root, protocol.ContextFilePath), tc.payload(fixture.context), 0o400)
			output, runErr := runFixture(t, binary, fixture.root, nil)
			if runErr == nil {
				t.Fatalf("fixture accepted %s: %s", tc.name, output)
			}
			if strings.Contains(output, token) {
				t.Fatal("fixture failure output exposed the session token")
			}
		})
	}
}

func TestProtocolDirectFixtureRejectsClosedContextShapeViolations(t *testing.T) {
	binary := buildProtocolDirectFixture(t)
	token := canonicalFixtureToken(t)
	for _, tc := range []struct {
		name   string
		mutate func(*protocol.EngineExecutionContext)
	}{
		{name: "missing Target", mutate: func(c *protocol.EngineExecutionContext) { c.Target = nil }},
		{name: "missing Target type", mutate: func(c *protocol.EngineExecutionContext) { c.Target.Type = "" }},
		{name: "missing Target value", mutate: func(c *protocol.EngineExecutionContext) { c.Target.Value = "" }},
		{name: "disabled section with scalar", mutate: func(c *protocol.EngineExecutionContext) {
			c.Config.Sections[0].Enabled = proto.Bool(false)
		}},
		{name: "disabled section with config resource", mutate: func(c *protocol.EngineExecutionContext) {
			c.Config.Sections[0].Enabled = proto.Bool(false)
			c.Config.Sections[0].Params = nil
		}},
		{name: "missing config resource file", mutate: func(c *protocol.EngineExecutionContext) {
			c.ConfigResources[0].Path = filepath.Join(t.TempDir(), "missing-wordlist")
		}},
		{name: "missing platform resource file", mutate: func(c *protocol.EngineExecutionContext) {
			c.PlatformResources[0].Path = filepath.Join(t.TempDir(), "missing-provider")
		}},
		{name: "non-positive limits", mutate: func(c *protocol.EngineExecutionContext) {
			c.Limits.ResultBatchMaxBytes = 0
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newFixture(t, true, false)
			tc.mutate(fixture.context)
			writeFixtureContext(t, fixture.root, fixture.context)
			writeFixtureCredential(t, fixture.root, []byte(token))
			output, runErr := runFixture(t, binary, fixture.root, nil)
			if runErr == nil {
				t.Fatalf("fixture accepted invalid Context: %s", output)
			}
			if strings.Contains(output, token) {
				t.Fatal("fixture failure output exposed the session token")
			}
		})
	}
}

func TestProtocolDirectFixtureRejectsInvalidCredentialsAndSocket(t *testing.T) {
	binary := buildProtocolDirectFixture(t)
	token := canonicalFixtureToken(t)
	for _, tc := range []struct {
		name       string
		credential []byte
	}{
		{name: "trailing newline", credential: []byte(token + "\n")},
		{name: "padding", credential: []byte(token + "=")},
		{name: "31 decoded bytes", credential: []byte(base64.RawURLEncoding.EncodeToString(make([]byte, 31)))},
		{name: "invalid alphabet", credential: []byte(strings.Repeat("!", 43))},
	} {
		t.Run("credential "+tc.name+" is rejected", func(t *testing.T) {
			fixture := newFixture(t, false, false)
			writeFixtureContext(t, fixture.root, fixture.context)
			writeFixtureCredential(t, fixture.root, tc.credential)
			output, runErr := runFixture(t, binary, fixture.root, nil)
			if runErr == nil {
				t.Fatalf("fixture accepted invalid credential: %s", output)
			}
			if strings.Contains(output, token) || strings.Contains(output, string(tc.credential)) {
				t.Fatal("fixture failure output exposed credential material")
			}
		})
	}

	t.Run("missing fixed ABI socket is rejected", func(t *testing.T) {
		fixture := newFixture(t, false, false)
		writeFixtureContext(t, fixture.root, fixture.context)
		writeFixtureCredential(t, fixture.root, []byte(token))
		output, runErr := runFixture(t, binary, fixture.root, nil)
		if runErr == nil {
			t.Fatalf("fixture accepted missing socket: %s", output)
		}
	})
}

func TestProtocolDirectFixtureIgnoresRetiredBootstrapEnvironment(t *testing.T) {
	binary := buildProtocolDirectFixture(t)
	token := canonicalFixtureToken(t)
	fixture := newFixture(t, false, false)
	writeFixtureContext(t, fixture.root, fixture.context)
	writeFixtureCredential(t, fixture.root, []byte(token))
	_, recorder := startReportingServer(t, fixture.root, token)
	overrides := map[string]string{
		"LUNAFOX_EXECUTION_CONTEXT_FILE":           "/invalid/context.json",
		"LUNAFOX_ENGINE_EXECUTION_ENDPOINT":        "/invalid/engine.sock",
		"LUNAFOX_ENGINE_EXECUTION_CREDENTIAL_FILE": "/invalid/token",
	}
	output, err := runFixture(t, binary, fixture.root, overrides)
	if err != nil {
		t.Fatalf("fixture honored a retired path override: %v\n%s", err, output)
	}
	if progress, _, _ := recorder.snapshot(); len(progress) != 1 {
		t.Fatalf("progress calls = %d, want 1", len(progress))
	}
}

func TestProtocolDirectFixtureImportsOnlyRawGeneratedBindings(t *testing.T) {
	path := filepath.Join("testdata", "protocol_direct_engine", "main.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse protocol-direct fixture: %v", err)
	}
	var foundRawBindings bool
	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			t.Fatalf("decode fixture import %s: %v", importSpec.Path.Value, err)
		}
		if importPath == rawBindingsImport {
			foundRawBindings = true
			continue
		}
		if strings.HasPrefix(importPath, "github.com/yyhuni/lunafox/") {
			t.Fatalf("protocol-direct fixture imports non-binding LunaFox package %q", importPath)
		}
	}
	if !foundRawBindings {
		t.Fatalf("protocol-direct fixture must import raw generated bindings %q", rawBindingsImport)
	}
}

type fixtureLayout struct {
	root    string
	context *protocol.EngineExecutionContext
}

func newFixture(t *testing.T, _, _ bool) fixtureLayout {
	t.Helper()
	// AF_UNIX paths are capped well below ordinary temporary-directory limits on
	// macOS. Keep the test-only ABI projection short while preserving its fixed
	// suffixes under /run/lunafox.
	root, err := os.MkdirTemp("/tmp", "lfpd-")
	if err != nil {
		t.Fatalf("create short fixture root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	wordlistPath := fixturePath(root, protocol.ConfigResourcesDirectoryPath+"/dns/resolvers.txt")
	providerPath := fixturePath(root, protocol.PlatformResourcesDirectoryPath+"/subfinderProviderConfig/config.yaml")
	writeFixtureBytes(t, wordlistPath, []byte("1.1.1.1\n"), 0o400)
	writeFixtureBytes(t, providerPath, []byte("github: []\n"), 0o400)
	return fixtureLayout{
		root: root,
		context: &protocol.EngineExecutionContext{
			CompatibilityRevision: protocol.EngineExecutionDiagnosticsCompatibilityRevision,
			Target:                &protocol.CanonicalTarget{Type: "domain", Value: "example.com"},
			Config: &protocol.EngineExecutionConfig{Sections: []*protocol.ConfigSection{{
				SectionId: "dns",
				Enabled:   proto.Bool(true),
				Params: []*protocol.ConfigValue{{
					ParamKey: "timeout",
					Value:    &protocol.ConfigValue_IntegerValue{IntegerValue: 10},
				}},
			}}},
			ConfigResources: []*protocol.ConfigResource{{
				SectionId: "dns", ParamKey: "resolvers", ContentType: "application/vnd.lunafox.wordlist.v1", Path: wordlistPath,
			}},
			PlatformResources: []*protocol.PlatformResource{{
				ResourceId: "subfinderProviderConfig", ContentType: "application/vnd.lunafox.subfinder-provider-config.v1+yaml", Path: providerPath,
			}},
			Limits: &protocol.ExecutionLimits{
				ProgressMessageMaxBytes: 4096,
				ResultBatchMaxItems:     100,
				ResultBatchMaxBytes:     64 * 1024,
			},
		},
	}
}

func fixturePath(root, canonical string) string {
	return filepath.Join(root, strings.TrimPrefix(canonical, "/"))
}

func buildProtocolDirectFixture(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "protocol-direct-engine")
	cmd := exec.Command("go", "build", "-o", binary, "./testdata/protocol_direct_engine")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build protocol-direct fixture: %v\n%s", err, output)
	}
	return binary
}

func canonicalFixtureToken(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i + 1)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	if len(token) != 43 {
		t.Fatalf("fixture token length = %d, want 43", len(token))
	}
	return token
}

func writeFixtureContext(t *testing.T, root string, context *protocol.EngineExecutionContext) {
	t.Helper()
	payload, err := proto.Marshal(context)
	if err != nil {
		t.Fatalf("marshal fixture Context: %v", err)
	}
	writeFixtureBytes(t, fixturePath(root, protocol.ContextFilePath), payload, 0o400)
}

func writeFixtureCredential(t *testing.T, root string, credential []byte) {
	t.Helper()
	writeFixtureBytes(t, fixturePath(root, protocol.CredentialFilePath), credential, 0o400)
}

func writeFixtureBytes(t *testing.T, path string, payload []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create fixture path parent: %v", err)
	}
	if _, err := os.Lstat(path); err == nil {
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatalf("make fixture %s writable: %v", path, err)
		}
	}
	if err := os.WriteFile(path, payload, mode); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("protect fixture %s: %v", path, err)
	}
}

func runFixture(t *testing.T, binary, root string, extraEnv map[string]string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "-test-runtime-root", root)
	cmd.Env = append([]string(nil), os.Environ()...)
	for name, value := range extraEnv {
		cmd.Env = append(cmd.Env, name+"="+value)
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

type reportingRecorder struct {
	protocol.UnimplementedEngineExecutionReportingServiceServer

	mu                    sync.Mutex
	wantAuthorization     string
	progress              []*protocol.ReportProgressRequest
	results               []*protocol.SubmitResultBatchRequest
	authorizationMetadata [][]string
	progressFailure       error
	resultFailure         error
}

func startReportingServer(t *testing.T, root, token string) (string, *reportingRecorder) {
	t.Helper()
	endpoint := fixturePath(root, protocol.ExecutionEndpointPath)
	if err := os.MkdirAll(filepath.Dir(endpoint), 0o700); err != nil {
		t.Fatalf("create UDS parent: %v", err)
	}
	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		t.Fatalf("listen on fixed task-scoped UDS: %v", err)
	}
	recorder := &reportingRecorder{wantAuthorization: "Bearer " + token}
	server := grpc.NewServer()
	protocol.RegisterEngineExecutionReportingServiceServer(server, recorder)
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
		select {
		case serveErr := <-serveErrCh:
			if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
				t.Errorf("serve task-scoped UDS: %v", serveErr)
			}
		case <-time.After(time.Second):
			t.Error("task-scoped UDS server did not stop")
		}
	})
	return endpoint, recorder
}

func (s *reportingRecorder) ReportProgress(ctx context.Context, request *protocol.ReportProgressRequest) (*protocol.ReportProgressResponse, error) {
	if err := s.authorize(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.progress = append(s.progress, proto.Clone(request).(*protocol.ReportProgressRequest))
	failure := s.progressFailure
	s.mu.Unlock()
	if failure != nil {
		return nil, failure
	}
	return &protocol.ReportProgressResponse{}, nil
}

func (s *reportingRecorder) SubmitResultBatch(ctx context.Context, request *protocol.SubmitResultBatchRequest) (*protocol.SubmitResultBatchResponse, error) {
	if err := s.authorize(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.results = append(s.results, proto.Clone(request).(*protocol.SubmitResultBatchRequest))
	failure := s.resultFailure
	s.mu.Unlock()
	if failure != nil {
		return nil, failure
	}
	return &protocol.SubmitResultBatchResponse{}, nil
}

func (s *reportingRecorder) authorize(ctx context.Context) error {
	values := metadata.ValueFromIncomingContext(ctx, "authorization")
	for _, forbiddenKey := range []string{"task", "scan", "execution", "target", "request_id", "sequence"} {
		if metadata.ValueFromIncomingContext(ctx, forbiddenKey) != nil {
			return status.Error(codes.InvalidArgument, "scope metadata is forbidden")
		}
	}
	s.mu.Lock()
	s.authorizationMetadata = append(s.authorizationMetadata, append([]string(nil), values...))
	s.mu.Unlock()
	if len(values) != 1 || values[0] != s.wantAuthorization {
		return status.Error(codes.Unauthenticated, "invalid execution session credential")
	}
	return nil
}

func (s *reportingRecorder) snapshot() ([]*protocol.ReportProgressRequest, []*protocol.SubmitResultBatchRequest, [][]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	progress := append([]*protocol.ReportProgressRequest(nil), s.progress...)
	results := append([]*protocol.SubmitResultBatchRequest(nil), s.results...)
	authorizations := make([][]string, len(s.authorizationMetadata))
	for i, values := range s.authorizationMetadata {
		authorizations[i] = append([]string(nil), values...)
	}
	return progress, results, authorizations
}

func equalByteSlices(left, right [][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if string(left[i]) != string(right[i]) {
			return false
		}
	}
	return true
}
