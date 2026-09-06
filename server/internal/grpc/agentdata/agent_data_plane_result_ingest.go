package agentdata

import (
	"context"
	"errors"

	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	contractresults "github.com/yyhuni/lunafox/contracts/results"
	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	pkg "github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxResultBatchItems     = contractresults.DefaultResultBatchMaxItems
	maxResultBatchJSONBytes = contractresults.DefaultResultBatchMaxBytes
)

type ResultIngestDataPlane interface {
	Ingest(context.Context, resultingestapp.ResultIngestCommand) (resultingestapp.ResultIngestOutcome, error)
}

type ResultTaskScopeDataPlane interface {
	GetResultTaskScope(ctx context.Context, request ResultTaskScopeRequest) (*ResultTaskScope, error)
}

type ResultTaskScopeRequest struct {
	TaskID       int
	AgentID      int
	SessionID    string
	SessionEpoch int64
}

type ResultTaskScope struct {
	TaskID   int
	ScanID   int
	TargetID int
}

type ResultIngestDataPlanes struct {
	TaskScopes ResultTaskScopeDataPlane
	Ingest     ResultIngestDataPlane
}

func (s *DataPlaneService) BatchIngestTaskResults(ctx context.Context, req *agentdatav1.BatchIngestTaskResultsRequest) (*agentdatav1.BatchIngestTaskResultsResponse, error) {
	lease, err := s.requireAuthenticatedAgentSession(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	scanID, taskID, err := resourcenames.ParseTask(req.GetTask())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	targetID, err := resourcenames.ParseTarget(req.GetTarget())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(req.ItemsJson) == 0 {
		return nil, status.Error(codes.InvalidArgument, "itemsJson must not be empty")
	}
	if err := validateResultIngestItemsJSON(req.ItemsJson); err != nil {
		return nil, err
	}
	scope, err := s.getResultTaskScope(ctx, ResultTaskScopeRequest{
		TaskID: taskID, AgentID: lease.AgentID, SessionID: lease.SessionID, SessionEpoch: lease.SessionEpoch,
	}, scanID, targetID)
	if err != nil {
		return nil, err
	}

	if s.resultIngest.Ingest == nil {
		return nil, status.Error(codes.Unimplemented, errDataPlaneUnimplemented)
	}
	// Result submission has no request id; the application boundary preserves
	// retry idempotency through task scope, result type, and natural-key upserts.
	items := make([][]byte, len(req.ItemsJson))
	for index := range req.ItemsJson {
		items[index] = []byte(req.ItemsJson[index])
	}
	outcome, err := s.resultIngest.Ingest.Ingest(ctx, resultingestapp.ResultIngestCommand{
		TaskID:       scope.TaskID,
		ScanID:       scope.ScanID,
		TargetID:     scope.TargetID,
		AgentID:      lease.AgentID,
		SessionID:    lease.SessionID,
		SessionEpoch: lease.SessionEpoch,
		ResultType:   req.ResultType,
		Items:        items,
	})
	if err != nil {
		return nil, mapResultIngestError(err)
	}
	pkg.Info("result ingest materialized",
		zap.Int("task.id", taskID),
		zap.Int("scan.id", scope.ScanID),
		zap.Int("target.id", scope.TargetID),
		zap.String("result.type", req.ResultType),
		zap.Int("result.received_items", outcome.ReceivedItems),
		zap.Int("result.duplicate_items", outcome.DuplicateItems),
		zap.Int("result.scope_filtered_items", outcome.ScopeFilteredItems),
		zap.Int("result.unsupported_items", outcome.UnsupportedItems),
		zap.Int64("result.snapshot_count", outcome.SnapshotCount),
		zap.Int64("result.asset_count", outcome.AssetCount),
	)

	return &agentdatav1.BatchIngestTaskResultsResponse{}, nil
}

func (s *DataPlaneService) getResultTaskScope(ctx context.Context, request ResultTaskScopeRequest, scanID, targetID int) (*ResultTaskScope, error) {
	if s.resultIngest.TaskScopes == nil {
		return nil, status.Error(codes.Unimplemented, errDataPlaneUnimplemented)
	}
	scope, err := s.resultIngest.TaskScopes.GetResultTaskScope(ctx, request)
	if err != nil {
		return nil, mapResultTaskScopeError(err)
	}
	if scope == nil {
		return nil, status.Error(codes.NotFound, scanapp.ErrScanTaskNotFound.Error())
	}
	if scope.TaskID != request.TaskID || scope.ScanID != scanID || scope.TargetID != targetID {
		return nil, status.Error(codes.PermissionDenied, "result task scope does not match request")
	}
	return scope, nil
}

func validateResultIngestItemsJSON(itemsJSON []string) error {
	if len(itemsJSON) > maxResultBatchItems {
		return status.Errorf(codes.ResourceExhausted, "itemsJson must not exceed %d", maxResultBatchItems)
	}
	totalBytes := 0
	for _, item := range itemsJSON {
		itemBytes := len([]byte(item))
		if itemBytes > maxResultBatchJSONBytes-totalBytes {
			return status.Errorf(codes.ResourceExhausted, "itemsJson total bytes must not exceed %d", maxResultBatchJSONBytes)
		}
		totalBytes += itemBytes
	}
	return nil
}

func mapResultTaskScopeError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "result task scope lookup cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "result task scope lookup deadline exceeded")
	case errors.Is(err, errResultTaskOwnershipMismatch):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, errResultTaskNotRunning), errors.Is(err, errResultTaskSessionMismatch):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, scanapp.ErrScanTaskNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, scanapp.ErrScanNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Unavailable, "result task scope authority is unavailable")
	}
}

func mapResultIngestError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "result ingest cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "result ingest deadline exceeded")
	case errors.Is(err, resultingestapp.ErrUnsupportedResultType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, resultingestapp.ErrInvalidResultItems):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, resultingestapp.ErrResultNotAuthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, resultingestapp.ErrResultMaterializerUnavailable):
		return status.Error(codes.Unimplemented, errDataPlaneUnimplemented)
	case errors.Is(err, resultingestapp.ErrInvalidResultScope):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, resultingestapp.ErrResultExecutionFenceRejected):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, snapshotapp.ErrScanNotFoundForSnapshot):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, snapshotapp.ErrTargetMismatch):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, snapshotapp.ErrInvalidTargetType):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
