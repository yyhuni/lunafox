package agentdata

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	taskprogressv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/taskprogress/v1"
	"github.com/yyhuni/lunafox/contracts/results"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type taskRuntimeLogRecorder struct {
	events []scanapp.TaskRuntimeLogEvent
}

func (recorder *taskRuntimeLogRecorder) AppendTaskRuntimeLog(_ context.Context, event scanapp.TaskRuntimeLogEvent) {
	recorder.events = append(recorder.events, event)
}

func (recorder *taskRuntimeLogRecorder) AppendTaskRuntimeLogs(ctx context.Context, events []scanapp.TaskRuntimeLogEvent) {
	for _, event := range events {
		recorder.AppendTaskRuntimeLog(ctx, event)
	}
}

type runtimeLogWriteStoreStub struct {
	requests []*scanapp.TaskProgressLogBatchCreateRequest
	err      error
}

type recoveringProgressLogService struct {
	progressLogApplicationStub
	err error
}

func (service *recoveringProgressLogService) BatchCreateTaskProgressLogs(_ context.Context, request *scanapp.TaskProgressLogBatchCreateRequest) (int, int, error) {
	service.calls++
	service.request = request
	if service.calls == 1 {
		return 0, 0, service.err
	}
	return len(request.Items), 0, nil
}

func (service *recoveringProgressLogService) BatchCreateTaskRuntimeLogs(ctx context.Context, request *scanapp.TaskProgressLogBatchCreateRequest) (int, int, error) {
	return service.BatchCreateTaskProgressLogs(ctx, request)
}

func (store *runtimeLogWriteStoreStub) BatchCreateTaskRuntimeLogs(_ context.Context, request *scanapp.TaskProgressLogBatchCreateRequest) (int, int, error) {
	store.requests = append(store.requests, request)
	return len(request.Items), 0, store.err
}

func TestResultRuntimeLogsRequireTaskScopeAndPreserveRawErrors(t *testing.T) {
	recorder := &taskRuntimeLogRecorder{}
	raw := errors.New("write /srv/private/results failed: token=raw-canary\noriginal database detail")
	ingest := &resultIngestRuntimeStub{err: raw}
	svc := NewDataPlaneService(nil, ResultIngestDataPlanes{TaskScopes: validResultTaskScope(7, 34, 101), Ingest: ingest}).WithTaskRuntimeLogAppender(recorder)
	req := agentBatchIngestTaskResultsRequest(8, 34, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"api.example.com"}`})
	if _, err := callAgentResultIngest(svc, context.Background(), req); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("scope mismatch = %v", err)
	}
	if len(recorder.events) != 0 {
		t.Fatal("unowned result request injected a runtime log")
	}
	req.Task = "scans/7/tasks/101"
	if _, err := callAgentResultIngest(svc, context.Background(), req); err == nil {
		t.Fatal("failed ingestion was acknowledged")
	}
	if len(recorder.events) != 1 || recorder.events[0].ScanID != 7 || recorder.events[0].TaskID != 101 || !strings.Contains(recorder.events[0].Content, raw.Error()) {
		t.Fatalf("raw result error was not retained: %#v", recorder.events)
	}
}

func TestResultRuntimeLogFailureDoesNotChangeStatusOnlyAcknowledgement(t *testing.T) {
	store := &runtimeLogWriteStoreStub{err: errors.New("log DB unavailable")}
	ingest := &resultIngestRuntimeStub{}
	svc := NewDataPlaneService(nil, ResultIngestDataPlanes{TaskScopes: validResultTaskScope(7, 34, 101), Ingest: ingest}).WithTaskRuntimeLogAppender(scanapp.NewTaskRuntimeLogAppender(store))
	req := agentBatchIngestTaskResultsRequest(7, 34, 101, results.ResultKindAssetSubdomain, []string{`{"dnsName":"api.example.com"}`})
	for range 2 {
		resp, err := callAgentResultIngest(svc, context.Background(), req)
		if err != nil || resp == nil || resp.ProtoReflect().Descriptor().Fields().Len() != 0 {
			t.Fatalf("runtime log failure changed result acknowledgement: %#v %v", resp, err)
		}
	}
	if len(store.requests) != 2 || store.requests[0].RequestID != store.requests[1].RequestID {
		t.Fatalf("result replay log identity changed: %#v", store.requests)
	}
}

func TestTaskProgressRawWhitespaceChunkRoundTrip(t *testing.T) {
	for _, raw := range []string{" \t\n\r", strings.Repeat(" ", maxTaskProgressLogContentBytes), "\x1b[31moriginal /workspace/path token=raw-canary\n"} {
		logs := &progressLogApplicationStub{}
		agentID, sessionID, epoch := 17, "session-17", int64(23)
		plane := NewTaskProgressLogDataPlane(logs, &progressTaskRepositoryStub{task: &scanapp.ScanTaskRecord{
			ID: 101, ScanID: 7, Status: "running", AgentID: &agentID, AssignedAgentID: &agentID, AssignedSessionID: &sessionID, AssignedSessionEpoch: &epoch,
		}})
		req := agentTaskProgressLogsRequest(7, 101, "raw-chunk", []*taskprogressv1.TaskProgressLogEntry{{
			Sequence: 1, Level: taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_INFO, Content: raw, EmittedAt: timestamppb.New(time.Now().UTC()),
		}})
		batch, err := taskProgressLogBatchFromProto(req)
		if err != nil {
			t.Fatalf("raw chunk rejected: %v", err)
		}
		batch.AgentID, batch.SessionID, batch.SessionEpoch = agentID, sessionID, epoch
		if _, _, err := plane.WriteTaskProgressLogs(context.Background(), batch); err != nil {
			t.Fatal(err)
		}
		if logs.calls != 1 || logs.request.Items[0].Content != raw {
			t.Fatal("raw chunk changed or duplicate Server reception event was emitted")
		}
	}
	for _, raw := range []string{"", string([]byte{0xff})} {
		_, err := taskProgressLogEntryFromProto(0, &taskprogressv1.TaskProgressLogEntry{Sequence: 1, Level: taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_INFO, Content: raw, EmittedAt: timestamppb.New(time.Now().UTC())})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("invalid raw content accepted: %q", raw)
		}
	}
}

func TestTaskProgressPersistenceErrorRetainsRawServerEvidence(t *testing.T) {
	agentID, sessionID, epoch := 17, "session-17", int64(23)
	raw := errors.New("original persistence failure /srv/private token=raw-canary")
	logs := &recoveringProgressLogService{err: raw}
	plane := NewTaskProgressLogDataPlane(logs, &progressTaskRepositoryStub{task: &scanapp.ScanTaskRecord{
		ID: 101, ScanID: 7, Status: "running", AgentID: &agentID, AssignedAgentID: &agentID, AssignedSessionID: &sessionID, AssignedSessionEpoch: &epoch,
	}})
	_, _, err := plane.WriteTaskProgressLogs(context.Background(), TaskProgressLogBatch{
		ScanID: 7, TaskID: 101, AgentID: agentID, SessionID: sessionID, SessionEpoch: epoch, RequestID: "original-event",
		Entries: []TaskProgressLogEntry{{Sequence: 1, Level: "info", Content: "input_ready", EmittedAt: time.Now().UTC()}},
	})
	if !errors.Is(err, raw) || logs.calls != 2 || logs.request.Items[0].Level != "error" || !strings.Contains(logs.request.Items[0].Content, raw.Error()) {
		t.Fatalf("progress failure evidence = %#v, error=%v", logs.request, err)
	}
}
