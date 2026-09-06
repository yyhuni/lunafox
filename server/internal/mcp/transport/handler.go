// Package transport exposes LunaFox's stateless Streamable HTTP MCP endpoint.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yyhuni/lunafox/server/internal/mcp/admission"
	"github.com/yyhuni/lunafox/server/internal/mcp/audit"
	"github.com/yyhuni/lunafox/server/internal/mcp/principal"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

const (
	// ProtocolVersion is the only MCP version exposed by the first release.
	ProtocolVersion = "2026-07-28"
	maxRequestBody  = 256 << 10
	toolDeadline    = 25 * time.Second
)

// KeyAuthenticator resolves a bearer credential without exposing its plaintext
// or digest to transport callers.
type KeyAuthenticator interface {
	Authenticate(context.Context, string) (*identitydomain.MCPKey, error)
}

// Dependencies supplies the explicit MCP boundary dependencies.
type Dependencies struct {
	Authenticator KeyAuthenticator
	Registry      *tools.Registry
	Limiter       *admission.Limiter
	Audit         audit.Emitter
}

// Handler enforces LunaFox admission policy before dispatching to the official SDK.
type Handler struct {
	authenticator KeyAuthenticator
	registry      *tools.Registry
	limiter       *admission.Limiter
	audit         audit.Emitter
	streamable    http.Handler
}

// NewHandler creates the stateless HTTP handler registered at /mcp.
func NewHandler(deps Dependencies) *Handler {
	if deps.Authenticator == nil || deps.Registry == nil || deps.Limiter == nil || deps.Audit == nil {
		panic("MCP transport dependencies are required")
	}
	handler := &Handler{
		authenticator: deps.Authenticator,
		registry:      deps.Registry,
		limiter:       deps.Limiter,
		audit:         deps.Audit,
	}
	handler.streamable = mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		server := mcp.NewServer(&mcp.Implementation{
			Name:        "lunafox",
			Title:       "LunaFox",
			Description: "LunaFox investigation and constrained-creation tools.",
			Version:     "1.0.0",
		}, &mcp.ServerOptions{Capabilities: &mcp.ServerCapabilities{}})
		deps.Registry.Register(server)
		server.AddReceivingMiddleware(currentProtocolDiscoveryMiddleware)
		server.AddReceivingMiddleware(toolDeadlineMiddleware)
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		DisableLocalhostProtection:   true,
		MaxRequestBodyBytes:          maxRequestBody,
		PropagateRequestCancellation: true,
	})
	return handler
}

// NewIdentityHandler is the production constructor for identity's key lifecycle service.
func NewIdentityHandler(keyService *identityapp.MCPKeyLifecycleService, registry *tools.Registry) *Handler {
	return NewHandler(Dependencies{
		Authenticator: keyService,
		Registry:      registry,
		Limiter:       admission.NewLimiter(60, 10, 4),
		Audit:         audit.NewLogger(),
	})
}

// ServeHTTP validates the fixed HTTP contract before letting the SDK parse MCP.
func (handler *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	startedAt := time.Now()
	observed := &responseObserver{ResponseWriter: response, ctx: request.Context()}
	requestID := request.Header.Get("Request-Id")
	inspection := inspectRequest(nil)

	if request.Method != http.MethodPost {
		observed.Header().Set("Allow", http.MethodPost)
		observed.WriteHeader(http.StatusMethodNotAllowed)
		handler.emitAudit(startedAt, requestID, principal.Principal{}, inspection, "METHOD_NOT_ALLOWED", "", observed.bytes)
		return
	}
	if request.ContentLength > maxRequestBody {
		observed.WriteHeader(http.StatusRequestEntityTooLarge)
		handler.emitAudit(startedAt, requestID, principal.Principal{}, inspection, "BODY_TOO_LARGE", "", observed.bytes)
		return
	}
	body, err := readRequestBody(observed, request)
	if err != nil {
		status := http.StatusBadRequest
		category := "BAD_REQUEST"
		if errors.As(err, new(*http.MaxBytesError)) {
			status = http.StatusRequestEntityTooLarge
			category = "BODY_TOO_LARGE"
		}
		observed.WriteHeader(status)
		handler.emitAudit(startedAt, requestID, principal.Principal{}, inspection, category, "", observed.bytes)
		return
	}
	mediaType, _, mediaErr := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if mediaErr != nil || !strings.EqualFold(mediaType, "application/json") {
		observed.WriteHeader(http.StatusBadRequest)
		handler.emitAudit(startedAt, requestID, principal.Principal{}, inspection, "BAD_REQUEST", "", observed.bytes)
		return
	}
	inspection = inspectRequest(body)
	if request.Header.Get("MCP-Protocol-Version") != ProtocolVersion || isLegacyInitialize(body) {
		observed.WriteHeader(http.StatusBadRequest)
		handler.emitAudit(startedAt, requestID, principal.Principal{}, inspection, "UNSUPPORTED_PROTOCOL", "", observed.bytes)
		return
	}
	if !validateStandardHeaders(body, request.Header) {
		writeInvalidRequest(observed, body, "invalid MCP request headers")
		handler.emitAudit(startedAt, requestID, principal.Principal{}, inspection, "BAD_REQUEST", "-32602", observed.bytes)
		return
	}

	secret, ok := bearerSecret(request.Header.Get("Authorization"))
	if !ok {
		writeUnauthorized(observed)
		handler.emitAudit(startedAt, requestID, principal.Principal{}, inspection, "AUTHENTICATION_FAILED", "", observed.bytes)
		return
	}
	key, err := handler.authenticator.Authenticate(request.Context(), secret)
	if err != nil || key == nil || key.ID <= 0 || key.UserID <= 0 {
		writeUnauthorized(observed)
		handler.emitAudit(startedAt, requestID, principal.Principal{}, inspection, "AUTHENTICATION_FAILED", "", observed.bytes)
		return
	}
	actor := principal.Principal{UserID: key.UserID, KeyRecordID: key.ID}
	if allowed, retryAfter := handler.limiter.Allow(key.ID); !allowed {
		writeRateLimited(observed, retryAfter)
		handler.emitAudit(startedAt, requestID, actor, inspection, "RATE_LIMITED", "", observed.bytes)
		return
	}

	release := func() {}
	if inspection.Method == "tools/call" {
		var acquired bool
		release, acquired = handler.limiter.Acquire(key.ID)
		if !acquired {
			writeRateLimited(observed, time.Second)
			handler.emitAudit(startedAt, requestID, actor, inspection, "CONCURRENCY_LIMITED", "", observed.bytes)
			return
		}
		defer release()
	}

	state := &tools.CallState{Tool: inspection.Tool}
	request.Body = io.NopCloser(bytes.NewReader(body))
	request = request.WithContext(tools.WithCallState(principal.With(request.Context(), actor), state))
	handler.streamable.ServeHTTP(observed, request)
	if request.Context().Err() != nil {
		state.Mark("CANCELLED", "")
	}

	tool, category, errorCode := state.Snapshot()
	if tool == "" {
		tool = inspection.Tool
	}
	if category == "" {
		category = "SUCCESS"
		if observed.status >= 400 {
			category = "PROTOCOL_ERROR"
		}
	}
	if errorCode == "" && observed.status == http.StatusNotFound {
		errorCode = "-32601"
	}
	if errorCode == "" && inspection.Method == "tools/call" && inspection.Tool != "" && !handler.registry.HasTool(inspection.Tool) {
		errorCode = "-32602"
	}
	handler.emitAudit(startedAt, requestID, actor, requestInspection{Method: inspection.Method, Tool: tool}, category, errorCode, observed.bytes)
}

func toolDeadlineMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
		if method != "tools/call" {
			return next(ctx, method, request)
		}
		deadlineCtx, cancel := context.WithTimeout(ctx, toolDeadline)
		defer cancel()
		return next(deadlineCtx, method, request)
	}
}

func readRequestBody(response http.ResponseWriter, request *http.Request) ([]byte, error) {
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	body, err := io.ReadAll(request.Body)
	if closeErr := request.Body.Close(); err == nil {
		err = closeErr
	}
	return body, err
}

func bearerSecret(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func writeUnauthorized(response http.ResponseWriter) {
	response.Header().Set("WWW-Authenticate", `Bearer realm="lunafox-mcp"`)
	response.WriteHeader(http.StatusUnauthorized)
}

func writeRateLimited(response http.ResponseWriter, retryAfter time.Duration) {
	response.Header().Set("Retry-After", strconv.Itoa(admission.RetryAfterSeconds(retryAfter)))
	response.WriteHeader(http.StatusTooManyRequests)
}

func (handler *Handler) emitAudit(startedAt time.Time, requestID string, actor principal.Principal, inspection requestInspection, category, errorCode string, responseBytes int) {
	handler.audit.Emit(audit.Event{
		RequestID: requestID, UserID: actor.UserID, KeyRecordID: actor.KeyRecordID,
		Method: inspection.Method, Tool: inspection.Tool, ResultCategory: category, ErrorCode: errorCode,
		Duration: time.Since(startedAt), ResponseBytes: responseBytes,
	})
}

type requestInspection struct {
	Method string
	Tool   string
}

func inspectRequest(body []byte) requestInspection {
	var envelope struct {
		Method string `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return requestInspection{}
	}
	return requestInspection{Method: envelope.Method, Tool: envelope.Params.Name}
}

// validateStandardHeaders prevents the SDK's transport-specific -32020 header
// code from escaping this boundary. LunaFox exposes only standard JSON-RPC
// invalid-params semantics for malformed MCP headers.
func validateStandardHeaders(body []byte, headers http.Header) bool {
	var envelope struct {
		Method string `json:"method"`
		Params struct {
			Name string         `json:"name"`
			Meta map[string]any `json:"_meta"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Method == "" {
		return true
	}
	if method := headers.Get("Mcp-Method"); method == "" || method != envelope.Method {
		return false
	}
	if envelope.Method == "tools/call" && (headers.Get("Mcp-Name") == "" || headers.Get("Mcp-Name") != envelope.Params.Name) {
		return false
	}
	if metaVersion, ok := envelope.Params.Meta["io.modelcontextprotocol/protocolVersion"].(string); ok && metaVersion != headers.Get("MCP-Protocol-Version") {
		return false
	}
	return true
}

func writeInvalidRequest(response http.ResponseWriter, body []byte, message string) {
	var envelope struct {
		ID json.RawMessage `json:"id"`
	}
	_ = json.Unmarshal(body, &envelope)
	if len(envelope.ID) == 0 {
		envelope.ID = json.RawMessage("null")
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusBadRequest)
	_, _ = response.Write([]byte(`{"jsonrpc":"2.0","id":` + string(envelope.ID) + `,"error":{"code":-32602,"message":"` + message + `"}}`))
}

// isLegacyInitialize rejects the deprecated handshake. MCP 2026-07-28 clients
// negotiate the stateless endpoint through server/discover instead.
func isLegacyInitialize(body []byte) bool {
	var envelope struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false // Let the SDK return the standard malformed-request mapping.
	}
	return envelope.Method == "initialize"
}

// currentProtocolDiscoveryMiddleware narrows the SDK's compatibility catalog
// to LunaFox's one supported wire version without replacing SDK protocol logic.
func currentProtocolDiscoveryMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
		result, err := next(ctx, method, request)
		if err != nil || method != "server/discover" {
			return result, err
		}
		if discovery, ok := result.(*mcp.DiscoverResult); ok {
			discovery.SupportedVersions = []string{ProtocolVersion}
		}
		return result, nil
	}
}
