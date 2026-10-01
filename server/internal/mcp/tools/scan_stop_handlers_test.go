package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
)

type testScanStopper struct {
	single     ScanStopInput
	batch      BatchScanStopInput
	singleErr  error
	batchErr   error
	batchCalls int
}

func (stopper *testScanStopper) Stop(_ context.Context, input ScanStopInput) (ScanStopOutput, error) {
	stopper.single = input
	if stopper.singleErr != nil {
		return ScanStopOutput{}, stopper.singleErr
	}
	return ScanStopOutput{Scan: input.Scan, Status: "cancelled", RevokedTaskCount: 2}, nil
}

func (stopper *testScanStopper) BatchStop(_ context.Context, input BatchScanStopInput) (BatchScanStopOutput, error) {
	stopper.batch = input
	stopper.batchCalls++
	if stopper.batchErr != nil {
		return BatchScanStopOutput{}, stopper.batchErr
	}
	return BatchScanStopOutput{StoppedCount: 1, SkippedCount: 2, RevokedTaskCount: 3}, nil
}

func TestStopScanHandlerAcceptsOnlyCanonicalScanReferences(t *testing.T) {
	stopper := &testScanStopper{}
	registry := NewRegistry(Dependencies{ScanStopper: stopper})
	for _, name := range []string{"123", "scans/0", "scans/01", "scans/abc", "targets/1", "operations/11111111-1111-1111-1111-111111111111", ""} {
		result, err := registry.stopScan(context.Background(), toolRequest(fmt.Sprintf(`{"scan":%q}`, name)))
		if err != nil {
			t.Fatalf("stop_scan(%q) returned protocol error %v", name, err)
		}
		if !result.IsError {
			t.Fatalf("stop_scan(%q) expected tool failure", name)
		}
		if got := toolFailureCategory(t, result); got != string(mcpErrors.CategoryInvalidInput) {
			t.Fatalf("stop_scan(%q) category = %v, want %s", name, got, mcpErrors.CategoryInvalidInput)
		}
	}
	if stopper.single.Scan != "" {
		t.Fatalf("port received input despite rejection: %+v", stopper.single)
	}
}

func TestStopScanHandlerStopsCanonicalReferenceAndReturnsChainableResult(t *testing.T) {
	stopper := &testScanStopper{}
	registry := NewRegistry(Dependencies{ScanStopper: stopper})
	result, err := registry.stopScan(context.Background(), toolRequest(`{"scan":"scans/12"}`))
	if err != nil || result == nil || result.IsError {
		t.Fatalf("stop_scan failed: err=%v result=%+v", err, result)
	}
	if stopper.single.Scan != "scans/12" {
		t.Fatalf("port input = %+v", stopper.single)
	}
	encoded, marshalErr := json.Marshal(result.StructuredContent)
	if marshalErr != nil {
		t.Fatalf("marshal result: %v", marshalErr)
	}
	var payload struct {
		Scan             string `json:"scan"`
		Status           string `json:"status"`
		RevokedTaskCount int    `json:"revokedTaskCount"`
	}
	if unmarshalErr := json.Unmarshal(encoded, &payload); unmarshalErr != nil {
		t.Fatalf("unmarshal result: %v", unmarshalErr)
	}
	if payload.Scan != "scans/12" || payload.Status != "cancelled" || payload.RevokedTaskCount != 2 {
		t.Fatalf("stop payload = %+v", payload)
	}
}

func TestStopScanHandlerMapsPortFailuresAndMissingDependency(t *testing.T) {
	registry := NewRegistry(Dependencies{ScanStopper: &testScanStopper{singleErr: mcpErrors.ErrCommandFailed}})
	terminal, err := registry.stopScan(context.Background(), toolRequest(`{"scan":"scans/7"}`))
	if err != nil || !terminal.IsError {
		t.Fatalf("terminal-stop expected tool failure: err=%v", err)
	}
	if got := toolFailureCategory(t, terminal); got != string(mcpErrors.CategoryInternal) {
		t.Fatalf("terminal-stop category = %v, want the existing business-failure channel", got)
	}

	// A missing dependency is not an expected tool failure; it follows the
	// shared investigation-tool pattern of surfacing a protocol-level error.
	if _, err := NewRegistry(Dependencies{}).stopScan(context.Background(), toolRequest(`{"scan":"scans/7"}`)); err == nil {
		t.Fatal("no-deps stop_scan must surface a protocol-level internal error")
	}
	if _, err := NewRegistry(Dependencies{}).batchStopScans(context.Background(), toolRequest(`{"scans":["scans/7"]}`)); err == nil {
		t.Fatal("no-deps batch_stop_scans must surface a protocol-level internal error")
	}
}

func TestBatchStopScansHandlerValidatesBoundsBeforeAnyStop(t *testing.T) {
	stopper := &testScanStopper{}
	registry := NewRegistry(Dependencies{ScanStopper: stopper})

	empty, err := registry.batchStopScans(context.Background(), toolRequest(`{"scans":[]}`))
	if err != nil || !empty.IsError {
		t.Fatalf("empty batch expected tool failure: err=%v", err)
	}

	oversized := make([]string, MaxBatchScanStopItems+1)
	for index := range oversized {
		oversized[index] = fmt.Sprintf("scans/%d", index+1)
	}
	arguments, marshalErr := json.Marshal(map[string][]string{"scans": oversized})
	if marshalErr != nil {
		t.Fatalf("marshal oversized batch: %v", marshalErr)
	}
	tooMany, err := registry.batchStopScans(context.Background(), toolRequest(string(arguments)))
	if err != nil || !tooMany.IsError {
		t.Fatalf("oversized batch expected tool failure: err=%v", err)
	}

	duplicate, err := registry.batchStopScans(context.Background(), toolRequest(`{"scans":["scans/1","scans/1"]}`))
	if err != nil || !duplicate.IsError {
		t.Fatalf("duplicate batch expected tool failure: err=%v", err)
	}

	nonCanonical, err := registry.batchStopScans(context.Background(), toolRequest(`{"scans":["scans/1","12"]}`))
	if err != nil || !nonCanonical.IsError {
		t.Fatalf("non-canonical batch expected tool failure: err=%v", err)
	}
	for _, result := range []*mcp.CallToolResult{empty, tooMany, duplicate, nonCanonical} {
		if got := toolFailureCategory(t, result); got != string(mcpErrors.CategoryInvalidInput) {
			t.Fatalf("batch validation category = %v, want %s", got, mcpErrors.CategoryInvalidInput)
		}
	}
	if stopper.batchCalls != 0 || stopper.batch.Scans != nil {
		t.Fatalf("port invoked despite validation failure: calls=%d input=%+v", stopper.batchCalls, stopper.batch)
	}
}

func TestBatchStopScansHandlerReturnsBoundedCountResult(t *testing.T) {
	stopper := &testScanStopper{}
	registry := NewRegistry(Dependencies{ScanStopper: stopper})
	result, err := registry.batchStopScans(context.Background(), toolRequest(`{"scans":["scans/3","scans/4"]}`))
	if err != nil || result == nil || result.IsError {
		t.Fatalf("batch_stop_scans failed: err=%v result=%+v", err, result)
	}
	if len(stopper.batch.Scans) != 2 || stopper.batch.Scans[0] != "scans/3" || stopper.batch.Scans[1] != "scans/4" {
		t.Fatalf("port input = %+v", stopper.batch)
	}
	encoded, marshalErr := json.Marshal(result.StructuredContent)
	if marshalErr != nil {
		t.Fatalf("marshal result: %v", marshalErr)
	}
	var payload map[string]any
	if unmarshalErr := json.Unmarshal(encoded, &payload); unmarshalErr != nil {
		t.Fatalf("unmarshal result: %v", unmarshalErr)
	}
	for _, key := range []string{"stoppedCount", "skippedCount", "revokedTaskCount"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("batch payload missing %q: %s", key, encoded)
		}
	}
	if len(payload) != 3 {
		t.Fatalf("batch payload must carry exactly the three counts, got %s", encoded)
	}
}

func TestBatchStopScansHandlerMapsWholeCallNotFound(t *testing.T) {
	registry := NewRegistry(Dependencies{ScanStopper: &testScanStopper{batchErr: mcpErrors.ErrNotFound}})
	result, err := registry.batchStopScans(context.Background(), toolRequest(`{"scans":["scans/3","scans/404"]}`))
	if err != nil {
		t.Fatalf("unknown-scan batch protocol error: %v", err)
	}
	if got := toolFailureCategory(t, result); got != string(mcpErrors.CategoryNotFound) {
		t.Fatalf("unknown-scan batch category = %v, want %s", got, mcpErrors.CategoryNotFound)
	}
}

func TestScanStopAnnotationsMatchTheApprovedToolHints(t *testing.T) {
	single := scanStopAnnotations()
	if single.ReadOnlyHint || single.IdempotentHint || *single.DestructiveHint {
		t.Fatalf("stop_scan annotations = %+v", single)
	}
	batch := batchScanStopAnnotations()
	if batch.ReadOnlyHint || !batch.IdempotentHint || *batch.DestructiveHint {
		t.Fatalf("batch_stop_scans annotations = %+v", batch)
	}
}

func TestStopScanHandlerPropagatesContextCancellation(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	stopper := &testScanStopper{singleErr: context.Canceled}
	_, err := NewRegistry(Dependencies{ScanStopper: stopper}).stopScan(cancelled, toolRequest(`{"scan":"scans/1"}`))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled stop_scan err = %v, want context.Canceled", err)
	}
}
