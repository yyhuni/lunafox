package sdk

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestValidateContextRejectsUnknownAndDuplicateBusinessBindings(t *testing.T) {
	root := t.TempDir()
	contextPath, credentialPath, endpointPath, paths := testRunnerPaths(t, root)
	_ = contextPath
	_ = credentialPath
	_ = endpointPath
	writeReadOnlyFixture(t, filepath.Join(paths.runtimeRoot, "input"), []byte("fixture input"))
	if err := validateContext(validContext(paths.runtimeRoot), paths); err != nil {
		t.Fatalf("validateContext() rejected valid Context: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*protocol.EngineExecutionContext)
	}{
		{
			name: "unknown field",
			mutate: func(snapshot *protocol.EngineExecutionContext) {
				snapshot.ProtoReflect().SetUnknown(protowire.AppendTag(nil, 99, protowire.VarintType))
			},
		},
		{
			name: "disabled section payload",
			mutate: func(snapshot *protocol.EngineExecutionContext) {
				snapshot.Config.Sections[0].Enabled = boolPointer(false)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validContext(paths.runtimeRoot)
			test.mutate(snapshot)
			if err := validateContext(snapshot, paths); err == nil {
				t.Fatal("validateContext() accepted malformed Context")
			}
		})
	}
}

func TestGracefulSignalCancellation(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	operational := errors.New("tool failed")
	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want error
	}{
		{
			name: "signal cancellation with context error",
			ctx:  cancelled,
			err:  context.Canceled,
			want: nil,
		},
		{
			name: "signal cancellation with grpc cancelled error",
			ctx:  cancelled,
			err:  status.Error(codes.Canceled, "reporting canceled"),
			want: nil,
		},
		{
			name: "signal cancellation with SDK cancelled error",
			ctx:  cancelled,
			err:  &Error{Code: codes.Canceled, Reason: protocol.ReasonCallerCanceled},
			want: nil,
		},
		{
			name: "signal cancellation preserves operational failure",
			ctx:  cancelled,
			err:  operational,
			want: operational,
		},
		{
			name: "ordinary cancellation remains an error",
			ctx:  context.Background(),
			err:  context.Canceled,
			want: context.Canceled,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := gracefulSignalCancellation(test.ctx, test.err)
			if test.want == nil {
				if got != nil {
					t.Fatalf("gracefulSignalCancellation() = %v, want nil", got)
				}
				return
			}
			if !errors.Is(got, test.want) {
				t.Fatalf("gracefulSignalCancellation() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestBootstrapFilesRejectFallbacksAndUnsafePaths(t *testing.T) {
	root := shortTempDir(t)
	contextPath, credentialPath, endpointPath, paths := testRunnerPaths(t, root)
	writeReadOnlyFixture(t, filepath.Join(paths.runtimeRoot, "input"), []byte("fixture input"))

	snapshot := validContext(paths.runtimeRoot)
	payload, err := proto.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal Context: %v", err)
	}
	writeReadOnlyFixture(t, contextPath, payload)
	if _, err := loadContext(paths); err != nil {
		t.Fatalf("loadContext() rejected valid binary Context: %v", err)
	}

	writeReadOnlyFixture(t, contextPath, []byte(`{"target":{"type":"domain","value":"example.com"}}`))
	if _, err := loadContext(paths); err == nil {
		t.Fatal("loadContext() accepted ProtoJSON fallback")
	}
	writeReadOnlyFixture(t, contextPath, payload)

	outsideContext := filepath.Join(paths.runtimeRoot, "outside-context")
	writeReadOnlyFixture(t, outsideContext, payload)
	if err := os.Remove(contextPath); err != nil {
		t.Fatalf("remove Context fixture: %v", err)
	}
	if err := os.Symlink(outsideContext, contextPath); err != nil {
		t.Fatalf("symlink Context fixture: %v", err)
	}
	if _, err := loadContext(paths); err == nil {
		t.Fatal("loadContext() accepted a symlinked Context")
	}
	if err := os.Remove(contextPath); err != nil {
		t.Fatalf("remove Context symlink: %v", err)
	}
	writeReadOnlyFixture(t, contextPath, payload)

	writeReadOnlyFixture(t, credentialPath, testCredential())
	credential, err := loadCredential(credentialPath)
	if err != nil {
		t.Fatalf("loadCredential() rejected canonical credential: %v", err)
	}
	zeroBytes(credential)
	writeReadOnlyFixture(t, credentialPath, append(testCredential(), '\n'))
	if _, err := loadCredential(credentialPath); err == nil {
		t.Fatal("loadCredential() accepted a non-canonical credential")
	}

	listener, err := net.Listen("unix", endpointPath)
	if err != nil {
		t.Fatalf("listen on test UDS: %v", err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(endpointPath)
	}()
	if err := validateSocket(endpointPath); err != nil {
		t.Fatalf("validateSocket() rejected filesystem UDS: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close test UDS: %v", err)
	}
	if err := os.Remove(endpointPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove test UDS: %v", err)
	}
	writeReadOnlyFixture(t, endpointPath, []byte("not a socket"))
	if err := validateSocket(endpointPath); err == nil {
		t.Fatal("validateSocket() accepted a regular file")
	}
}

func TestReporterRejectsCancelledAndOverLimitCallsBeforeForwarding(t *testing.T) {
	client := newReporter(validContext(t.TempDir()), &recordingReportingClient{}, testCredential())
	recording := client.reporting.(*recordingReportingClient)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.ReportProgress(cancelled, "started"); !hasProtocolError(err, codes.Canceled, protocol.ReasonCallerCanceled) {
		t.Fatalf("cancelled progress error = %#v", err)
	}

	client.snapshot.Limits.ProgressMessageMaxBytes = 1
	if err := client.ReportProgress(context.Background(), "too large"); !hasProtocolError(err, codes.ResourceExhausted, protocol.ReasonReportingLimitExceeded) {
		t.Fatalf("over-limit progress error = %#v", err)
	}
	client.snapshot.Limits.ResultBatchMaxBytes = 4
	registerGeneratedResultType(t, client, "asset.example.v1")
	sink, err := client.ResultSink("asset.example.v1")
	if err != nil {
		t.Fatalf("ResultSink() error = %v", err)
	}
	items := make(chan []byte, 1)
	items <- []byte(`{"id":1}`)
	close(items)
	if err := sink.Submit(context.Background(), items); !hasProtocolError(err, codes.ResourceExhausted, protocol.ReasonReportingLimitExceeded) {
		t.Fatalf("over-limit item error = %#v", err)
	}
	if len(recording.resultCalls) != 0 {
		t.Fatalf("over-limit item reached reporting client: %d calls", len(recording.resultCalls))
	}
}

func TestResultSinkBatchesInOrderAndWaitsForAcknowledgement(t *testing.T) {
	client := newReporter(validContext(t.TempDir()), &recordingReportingClient{}, testCredential())
	recording := client.reporting.(*recordingReportingClient)
	registerGeneratedResultType(t, client, "asset.example.v1")
	sink, err := client.ResultSink("asset.example.v1")
	if err != nil {
		t.Fatalf("ResultSink() error = %v", err)
	}
	items := make(chan []byte, 3)
	items <- []byte(`{"id":1}`)
	items <- []byte(`{"id":2}`)
	items <- []byte(`{"id":3}`)
	close(items)
	client.snapshot.Limits.ResultBatchMaxItems = 2
	client.snapshot.Limits.ResultBatchMaxBytes = 1024
	if err := sink.Submit(context.Background(), items); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if len(recording.resultCalls) != 2 {
		t.Fatalf("SubmitResultBatch calls = %d, want 2", len(recording.resultCalls))
	}
	if got := string(recording.resultCalls[0].Items[0]) + string(recording.resultCalls[0].Items[1]) + string(recording.resultCalls[1].Items[0]); got != `{"id":1}{"id":2}{"id":3}` {
		t.Fatalf("batch item order = %q", got)
	}
}

func TestResultSinkReplaysOnlyStrictIndeterminateUnavailable(t *testing.T) {
	client := newReporter(validContext(t.TempDir()), &recordingReportingClient{resultErrors: []error{
		status.Error(codes.Unavailable, "connection dropped"), nil,
	}}, testCredential())
	recording := client.reporting.(*recordingReportingClient)
	registerGeneratedResultType(t, client, "asset.example.v1")
	sink, err := client.ResultSink("asset.example.v1")
	if err != nil {
		t.Fatalf("ResultSink() error = %v", err)
	}
	items := make(chan []byte, 1)
	items <- []byte(`{"id":1}`)
	close(items)
	if err := sink.Submit(context.Background(), items); err != nil {
		t.Fatalf("Submit() strict replay error = %v", err)
	}
	if len(recording.resultCalls) != 2 {
		t.Fatalf("strict indeterminate calls = %d, want 2", len(recording.resultCalls))
	}

	client = newReporter(validContext(t.TempDir()), &recordingReportingClient{resultErrors: []error{
		status.Error(codes.InvalidArgument, "application rejected"),
	}}, testCredential())
	recording = client.reporting.(*recordingReportingClient)
	registerGeneratedResultType(t, client, "asset.example.v1")
	sink, err = client.ResultSink("asset.example.v1")
	if err != nil {
		t.Fatalf("ResultSink() error = %v", err)
	}
	items = make(chan []byte, 1)
	items <- []byte(`{"id":1}`)
	close(items)
	err = sink.Submit(context.Background(), items)
	if len(recording.resultCalls) != 1 {
		t.Fatalf("application failure calls = %d, want 1", len(recording.resultCalls))
	}
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != codes.Internal || typed.Reason != protocol.ReasonReportingInternalInvariant {
		t.Fatalf("application error = %#v, want redacted internal error", err)
	}
}

func TestReporterRevokeCancelsInflightAndRejectsFutureCalls(t *testing.T) {
	client := newReporter(validContext(t.TempDir()), &blockingReportingClient{started: make(chan struct{})}, testCredential())
	reporting := client.reporting.(*blockingReportingClient)
	result := make(chan error, 1)
	go func() { result <- client.ReportProgress(context.Background(), "started") }()
	<-reporting.started
	client.revoke()
	err := <-result
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != codes.Canceled || typed.Reason != protocol.ReasonSessionRevoked {
		t.Fatalf("in-flight revoke error = %#v", err)
	}
	if err := client.ReportProgress(context.Background(), "after revoke"); !errors.As(err, &typed) || typed.Reason != protocol.ReasonSessionRevoked {
		t.Fatalf("post-revoke error = %#v", err)
	}
}

func TestMapErrorDoesNotLeakPrivateStatus(t *testing.T) {
	private := status.Error(codes.PermissionDenied, "private Agent path /run/lunafox/socket/engine.sock")
	err := mapError(private)
	if err.Error() == private.Error() {
		t.Fatalf("mapError() leaked raw status: %v", err)
	}
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != codes.Internal || typed.Reason != protocol.ReasonReportingInternalInvariant {
		t.Fatalf("mapError() = %#v", err)
	}
}

func TestReporterEstablishesMatchingDiagnosticsSessionBeforeHandler(t *testing.T) {
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), &recordingReportingClient{}, diagnostics, testCredential())
	if err := client.establishExecutionDiagnostics(context.Background()); err != nil {
		t.Fatalf("establishExecutionDiagnostics() error = %v", err)
	}
	requests := diagnostics.establishmentSnapshot()
	if len(requests) != 1 || requests[0].GetCompatibilityRevision() != protocol.EngineExecutionDiagnosticsCompatibilityRevision {
		t.Fatalf("establishment requests = %#v", requests)
	}

	diagnostics.establishErr = status.Error(codes.FailedPrecondition, "pre-cut Engine")
	if err := client.establishExecutionDiagnostics(context.Background()); err == nil || strings.Contains(err.Error(), "pre-cut Engine") {
		t.Fatalf("failed establishment = %v, want fixed bootstrap error without private status", err)
	}
	if err := newReporter(validContext(t.TempDir()), &recordingReportingClient{}, testCredential()).establishExecutionDiagnostics(context.Background()); err == nil {
		t.Fatal("reporter without diagnostics client established a session")
	}
}

func TestGeneratedDiagnosticsTracksBoundariesAndUploadsOneSnapshot(t *testing.T) {
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), &recordingReportingClient{}, diagnostics, testCredential())
	collector, err := client.GeneratedDiagnostics()
	if err != nil {
		t.Fatalf("GeneratedDiagnostics() error = %v", err)
	}
	const resultType = "asset.example.v1"
	if err := collector.RegisterResultType(resultType); err != nil {
		t.Fatalf("RegisterResultType() error = %v", err)
	}
	for index := 0; index < 2; index++ {
		if err := collector.ObserveResultReceived(resultType); err != nil {
			t.Fatalf("ObserveResultReceived() error = %v", err)
		}
		if err := collector.ObserveResultEncoded(resultType); err != nil {
			t.Fatalf("ObserveResultEncoded() error = %v", err)
		}
	}
	sink, err := client.ResultSink(resultType)
	if err != nil {
		t.Fatalf("ResultSink() error = %v", err)
	}
	items := make(chan []byte, 2)
	items <- []byte(`{"id":1}`)
	items <- []byte(`{"id":2}`)
	close(items)
	if err := sink.Submit(context.Background(), items); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	client.reportTerminalDiagnostics(nil, context.Background())
	client.reportTerminalDiagnostics(nil, context.Background())
	calls := diagnostics.snapshot()
	if len(calls) != 1 {
		t.Fatalf("terminal diagnostics calls = %d, want 1", len(calls))
	}
	snapshot := calls[0].GetSnapshot()
	if snapshot.GetFailedStage() != protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_UNKNOWN || snapshot.FailedStage != nil || snapshot.ErrorType != nil {
		t.Fatalf("successful diagnostic failure fields = %#v", snapshot)
	}
	if len(snapshot.GetResultTypeWatermarks()) != 1 {
		t.Fatalf("result watermarks = %#v", snapshot.GetResultTypeWatermarks())
	}
	watermark := snapshot.GetResultTypeWatermarks()[0]
	if watermark.GetResultType() != resultType || watermark.GetReceivedItems() != 2 || watermark.GetEncodedItems() != 2 || watermark.GetSubmittedItems() != 2 || watermark.GetAcknowledgedItems() != 2 || watermark.GetSubmittedBatches() != 1 || watermark.GetAcknowledgedBatches() != 1 {
		t.Fatalf("watermark = %#v", watermark)
	}
}

func TestGeneratedDiagnosticsCountsLogicalBatchOnlyOnceAcrossTransparentRetry(t *testing.T) {
	reporting := &recordingReportingClient{resultErrors: []error{status.Error(codes.Unavailable, "lost response"), nil}}
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), reporting, diagnostics, testCredential())
	collector, err := client.GeneratedDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	const resultType = "asset.example.v1"
	if err := collector.RegisterResultType(resultType); err != nil {
		t.Fatal(err)
	}
	if err := collector.ObserveResultReceived(resultType); err != nil {
		t.Fatal(err)
	}
	if err := collector.ObserveResultEncoded(resultType); err != nil {
		t.Fatal(err)
	}
	sink, err := client.ResultSink(resultType)
	if err != nil {
		t.Fatal(err)
	}
	items := make(chan []byte, 1)
	items <- []byte(`{"id":1}`)
	close(items)
	if err := sink.Submit(context.Background(), items); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if got := len(reporting.resultCalls); got != 2 {
		t.Fatalf("SubmitResultBatch attempts = %d, want 2", got)
	}
	client.reportTerminalDiagnostics(nil, context.Background())
	watermark := diagnostics.snapshot()[0].GetSnapshot().GetResultTypeWatermarks()[0]
	if watermark.GetSubmittedItems() != 1 || watermark.GetAcknowledgedItems() != 1 || watermark.GetSubmittedBatches() != 1 || watermark.GetAcknowledgedBatches() != 1 {
		t.Fatalf("retry watermark double counted: %#v", watermark)
	}
}

func TestGeneratedDiagnosticsClassifiesBoundaryFailureWithoutRawError(t *testing.T) {
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), &recordingReportingClient{}, diagnostics, testCredential())
	collector, err := client.GeneratedDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	const resultType = "asset.example.v1"
	if err := collector.RegisterResultType(resultType); err != nil {
		t.Fatal(err)
	}
	if err := collector.ObserveResultReceived(resultType); err != nil {
		t.Fatal(err)
	}
	raw := errors.New("private tool path /workspace/secret-token")
	client.reportTerminalDiagnostics(collector.WrapFailure(GeneratedFailureResultEncode, raw), context.Background())
	calls := diagnostics.snapshot()
	if len(calls) != 1 {
		t.Fatalf("terminal diagnostics calls = %d", len(calls))
	}
	snapshot := calls[0].GetSnapshot()
	if snapshot.GetFailedStage() != protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_RESULT_ENCODE || snapshot.GetErrorType() != protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESULT_ENCODE_FAILED {
		t.Fatalf("boundary classification = %#v", snapshot)
	}
	if encoded := snapshot.String(); strings.Contains(encoded, "workspace") || strings.Contains(encoded, "secret-token") {
		t.Fatalf("diagnostic snapshot leaked raw error: %s", encoded)
	}
}

func TestGeneratedDiagnosticsClassifiesResultProtocolMarker(t *testing.T) {
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), &recordingReportingClient{}, diagnostics, testCredential())
	collector, err := client.GeneratedDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	client.reportTerminalDiagnostics(collector.WrapFailure(GeneratedFailureHandler, testResultProtocolFailure{}), context.Background())
	calls := diagnostics.snapshot()
	if len(calls) != 1 {
		t.Fatalf("terminal diagnostics calls = %d", len(calls))
	}
	snapshot := calls[0].GetSnapshot()
	if snapshot.GetFailedStage() != protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_HANDLER || snapshot.GetErrorType() != protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESULT_PROTOCOL_FAILED {
		t.Fatalf("result protocol classification = %#v", snapshot)
	}
}

func TestGeneratedDiagnosticsClassifiesWrappedHandlerDeadlineWithoutRawError(t *testing.T) {
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), &recordingReportingClient{}, diagnostics, testCredential())
	collector, err := client.GeneratedDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	raw := fmt.Errorf("private tool path /workspace/secret-token: %w", context.DeadlineExceeded)
	client.reportTerminalDiagnostics(collector.WrapFailure(GeneratedFailureHandler, raw), context.Background())
	calls := diagnostics.snapshot()
	if len(calls) != 1 {
		t.Fatalf("terminal diagnostics calls = %d", len(calls))
	}
	snapshot := calls[0].GetSnapshot()
	if snapshot.GetFailedStage() != protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_UNKNOWN || snapshot.GetErrorType() != protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_TIMEOUT {
		t.Fatalf("deadline classification = %#v", snapshot)
	}
	if encoded := snapshot.String(); strings.Contains(encoded, "workspace") || strings.Contains(encoded, "secret-token") {
		t.Fatalf("deadline diagnostic snapshot leaked raw error: %s", encoded)
	}
}

type testResultProtocolFailure struct{}

func (testResultProtocolFailure) Error() string { return "private result protocol detail" }

func (testResultProtocolFailure) IsResultProtocolFailure() {}

func TestGeneratedDiagnosticsDoesNotAcknowledgeLostResponse(t *testing.T) {
	reporting := &recordingReportingClient{nilResultResponses: map[int]bool{0: true}}
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), reporting, diagnostics, testCredential())
	collector, err := client.GeneratedDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	const resultType = "asset.example.v1"
	if err := collector.RegisterResultType(resultType); err != nil {
		t.Fatal(err)
	}
	if err := collector.ObserveResultReceived(resultType); err != nil {
		t.Fatal(err)
	}
	if err := collector.ObserveResultEncoded(resultType); err != nil {
		t.Fatal(err)
	}
	sink, err := client.ResultSink(resultType)
	if err != nil {
		t.Fatal(err)
	}
	items := make(chan []byte, 1)
	items <- []byte(`{"id":1}`)
	close(items)
	submitErr := sink.Submit(context.Background(), items)
	if !hasProtocolError(submitErr, codes.Internal, protocol.ReasonReportingInternalInvariant) {
		t.Fatalf("Submit() error = %#v, want missing acknowledgement error", submitErr)
	}

	client.reportTerminalDiagnostics(collector.WrapFailure(GeneratedFailureResultSubmit, submitErr), context.Background())
	snapshot := diagnostics.snapshot()[0].GetSnapshot()
	if snapshot.GetFailedStage() != protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_RESULT_SUBMIT || snapshot.GetErrorType() != protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESULT_SUBMIT_FAILED {
		t.Fatalf("lost-response classification = %#v", snapshot)
	}
	watermark := snapshot.GetResultTypeWatermarks()[0]
	if watermark.GetSubmittedItems() != 1 || watermark.GetAcknowledgedItems() != 0 || watermark.GetSubmittedBatches() != 1 || watermark.GetAcknowledgedBatches() != 0 {
		t.Fatalf("lost response must remain unacknowledged: %#v", watermark)
	}
}

func TestGeneratedDiagnosticsPreservesSignalCancellationWithoutFailureClassification(t *testing.T) {
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), &recordingReportingClient{}, diagnostics, testCredential())
	collector, err := client.GeneratedDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	const resultType = "asset.example.v1"
	if err := collector.RegisterResultType(resultType); err != nil {
		t.Fatal(err)
	}
	sink, err := client.ResultSink(resultType)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sink.Submit(cancelled, make(chan []byte)); !hasProtocolError(err, codes.Canceled, protocol.ReasonCallerCanceled) {
		t.Fatalf("cancelled Submit() error = %#v", err)
	}
	client.reportTerminalDiagnostics(collector.WrapFailure(GeneratedFailureResultSubmit, context.Canceled), cancelled)
	snapshot := diagnostics.snapshot()[0].GetSnapshot()
	if snapshot.FailedStage != nil || snapshot.ErrorType != nil {
		t.Fatalf("signal cancellation must not become an Engine failure: %#v", snapshot)
	}
	if len(snapshot.GetResultTypeWatermarks()) != 1 {
		t.Fatalf("cancelled snapshot watermarks = %#v", snapshot.GetResultTypeWatermarks())
	}
}

func TestGeneratedDiagnosticsRecordsLegalZeroResults(t *testing.T) {
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), &recordingReportingClient{}, diagnostics, testCredential())
	collector, err := client.GeneratedDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	if err := collector.RegisterResultType("asset.example.v1"); err != nil {
		t.Fatal(err)
	}
	client.reportTerminalDiagnostics(nil, context.Background())
	snapshot := diagnostics.snapshot()[0].GetSnapshot()
	if snapshot.FailedStage != nil || snapshot.ErrorType != nil {
		t.Fatalf("zero-result success failure fields = %#v", snapshot)
	}
	if watermarks := snapshot.GetResultTypeWatermarks(); len(watermarks) != 1 || watermarks[0].GetReceivedItems() != 0 || watermarks[0].GetAcknowledgedBatches() != 0 {
		t.Fatalf("legal zero-result watermark = %#v", watermarks)
	}
}

func TestGeneratedDiagnosticsAttributesMultipleResultTypesIndependently(t *testing.T) {
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), &recordingReportingClient{}, diagnostics, testCredential())
	collector, err := client.GeneratedDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	resultTypes := []string{"asset.alpha.v1", "asset.beta.v1"}
	for _, resultType := range resultTypes {
		if err := collector.RegisterResultType(resultType); err != nil {
			t.Fatal(err)
		}
		if err := collector.ObserveResultReceived(resultType); err != nil {
			t.Fatal(err)
		}
		if err := collector.ObserveResultEncoded(resultType); err != nil {
			t.Fatal(err)
		}
		sink, err := client.ResultSink(resultType)
		if err != nil {
			t.Fatal(err)
		}
		items := make(chan []byte, 1)
		items <- []byte(`{"id":1}`)
		close(items)
		if err := sink.Submit(context.Background(), items); err != nil {
			t.Fatal(err)
		}
	}
	client.reportTerminalDiagnostics(nil, context.Background())
	watermarks := diagnostics.snapshot()[0].GetSnapshot().GetResultTypeWatermarks()
	if len(watermarks) != len(resultTypes) {
		t.Fatalf("watermark count = %d, want %d", len(watermarks), len(resultTypes))
	}
	for index, resultType := range resultTypes {
		watermark := watermarks[index]
		if watermark.GetResultType() != resultType || watermark.GetReceivedItems() != 1 || watermark.GetEncodedItems() != 1 || watermark.GetSubmittedItems() != 1 || watermark.GetAcknowledgedItems() != 1 {
			t.Fatalf("watermark[%d] = %#v", index, watermark)
		}
	}
}

func TestDiagnosticAccumulatorRejectsOutOfOrderWatermarks(t *testing.T) {
	accumulator := newDiagnosticAccumulator()
	if err := accumulator.register("asset.example.v1"); err != nil {
		t.Fatal(err)
	}
	if err := accumulator.recordSubmitted("asset.example.v1", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := accumulator.terminalSnapshot(nil, context.Background()); err == nil {
		t.Fatal("terminalSnapshot() accepted submitted items without encoded items")
	}
}

func TestResultSinkRequiresGeneratedResultTypeRegistration(t *testing.T) {
	client := newReporter(validContext(t.TempDir()), &recordingReportingClient{}, testCredential())
	if _, err := client.ResultSink("asset.example.v1"); !hasProtocolError(err, codes.Internal, protocol.ReasonReportingInternalInvariant) {
		t.Fatalf("unregistered ResultSink() error = %#v", err)
	}
}

func TestDiagnosticAccumulatorIsFixedSize(t *testing.T) {
	diagnostics := &recordingDiagnosticsClient{}
	client := newReporterWithDiagnostics(validContext(t.TempDir()), &recordingReportingClient{}, diagnostics, testCredential())
	accumulator := client.diagnosticState
	const resultType = "asset.example.v1"
	if err := accumulator.register(resultType); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 10000; index++ {
		if err := accumulator.increment(resultType, diagnosticCounterReceived); err != nil {
			t.Fatal(err)
		}
	}
	if accumulator.count != 1 || len(accumulator.entries) != protocol.DiagnosticResultTypeLimit {
		t.Fatalf("diagnostic state cardinality = count:%d capacity:%d", accumulator.count, len(accumulator.entries))
	}
	client.reportTerminalDiagnostics(nil, context.Background())
	if calls := diagnostics.snapshot(); len(calls) != 1 || len(calls[0].GetSnapshot().GetResultTypeWatermarks()) != 1 {
		t.Fatalf("result-volume stream emitted unexpected diagnostic events: %#v", calls)
	}
}

type recordingReportingClient struct {
	mu                 sync.Mutex
	resultCalls        []*protocol.SubmitResultBatchRequest
	resultErrors       []error
	nilResultResponses map[int]bool
}

func (client *recordingReportingClient) ReportProgress(context.Context, *protocol.ReportProgressRequest, ...grpc.CallOption) (*protocol.ReportProgressResponse, error) {
	return &protocol.ReportProgressResponse{}, nil
}

func (client *recordingReportingClient) SubmitResultBatch(_ context.Context, request *protocol.SubmitResultBatchRequest, _ ...grpc.CallOption) (*protocol.SubmitResultBatchResponse, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	copyRequest := &protocol.SubmitResultBatchRequest{ResultType: request.GetResultType(), Items: cloneBytes(request.GetItems())}
	client.resultCalls = append(client.resultCalls, copyRequest)
	index := len(client.resultCalls) - 1
	if index < len(client.resultErrors) && client.resultErrors[index] != nil {
		return nil, client.resultErrors[index]
	}
	if client.nilResultResponses[index] {
		return nil, nil
	}
	return &protocol.SubmitResultBatchResponse{}, nil
}

type blockingReportingClient struct {
	started chan struct{}
}

type recordingDiagnosticsClient struct {
	mu             sync.Mutex
	calls          []*protocol.ReportTerminalDiagnosticsRequest
	establishCalls []*protocol.EstablishExecutionDiagnosticsRequest
	err            error
	establishErr   error
}

func (client *recordingDiagnosticsClient) EstablishExecutionDiagnostics(_ context.Context, request *protocol.EstablishExecutionDiagnosticsRequest, _ ...grpc.CallOption) (*protocol.EstablishExecutionDiagnosticsResponse, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.establishCalls = append(client.establishCalls, proto.Clone(request).(*protocol.EstablishExecutionDiagnosticsRequest))
	if client.establishErr != nil {
		return nil, client.establishErr
	}
	return &protocol.EstablishExecutionDiagnosticsResponse{}, nil
}

func (client *recordingDiagnosticsClient) ReportTerminalDiagnostics(_ context.Context, request *protocol.ReportTerminalDiagnosticsRequest, _ ...grpc.CallOption) (*protocol.ReportTerminalDiagnosticsResponse, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.calls = append(client.calls, proto.Clone(request).(*protocol.ReportTerminalDiagnosticsRequest))
	if client.err != nil {
		return nil, client.err
	}
	return &protocol.ReportTerminalDiagnosticsResponse{}, nil
}

func (client *recordingDiagnosticsClient) snapshot() []*protocol.ReportTerminalDiagnosticsRequest {
	client.mu.Lock()
	defer client.mu.Unlock()
	result := make([]*protocol.ReportTerminalDiagnosticsRequest, len(client.calls))
	for index, request := range client.calls {
		result[index] = proto.Clone(request).(*protocol.ReportTerminalDiagnosticsRequest)
	}
	return result
}

func (client *recordingDiagnosticsClient) establishmentSnapshot() []*protocol.EstablishExecutionDiagnosticsRequest {
	client.mu.Lock()
	defer client.mu.Unlock()
	result := make([]*protocol.EstablishExecutionDiagnosticsRequest, len(client.establishCalls))
	for index, request := range client.establishCalls {
		result[index] = proto.Clone(request).(*protocol.EstablishExecutionDiagnosticsRequest)
	}
	return result
}

func (client *blockingReportingClient) ReportProgress(ctx context.Context, _ *protocol.ReportProgressRequest, _ ...grpc.CallOption) (*protocol.ReportProgressResponse, error) {
	close(client.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

func (client *blockingReportingClient) SubmitResultBatch(context.Context, *protocol.SubmitResultBatchRequest, ...grpc.CallOption) (*protocol.SubmitResultBatchResponse, error) {
	return &protocol.SubmitResultBatchResponse{}, nil
}

func validContext(root string) *protocol.EngineExecutionContext {
	return &protocol.EngineExecutionContext{
		CompatibilityRevision: protocol.EngineExecutionDiagnosticsCompatibilityRevision,
		Target:                &protocol.CanonicalTarget{Type: "domain", Value: "example.com"},
		Config: &protocol.EngineExecutionConfig{Sections: []*protocol.ConfigSection{{
			SectionId: "scan", Enabled: boolPointer(true), Params: []*protocol.ConfigValue{{
				ParamKey: "threads", Value: &protocol.ConfigValue_IntegerValue{IntegerValue: 1},
			}},
		}}},
		Limits: &protocol.ExecutionLimits{ProgressMessageMaxBytes: 1024, ResultBatchMaxItems: 16, ResultBatchMaxBytes: 4096},
	}
}

func boolPointer(value bool) *bool { return &value }

func testCredential() []byte { return []byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA") }

func writeReadOnlyFixture(t *testing.T, path string, payload []byte) {
	t.Helper()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove fixture %s: %v", path, err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatalf("chmod fixture %s: %v", path, err)
	}
}

func hasProtocolError(err error, code codes.Code, reason protocol.ErrorReason) bool {
	var typed *Error
	return errors.As(err, &typed) && typed.Code == code && typed.Reason == reason
}

func registerGeneratedResultType(t *testing.T, client *reporter, resultType string) {
	t.Helper()
	collector, err := client.GeneratedDiagnostics()
	if err != nil {
		t.Fatalf("GeneratedDiagnostics() error = %v", err)
	}
	if err := collector.RegisterResultType(resultType); err != nil {
		t.Fatalf("RegisterResultType(%q) error = %v", resultType, err)
	}
}

func testRunnerPaths(t *testing.T, root string) (string, string, string, runnerPaths) {
	t.Helper()
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve test runtime root: %v", err)
	}
	return resolvedRoot + "/context", resolvedRoot + "/credential", resolvedRoot + "/socket", runnerPaths{
		contextPath: resolvedRoot + "/context", credentialPath: resolvedRoot + "/credential", endpointPath: resolvedRoot + "/socket", runtimeRoot: resolvedRoot,
	}
}

func shortTempDir(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "lfsdk-")
	if err != nil {
		t.Fatalf("create short test runtime root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}

var _ protocol.EngineExecutionReportingServiceClient = (*recordingReportingClient)(nil)
var _ protocol.EngineExecutionReportingServiceClient = (*blockingReportingClient)(nil)
var _ protocol.EngineExecutionDiagnosticsServiceClient = (*recordingDiagnosticsClient)(nil)
var _ = metadata.MD{}
