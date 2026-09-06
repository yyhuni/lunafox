package sdk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unicode/utf8"

	"github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	authorizationMetadataKey = "authorization"
	credentialLength         = 43
	credentialTokenBytes     = 32
	maxResultTypeBytes       = 128
)

// Adapter is the generated bridge between raw Protocol and one Engine-local
// typed execution contract. It receives a detached Context, narrow reporting
// port, and the closed Registry's role-specific path operations. It never
// receives a credential, connection, Agent identity, role string, or generic
// input client; generated code projects these operations into handler-only
// typed Path(ctx) handles.
type Adapter func(context.Context, *protocol.EngineExecutionContext, Reporter, interface {
	SubdomainsPath(context.Context) (string, error)
	HostPortsPath(context.Context) (string, error)
	WebsiteURLsPath(context.Context) (string, error)
	EndpointURLsPath(context.Context) (string, error)
}) error

// Reporter is the narrow reporting capability passed only to generated code.
// Engine-local adapters construct a ResultSink with a fixed canonical type.
type Reporter interface {
	ReportProgress(context.Context, string) error
	ResultSink(string) (ResultSink, error)
	GeneratedDiagnostics() (GeneratedDiagnostics, error)
}

// ResultSink submits ordered, already-encoded JSON object items. A nil result
// means all unary batches were acknowledged; earlier accepted batches are not
// rolled back when a later call fails.
type ResultSink interface {
	Submit(context.Context, <-chan []byte) error
}

// Error is the transport-neutral reporting error exposed to Engine code.
// Reason is empty only for a strict indeterminate transport failure.
type Error struct {
	Code   codes.Code
	Reason protocol.ErrorReason
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	if err.Reason == "" {
		return err.Code.String()
	}
	return err.Code.String() + ": " + string(err.Reason)
}

func (err *Error) Is(target error) bool {
	if err == nil {
		return false
	}
	return (errors.Is(target, context.Canceled) && err.Code == codes.Canceled) ||
		(errors.Is(target, context.DeadlineExceeded) && err.Code == codes.DeadlineExceeded)
}

type runnerPaths struct {
	contextPath    string
	credentialPath string
	endpointPath   string
	runtimeRoot    string
}

func canonicalRunnerPaths() runnerPaths {
	return runnerPaths{
		contextPath:    protocol.ContextFilePath,
		credentialPath: protocol.CredentialFilePath,
		endpointPath:   protocol.ExecutionEndpointPath,
		runtimeRoot:    protocol.RuntimeRootPath,
	}
}

// Run is the single production lifecycle entry for a Go Engine. It owns
// process cancellation, fixed bootstrap, transport cleanup, and credential
// clearing; the caller maps its returned error to process behavior.
func Run(adapter Adapter) error {
	return run(adapter, canonicalRunnerPaths())
}

func run(adapter Adapter, paths runnerPaths) (runErr error) {
	if adapter == nil {
		return errors.New("engine adapter is required")
	}
	if err := validateRunnerPaths(paths); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	snapshot, err := loadContext(paths)
	if err != nil {
		return err
	}
	credential, err := loadCredential(paths.credentialPath)
	if err != nil {
		return err
	}
	defer zeroBytes(credential)

	connection, err := dial(ctx, paths.endpointPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := connection.Close(); closeErr != nil && runErr == nil {
			runErr = errors.New("close engine reporting connection")
		}
	}()

	client := newReporterWithDiagnostics(
		snapshot,
		protocol.NewEngineExecutionReportingServiceClient(connection),
		protocol.NewEngineExecutionDiagnosticsServiceClient(connection),
		credential,
	)
	defer client.close()
	if err := client.establishExecutionDiagnostics(ctx); err != nil {
		return err
	}
	inputs := newInputResolver(client, protocol.NewEngineExecutionInputServiceClient(connection))

	adapterErr := adapter(ctx, protocolClone(snapshot), client, inputs)
	client.reportTerminalDiagnostics(adapterErr, ctx)
	return gracefulSignalCancellation(ctx, adapterErr)
}

// gracefulSignalCancellation preserves the Engine container lifecycle
// contract: a handler interrupted by SIGINT or SIGTERM exits successfully
// when it returns cancellation rather than an operational failure.
func gracefulSignalCancellation(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx != nil && errors.Is(ctx.Err(), context.Canceled) &&
		(errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled) {
		return nil
	}
	return err
}

func newReporter(
	snapshot *protocol.EngineExecutionContext,
	reporting protocol.EngineExecutionReportingServiceClient,
	credential []byte,
) *reporter {
	return newReporterWithDiagnostics(snapshot, reporting, nil, credential)
}

func newReporterWithDiagnostics(
	snapshot *protocol.EngineExecutionContext,
	reporting protocol.EngineExecutionReportingServiceClient,
	diagnosticClient protocol.EngineExecutionDiagnosticsServiceClient,
	credential []byte,
) *reporter {
	sessionContext, sessionCancel := context.WithCancel(context.Background())
	return &reporter{
		snapshot:         protocolClone(snapshot),
		reporting:        reporting,
		diagnosticClient: diagnosticClient,
		diagnosticState:  newDiagnosticAccumulator(),
		credential:       append([]byte(nil), credential...),
		sessionContext:   sessionContext,
		sessionCancel:    sessionCancel,
		activeCalls:      make(map[*activeReporterCall]struct{}),
	}
}

func protocolClone(source *protocol.EngineExecutionContext) *protocol.EngineExecutionContext {
	if source == nil {
		return nil
	}
	return proto.Clone(source).(*protocol.EngineExecutionContext)
}

func validateRunnerPaths(paths runnerPaths) error {
	if paths.contextPath == "" || paths.credentialPath == "" || paths.endpointPath == "" || paths.runtimeRoot == "" {
		return errors.New("engine runner paths are required")
	}
	for _, value := range []string{paths.contextPath, paths.credentialPath, paths.endpointPath, paths.runtimeRoot} {
		if err := validateAbsoluteCleanPath(value); err != nil {
			return errors.New("engine runner path must be absolute and clean")
		}
	}
	if !pathWithin(paths.runtimeRoot, paths.contextPath) || !pathWithin(paths.runtimeRoot, paths.credentialPath) || !pathWithin(paths.runtimeRoot, paths.endpointPath) {
		return errors.New("engine bootstrap paths must remain below runtime root")
	}
	return nil
}

func loadContext(paths runnerPaths) (*protocol.EngineExecutionContext, error) {
	if err := validateReadOnlyRegularFile("engine context", paths.contextPath); err != nil {
		return nil, err
	}
	payload, err := os.ReadFile(paths.contextPath)
	if err != nil {
		return nil, redactFilesystemError("read engine context", err)
	}
	var snapshot protocol.EngineExecutionContext
	if err := proto.Unmarshal(payload, &snapshot); err != nil {
		return nil, errors.New("decode binary engine context")
	}
	if err := validateContext(&snapshot, paths); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func validateContext(snapshot *protocol.EngineExecutionContext, paths runnerPaths) error {
	if snapshot == nil || snapshot.GetTarget() == nil || snapshot.GetConfig() == nil {
		return errors.New("engine context required fields are missing")
	}
	if hasUnknownFields(snapshot.ProtoReflect()) {
		return errors.New("engine context contains unsupported fields")
	}
	if snapshot.GetCompatibilityRevision() != protocol.EngineExecutionDiagnosticsCompatibilityRevision {
		return errors.New("engine context compatibility revision is invalid")
	}
	if err := requireCanonicalText(snapshot.GetTarget().GetType()); err != nil {
		return errors.New("engine target type is invalid")
	}
	if err := requireCanonicalText(snapshot.GetTarget().GetValue()); err != nil {
		return errors.New("engine target value is invalid")
	}
	if err := protocol.ValidateLimits(snapshot.GetLimits()); err != nil {
		return err
	}

	seenPaths := make(map[string]struct{})
	seenSections := make(map[string]struct{})
	for _, section := range snapshot.GetConfig().GetSections() {
		if section == nil || !validIdentifier(section.GetSectionId()) || section.Enabled == nil {
			return errors.New("engine config section is invalid")
		}
		if _, duplicate := seenSections[section.GetSectionId()]; duplicate {
			return errors.New("engine config section IDs must be unique")
		}
		seenParams := make(map[string]struct{})
		for _, param := range section.GetParams() {
			if param == nil || !validIdentifier(param.GetParamKey()) || param.GetValue() == nil {
				return errors.New("engine config value is invalid")
			}
			if !section.GetEnabled() {
				return errors.New("disabled engine config section contains scalar values")
			}
			if _, duplicate := seenParams[param.GetParamKey()]; duplicate {
				return errors.New("engine config parameter keys must be unique")
			}
			if array := param.GetStringArrayValue(); array != nil {
				for _, value := range array.GetValues() {
					if !utf8.ValidString(value) {
						return errors.New("engine config string array value is invalid")
					}
				}
			}
			seenParams[param.GetParamKey()] = struct{}{}
		}
		seenSections[section.GetSectionId()] = struct{}{}
	}
	seenConfigResources := make(map[string]struct{})
	for _, resource := range snapshot.GetConfigResources() {
		if resource == nil || !validIdentifier(resource.GetSectionId()) || !validIdentifier(resource.GetParamKey()) || !validContentType(resource.GetContentType()) {
			return errors.New("engine config resource is invalid")
		}
		key := resource.GetSectionId() + "\x00" + resource.GetParamKey()
		if _, duplicate := seenConfigResources[key]; duplicate {
			return errors.New("engine config resource bindings must be unique")
		}
		if err := validateBoundFile(resource.GetPath(), paths.runtimeRoot, seenPaths); err != nil {
			return err
		}
		seenConfigResources[key] = struct{}{}
	}
	seenPlatformResources := make(map[string]struct{})
	for _, resource := range snapshot.GetPlatformResources() {
		if resource == nil || !validIdentifier(resource.GetResourceId()) || !validContentType(resource.GetContentType()) {
			return errors.New("engine platform resource is invalid")
		}
		if _, duplicate := seenPlatformResources[resource.GetResourceId()]; duplicate {
			return errors.New("engine platform resource IDs must be unique")
		}
		if err := validateBoundFile(resource.GetPath(), paths.runtimeRoot, seenPaths); err != nil {
			return err
		}
		seenPlatformResources[resource.GetResourceId()] = struct{}{}
	}
	return nil
}

func hasUnknownFields(message protoreflect.Message) bool {
	if !message.IsValid() || len(message.GetUnknown()) != 0 {
		return true
	}
	var unknown bool
	message.Range(func(descriptor protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if descriptor.IsList() {
			list := value.List()
			for index := 0; index < list.Len(); index++ {
				if descriptor.Kind() == protoreflect.MessageKind && hasUnknownFields(list.Get(index).Message()) {
					unknown = true
					return false
				}
			}
			return true
		}
		if descriptor.Kind() == protoreflect.MessageKind && hasUnknownFields(value.Message()) {
			unknown = true
			return false
		}
		return true
	})
	return unknown
}

func validIdentifier(value string) bool {
	return requireCanonicalText(value) == nil && len(value) <= 128
}

func validContentType(value string) bool {
	return requireCanonicalText(value) == nil && strings.Contains(value, "/") && len(value) <= 256
}

func validateBoundFile(path, root string, seen map[string]struct{}) error {
	if err := validateAbsoluteCleanPath(path); err != nil || !pathWithin(root, path) {
		return errors.New("engine bound file path is invalid")
	}
	if _, exists := seen[path]; exists {
		return errors.New("engine bound file paths must be unique")
	}
	if err := validateReadOnlyRegularFile("engine bound file", path); err != nil {
		return err
	}
	seen[path] = struct{}{}
	return nil
}

func loadCredential(path string) ([]byte, error) {
	if err := validateReadOnlyRegularFile("engine credential", path); err != nil {
		return nil, err
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, redactFilesystemError("read engine credential", err)
	}
	if len(payload) != credentialLength {
		zeroBytes(payload)
		return nil, errors.New("engine credential is invalid")
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(string(payload))
	if err != nil || len(decoded) != credentialTokenBytes || base64.RawURLEncoding.EncodeToString(decoded) != string(payload) {
		zeroBytes(decoded)
		zeroBytes(payload)
		return nil, errors.New("engine credential is invalid")
	}
	zeroBytes(decoded)
	return payload, nil
}

func dial(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	if err := validateSocket(endpoint); err != nil {
		return nil, err
	}
	connection, err := grpc.NewClient(
		"passthrough:///lunafox-engine-execution-v2",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDisableRetry(),
		grpc.WithStatsHandler(rpcStatsHandler{}),
		grpc.WithContextDialer(func(dialContext context.Context, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(dialContext, "unix", endpoint)
		}),
	)
	if err != nil {
		return nil, errors.New("create engine reporting client")
	}
	if err := waitForReady(ctx, connection); err != nil {
		_ = connection.Close()
		return nil, err
	}
	return connection, nil
}

func validateSocket(path string) error {
	if err := validateAbsoluteCleanPath(path); err != nil {
		return errors.New("engine reporting socket path is invalid")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return redactFilesystemError("inspect engine reporting socket", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeSocket == 0 {
		return errors.New("engine reporting endpoint must be a filesystem socket")
	}
	return validatePathWithoutSymlinks(path)
}

func waitForReady(ctx context.Context, connection *grpc.ClientConn) error {
	connection.Connect()
	for {
		state := connection.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if state == connectivity.Shutdown {
			return errors.New("engine reporting connection shut down before ready")
		}
		if !connection.WaitForStateChange(ctx, state) {
			if err := ctx.Err(); err != nil {
				return err
			}
			return errors.New("engine reporting connection did not become ready")
		}
	}
}

type reporter struct {
	snapshot         *protocol.EngineExecutionContext
	reporting        protocol.EngineExecutionReportingServiceClient
	diagnosticClient protocol.EngineExecutionDiagnosticsServiceClient
	diagnosticState  *diagnosticAccumulator
	credential       []byte
	sessionContext   context.Context
	sessionCancel    context.CancelFunc
	mu               sync.RWMutex
	closed           bool
	revoked          bool
	activeCalls      map[*activeReporterCall]struct{}
}

type activeReporterCall struct {
	cancel context.CancelFunc
}

func (client *reporter) close() {
	if client == nil {
		return
	}
	client.mu.Lock()
	if client.closed {
		client.mu.Unlock()
		return
	}
	client.closed = true
	for call := range client.activeCalls {
		call.cancel()
	}
	clear(client.activeCalls)
	client.activeCalls = nil
	zeroBytes(client.credential)
	client.credential = nil
	if client.sessionCancel != nil {
		client.sessionCancel()
	}
	client.mu.Unlock()
}

// revoke is linearized with call registration. An Agent-provided revoke status
// makes all existing calls observe cancellation before future calls are denied.
func (client *reporter) revoke() {
	if client == nil {
		return
	}
	client.mu.Lock()
	if client.revoked {
		client.mu.Unlock()
		return
	}
	client.revoked = true
	client.closed = true
	for call := range client.activeCalls {
		call.cancel()
	}
	clear(client.activeCalls)
	client.activeCalls = nil
	zeroBytes(client.credential)
	client.credential = nil
	if client.sessionCancel != nil {
		client.sessionCancel()
	}
	client.mu.Unlock()
}

func (client *reporter) isRevoked() bool {
	if client == nil {
		return false
	}
	client.mu.RLock()
	defer client.mu.RUnlock()
	return client.revoked
}

func (client *reporter) ReportProgress(ctx context.Context, message string) error {
	if ctx == nil {
		return errors.New("progress context is required")
	}
	if message == "" || !utf8.ValidString(message) {
		return &Error{Code: codes.InvalidArgument, Reason: protocol.ReasonInvalidReportingRequest}
	}
	if uint64(len([]byte(message))) > uint64(client.snapshot.GetLimits().GetProgressMessageMaxBytes()) {
		return &Error{Code: codes.ResourceExhausted, Reason: protocol.ReasonReportingLimitExceeded}
	}
	callContext, cleanup, err := client.callContext(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	response, err := client.reporting.ReportProgress(callContext, &protocol.ReportProgressRequest{Message: message})
	if err != nil {
		return client.mapCallError(err)
	}
	if response == nil {
		return &Error{Code: codes.Internal, Reason: protocol.ReasonReportingInternalInvariant}
	}
	return nil
}

func (client *reporter) ResultSink(resultType string) (ResultSink, error) {
	if err := validateResultType(resultType); err != nil {
		return nil, err
	}
	if err := client.diagnosticState.requireRegistered(resultType); err != nil {
		return nil, diagnosticError(err)
	}
	client.mu.RLock()
	closed := client.closed
	client.mu.RUnlock()
	if closed {
		return nil, errors.New("engine reporter is closed")
	}
	return &resultSink{client: client, resultType: resultType}, nil
}

func (client *reporter) GeneratedDiagnostics() (GeneratedDiagnostics, error) {
	return client.generatedDiagnostics()
}

type resultSink struct {
	client     *reporter
	resultType string
}

func (sink *resultSink) Submit(ctx context.Context, items <-chan []byte) error {
	if sink == nil || sink.client == nil || items == nil {
		return errors.New("result item stream is required")
	}
	if ctx == nil {
		return errors.New("result submission context is required")
	}
	limits := sink.client.snapshot.GetLimits()
	batch := make([][]byte, 0, limits.GetResultBatchMaxItems())
	batchBytes := 0
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := sink.client.submitBatch(ctx, sink.resultType, batch); err != nil {
			return err
		}
		batch = make([][]byte, 0, limits.GetResultBatchMaxItems())
		batchBytes = 0
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return mapContextError(ctx.Err())
		case item, open := <-items:
			if !open {
				return flush()
			}
			if !isJSONObject(item) {
				return &Error{Code: codes.InvalidArgument, Reason: protocol.ReasonInvalidReportingRequest}
			}
			if len(item) > int(limits.GetResultBatchMaxBytes()) {
				return &Error{Code: codes.ResourceExhausted, Reason: protocol.ReasonReportingLimitExceeded}
			}
			if len(batch) > 0 && (len(batch) >= int(limits.GetResultBatchMaxItems()) || batchBytes > int(limits.GetResultBatchMaxBytes())-len(item)) {
				if err := flush(); err != nil {
					return err
				}
			}
			batch = append(batch, append([]byte(nil), item...))
			batchBytes += len(item)
			if len(batch) == int(limits.GetResultBatchMaxItems()) || batchBytes == int(limits.GetResultBatchMaxBytes()) {
				if err := flush(); err != nil {
					return err
				}
			}
		}
	}
}

func (client *reporter) submitBatch(ctx context.Context, resultType string, items [][]byte) error {
	callContext, cleanup, err := client.callContext(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	request := &protocol.SubmitResultBatchRequest{ResultType: resultType, Items: cloneBytes(items)}
	if err := client.diagnosticState.recordSubmitted(resultType, uint64(len(items))); err != nil {
		return diagnosticError(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if client.isRevoked() {
			return &Error{Code: codes.Canceled, Reason: protocol.ReasonSessionRevoked}
		}
		observation := new(rpcObservation)
		attemptContext := context.WithValue(callContext, observationContextKey{}, observation)
		var header metadata.MD
		var trailer metadata.MD
		response, callErr := client.reporting.SubmitResultBatch(attemptContext, request, grpc.Header(&header), grpc.Trailer(&trailer))
		if callErr == nil {
			if response == nil {
				return &Error{Code: codes.Internal, Reason: protocol.ReasonReportingInternalInvariant}
			}
			if err := client.diagnosticState.recordAcknowledged(resultType, uint64(len(items))); err != nil {
				return diagnosticError(err)
			}
			return nil
		}
		if attempt == 1 || !indeterminateUnavailable(callContext, callErr, header, trailer, observation) {
			return client.mapCallError(callErr)
		}
	}
	return &Error{Code: codes.Internal, Reason: protocol.ReasonReportingInternalInvariant}
}

func (client *reporter) callContext(parent context.Context) (context.Context, func(), error) {
	if parent == nil {
		return nil, nil, errors.New("engine reporting context is required")
	}
	if err := parent.Err(); err != nil {
		return nil, nil, mapContextError(err)
	}
	callContext, callCancel := context.WithCancel(parent)
	call := &activeReporterCall{cancel: callCancel}
	client.mu.Lock()
	if client.revoked {
		client.mu.Unlock()
		callCancel()
		return nil, nil, &Error{Code: codes.Canceled, Reason: protocol.ReasonSessionRevoked}
	}
	if client.closed || len(client.credential) != credentialLength || client.reporting == nil || client.sessionContext == nil {
		client.mu.Unlock()
		callCancel()
		return nil, nil, errors.New("engine reporter is closed")
	}
	client.activeCalls[call] = struct{}{}
	sessionContext := client.sessionContext
	metadataValue := "Bearer " + string(client.credential)
	client.mu.Unlock()
	stopSession := context.AfterFunc(sessionContext, callCancel)
	cleanup := func() {
		stopSession()
		client.mu.Lock()
		delete(client.activeCalls, call)
		client.mu.Unlock()
		callCancel()
	}
	return metadata.AppendToOutgoingContext(callContext, authorizationMetadataKey, metadataValue), cleanup, nil
}

func (client *reporter) mapCallError(err error) error {
	if client.isRevoked() {
		return &Error{Code: codes.Canceled, Reason: protocol.ReasonSessionRevoked}
	}
	mapped := mapError(err)
	if typed, ok := mapped.(*Error); ok && typed.Reason == protocol.ReasonSessionRevoked {
		client.revoke()
	}
	return mapped
}

func validateResultType(value string) error {
	if value == "" || value != strings.TrimSpace(value) || !utf8.ValidString(value) || len([]byte(value)) > maxResultTypeBytes {
		return &Error{Code: codes.InvalidArgument, Reason: protocol.ReasonInvalidReportingRequest}
	}
	segments := strings.Split(value, ".")
	if len(segments) < 2 {
		return &Error{Code: codes.InvalidArgument, Reason: protocol.ReasonInvalidReportingRequest}
	}
	for _, segment := range segments {
		if segment == "" || segment[0] == '_' || segment[0] == '-' || segment[len(segment)-1] == '_' || segment[len(segment)-1] == '-' {
			return &Error{Code: codes.InvalidArgument, Reason: protocol.ReasonInvalidReportingRequest}
		}
		for _, runeValue := range segment {
			if (runeValue >= 'a' && runeValue <= 'z') || (runeValue >= '0' && runeValue <= '9') || runeValue == '_' || runeValue == '-' {
				continue
			}
			return &Error{Code: codes.InvalidArgument, Reason: protocol.ReasonInvalidReportingRequest}
		}
	}
	return nil
}

func isJSONObject(item []byte) bool {
	trimmed := strings.TrimSpace(string(item))
	return utf8.Valid(item) && json.Valid(item) && len(trimmed) >= 2 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}'
}

func cloneBytes(source [][]byte) [][]byte {
	result := make([][]byte, len(source))
	for index := range source {
		result[index] = append([]byte(nil), source[index]...)
	}
	return result
}

type observationContextKey struct{}

type rpcObservation struct {
	mu        sync.Mutex
	responded bool
}

func (observation *rpcObservation) markResponded() {
	observation.mu.Lock()
	observation.responded = true
	observation.mu.Unlock()
}

func (observation *rpcObservation) receivedResponse() bool {
	observation.mu.Lock()
	defer observation.mu.Unlock()
	return observation.responded
}

type rpcStatsHandler struct{}

func (rpcStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context   { return ctx }
func (rpcStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context { return ctx }
func (rpcStatsHandler) HandleConn(context.Context, stats.ConnStats)                       {}
func (rpcStatsHandler) HandleRPC(ctx context.Context, event stats.RPCStats) {
	observation, _ := ctx.Value(observationContextKey{}).(*rpcObservation)
	if observation == nil {
		return
	}
	switch event.(type) {
	case *stats.InHeader, *stats.InTrailer:
		observation.markResponded()
	}
}

func indeterminateUnavailable(ctx context.Context, err error, header, trailer metadata.MD, observation *rpcObservation) bool {
	if ctx.Err() != nil || len(header) != 0 || len(trailer) != 0 || observation == nil || observation.receivedResponse() {
		return false
	}
	grpcStatus, ok := status.FromError(err)
	return ok && grpcStatus.Code() == codes.Unavailable && len(grpcStatus.Details()) == 0
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return &Error{Code: codes.Canceled, Reason: protocol.ReasonCallerCanceled}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{Code: codes.DeadlineExceeded, Reason: protocol.ReasonCallerDeadlineExceeded}
	}
	grpcStatus, ok := status.FromError(err)
	if !ok {
		return &Error{Code: codes.Internal, Reason: protocol.ReasonReportingInternalInvariant}
	}
	for _, detail := range grpcStatus.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok && info.GetDomain() == protocol.ErrorDomain {
			reason := protocol.ErrorReason(info.GetReason())
			if protocol.ValidErrorReason(reason) {
				return &Error{Code: grpcStatus.Code(), Reason: reason}
			}
		}
	}
	if grpcStatus.Code() == codes.Unavailable {
		return &Error{Code: codes.Unavailable}
	}
	return &Error{Code: codes.Internal, Reason: protocol.ReasonReportingInternalInvariant}
}

func mapContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{Code: codes.DeadlineExceeded, Reason: protocol.ReasonCallerDeadlineExceeded}
	}
	return &Error{Code: codes.Canceled, Reason: protocol.ReasonCallerCanceled}
}

func validateReadOnlyRegularFile(label, path string) error {
	if err := validateAbsoluteCleanPath(path); err != nil {
		return fmt.Errorf("%s path is invalid", label)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return redactFilesystemError("inspect "+label, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%s must be a regular file", label)
	}
	if err := validatePathWithoutSymlinks(path); err != nil {
		return err
	}
	probe, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err == nil {
		_ = probe.Close()
		return fmt.Errorf("%s must be read-only", label)
	}
	if !errors.Is(err, fs.ErrPermission) && !errors.Is(err, syscall.EROFS) {
		return redactFilesystemError("probe "+label, err)
	}
	return nil
}

func validatePathWithoutSymlinks(path string) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return redactFilesystemError("resolve engine path", err)
	}
	if resolved != path {
		return errors.New("engine path must not contain symlinks")
	}
	return nil
}

func validateAbsoluteCleanPath(path string) error {
	if path == "" || path != strings.TrimSpace(path) || !filepath.IsAbs(path) || filepath.Clean(path) != path || !utf8.ValidString(path) {
		return errors.New("path must be absolute and clean")
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func requireCanonicalText(value string) error {
	if value == "" || value != strings.TrimSpace(value) || !utf8.ValidString(value) {
		return errors.New("text must be canonical")
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return errors.New("text must be canonical")
		}
	}
	return nil
}

func redactFilesystemError(operation string, err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%s: %w", operation, fs.ErrNotExist)
	case errors.Is(err, fs.ErrPermission):
		return fmt.Errorf("%s: %w", operation, fs.ErrPermission)
	default:
		return errors.New(operation + " failed")
	}
}

func zeroBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
