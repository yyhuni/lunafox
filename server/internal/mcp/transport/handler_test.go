package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yyhuni/lunafox/server/internal/mcp/admission"
	"github.com/yyhuni/lunafox/server/internal/mcp/audit"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

const transportTestSecret = "lf_mcp_transport_test_secret"

type transportAuthenticator struct {
	key *identitydomain.MCPKey
	err error
}

func (authenticator transportAuthenticator) Authenticate(_ context.Context, secret string) (*identitydomain.MCPKey, error) {
	if secret != transportTestSecret {
		return nil, identitydomain.ErrMCPKeyNotFound
	}
	return authenticator.key, authenticator.err
}

type transportAuditEmitter struct {
	mu     sync.Mutex
	events []audit.Event
}

func (emitter *transportAuditEmitter) Emit(event audit.Event) {
	emitter.mu.Lock()
	defer emitter.mu.Unlock()
	emitter.events = append(emitter.events, event)
}

func (emitter *transportAuditEmitter) last() audit.Event {
	emitter.mu.Lock()
	defer emitter.mu.Unlock()
	if len(emitter.events) == 0 {
		return audit.Event{}
	}
	return emitter.events[len(emitter.events)-1]
}

type transportTargetReader struct {
	list func(context.Context, tools.TargetQuery) (tools.Page[tools.TargetRecord], error)
}

type transportOrganizationCreator struct {
	create func(context.Context, tools.OrganizationCreateInput) (tools.OrganizationCreateOutput, error)
}

func (creator transportOrganizationCreator) Create(ctx context.Context, input tools.OrganizationCreateInput) (tools.OrganizationCreateOutput, error) {
	return creator.create(ctx, input)
}

type transportTargetBatchCreator struct {
	create func(context.Context, tools.TargetBatchCreateInput) (tools.TargetBatchCreateOutput, error)
}

func (creator transportTargetBatchCreator) Create(ctx context.Context, input tools.TargetBatchCreateInput) (tools.TargetBatchCreateOutput, error) {
	return creator.create(ctx, input)
}

func (reader transportTargetReader) List(ctx context.Context, query tools.TargetQuery) (tools.Page[tools.TargetRecord], error) {
	if reader.list == nil {
		return tools.Page[tools.TargetRecord]{}, nil
	}
	return reader.list(ctx, query)
}

func (transportTargetReader) Get(context.Context, int) (tools.TargetRecord, error) {
	return tools.TargetRecord{}, errors.New("not implemented")
}

func newTransportHandlerForTest(reader tools.TargetReader, limiter *admission.Limiter) (*Handler, *transportAuditEmitter) {
	if limiter == nil {
		limiter = admission.NewLimiter(60, 10, 4)
	}
	emitter := &transportAuditEmitter{}
	registry := tools.NewRegistry(tools.Dependencies{Targets: reader})
	return NewHandler(Dependencies{
		Authenticator: transportAuthenticator{key: &identitydomain.MCPKey{ID: 41, UserID: 7}},
		Registry:      registry,
		Limiter:       limiter,
		Audit:         emitter,
	}), emitter
}

type bearerRoundTripper struct {
	base http.RoundTripper
}

func (roundTripper bearerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	cloned := request.Clone(request.Context())
	cloned.Header = request.Header.Clone()
	cloned.Header.Set("Authorization", "Bearer "+transportTestSecret)
	base := roundTripper.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(cloned)
}

func TestHandlerInteroperatesWithOfficialSDK(t *testing.T) {
	handler, emitter := newTransportHandlerForTest(transportTargetReader{
		list: func(_ context.Context, query tools.TargetQuery) (tools.Page[tools.TargetRecord], error) {
			if query.PageSize != 20 {
				t.Fatalf("page size = %d, want default 20", query.PageSize)
			}
			return tools.Page[tools.TargetRecord]{
				Items:     []tools.TargetRecord{{ID: 1, Name: "example.test", Type: "domain"}},
				TotalSize: 1,
			}, nil
		},
	}, nil)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "transport-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   server.URL,
		HTTPClient: &http.Client{Transport: bearerRoundTripper{}},
		MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("official MCP client connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	catalog, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(catalog.Tools) != len(tools.ToolNames()) {
		t.Fatalf("tool count = %d, want %d", len(catalog.Tools), len(tools.ToolNames()))
	}
	foundListTargets := false
	for _, tool := range catalog.Tools {
		if tool.Name == tools.ToolListTargets {
			foundListTargets = true
			break
		}
	}
	if !foundListTargets {
		t.Fatalf("catalog does not include %q", tools.ToolListTargets)
	}
	assertClosedToolSchemas(t, catalog.Tools)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tools.ToolListTargets})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("list targets returned tool error: %+v", result)
	}
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	if !strings.Contains(string(encoded), `"items"`) || !strings.Contains(string(encoded), `"total_size"`) {
		t.Fatalf("list response does not use the MCP page contract: %s", encoded)
	}

	event := emitter.last()
	if event.Method != "tools/call" || event.Tool != tools.ToolListTargets || event.UserID != 7 || event.KeyRecordID != 41 {
		t.Fatalf("unexpected audit event: %+v", event)
	}
}

func TestHandlerExecutesConstrainedCreationThroughStreamableHTTP(t *testing.T) {
	emitter := &transportAuditEmitter{}
	organizationCreated := false
	var receivedOrganizationID *int
	registry := tools.NewRegistry(tools.Dependencies{
		Organizations: transportOrganizationCreator{create: func(_ context.Context, input tools.OrganizationCreateInput) (tools.OrganizationCreateOutput, error) {
			if input.Name == "Taken" {
				return tools.OrganizationCreateOutput{}, mcpErrors.ErrAlreadyExists
			}
			organizationCreated = true
			return tools.OrganizationCreateOutput{ResourceName: "organizations/71", DisplayName: "Platform"}, nil
		}},
		TargetCreator: transportTargetBatchCreator{create: func(_ context.Context, input tools.TargetBatchCreateInput) (tools.TargetBatchCreateOutput, error) {
			if !organizationCreated || input.OrganizationID == nil || *input.OrganizationID != 71 {
				t.Fatalf("target command did not receive chained organization reference: %+v", input)
			}
			receivedOrganizationID = input.OrganizationID
			return tools.TargetBatchCreateOutput{
				CreatedCount:         1,
				Organization:         "organizations/71",
				AssociationCompleted: true,
			}, nil
		}},
	})
	handler := NewHandler(Dependencies{
		Authenticator: transportAuthenticator{key: &identitydomain.MCPKey{ID: 41, UserID: 7}},
		Registry:      registry,
		Limiter:       admission.NewLimiter(60, 10, 4),
		Audit:         emitter,
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "transport-write-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   server.URL,
		HTTPClient: &http.Client{Transport: bearerRoundTripper{}},
		MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("official MCP client connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	catalog, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	assertClosedToolSchemas(t, catalog.Tools)
	assertConstrainedCreationTools(t, catalog.Tools)

	organizationResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      tools.ToolCreateOrganization,
		Arguments: map[string]any{"name": "Platform", "description": "business group"},
	})
	if err != nil || organizationResult.IsError {
		t.Fatalf("create organization = %+v, %v", organizationResult, err)
	}
	organizationOutput := structuredObject(t, organizationResult.StructuredContent)
	organizationName, _ := organizationOutput["name"].(string)
	if organizationName != "organizations/71" {
		t.Fatalf("organization result name = %q", organizationName)
	}

	targetResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: tools.ToolCreateTarget,
		Arguments: map[string]any{
			"targets":      []map[string]any{{"name": "example.com"}},
			"organization": organizationName,
		},
	})
	if err != nil || targetResult.IsError {
		t.Fatalf("create target = %+v, %v", targetResult, err)
	}
	if receivedOrganizationID == nil || *receivedOrganizationID != 71 {
		t.Fatalf("target creator did not receive organization ID: %v", receivedOrganizationID)
	}
	targetOutput := structuredObject(t, targetResult.StructuredContent)
	if targetOutput["organization"] != "organizations/71" || targetOutput["associationCompleted"] != true {
		t.Fatalf("unexpected target result: %#v", targetOutput)
	}

	duplicateResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      tools.ToolCreateOrganization,
		Arguments: map[string]any{"name": "Taken"},
	})
	if err != nil || !duplicateResult.IsError {
		t.Fatalf("duplicate organization = %+v, %v", duplicateResult, err)
	}
	encodedDuplicate, marshalErr := json.Marshal(duplicateResult)
	if marshalErr != nil {
		t.Fatalf("marshal duplicate result: %v", marshalErr)
	}
	if strings.Contains(string(encodedDuplicate), "Taken") || strings.Contains(string(encodedDuplicate), transportTestSecret) {
		t.Fatalf("duplicate tool result leaked sensitive input: %s", encodedDuplicate)
	}
	event := emitter.last()
	if event.Tool != tools.ToolCreateOrganization || event.ResultCategory != string(mcpErrors.CategoryAlreadyExists) || event.UserID != 7 || event.KeyRecordID != 41 {
		t.Fatalf("duplicate audit event = %+v", event)
	}
}

func TestHandlerCancelsConstrainedCreationBeforeCommit(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	emitter := &transportAuditEmitter{}
	registry := tools.NewRegistry(tools.Dependencies{
		Organizations: transportOrganizationCreator{create: func(ctx context.Context, _ tools.OrganizationCreateInput) (tools.OrganizationCreateOutput, error) {
			close(started)
			<-ctx.Done()
			close(cancelled)
			return tools.OrganizationCreateOutput{}, ctx.Err()
		}},
	})
	handler := NewHandler(Dependencies{
		Authenticator: transportAuthenticator{key: &identitydomain.MCPKey{ID: 41, UserID: 7}},
		Registry:      registry,
		Limiter:       admission.NewLimiter(60, 10, 1),
		Audit:         emitter,
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	request := newMCPRequest(t, "tools/call", tools.ToolCreateOrganization, true).WithContext(ctx)
	request.URL = mustParseURL(t, server.URL)
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"create_organization","arguments":{"name":"Cancel"},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"transport-test","version":"1.0.0"},"io.modelcontextprotocol/clientCapabilities":{}}}}`
	request.Body = io.NopCloser(strings.NewReader(body))
	request.ContentLength = int64(len(body))
	responseDone := make(chan error, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		if response != nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		responseDone <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("creation tool did not start")
	}
	cancel()
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("creation tool context was not cancelled")
	}
	select {
	case <-responseDone:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled creation request did not terminate")
	}
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		event := emitter.last()
		if event.Tool == tools.ToolCreateOrganization && event.ResultCategory == "CANCELLED" {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("cancelled creation audit event = %+v", event)
		case <-ticker.C:
		}
	}
}

func TestHandlerRequiresStaticKeyForWriteAndIgnoresOrigin(t *testing.T) {
	called := 0
	registry := tools.NewRegistry(tools.Dependencies{
		TargetCreator: transportTargetBatchCreator{create: func(_ context.Context, input tools.TargetBatchCreateInput) (tools.TargetBatchCreateOutput, error) {
			called++
			if len(input.Names) != 1 || input.OrganizationID != nil {
				t.Fatalf("unexpected ungrouped target command: %+v", input)
			}
			return tools.TargetBatchCreateOutput{CreatedCount: 1}, nil
		}},
	})
	emitter := &transportAuditEmitter{}
	handler := NewHandler(Dependencies{
		Authenticator: transportAuthenticator{key: &identitydomain.MCPKey{ID: 41, UserID: 7}},
		Registry:      registry,
		Limiter:       admission.NewLimiter(60, 10, 4),
		Audit:         emitter,
	})

	missingKey := newMCPToolCallRequest(t, tools.ToolCreateTarget, map[string]any{
		"targets": []map[string]any{{"name": "example.test"}},
	}, false)
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, missingKey)
	if missingResponse.Code != http.StatusUnauthorized || called != 0 {
		t.Fatalf("missing write key response = %d, calls=%d", missingResponse.Code, called)
	}

	withOrigin := newMCPToolCallRequest(t, tools.ToolCreateTarget, map[string]any{
		"targets": []map[string]any{{"name": "example.test"}},
	}, true)
	withOrigin.Header.Set("Origin", "https://untrusted.example")
	originResponse := httptest.NewRecorder()
	handler.ServeHTTP(originResponse, withOrigin)
	if originResponse.Code != http.StatusOK || called != 1 {
		t.Fatalf("origin write response = %d, body=%s, calls=%d", originResponse.Code, originResponse.Body.String(), called)
	}
	if event := emitter.last(); event.Tool != tools.ToolCreateTarget || event.ResultCategory != "success" || event.UserID != 7 || event.KeyRecordID != 41 {
		t.Fatalf("write audit event = %+v", event)
	}
}

func TestHandlerAdmissionBoundaries(t *testing.T) {
	handler, emitter := newTransportHandlerForTest(transportTargetReader{}, nil)

	t.Run("post only", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodPost {
			t.Fatalf("GET response = %d allow=%q", response.Code, response.Header().Get("Allow"))
		}
	})

	t.Run("requires the issued bearer key", func(t *testing.T) {
		request := newMCPRequest(t, "server/discover", "", false)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("missing bearer response = %d, body=%s", response.Code, response.Body.String())
		}
		if response.Header().Get("WWW-Authenticate") == "" {
			t.Fatal("missing WWW-Authenticate challenge")
		}
	})

	t.Run("does not use Origin as an admission condition", func(t *testing.T) {
		request := newMCPRequest(t, "server/discover", "", true)
		request.Header.Set("Origin", "https://untrusted.example")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("arbitrary Origin response = %d, body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("advertises only the supported protocol version", func(t *testing.T) {
		request := newMCPRequest(t, "server/discover", "", true)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("discover response = %d body=%s", response.Code, response.Body.String())
		}
		var envelope struct {
			Result struct {
				SupportedVersions []string `json:"supportedVersions"`
			} `json:"result"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode discover response: %v", err)
		}
		if len(envelope.Result.SupportedVersions) != 1 || envelope.Result.SupportedVersions[0] != ProtocolVersion {
			t.Fatalf("supported versions = %#v, want only %q", envelope.Result.SupportedVersions, ProtocolVersion)
		}
	})

	t.Run("rejects unsupported protocol versions", func(t *testing.T) {
		request := newMCPRequest(t, "server/discover", "", true)
		request.Header.Set("MCP-Protocol-Version", "2025-11-25")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("unsupported protocol response = %d", response.Code)
		}
	})

	t.Run("maps malformed standard headers without reserved MCP codes", func(t *testing.T) {
		request := newMCPRequest(t, "tools/call", tools.ToolListTargets, true)
		request.Header.Del("Mcp-Method")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":-32602`) || strings.Contains(response.Body.String(), "-32020") {
			t.Fatalf("missing method header response = %d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("maps mismatched protocol metadata without reserved MCP codes", func(t *testing.T) {
		request := newMCPRequest(t, "server/discover", "", true)
		body := strings.ReplaceAll(readRequestString(t, request), ProtocolVersion, "2025-11-25")
		request.Body = io.NopCloser(strings.NewReader(body))
		request.ContentLength = int64(len(body))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":-32602`) || strings.Contains(response.Body.String(), "-32020") {
			t.Fatalf("mismatched metadata response = %d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("rejects the legacy initialize handshake", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2026-07-28"}}`
		request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		request.Header.Set("MCP-Protocol-Version", ProtocolVersion)
		request.Header.Set("Authorization", "Bearer "+transportTestSecret)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("legacy initialize response = %d", response.Code)
		}
	})

	t.Run("rejects oversized bodies before parsing", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(strings.Repeat("x", maxRequestBody+1)))
		request.Header.Set("MCP-Protocol-Version", ProtocolVersion)
		request.Header.Set("Authorization", "Bearer "+transportTestSecret)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("large body response = %d", response.Code)
		}
	})

	t.Run("rejects chunked oversized bodies before tool execution", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(strings.Repeat("x", maxRequestBody+1)))
		request.ContentLength = -1
		request.Header.Set("MCP-Protocol-Version", ProtocolVersion)
		request.Header.Set("Authorization", "Bearer "+transportTestSecret)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("chunked large body response = %d", response.Code)
		}
	})

	t.Run("rejects malformed protocol payloads without invoking a tool", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{`))
		request.Header.Set("MCP-Protocol-Version", ProtocolVersion)
		request.Header.Set("Authorization", "Bearer "+transportTestSecret)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("malformed request response = %d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("maps unknown RPC methods to standard JSON-RPC", func(t *testing.T) {
		request := newMCPRequest(t, "unknown/method", "", true)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"code":-32601`) {
			t.Fatalf("unknown method response = %d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("maps unknown tools to standard invalid params", func(t *testing.T) {
		request := newMCPRequest(t, "tools/call", "missing_tool", true)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":-32602`) {
			t.Fatalf("unknown tool response = %d body=%s", response.Code, response.Body.String())
		}
	})

	if event := emitter.last(); event.Method != "tools/call" || event.ErrorCode != "-32602" {
		t.Fatalf("unknown tool audit event = %+v", event)
	}
}

func TestHandlerReturnsHTTPRateLimitWithRetryAfter(t *testing.T) {
	handler, emitter := newTransportHandlerForTest(transportTargetReader{}, admission.NewLimiter(60, 1, 4))
	first := newMCPRequest(t, "server/discover", "", true)
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first admission response = %d", firstResponse.Code)
	}

	second := newMCPRequest(t, "server/discover", "", true)
	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, second)
	if secondResponse.Code != http.StatusTooManyRequests || secondResponse.Header().Get("Retry-After") == "" {
		t.Fatalf("rate-limited response = %d retry-after=%q", secondResponse.Code, secondResponse.Header().Get("Retry-After"))
	}
	if event := emitter.last(); event.ResultCategory != "RATE_LIMITED" || event.UserID != 7 || event.KeyRecordID != 41 {
		t.Fatalf("rate-limit audit event = %+v", event)
	}
}

func TestHandlerReturnsHTTPConcurrencyLimitWithRetryAfter(t *testing.T) {
	started := make(chan struct{})
	releaseWork := make(chan struct{})
	handler, emitter := newTransportHandlerForTest(transportTargetReader{
		list: func(context.Context, tools.TargetQuery) (tools.Page[tools.TargetRecord], error) {
			close(started)
			<-releaseWork
			return tools.Page[tools.TargetRecord]{}, nil
		},
	}, admission.NewLimiter(60, 10, 1))
	first := newMCPRequest(t, "tools/call", tools.ToolListTargets, true)
	firstResponse := httptest.NewRecorder()
	firstDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(firstResponse, first)
		close(firstDone)
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first tool did not occupy a concurrency permit")
	}

	second := newMCPRequest(t, "tools/call", tools.ToolListTargets, true)
	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, second)
	if secondResponse.Code != http.StatusTooManyRequests || secondResponse.Header().Get("Retry-After") == "" {
		t.Fatalf("concurrency-limited response = %d retry-after=%q", secondResponse.Code, secondResponse.Header().Get("Retry-After"))
	}
	if event := emitter.last(); event.ResultCategory != "CONCURRENCY_LIMITED" || event.UserID != 7 || event.KeyRecordID != 41 {
		t.Fatalf("concurrency audit event = %+v", event)
	}
	close(releaseWork)
	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("first tool did not finish")
	}
}

func TestHandlerCancelsToolWorkAndReleasesConcurrencyPermit(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	limiter := admission.NewLimiter(60, 10, 1)
	handler, _ := newTransportHandlerForTest(transportTargetReader{
		list: func(ctx context.Context, _ tools.TargetQuery) (tools.Page[tools.TargetRecord], error) {
			close(started)
			<-ctx.Done()
			close(cancelled)
			return tools.Page[tools.TargetRecord]{}, ctx.Err()
		},
	}, limiter)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	request := newMCPRequest(t, "tools/call", tools.ToolListTargets, true).WithContext(ctx)
	request.URL = mustParseURL(t, server.URL)
	responseDone := make(chan error, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		if response != nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		responseDone <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("tool did not start")
	}
	cancel()
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("tool context was not cancelled after client disconnect")
	}
	select {
	case <-responseDone:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled request did not terminate")
	}

	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		release, allowed := limiter.Acquire(41)
		if allowed {
			release()
			return
		}
		select {
		case <-deadline.C:
			t.Fatal("concurrency permit was not released after cancellation")
		case <-ticker.C:
		}
	}
}

func TestToolDeadlineMiddlewareUsesBusinessDeadline(t *testing.T) {
	var observed time.Duration
	next := toolDeadlineMiddleware(func(ctx context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("tool context has no deadline")
		}
		observed = time.Until(deadline)
		return nil, nil
	})
	if _, err := next(context.Background(), "tools/call", nil); err != nil {
		t.Fatalf("deadline middleware: %v", err)
	}
	if observed < toolDeadline-time.Second || observed > toolDeadline {
		t.Fatalf("tool deadline = %s, want approximately %s", observed, toolDeadline)
	}
}

func newMCPRequest(t *testing.T, method, tool string, authenticated bool) *http.Request {
	t.Helper()
	var arguments any
	if method == "tools/call" {
		arguments = map[string]any{}
	}
	return newMCPJSONRequest(t, method, tool, arguments, authenticated)
}

func newMCPToolCallRequest(t *testing.T, tool string, arguments map[string]any, authenticated bool) *http.Request {
	t.Helper()
	return newMCPJSONRequest(t, "tools/call", tool, arguments, authenticated)
}

func newMCPJSONRequest(t *testing.T, method, tool string, arguments any, authenticated bool) *http.Request {
	t.Helper()
	params := map[string]any{
		"_meta": map[string]any{
			"io.modelcontextprotocol/protocolVersion":    ProtocolVersion,
			"io.modelcontextprotocol/clientInfo":         map[string]any{"name": "transport-test", "version": "1.0.0"},
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		},
	}
	if method == "tools/call" {
		params["name"] = tool
		params["arguments"] = arguments
	}
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		t.Fatalf("marshal test request: %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, "http://example.test/mcp", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("create test request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("MCP-Protocol-Version", ProtocolVersion)
	request.Header.Set("Mcp-Method", method)
	if tool != "" {
		request.Header.Set("Mcp-Name", tool)
	}
	if authenticated {
		request.Header.Set("Authorization", "Bearer "+transportTestSecret)
	}
	return request
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return parsed
}

func readRequestString(t *testing.T, request *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	return string(body)
}

func assertClosedToolSchemas(t *testing.T, catalog []*mcp.Tool) {
	t.Helper()
	expected := map[string]struct{}{}
	for _, name := range tools.ToolNames() {
		expected[name] = struct{}{}
	}
	for _, tool := range catalog {
		if _, ok := expected[tool.Name]; !ok {
			t.Fatalf("unexpected tool in catalog: %q", tool.Name)
		}
		delete(expected, tool.Name)
		encoded, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("marshal %s schema: %v", tool.Name, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(encoded, &schema); err != nil {
			t.Fatalf("decode %s schema: %v", tool.Name, err)
		}
		if schema["type"] != "object" || schema["additionalProperties"] != false {
			t.Fatalf("%s schema is not a closed object: %#v", tool.Name, schema)
		}
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s schema properties = %#v", tool.Name, schema["properties"])
		}
		if _, found := properties["organizationId"]; found {
			t.Fatalf("%s schema must not accept organizationId", tool.Name)
		}
		if strings.HasPrefix(tool.Name, "list_") {
			pageSize, ok := properties["page_size"].(map[string]any)
			defaultSize, maximumSize := float64(20), float64(100)
			if tool.Name == tools.ToolListServerLogEntries || tool.Name == tools.ToolListAgentLogEntries {
				defaultSize, maximumSize = 200, 500
			}
			if !ok || pageSize["default"] != defaultSize || pageSize["maximum"] != maximumSize {
				t.Fatalf("%s page_size schema = %#v", tool.Name, pageSize)
			}
			if _, ok := properties["page_token"].(map[string]any); !ok {
				t.Fatalf("%s missing page_token schema", tool.Name)
			}
		}
		if strings.HasPrefix(tool.Name, "get_") && isLegacyNumericDetailTool(tool.Name) {
			required, ok := schema["required"].([]any)
			if !ok || len(required) != 1 || required[0] != "id" {
				t.Fatalf("%s required schema = %#v", tool.Name, schema["required"])
			}
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing expected tools: %#v", expected)
	}
}

func isLegacyNumericDetailTool(name string) bool {
	switch name {
	case tools.ToolGetTarget, tools.ToolGetScan, tools.ToolGetVulnerability:
		return true
	default:
		return false
	}
}

func assertConstrainedCreationTools(t *testing.T, catalog []*mcp.Tool) {
	t.Helper()
	byName := make(map[string]*mcp.Tool, len(catalog))
	for _, tool := range catalog {
		byName[tool.Name] = tool
	}
	for _, name := range []string{tools.ToolCreateOrganization, tools.ToolCreateTarget} {
		tool := byName[name]
		if tool == nil {
			t.Fatalf("missing constrained creation tool %q", name)
		}
		if !strings.Contains(strings.ToLower(tool.Description), "mutat") {
			t.Fatalf("%s description does not identify the mutation: %q", name, tool.Description)
		}
		if tool.Annotations == nil || tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint {
			t.Fatalf("unexpected mutation annotations for %q: %+v", name, tool.Annotations)
		}
	}
	targetSchema, err := json.Marshal(byName[tools.ToolCreateTarget].InputSchema)
	if err != nil {
		t.Fatalf("marshal create_target schema: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(targetSchema, &decoded); err != nil {
		t.Fatalf("decode create_target schema: %v", err)
	}
	properties := decoded["properties"].(map[string]any)
	if _, ok := properties["organizationId"]; ok {
		t.Fatal("create_target schema unexpectedly accepts organizationId")
	}
	if _, ok := properties["dryRun"]; ok {
		t.Fatal("create_target schema unexpectedly accepts dryRun")
	}
	targets := properties["targets"].(map[string]any)
	if targets["minItems"] != float64(1) || targets["maxItems"] != float64(5000) {
		t.Fatalf("create_target bounds = %#v", targets)
	}
}

func structuredObject(t *testing.T, value any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatalf("decode structured content: %v", err)
	}
	return result
}
