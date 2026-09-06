package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

var (
	ErrInvalidScanWorkflowPageToken = errors.New("invalid scan workflow pageToken")
	ErrScanWorkflowAlreadyExists    = errors.New("scan workflow already exists")
	ErrScanWorkflowImmutable        = errors.New("built-in scan workflow is immutable")
	ErrScanWorkflowConflict         = errors.New("scan workflow etag conflict")
	ErrInvalidScanWorkflowUpdate    = errors.New("invalid scan workflow update")
)

const (
	scanWorkflowListDefaultPageSize = 25
	scanWorkflowListMaxPageSize     = 100
	scanWorkflowListTokenVersion    = 1
)

type ManagedScanWorkflowStore interface {
	GetScanWorkflowByID(scanWorkflowID string) (*catalogdomain.ManagedScanWorkflow, error)
	ListScanWorkflows(filter catalogdomain.ScanWorkflowListFilter) ([]catalogdomain.ManagedScanWorkflow, int64, error)
	CreateScanWorkflow(workflow *catalogdomain.ManagedScanWorkflow) error
	FindScanWorkflowByRequestID(requestID string) (*catalogdomain.ManagedScanWorkflow, error)
	UpdateUserScanWorkflow(workflow *catalogdomain.ManagedScanWorkflow, expectedVersion int64) (bool, error)
}

// ManagedScanWorkflowStoreContext is the request-aware read extension used by
// MCP.  It is intentionally additive so existing command/query test doubles
// that implement the legacy store remain source-compatible.
type ManagedScanWorkflowStoreContext interface {
	GetScanWorkflowByIDContext(context.Context, string) (*catalogdomain.ManagedScanWorkflow, error)
	ListScanWorkflowsContext(context.Context, catalogdomain.ScanWorkflowListFilter) ([]catalogdomain.ManagedScanWorkflow, int64, error)
}

type ScanWorkflowEngineResolver interface {
	HasEngine(ctx context.Context, engineID string) (bool, error)
}

type ManagedScanWorkflow struct {
	catalogdomain.ManagedScanWorkflow
	ETag         string
	IsExecutable bool
}

type ListManagedScanWorkflowsInput struct {
	PageSize  int
	PageToken string
	Filter    string
}

type ListManagedScanWorkflowsResult struct {
	Results       []ManagedScanWorkflow
	NextPageToken string
	TotalSize     int64
}

type CreateManagedScanWorkflowInput struct {
	ScanWorkflowID string
	RequestID      string
	DisplayName    string
	Description    string
	Stages         []scanworkflow.Stage
}

type UpdateManagedScanWorkflowInput struct {
	ScanWorkflowID string
	ETag           string
	UpdateMask     []string
	DisplayName    *string
	Description    *string
	Stages         *[]scanworkflow.Stage
}

type scanWorkflowListPageToken struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	Filter   string `json:"f"`
	Order    string `json:"o"`
}

type ScanWorkflowManagementService struct {
	store   ManagedScanWorkflowStore
	engines ScanWorkflowEngineResolver
}

func NewScanWorkflowManagementService(store ManagedScanWorkflowStore, engines ScanWorkflowEngineResolver) (*ScanWorkflowManagementService, error) {
	if store == nil {
		return nil, fmt.Errorf("scan workflow store is required")
	}
	if engines == nil {
		return nil, fmt.Errorf("scan workflow engine resolver is required")
	}
	return &ScanWorkflowManagementService{store: store, engines: engines}, nil
}

func (service *ScanWorkflowManagementService) GetScanWorkflow(ctx context.Context, scanWorkflowID string) (*ManagedScanWorkflow, error) {
	workflow, err := service.getScanWorkflow(ctx, strings.TrimSpace(scanWorkflowID))
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, catalogdomain.ErrScanWorkflowNotFound
	}
	return service.present(ctx, *workflow)
}

func (service *ScanWorkflowManagementService) ListScanWorkflows(ctx context.Context, input ListManagedScanWorkflowsInput) (*ListManagedScanWorkflowsResult, error) {
	pageSize := normalizeScanWorkflowPageSize(input.PageSize)
	filter := strings.TrimSpace(input.Filter)
	page := 1
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeScanWorkflowListPageToken(input.PageToken)
		if err != nil || payload.Filter != filter || payload.PageSize != pageSize || payload.Order != "builtin-display-name" {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidScanWorkflowPageToken)
		}
		page = payload.Page
	}
	workflows, total, err := service.listScanWorkflows(ctx, catalogdomain.ScanWorkflowListFilter{Page: page, PageSize: pageSize, Filter: filter})
	if err != nil {
		return nil, err
	}
	results := make([]ManagedScanWorkflow, 0, len(workflows))
	for _, workflow := range workflows {
		presented, err := service.present(ctx, workflow)
		if err != nil {
			return nil, err
		}
		results = append(results, *presented)
	}
	next := ""
	if int64(page*pageSize) < total {
		next, err = encodeScanWorkflowListPageToken(scanWorkflowListPageToken{Version: scanWorkflowListTokenVersion, Page: page + 1, PageSize: pageSize, Filter: filter, Order: "builtin-display-name"})
		if err != nil {
			return nil, err
		}
	}
	return &ListManagedScanWorkflowsResult{Results: results, NextPageToken: next, TotalSize: total}, nil
}

func (service *ScanWorkflowManagementService) getScanWorkflow(ctx context.Context, scanWorkflowID string) (*catalogdomain.ManagedScanWorkflow, error) {
	if service == nil || service.store == nil {
		return nil, fmt.Errorf("scan workflow store is required")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	if store, ok := service.store.(ManagedScanWorkflowStoreContext); ok {
		return store.GetScanWorkflowByIDContext(ctx, scanWorkflowID)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return service.store.GetScanWorkflowByID(scanWorkflowID)
}

func (service *ScanWorkflowManagementService) listScanWorkflows(ctx context.Context, filter catalogdomain.ScanWorkflowListFilter) ([]catalogdomain.ManagedScanWorkflow, int64, error) {
	if service == nil || service.store == nil {
		return nil, 0, fmt.Errorf("scan workflow store is required")
	}
	if ctx == nil {
		return nil, 0, context.Canceled
	}
	if store, ok := service.store.(ManagedScanWorkflowStoreContext); ok {
		return store.ListScanWorkflowsContext(ctx, filter)
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	return service.store.ListScanWorkflows(filter)
}

func (service *ScanWorkflowManagementService) CreateScanWorkflow(ctx context.Context, input CreateManagedScanWorkflowInput) (*ManagedScanWorkflow, error) {
	requestID := strings.TrimSpace(input.RequestID)
	if requestID == "" {
		return nil, fmt.Errorf("%w: requestId is required", ErrInvalidScanWorkflowUpdate)
	}
	parsedRequestID, err := uuid.Parse(requestID)
	if err != nil || parsedRequestID.String() != requestID {
		return nil, fmt.Errorf("%w: requestId must be a canonical UUID", ErrInvalidScanWorkflowUpdate)
	}
	if existing, err := service.store.FindScanWorkflowByRequestID(requestID); err != nil {
		return nil, err
	} else if existing != nil {
		return service.present(ctx, *existing)
	}
	id := strings.TrimSpace(input.ScanWorkflowID)
	if id == "" {
		id = "wf-" + uuid.NewString()
	}
	workflow := &catalogdomain.ManagedScanWorkflow{ScanWorkflowID: id, DisplayName: input.DisplayName, Description: input.Description, Stages: input.Stages, RequestID: requestID, Version: 1}
	if err := workflow.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidScanWorkflowUpdate, err)
	}
	if err := service.ensureEngines(ctx, workflow.Stages); err != nil {
		return nil, err
	}
	if err := service.store.CreateScanWorkflow(workflow); err != nil {
		if existing, lookupErr := service.store.FindScanWorkflowByRequestID(requestID); lookupErr == nil && existing != nil {
			return service.present(ctx, *existing)
		}
		return nil, fmt.Errorf("%w: %v", ErrScanWorkflowAlreadyExists, err)
	}
	return service.present(ctx, *workflow)
}

func (service *ScanWorkflowManagementService) UpdateScanWorkflow(ctx context.Context, input UpdateManagedScanWorkflowInput) (*ManagedScanWorkflow, error) {
	workflow, err := service.getScanWorkflow(ctx, strings.TrimSpace(input.ScanWorkflowID))
	if err != nil {
		return nil, err
	}
	if workflow.IsBuiltin {
		return nil, ErrScanWorkflowImmutable
	}
	version, _, err := scanworkflow.ParseETag(workflow.ScanWorkflowID, input.ETag)
	if err != nil {
		return nil, fmt.Errorf("%w: etag is required and must match the resource", ErrInvalidScanWorkflowUpdate)
	}
	if version != workflow.Version {
		return nil, ErrScanWorkflowConflict
	}
	if err := applyScanWorkflowUpdateMask(workflow, input); err != nil {
		return nil, err
	}
	if err := workflow.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidScanWorkflowUpdate, err)
	}
	if err := service.ensureEngines(ctx, workflow.Stages); err != nil {
		return nil, err
	}
	updated, err := service.store.UpdateUserScanWorkflow(workflow, version)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, ErrScanWorkflowConflict
	}
	return service.GetScanWorkflow(ctx, workflow.ScanWorkflowID)
}

func (service *ScanWorkflowManagementService) present(ctx context.Context, workflow catalogdomain.ManagedScanWorkflow) (*ManagedScanWorkflow, error) {
	digest, err := scanworkflow.CanonicalWorkflowDigest(workflow.ScanWorkflowID, workflow.DisplayName, workflow.Description, workflow.Stages)
	if err != nil {
		return nil, err
	}
	etag, err := scanworkflow.NewETag(workflow.ScanWorkflowID, workflow.Version, digest)
	if err != nil {
		return nil, err
	}
	executable := true
	for _, stage := range workflow.Stages {
		for _, step := range stage.Steps {
			available, err := service.engines.HasEngine(ctx, step.EngineID)
			if err != nil {
				return nil, err
			}
			if !available {
				executable = false
			}
		}
	}
	return &ManagedScanWorkflow{ManagedScanWorkflow: workflow, ETag: etag, IsExecutable: executable}, nil
}

func (service *ScanWorkflowManagementService) ensureEngines(ctx context.Context, stages []scanworkflow.Stage) error {
	for _, stage := range stages {
		for _, step := range stage.Steps {
			available, err := service.engines.HasEngine(ctx, step.EngineID)
			if err != nil {
				return err
			}
			if !available {
				return fmt.Errorf("%w: engine %q is unavailable", ErrInvalidScanWorkflowUpdate, step.EngineID)
			}
		}
	}
	return nil
}

func applyScanWorkflowUpdateMask(workflow *catalogdomain.ManagedScanWorkflow, input UpdateManagedScanWorkflowInput) error {
	if len(input.UpdateMask) == 0 {
		return fmt.Errorf("%w: updateMask is required", ErrInvalidScanWorkflowUpdate)
	}
	seen := map[string]struct{}{}
	for _, path := range input.UpdateMask {
		switch path {
		case "displayName":
			if input.DisplayName == nil {
				return fmt.Errorf("%w: displayName is required by updateMask", ErrInvalidScanWorkflowUpdate)
			}
			workflow.DisplayName = *input.DisplayName
		case "description":
			if input.Description == nil {
				return fmt.Errorf("%w: description is required by updateMask", ErrInvalidScanWorkflowUpdate)
			}
			workflow.Description = *input.Description
		case "stages":
			if input.Stages == nil {
				return fmt.Errorf("%w: stages is required by updateMask", ErrInvalidScanWorkflowUpdate)
			}
			workflow.Stages = *input.Stages
		default:
			return fmt.Errorf("%w: unsupported update path %q", ErrInvalidScanWorkflowUpdate, path)
		}
		if _, exists := seen[path]; exists {
			return fmt.Errorf("%w: duplicate update path %q", ErrInvalidScanWorkflowUpdate, path)
		}
		seen[path] = struct{}{}
	}
	return nil
}

func normalizeScanWorkflowPageSize(pageSize int) int {
	if pageSize <= 0 {
		return scanWorkflowListDefaultPageSize
	}
	if pageSize > scanWorkflowListMaxPageSize {
		return scanWorkflowListMaxPageSize
	}
	return pageSize
}

func encodeScanWorkflowListPageToken(token scanWorkflowListPageToken) (string, error) {
	encoded, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeScanWorkflowListPageToken(raw string) (scanWorkflowListPageToken, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return scanWorkflowListPageToken{}, ErrInvalidScanWorkflowPageToken
	}
	var token scanWorkflowListPageToken
	if err := json.Unmarshal(decoded, &token); err != nil || token.Version != scanWorkflowListTokenVersion || token.Page < 1 || token.PageSize < 1 {
		return scanWorkflowListPageToken{}, ErrInvalidScanWorkflowPageToken
	}
	return token, nil
}
