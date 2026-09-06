package sdk

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/grpc/codes"
)

const diagnosticReportTimeout = 2 * time.Second

// GeneratedFailureBoundary is the closed set of generated Facade boundaries
// that can refine an otherwise generic Engine handler failure. It is not an
// Engine authoring API and never carries arbitrary error text on the wire.
type GeneratedFailureBoundary uint8

const (
	GeneratedFailureExecutionProjection GeneratedFailureBoundary = iota + 1
	GeneratedFailureHandler
	GeneratedFailureProgressReport
	GeneratedFailureResultEncode
	GeneratedFailureResultSubmit
)

// ResultProtocolFailure is a controlled marker used by an Engine runtime when
// it has already classified a non-empty output stream as having no valid
// typed result rows. The marker carries no dynamic output or error details;
// the Agent may use it only to refine a generic non-zero Engine exit.
type ResultProtocolFailure interface {
	IsResultProtocolFailure()
}

// GeneratedDiagnostics is consumed only by generated Facades. Hand-written
// handlers continue to receive the typed Progress and Results ports rather
// than this adapter-only observation capability.
type GeneratedDiagnostics interface {
	RegisterResultType(string) error
	ObserveResultReceived(string) error
	ObserveResultEncoded(string) error
	WrapFailure(GeneratedFailureBoundary, error) error
}

type generatedDiagnostics struct {
	state *diagnosticAccumulator
}

func (diagnostics *generatedDiagnostics) RegisterResultType(resultType string) error {
	if diagnostics == nil || diagnostics.state == nil {
		return errors.New("generated diagnostic collector is required")
	}
	return diagnostics.state.register(resultType)
}

func (diagnostics *generatedDiagnostics) ObserveResultReceived(resultType string) error {
	if diagnostics == nil || diagnostics.state == nil {
		return errors.New("generated diagnostic collector is required")
	}
	return diagnostics.state.increment(resultType, diagnosticCounterReceived)
}

func (diagnostics *generatedDiagnostics) ObserveResultEncoded(resultType string) error {
	if diagnostics == nil || diagnostics.state == nil {
		return errors.New("generated diagnostic collector is required")
	}
	return diagnostics.state.increment(resultType, diagnosticCounterEncoded)
}

func (diagnostics *generatedDiagnostics) WrapFailure(boundary GeneratedFailureBoundary, err error) error {
	if err == nil {
		return nil
	}
	if !validGeneratedFailureBoundary(boundary) {
		return &generatedBoundaryError{boundary: GeneratedFailureHandler, cause: err}
	}
	var existing *generatedBoundaryError
	if errors.As(err, &existing) {
		return err
	}
	return &generatedBoundaryError{boundary: boundary, cause: err}
}

type generatedBoundaryError struct {
	boundary GeneratedFailureBoundary
	cause    error
}

func (err *generatedBoundaryError) Error() string {
	if err == nil || err.cause == nil {
		return "engine generated boundary failure"
	}
	return err.cause.Error()
}

func (err *generatedBoundaryError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

type diagnosticCounter uint8

const (
	diagnosticCounterReceived diagnosticCounter = iota + 1
	diagnosticCounterEncoded
	diagnosticCounterSubmittedItems
	diagnosticCounterAcknowledgedItems
	diagnosticCounterSubmittedBatches
	diagnosticCounterAcknowledgedBatches
)

type diagnosticWatermark struct {
	resultType          string
	receivedItems       uint64
	encodedItems        uint64
	submittedItems      uint64
	acknowledgedItems   uint64
	submittedBatches    uint64
	acknowledgedBatches uint64
}

// diagnosticAccumulator deliberately uses an array rather than a map: result
// type count is a protocol constant, so neither retained state nor emitted
// terminal data can grow with streamed result volume.
type diagnosticAccumulator struct {
	mu      sync.Mutex
	entries [protocol.DiagnosticResultTypeLimit]diagnosticWatermark
	count   int
	emitted bool
}

func newDiagnosticAccumulator() *diagnosticAccumulator {
	return &diagnosticAccumulator{}
}

func (accumulator *diagnosticAccumulator) register(resultType string) error {
	if err := protocol.ValidateResultTypeSyntax(resultType); err != nil {
		return &Error{Code: codes.InvalidArgument, Reason: protocol.ReasonInvalidReportingRequest}
	}
	if accumulator == nil {
		return errors.New("diagnostic accumulator is required")
	}
	accumulator.mu.Lock()
	defer accumulator.mu.Unlock()
	_, err := accumulator.entryLocked(resultType, true)
	return err
}

func (accumulator *diagnosticAccumulator) increment(resultType string, counter diagnosticCounter) error {
	if accumulator == nil {
		return errors.New("diagnostic accumulator is required")
	}
	accumulator.mu.Lock()
	defer accumulator.mu.Unlock()
	entry, err := accumulator.entryLocked(resultType, false)
	if err != nil {
		return err
	}
	return incrementDiagnosticCounter(entry, counter, 1)
}

func (accumulator *diagnosticAccumulator) recordSubmitted(resultType string, itemCount uint64) error {
	if accumulator == nil || itemCount == 0 {
		return errors.New("diagnostic result batch is invalid")
	}
	accumulator.mu.Lock()
	defer accumulator.mu.Unlock()
	entry, err := accumulator.entryLocked(resultType, false)
	if err != nil {
		return err
	}
	if err := incrementDiagnosticCounter(entry, diagnosticCounterSubmittedItems, itemCount); err != nil {
		return err
	}
	return incrementDiagnosticCounter(entry, diagnosticCounterSubmittedBatches, 1)
}

func (accumulator *diagnosticAccumulator) requireRegistered(resultType string) error {
	if accumulator == nil {
		return errors.New("diagnostic accumulator is required")
	}
	accumulator.mu.Lock()
	defer accumulator.mu.Unlock()
	_, err := accumulator.entryLocked(resultType, false)
	return err
}

func (accumulator *diagnosticAccumulator) recordAcknowledged(resultType string, itemCount uint64) error {
	if accumulator == nil || itemCount == 0 {
		return errors.New("diagnostic result batch is invalid")
	}
	accumulator.mu.Lock()
	defer accumulator.mu.Unlock()
	entry, err := accumulator.entryLocked(resultType, false)
	if err != nil {
		return err
	}
	if err := incrementDiagnosticCounter(entry, diagnosticCounterAcknowledgedItems, itemCount); err != nil {
		return err
	}
	return incrementDiagnosticCounter(entry, diagnosticCounterAcknowledgedBatches, 1)
}

func (accumulator *diagnosticAccumulator) entryLocked(resultType string, create bool) (*diagnosticWatermark, error) {
	for index := 0; index < accumulator.count; index++ {
		if accumulator.entries[index].resultType == resultType {
			return &accumulator.entries[index], nil
		}
	}
	if !create {
		return nil, errors.New("generated diagnostic result type is not registered")
	}
	if accumulator.count >= len(accumulator.entries) {
		return nil, errors.New("Engine diagnostic result type limit exceeded")
	}
	entry := &accumulator.entries[accumulator.count]
	entry.resultType = resultType
	accumulator.count++
	return entry, nil
}

func incrementDiagnosticCounter(entry *diagnosticWatermark, counter diagnosticCounter, amount uint64) error {
	if entry == nil || amount == 0 {
		return errors.New("diagnostic counter increment is invalid")
	}
	var value *uint64
	switch counter {
	case diagnosticCounterReceived:
		value = &entry.receivedItems
	case diagnosticCounterEncoded:
		value = &entry.encodedItems
	case diagnosticCounterSubmittedItems:
		value = &entry.submittedItems
	case diagnosticCounterAcknowledgedItems:
		value = &entry.acknowledgedItems
	case diagnosticCounterSubmittedBatches:
		value = &entry.submittedBatches
	case diagnosticCounterAcknowledgedBatches:
		value = &entry.acknowledgedBatches
	default:
		return errors.New("diagnostic counter is unsupported")
	}
	if ^uint64(0)-*value < amount {
		return errors.New("diagnostic counter overflow")
	}
	*value += amount
	return nil
}

func (accumulator *diagnosticAccumulator) terminalSnapshot(runErr error, signalContext context.Context) (*protocol.EngineTerminalDiagnosticSnapshot, error) {
	if accumulator == nil {
		return nil, errors.New("diagnostic accumulator is required")
	}
	accumulator.mu.Lock()
	defer accumulator.mu.Unlock()
	if accumulator.emitted {
		return nil, errors.New("terminal diagnostics already emitted")
	}
	accumulator.emitted = true
	snapshot := &protocol.EngineTerminalDiagnosticSnapshot{
		CompatibilityRevision: protocol.EngineExecutionDiagnosticsCompatibilityRevision,
		ResultTypeWatermarks:  make([]*protocol.ResultTypeWatermark, accumulator.count),
	}
	for index := 0; index < accumulator.count; index++ {
		entry := accumulator.entries[index]
		snapshot.ResultTypeWatermarks[index] = &protocol.ResultTypeWatermark{
			ResultType:          entry.resultType,
			ReceivedItems:       entry.receivedItems,
			EncodedItems:        entry.encodedItems,
			SubmittedItems:      entry.submittedItems,
			AcknowledgedItems:   entry.acknowledgedItems,
			SubmittedBatches:    entry.submittedBatches,
			AcknowledgedBatches: entry.acknowledgedBatches,
		}
	}
	if runErr != nil && !isSignalCancellation(signalContext, runErr) {
		stage, errorType := generatedFailureClassification(runErr)
		snapshot.FailedStage = stage.Enum()
		snapshot.ErrorType = errorType.Enum()
	}
	if err := protocol.ValidateEngineTerminalDiagnosticSnapshot(snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func isSignalCancellation(signalContext context.Context, err error) bool {
	return signalContext != nil && errors.Is(signalContext.Err(), context.Canceled) && errors.Is(err, context.Canceled)
}

func generatedFailureClassification(err error) (protocol.EngineExecutionFailedStage, protocol.EngineExecutionErrorType) {
	var protocolFailure ResultProtocolFailure
	boundary := GeneratedFailureHandler
	var generated *generatedBoundaryError
	if errors.As(err, &generated) && validGeneratedFailureBoundary(generated.boundary) {
		boundary = generated.boundary
	}
	if boundary == GeneratedFailureHandler && errors.Is(err, context.DeadlineExceeded) {
		return protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_UNKNOWN, protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_TIMEOUT
	}
	if errors.As(err, &protocolFailure) {
		return protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_HANDLER, protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESULT_PROTOCOL_FAILED
	}
	switch boundary {
	case GeneratedFailureExecutionProjection:
		return protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_EXECUTION_PROJECTION, protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_EXECUTION_FAILED
	case GeneratedFailureProgressReport:
		return protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_PROGRESS_REPORT, protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_PROGRESS_REPORT_FAILED
	case GeneratedFailureResultEncode:
		return protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_RESULT_ENCODE, protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESULT_ENCODE_FAILED
	case GeneratedFailureResultSubmit:
		return protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_RESULT_SUBMIT, protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_RESULT_SUBMIT_FAILED
	case GeneratedFailureHandler:
		fallthrough
	default:
		return protocol.EngineExecutionFailedStage_ENGINE_EXECUTION_FAILED_STAGE_HANDLER, protocol.EngineExecutionErrorType_ENGINE_EXECUTION_ERROR_TYPE_HANDLER_FAILED
	}
}

func validGeneratedFailureBoundary(boundary GeneratedFailureBoundary) bool {
	switch boundary {
	case GeneratedFailureExecutionProjection, GeneratedFailureHandler, GeneratedFailureProgressReport, GeneratedFailureResultEncode, GeneratedFailureResultSubmit:
		return true
	default:
		return false
	}
}

func (client *reporter) generatedDiagnostics() (GeneratedDiagnostics, error) {
	if client == nil || client.diagnosticState == nil {
		return nil, errors.New("generated diagnostic collector is required")
	}
	client.mu.RLock()
	closed := client.closed
	client.mu.RUnlock()
	if closed {
		return nil, errors.New("engine reporter is closed")
	}
	return &generatedDiagnostics{state: client.diagnosticState}, nil
}

// establishExecutionDiagnostics is a bootstrap boundary rather than terminal
// evidence transport. A missing acknowledgement means the Engine may be from
// another pre-release revision, so execution must stop before handler code can
// use reporting or input services.
func (client *reporter) establishExecutionDiagnostics(parent context.Context) error {
	if client == nil || client.diagnosticClient == nil {
		return errors.New("Engine execution diagnostics client is required")
	}
	if parent == nil {
		return errors.New("Engine execution diagnostics context is required")
	}
	callContext, cancel := context.WithTimeout(parent, diagnosticReportTimeout)
	defer cancel()
	authenticatedContext, cleanup, err := client.callContext(callContext)
	if err != nil {
		return errors.New("establish Engine execution diagnostics session")
	}
	defer cleanup()
	response, err := client.diagnosticClient.EstablishExecutionDiagnostics(authenticatedContext, &protocol.EstablishExecutionDiagnosticsRequest{
		CompatibilityRevision: protocol.EngineExecutionDiagnosticsCompatibilityRevision,
	})
	if err != nil || response == nil {
		return errors.New("establish Engine execution diagnostics session")
	}
	return nil
}

// reportTerminalDiagnostics is intentionally best-effort. Diagnostic channel
// failure is evidence loss only and must never alter the Engine process result.
func (client *reporter) reportTerminalDiagnostics(runErr error, signalContext context.Context) {
	if client == nil || client.diagnosticClient == nil || client.diagnosticState == nil {
		return
	}
	snapshot, err := client.diagnosticState.terminalSnapshot(runErr, signalContext)
	if err != nil {
		return
	}
	callContext, cancel := context.WithTimeout(context.Background(), diagnosticReportTimeout)
	defer cancel()
	authenticatedContext, cleanup, err := client.callContext(callContext)
	if err != nil {
		return
	}
	defer cleanup()
	response, err := client.diagnosticClient.ReportTerminalDiagnostics(authenticatedContext, &protocol.ReportTerminalDiagnosticsRequest{Snapshot: snapshot})
	if err != nil || response == nil {
		return
	}
}

func diagnosticError(err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: codes.Internal, Reason: protocol.ReasonReportingInternalInvariant}
}
