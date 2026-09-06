package agentdata

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	taskprogressv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/taskprogress/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	"github.com/yyhuni/lunafox/contracts/results"
	engineexecution "github.com/yyhuni/lunafox/engine-go/protocol"
	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	pkg "github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func agentBatchIngestTaskResultsRequest(scanID, targetID, taskID int, resultKind string, items []string) *agentdatav1.BatchIngestTaskResultsRequest {
	return &agentdatav1.BatchIngestTaskResultsRequest{
		ResultType: resultKind,
		ItemsJson:  items,
		Task:       resourcenames.Task(scanID, taskID),
		Target:     resourcenames.Target(targetID),
	}
}

func callAgentResultIngest(svc *DataPlaneService, ctx context.Context, req *agentdatav1.BatchIngestTaskResultsRequest) (*agentdatav1.BatchIngestTaskResultsResponse, error) {
	if svc.agentFinder == nil {
		svc.agentFinder = &agentFinderStub{agent: &agentdomain.Agent{ID: 1, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}}
	}
	if svc.sessions == nil {
		svc.sessions = executionArtifactSessionReaderStub{ready: true, session: agentcontrol.ActiveControlSession{
			AgentID: 1, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch,
		}}
	}
	return svc.BatchIngestTaskResults(agentAuthContextFrom(ctx), req)
}

func callAgentTaskProgressLogs(svc *DataPlaneService, ctx context.Context, req *agentdatav1.BatchWriteTaskProgressLogsRequest) (*agentdatav1.BatchWriteTaskProgressLogsResponse, error) {
	if svc.agentFinder == nil {
		svc.agentFinder = &agentFinderStub{agent: &agentdomain.Agent{ID: 1, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}}
	}
	if svc.sessions == nil {
		svc.sessions = executionArtifactSessionReaderStub{ready: true, session: agentcontrol.ActiveControlSession{
			AgentID: 1, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch,
		}}
	}
	return svc.BatchWriteTaskProgressLogs(agentAuthContextFrom(ctx), req)
}

func agentTaskProgressLogsRequest(scanID, taskID int, requestID string, entries []*taskprogressv1.TaskProgressLogEntry) *agentdatav1.BatchWriteTaskProgressLogsRequest {
	return &agentdatav1.BatchWriteTaskProgressLogsRequest{
		RequestId: requestID,
		Entries:   entries,
		Task:      resourcenames.Task(scanID, taskID),
	}
}

func TestAgentDataPlaneSubmitResultBatchRoutesByKind(t *testing.T) {
	ingest := &resultIngestRuntimeStub{}
	taskScopes := validResultTaskScope(12, 34, 101)
	svc := NewDataPlaneService(nil,

		ResultIngestDataPlanes{
			TaskScopes: taskScopes,
			Ingest:     ingest,
		})

	cases := []struct {
		name       string
		resultKind string
		items      []string
		assert     func(t *testing.T)
	}{
		{
			name:       "subdomain",
			resultKind: results.ResultKindAssetSubdomain,
			items:      []string{`{"dnsName":"api.example.com"}`},
			assert: func(t *testing.T) {
				if ingest.last.TaskID != 101 || ingest.last.ScanID != 12 || ingest.last.TargetID != 34 || ingest.last.ResultType != results.ResultKindAssetSubdomain {
					t.Fatalf("unexpected ingest scope: %+v", ingest.last)
				}
				if ingest.last.AgentID != 1 || ingest.last.SessionID != testAgentSessionID || ingest.last.SessionEpoch != testAgentSessionEpoch {
					t.Fatalf("authenticated execution lease was not forwarded: %+v", ingest.last)
				}
				if len(ingest.last.Items) != 1 || string(ingest.last.Items[0]) != `{"dnsName":"api.example.com"}` {
					t.Fatalf("unexpected ingest payload: %q", ingest.last.Items)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(12, 34, 101, tc.resultKind, tc.items))
			if err != nil {
				t.Fatalf("result ingest failed: %v", err)
			}
			if resp == nil || resp.ProtoReflect().Descriptor().Fields().Len() != 0 {
				t.Fatalf("result acknowledgement must be an empty response: %#v", resp)
			}
			tc.assert(t)
		})
	}
	if taskScopes.last.AgentID != 1 || taskScopes.last.SessionID != testAgentSessionID || taskScopes.last.SessionEpoch != testAgentSessionEpoch || taskScopes.last.TaskID != 101 {
		t.Fatalf("authenticated Agent scope was not forwarded: %+v", taskScopes.last)
	}
}

func TestAgentDataPlaneSubmitResultBatchLogsMaterializationSummary(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	previousLogger := pkg.Logger
	previousSugar := pkg.Sugar
	pkg.Logger = logger
	pkg.Sugar = logger.Sugar()
	t.Cleanup(func() {
		pkg.Logger = previousLogger
		pkg.Sugar = previousSugar
	})

	ingest := &resultIngestRuntimeStub{output: resultingestapp.ResultIngestOutcome{ReceivedItems: 2, SnapshotCount: 1, AssetCount: 1}}
	svc := NewDataPlaneService(nil,

		ResultIngestDataPlanes{TaskScopes: validResultTaskScope(12, 34, 101), Ingest: ingest})

	_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(12, 34, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"api.example.com"}`, `{"dnsName":"www.example.com"}`}))
	if err != nil {
		t.Fatalf("result ingest failed: %v", err)
	}

	entries := logs.FilterMessage("result ingest materialized").All()
	if len(entries) != 1 {
		t.Fatalf("expected one result materialization log, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	for _, key := range []string{"task.id", "scan.id", "target.id", "result.type", "result.received_items", "result.scope_filtered_items", "result.unsupported_items", "result.snapshot_count", "result.asset_count"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("expected %s field, got %v", key, fields)
		}
	}
}

func TestAgentDataPlaneSubmitResultBatchErrorMapping(t *testing.T) {
	t.Run("missing authenticated Agent", func(t *testing.T) {
		svc := NewDataPlaneService(nil, ResultIngestDataPlanes{
			TaskScopes: validResultTaskScope(1, 2, 101), Ingest: &resultIngestRuntimeStub{},
		})

		_, err := svc.BatchIngestTaskResults(context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("expected unauthenticated, got=%v", err)
		}
	})

	t.Run("missing process session metadata", func(t *testing.T) {
		ingest := &resultIngestRuntimeStub{}
		svc := NewDataPlaneService(&agentFinderStub{agent: &agentdomain.Agent{ID: 1, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}}, ResultIngestDataPlanes{
			TaskScopes: validResultTaskScope(1, 2, 101), Ingest: ingest,
		})

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token"))
		_, err := svc.BatchIngestTaskResults(ctx, agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.FailedPrecondition || ingest.calls != 0 {
			t.Fatalf("missing process session result = %v, ingest calls=%d", err, ingest.calls)
		}
	})

	t.Run("superseded process session", func(t *testing.T) {
		ingest := &resultIngestRuntimeStub{}
		svc := NewDataPlaneService(&agentFinderStub{agent: &agentdomain.Agent{ID: 1, SessionID: "session-new", SessionEpoch: testAgentSessionEpoch + 1}}, ResultIngestDataPlanes{
			TaskScopes: validResultTaskScope(1, 2, 101), Ingest: ingest,
		})

		_, err := svc.BatchIngestTaskResults(agentAuthContext(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.FailedPrecondition || ingest.calls != 0 {
			t.Fatalf("superseded process result = %v, ingest calls=%d", err, ingest.calls)
		}
	})

	t.Run("stale task session", func(t *testing.T) {
		ingest := &resultIngestRuntimeStub{}
		svc := NewDataPlaneService(nil, ResultIngestDataPlanes{
			TaskScopes: &resultTaskScopeStub{err: errResultTaskSessionMismatch}, Ingest: ingest,
		})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.FailedPrecondition || ingest.calls != 0 {
			t.Fatalf("stale session result = %v, ingest calls=%d", err, ingest.calls)
		}
	})

	t.Run("caller cancellation", func(t *testing.T) {
		ingest := &resultIngestRuntimeStub{returnContextError: true}
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{TaskScopes: validResultTaskScope(1, 2, 101), Ingest: ingest})

		ctx, cancel := context.WithCancel(context.Background())
		ctx = agentAuthContextFrom(ctx)
		cancel()

		_, err := callAgentResultIngest(svc, ctx, agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.Canceled {
			t.Fatalf("expected cancelled, got=%v", err)
		}
		if ingest.ctx != ctx {
			t.Fatal("result ingest did not receive the original caller context")
		}
	})

	t.Run("caller deadline", func(t *testing.T) {
		ingest := &resultIngestRuntimeStub{returnContextError: true}
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{TaskScopes: validResultTaskScope(1, 2, 101), Ingest: ingest})

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		ctx = agentAuthContextFrom(ctx)
		defer cancel()

		_, err := callAgentResultIngest(svc, ctx, agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.DeadlineExceeded {
			t.Fatalf("expected deadline exceeded, got=%v", err)
		}
		if ingest.ctx != ctx {
			t.Fatal("result ingest did not receive the original caller context")
		}
	})

	t.Run("nil request", func(t *testing.T) {
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{TaskScopes: validResultTaskScope(1, 2, 101), Ingest: &resultIngestRuntimeStub{}})

		_, err := callAgentResultIngest(svc, context.Background(), nil)
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
	})

	t.Run("missing task scope", func(t *testing.T) {
		ingest := &resultIngestRuntimeStub{}
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{TaskScopes: validResultTaskScope(1, 2, 101), Ingest: ingest})

		_, err := callAgentResultIngest(svc, context.Background(), &agentdatav1.BatchIngestTaskResultsRequest{
			ResultType: results.ResultKindAssetSubdomain,
			ItemsJson:  []string{`{"dnsName":"a.example.com"}`},
			Target:     resourcenames.Target(2),
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
		if ingest.calls != 0 {
			t.Fatalf("result ingest backend must not be called without task scope, got %d calls", ingest.calls)
		}
	})

	t.Run("missing task scope dependency", func(t *testing.T) {
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{Ingest: &resultIngestRuntimeStub{}})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.Unimplemented {
			t.Fatalf("expected unimplemented, got=%v", err)
		}
	})

	for _, test := range []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "task scope cancelled", err: context.Canceled, want: codes.Canceled},
		{name: "task scope deadline", err: context.DeadlineExceeded, want: codes.DeadlineExceeded},
		{name: "task scope unavailable", err: errors.New("database unavailable"), want: codes.Unavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := NewDataPlaneService(nil, ResultIngestDataPlanes{
				TaskScopes: &resultTaskScopeStub{err: test.err}, Ingest: &resultIngestRuntimeStub{},
			})

			_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
			if status.Code(err) != test.want {
				t.Fatalf("expected %s, got=%v", test.want, err)
			}
		})
	}

	t.Run("task scope mismatch", func(t *testing.T) {
		ingest := &resultIngestRuntimeStub{}
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{
				TaskScopes: validResultTaskScope(9, 2, 101),
				Ingest:     ingest,
			})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected permission denied, got=%v", err)
		}
		if ingest.calls != 0 {
			t.Fatalf("result ingest backend must not be called on task scope mismatch, got %d calls", ingest.calls)
		}
	})

	t.Run("task not found", func(t *testing.T) {
		ingest := &resultIngestRuntimeStub{}
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{
				TaskScopes: &resultTaskScopeStub{},
				Ingest:     ingest,
			})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 404, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.NotFound {
			t.Fatalf("expected not found, got=%v", err)
		}
		if ingest.calls != 0 {
			t.Fatalf("result ingest backend must not be called when task is not found, got %d calls", ingest.calls)
		}
	})

	t.Run("unsupported result kind", func(t *testing.T) {
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{
				TaskScopes: validResultTaskScope(1, 2, 101),
				Ingest:     &resultIngestRuntimeStub{err: resultingestapp.ErrUnsupportedResultType},
			})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, "directory", []string{`{"url":"https://example.com"}`}))
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{
				TaskScopes: validResultTaskScope(1, 2, 101),
				Ingest:     &resultIngestRuntimeStub{err: resultingestapp.ErrInvalidResultItems},
			})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":`}))
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
	})

	t.Run("result scope is not authorized", func(t *testing.T) {
		svc := NewDataPlaneService(nil,
			ResultIngestDataPlanes{
				TaskScopes: validResultTaskScope(1, 2, 101),
				Ingest:     &resultIngestRuntimeStub{err: resultingestapp.ErrResultNotAuthorized},
			})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"outside.example.net"}`}))
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected permission denied, got=%v", err)
		}
	})

	t.Run("final execution fence rejected", func(t *testing.T) {
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{
				TaskScopes: validResultTaskScope(1, 2, 101),
				Ingest:     &resultIngestRuntimeStub{err: resultingestapp.ErrResultExecutionFenceRejected},
			})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("expected failed precondition, got=%v", err)
		}
	})

	t.Run("scan not found", func(t *testing.T) {
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{
				TaskScopes: validResultTaskScope(1, 2, 101),
				Ingest:     &resultIngestRuntimeStub{err: snapshotapp.ErrScanNotFoundForSnapshot},
			})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.NotFound {
			t.Fatalf("expected not found, got=%v", err)
		}
	})

	t.Run("target mismatch", func(t *testing.T) {
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{
				TaskScopes: validResultTaskScope(1, 2, 101),
				Ingest:     &resultIngestRuntimeStub{err: snapshotapp.ErrTargetMismatch},
			})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
	})

	t.Run("invalid target type", func(t *testing.T) {
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{
				TaskScopes: validResultTaskScope(1, 2, 101),
				Ingest:     &resultIngestRuntimeStub{err: snapshotapp.ErrInvalidTargetType},
			})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		svc := NewDataPlaneService(nil,

			ResultIngestDataPlanes{
				TaskScopes: validResultTaskScope(1, 2, 101),
				Ingest:     &resultIngestRuntimeStub{err: errors.New("boom")},
			})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal, got=%v", err)
		}
	})

	t.Run("missing batch deps", func(t *testing.T) {
		svc := NewDataPlaneService(nil, ResultIngestDataPlanes{
			TaskScopes: validResultTaskScope(1, 2, 101),
		})

		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"a.example.com"}`}))
		if status.Code(err) != codes.Unimplemented {
			t.Fatalf("expected unimplemented, got=%v", err)
		}
	})
}

func TestAgentDataPlaneSubmitResultBatchRejectsResourceLimits(t *testing.T) {
	tests := []struct {
		name  string
		items []string
	}{
		{
			name:  "too many items",
			items: make([]string, maxResultBatchItems+1),
		},
		{
			name:  "total payload too large",
			items: []string{strings.Repeat("x", maxResultBatchJSONBytes+1)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for index := range tc.items {
				if tc.items[index] == "" {
					tc.items[index] = `{"dnsName":"api.example.com"}`
				}
			}
			ingest := &resultIngestRuntimeStub{}
			svc := NewDataPlaneService(nil,

				ResultIngestDataPlanes{TaskScopes: validResultTaskScope(1, 2, 101), Ingest: ingest})

			_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, tc.items))
			if status.Code(err) != codes.ResourceExhausted {
				t.Fatalf("expected resource exhausted, got=%v", err)
			}
			if ingest.calls != 0 {
				t.Fatalf("result ingest backend must not be called for oversized request, got %d calls", ingest.calls)
			}
		})
	}
}

func TestAgentDataPlaneResultIngressLimitsMatchEngineProtocol(t *testing.T) {
	if maxResultBatchItems != int(engineexecution.ResultBatchMaxItemsCeiling) || maxResultBatchJSONBytes != int(engineexecution.ResultBatchMaxBytesCeiling) {
		t.Fatalf("result ingress limits = items:%d bytes:%d; want Engine Protocol ceilings", maxResultBatchItems, maxResultBatchJSONBytes)
	}
}

func TestAgentDataPlaneWriteTaskProgressLogsPersistsValidBatch(t *testing.T) {
	emittedAt := time.Date(2026, 4, 21, 8, 30, 0, 0, time.UTC)
	logs := &taskProgressLogRuntimeStub{accepted: 1, duplicates: 1}
	svc := NewDataPlaneService(nil, ResultIngestDataPlanes{})
	svc.taskProgressLogs = logs

	resp, err := callAgentTaskProgressLogs(svc, context.Background(), agentTaskProgressLogsRequest(7, 101, "request-1", []*taskprogressv1.TaskProgressLogEntry{
		{Sequence: 1, Level: taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_INFO, Content: "started", EmittedAt: timestamppb.New(emittedAt)},
		{Sequence: 2, Level: taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_WARNING, Content: "already stored", EmittedAt: timestamppb.New(emittedAt.Add(time.Second))},
	}))
	if err != nil {
		t.Fatalf("write task progress logs failed: %v", err)
	}
	if resp.GetSummary().GetAcceptedItems() != 1 || resp.GetSummary().GetDuplicateItems() != 1 || resp.GetSummary().GetTotalItems() != 2 {
		t.Fatalf("unexpected response summary: %+v", resp.GetSummary())
	}
	if logs.last.ScanID != 7 || logs.last.TaskID != 101 || logs.last.AgentID != 1 || logs.last.SessionID != testAgentSessionID || logs.last.SessionEpoch != testAgentSessionEpoch || logs.last.RequestID != "request-1" {
		t.Fatalf("unexpected task progress log batch identity: %+v", logs.last)
	}
	if len(logs.last.Entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(logs.last.Entries))
	}
	if logs.last.Entries[0].Level != "info" || logs.last.Entries[0].Content != "started" || !logs.last.Entries[0].EmittedAt.Equal(emittedAt) {
		t.Fatalf("unexpected first log entry: %+v", logs.last.Entries[0])
	}
}

func TestAgentDataPlaneWriteTaskProgressLogsRequiresCurrentAgentSession(t *testing.T) {
	validRequest := func() *agentdatav1.BatchWriteTaskProgressLogsRequest {
		return agentTaskProgressLogsRequest(7, 101, "request-1", []*taskprogressv1.TaskProgressLogEntry{{
			Sequence: 1, Level: taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_INFO, Content: "started", EmittedAt: timestamppb.Now(),
		}})
	}
	newService := func(sessions ExecutionArtifactSessionReader) *DataPlaneService {
		return NewDataPlaneService(
			&agentFinderStub{agent: &agentdomain.Agent{ID: 1, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}},
			ResultIngestDataPlanes{}).
			WithTaskProgressLogDataPlane(&taskProgressLogRuntimeStub{}).WithAgentSessionReader(sessions)
	}
	tests := []struct {
		name     string
		ctx      context.Context
		sessions ExecutionArtifactSessionReader
		want     codes.Code
	}{
		{name: "missing authentication", ctx: context.Background(), sessions: executionArtifactSessionReaderStub{ready: true}, want: codes.Unauthenticated},
		{name: "wrong process session", ctx: agentSessionAuthContext("session-other", testAgentSessionEpoch), sessions: executionArtifactSessionReaderStub{ready: true}, want: codes.FailedPrecondition},
		{name: "wrong epoch", ctx: agentSessionAuthContext(testAgentSessionID, testAgentSessionEpoch+1), sessions: executionArtifactSessionReaderStub{ready: true}, want: codes.FailedPrecondition},
		{name: "expired or unavailable lease", ctx: agentAuthContext(), sessions: executionArtifactSessionReaderStub{}, want: codes.FailedPrecondition},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newService(test.sessions).BatchWriteTaskProgressLogs(test.ctx, validRequest())
			if status.Code(err) != test.want {
				t.Fatalf("BatchWriteTaskProgressLogs() code = %s, want %s: %v", status.Code(err), test.want, err)
			}
		})
	}
}

func TestAgentDataPlaneWriteTaskProgressLogsAllowsCurrentDetachedLease(t *testing.T) {
	registry := agentcontrol.NewActiveSessionRegistry()
	registered, _ := registry.Register(1, 77, "session-detached")
	if _, ok := registry.MarkReadyIfCurrent(1, registered.SessionID, registered.SessionEpoch, registered.StreamID); !ok {
		t.Fatal("mark session ready")
	}
	if !registry.DetachStreamIfCurrent(1, registered.SessionID, registered.SessionEpoch, registered.StreamID) {
		t.Fatal("detach current session")
	}
	svc := NewDataPlaneService(
		&agentFinderStub{agent: &agentdomain.Agent{ID: 1, SessionID: registered.SessionID, SessionEpoch: registered.SessionEpoch}},
		ResultIngestDataPlanes{}).
		WithTaskProgressLogDataPlane(&taskProgressLogRuntimeStub{}).WithAgentSessionReader(registry)

	_, err := svc.BatchWriteTaskProgressLogs(agentSessionAuthContext(registered.SessionID, registered.SessionEpoch), agentTaskProgressLogsRequest(7, 101, "request-1", []*taskprogressv1.TaskProgressLogEntry{{
		Sequence: 1, Level: taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_INFO, Content: "started", EmittedAt: timestamppb.Now(),
	}}))
	if err != nil {
		t.Fatalf("detached current lease was rejected: %v", err)
	}
}

func TestAgentDataPlaneWriteTaskProgressLogsValidation(t *testing.T) {
	emittedAt := timestamppb.Now()
	baseReq := func() *agentdatav1.BatchWriteTaskProgressLogsRequest {
		return agentTaskProgressLogsRequest(7, 101, "request-1", []*taskprogressv1.TaskProgressLogEntry{
			{Sequence: 1, Level: taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_INFO, Content: "started", EmittedAt: emittedAt},
		})
	}
	longContent := make([]byte, maxTaskProgressLogContentBytes+1)
	for i := range longContent {
		longContent[i] = 'x'
	}
	tooManyEntries := make([]*taskprogressv1.TaskProgressLogEntry, maxTaskProgressLogEntries+1)
	for index := range tooManyEntries {
		tooManyEntries[index] = &taskprogressv1.TaskProgressLogEntry{
			Sequence:  int64(index + 1),
			Level:     taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_INFO,
			Content:   "line",
			EmittedAt: emittedAt,
		}
	}
	tooLargeBatchEntries := make([]*taskprogressv1.TaskProgressLogEntry, maxTaskProgressLogBatchContentBytes/maxTaskProgressLogContentBytes+1)
	for index := range tooLargeBatchEntries {
		tooLargeBatchEntries[index] = &taskprogressv1.TaskProgressLogEntry{
			Sequence:  int64(index + 1),
			Level:     taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_INFO,
			Content:   strings.Repeat("x", maxTaskProgressLogContentBytes),
			EmittedAt: emittedAt,
		}
	}

	cases := []struct {
		name   string
		mutate func(*agentdatav1.BatchWriteTaskProgressLogsRequest)
		want   codes.Code
	}{
		{name: "nil request", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) {
			*req = agentdatav1.BatchWriteTaskProgressLogsRequest{}
		}, want: codes.InvalidArgument},
		{name: "missing task name", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) { req.Task = "" }, want: codes.InvalidArgument},
		{name: "invalid task name", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) { req.Task = "tasks/101" }, want: codes.InvalidArgument},
		{name: "missing request", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) { req.RequestId = "" }, want: codes.InvalidArgument},
		{name: "empty entries", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) { req.Entries = nil }, want: codes.InvalidArgument},
		{name: "too many entries", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) { req.Entries = tooManyEntries }, want: codes.ResourceExhausted},
		{name: "batch content too large", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) { req.Entries = tooLargeBatchEntries }, want: codes.ResourceExhausted},
		{name: "missing sequence", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) { req.Entries[0].Sequence = 0 }, want: codes.InvalidArgument},
		{name: "unsupported level", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) {
			req.Entries[0].Level = taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_UNSPECIFIED
		}, want: codes.InvalidArgument},
		{name: "missing content", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) { req.Entries[0].Content = "" }, want: codes.InvalidArgument},
		{name: "content too large", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) { req.Entries[0].Content = string(longContent) }, want: codes.ResourceExhausted},
		{name: "missing emitted at", mutate: func(req *agentdatav1.BatchWriteTaskProgressLogsRequest) { req.Entries[0].EmittedAt = nil }, want: codes.InvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewDataPlaneService(nil, ResultIngestDataPlanes{})
			svc.taskProgressLogs = &taskProgressLogRuntimeStub{}
			req := baseReq()
			tc.mutate(req)
			if tc.name == "nil request" {
				req = nil
			}
			_, err := callAgentTaskProgressLogs(svc, context.Background(), req)
			if status.Code(err) != tc.want {
				t.Fatalf("expected %s, got %v", tc.want, err)
			}
		})
	}
}

func TestAgentDataPlaneWriteTaskProgressLogsErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "missing dependency", err: nil, want: codes.Unimplemented},
		{name: "scan not found", err: scanapp.ErrScanNotFound, want: codes.NotFound},
		{name: "task not found", err: scanapp.ErrScanTaskNotFound, want: codes.NotFound},
		{name: "task scan mismatch", err: scanapp.ErrScanTaskNotOwned, want: codes.PermissionDenied},
		{name: "task owner mismatch", err: errTaskProgressLogOwnershipMismatch, want: codes.PermissionDenied},
		{name: "task not running", err: errTaskProgressLogNotRunning, want: codes.FailedPrecondition},
		{name: "task session mismatch", err: errTaskProgressLogSessionMismatch, want: codes.FailedPrecondition},
		{name: "cancelled", err: context.Canceled, want: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, want: codes.DeadlineExceeded},
		{name: "unavailable", err: errors.New("boom"), want: codes.Unavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewDataPlaneService(nil, ResultIngestDataPlanes{})
			if tc.name != "missing dependency" {
				svc.taskProgressLogs = &taskProgressLogRuntimeStub{err: tc.err}
			}
			_, err := callAgentTaskProgressLogs(svc, context.Background(), agentTaskProgressLogsRequest(7, 101, "request-1", []*taskprogressv1.TaskProgressLogEntry{
				{Sequence: 1, Level: taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_ERROR, Content: "failed", EmittedAt: timestamppb.Now()},
			}))
			if status.Code(err) != tc.want {
				t.Fatalf("expected %s, got %v", tc.want, err)
			}
		})
	}
}

type taskProgressLogRuntimeStub struct {
	accepted   int
	duplicates int
	err        error
	last       TaskProgressLogBatch
}

func (s *taskProgressLogRuntimeStub) WriteTaskProgressLogs(_ context.Context, batch TaskProgressLogBatch) (int, int, error) {
	s.last = batch
	if s.err != nil {
		return 0, 0, s.err
	}
	return s.accepted, s.duplicates, nil
}

type resultTaskScopeStub struct {
	scopes map[int]ResultTaskScope
	err    error
	last   ResultTaskScopeRequest
}

func validResultTaskScope(scanID, targetID, taskID int) *resultTaskScopeStub {
	return &resultTaskScopeStub{
		scopes: map[int]ResultTaskScope{
			taskID: {TaskID: taskID, ScanID: scanID, TargetID: targetID},
		},
	}
}

func (s *resultTaskScopeStub) GetResultTaskScope(_ context.Context, request ResultTaskScopeRequest) (*ResultTaskScope, error) {
	s.last = request
	if s.err != nil {
		return nil, s.err
	}
	scope, ok := s.scopes[request.TaskID]
	if !ok {
		return nil, scanapp.ErrScanTaskNotFound
	}
	return &scope, nil
}

type resultIngestRuntimeStub struct {
	last               resultingestapp.ResultIngestCommand
	output             resultingestapp.ResultIngestOutcome
	err                error
	calls              int
	ctx                context.Context
	returnContextError bool
}

func (s *resultIngestRuntimeStub) Ingest(ctx context.Context, input resultingestapp.ResultIngestCommand) (resultingestapp.ResultIngestOutcome, error) {
	s.calls++
	s.ctx = ctx
	s.last = input
	if s.returnContextError && ctx.Err() != nil {
		return s.output, ctx.Err()
	}
	if s.err != nil {
		return s.output, s.err
	}
	if s.output.ReceivedItems == 0 {
		s.output.ReceivedItems = len(input.Items)
	}
	return s.output, nil
}

func agentAuthContext() context.Context {
	return agentAuthContextFrom(context.Background())
}

func agentSessionAuthContext(sessionID string, sessionEpoch int64) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token",
		grpcauth.AgentSessionIDMetadataKey, sessionID,
		grpcauth.AgentSessionEpochMetadataKey, strconv.FormatInt(sessionEpoch, 10),
	))
}

const (
	testAgentSessionID    = "session-42"
	testAgentSessionEpoch = int64(11)
)

func agentAuthContextFrom(ctx context.Context) context.Context {
	if existing, ok := metadata.FromIncomingContext(ctx); ok {
		if len(existing.Get(grpcauth.AgentAuthenticationTokenMetadataKey)) == 1 &&
			len(existing.Get(grpcauth.AgentSessionIDMetadataKey)) == 1 &&
			len(existing.Get(grpcauth.AgentSessionEpochMetadataKey)) == 1 {
			return ctx
		}
		merged := existing.Copy()
		if len(merged.Get(grpcauth.AgentAuthenticationTokenMetadataKey)) == 0 {
			merged.Append(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token")
		}
		if len(merged.Get(grpcauth.AgentSessionIDMetadataKey)) == 0 {
			merged.Append(grpcauth.AgentSessionIDMetadataKey, testAgentSessionID)
		}
		if len(merged.Get(grpcauth.AgentSessionEpochMetadataKey)) == 0 {
			merged.Append(grpcauth.AgentSessionEpochMetadataKey, "11")
		}
		return metadata.NewIncomingContext(ctx, merged)
	}
	return metadata.NewIncomingContext(ctx, metadata.Pairs(
		grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token",
		grpcauth.AgentSessionIDMetadataKey, testAgentSessionID,
		grpcauth.AgentSessionEpochMetadataKey, "11",
	))
}

func TestAgentDataPlaneSubmitResultBatchValidationMessagesUseCamelCase(t *testing.T) {
	svc := NewDataPlaneService(nil,

		ResultIngestDataPlanes{TaskScopes: validResultTaskScope(1, 2, 101), Ingest: &resultIngestRuntimeStub{}})

	t.Run("missing task resource", func(t *testing.T) {
		_, err := callAgentResultIngest(svc, context.Background(), &agentdatav1.BatchIngestTaskResultsRequest{
			ResultType: results.ResultKindAssetSubdomain,
			ItemsJson:  []string{`{"dnsName":"a.example.com"}`},
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
		if got := status.Convert(err).Message(); !strings.Contains(got, "task name") {
			t.Fatalf("expected task resource validation message, got %q", got)
		}
	})

	t.Run("empty items", func(t *testing.T) {
		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, nil))
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
		if got := status.Convert(err).Message(); got != "itemsJson must not be empty" {
			t.Fatalf("expected camelCase validation message, got %q", got)
		}
	})

	t.Run("missing target resource", func(t *testing.T) {
		_, err := callAgentResultIngest(svc, context.Background(), &agentdatav1.BatchIngestTaskResultsRequest{
			ResultType: results.ResultKindAssetSubdomain,
			ItemsJson:  []string{`{"dnsName":"a.example.com"}`},
			Task:       resourcenames.Task(1, 101),
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
		if got := status.Convert(err).Message(); !strings.Contains(got, "targets/{resource}") {
			t.Fatalf("expected target resource validation message, got %q", got)
		}
	})

	t.Run("invalid payload", func(t *testing.T) {
		svc.resultIngest.Ingest = &resultIngestRuntimeStub{err: resultingestapp.ErrInvalidResultItems}
		_, err := callAgentResultIngest(svc, context.Background(), agentBatchIngestTaskResultsRequest(1, 2, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":`}))
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
		if got := status.Convert(err).Message(); !strings.Contains(got, resultingestapp.ErrInvalidResultItems.Error()) {
			t.Fatalf("expected invalid result items message, got %q", got)
		}
	})
}

type agentFinderStub struct {
	agent                   *agentdomain.Agent
	err                     error
	lastAuthenticationToken string
}

func (stub *agentFinderStub) FindByAuthenticationToken(_ context.Context, authenticationToken string) (*agentdomain.Agent, error) {
	stub.lastAuthenticationToken = authenticationToken
	return stub.agent, stub.err
}
