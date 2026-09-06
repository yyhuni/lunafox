package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	app "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/application"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/dto"
)

type nucleiPocService interface {
	CurrentSource(context.Context) (*domain.Source, error)
	CreateSync(context.Context, app.CreateSyncInput) (*domain.SyncTask, error)
	GetSyncTask(context.Context, uuid.UUID) (*domain.SyncTask, error)
	List(context.Context, app.POCListQuery) (*app.POCListResult, error)
	ListFilterOptions(context.Context, string) ([]domain.FilterOption, error)
	Get(context.Context, string) (*domain.POC, error)
	UpdateEnabled(context.Context, string, bool) (*domain.POC, error)
	SetActivation(context.Context, app.SetPOCActivationInput) (*app.SetPOCActivationResult, error)
}

type NucleiPOCHandler struct{ service nucleiPocService }

func NewNucleiPOCHandler(service nucleiPocService) *NucleiPOCHandler {
	return &NucleiPOCHandler{service: service}
}

func (handler *NucleiPOCHandler) Sync(c *gin.Context) {
	var request dto.SyncSourceRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	requestID, ok := parseCanonicalUUID(request.RequestID)
	if !ok {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "requestId must be a canonical UUID")
		return
	}
	sourceType := domain.SourceType(strings.TrimSpace(request.SourceType))
	if !sourceType.Valid() {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "sourceType must be git, gitee, or custom")
		return
	}
	task, err := handler.service.CreateSync(c.Request.Context(), app.CreateSyncInput{RequestID: requestID, SourceType: sourceType, RepoURL: request.RepoURL})
	if err != nil {
		handler.writeError(c, err)
		return
	}
	httpdto.Created(c, taskResponse(task))
}

func (handler *NucleiPOCHandler) CurrentSource(c *gin.Context) {
	source, err := handler.service.CurrentSource(c.Request.Context())
	if err != nil {
		handler.writeError(c, err)
		return
	}
	httpdto.Success(c, sourceResponse(source))
}

func (handler *NucleiPOCHandler) GetSyncTask(c *gin.Context) {
	// Keep path extraction separate from UUID parsing so the HTTP boundary
	// remains explicit about the canonical task resource segment.
	taskSegment := c.Param("task")
	id, ok := parseCanonicalUUID(taskSegment)
	if !ok {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "task must be a canonical UUID")
		return
	}
	task, err := handler.service.GetSyncTask(c.Request.Context(), id)
	if err != nil {
		handler.writeError(c, err)
		return
	}
	httpdto.Success(c, taskResponse(task))
}

func (handler *NucleiPOCHandler) List(c *gin.Context) {
	if !hasOnlyListQueryFields(c) {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "unsupported Nuclei POC query field")
		return
	}
	var query dto.NucleiPocListQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}
	result, err := handler.service.List(c.Request.Context(), app.POCListQuery{PageSize: query.PageSize, PageToken: query.PageToken, Filter: query.Filter, OrderBy: query.OrderBy})
	if err != nil {
		handler.writeError(c, err)
		return
	}
	items := make([]dto.NucleiPocResponse, 0, len(result.Results))
	for index := range result.Results {
		items = append(items, pocResponse(&result.Results[index], false))
	}
	httpdto.Success(c, dto.NucleiPocListResponse{Results: items, NextPageToken: result.NextPageToken, TotalSize: result.TotalSize})
}

// FilterOptions returns complete committed-catalog options for one approved
// Nuclei POC facet. GET /v1/nucleiPocs/filterOptions?field=tags
func (handler *NucleiPOCHandler) FilterOptions(c *gin.Context) {
	if !hasOnlyFilterOptionsQueryFields(c) {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "field must be tags")
		return
	}
	field, ok := c.GetQuery("field")
	if !ok || strings.TrimSpace(field) == "" {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "field is required")
		return
	}
	options, err := handler.service.ListFilterOptions(c.Request.Context(), field)
	if err != nil {
		handler.writeError(c, err)
		return
	}
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: results})
}

func (handler *NucleiPOCHandler) Get(c *gin.Context) {
	templateID := c.Param("nucleiPoc")
	if !isCanonicalTemplateIDSegment(templateID) {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "invalid nuclei POC resource name")
		return
	}
	resourceName := "nucleiPocs/" + templateID
	poc, err := handler.service.Get(c.Request.Context(), resourceName)
	if err != nil {
		handler.writeError(c, err)
		return
	}
	httpdto.Success(c, pocResponse(poc, true))
}

func (handler *NucleiPOCHandler) Update(c *gin.Context) {
	templateID := c.Param("nucleiPoc")
	if !isCanonicalTemplateIDSegment(templateID) {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "invalid nuclei POC resource name")
		return
	}
	resourceName := "nucleiPocs/" + templateID
	var request dto.UpdateNucleiPocRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	if request.Name != resourceName {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "name must match the resource path")
		return
	}
	mask := request.MaskSet()
	if len(request.UpdateMask) != 1 || request.UpdateMask[0] != "isEnabled" || len(mask) != 1 {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "updateMask must contain only isEnabled")
		return
	}
	if _, ok := mask["isEnabled"]; !ok || request.IsEnabled == nil {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "isEnabled must be declared in updateMask")
		return
	}
	poc, err := handler.service.UpdateEnabled(c.Request.Context(), resourceName, *request.IsEnabled)
	if err != nil {
		handler.writeError(c, err)
		return
	}
	httpdto.Success(c, pocResponse(poc, false))
}

func (handler *NucleiPOCHandler) SetActivation(c *gin.Context) {
	var request dto.SetNucleiPocActivationRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	if request.Enabled == nil {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "enabled is required")
		return
	}
	var names []string
	if request.Names != nil {
		names = append([]string{}, (*request.Names)...)
		if names == nil {
			// A present JSON null is an invalid selected scope, not the legacy
			// omitted-name full-catalog command.
			names = []string{}
		}
	}
	result, err := handler.service.SetActivation(c.Request.Context(), app.SetPOCActivationInput{Enabled: *request.Enabled, Names: names})
	if err != nil {
		handler.writeError(c, err)
		return
	}
	httpdto.Success(c, dto.SetNucleiPocActivationResponse{Enabled: result.Enabled, AffectedCount: result.AffectedCount})
}

func (handler *NucleiPOCHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, app.ErrInvalidArgument), errors.Is(err, app.ErrInvalidSourceURL), errors.Is(err, app.ErrInvalidQuery), errors.Is(err, app.ErrInvalidUpdateMask):
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", safeErrorMessage(err, "Invalid Nuclei POC request"))
	case errors.Is(err, app.ErrSyncAlreadyRunning):
		metadata := map[string]string{}
		var active *app.SyncAlreadyRunningError
		if errors.As(err, &active) && active.TaskID != uuid.Nil {
			metadata["task"] = "nucleiPocSyncTasks/" + active.TaskID.String()
		}
		httpdto.ErrorWithStatusAndTypedDetails(c, http.StatusConflict, "SYNC_ALREADY_RUNNING", "ABORTED", "Another Nuclei POC sync is already running.", metadata)
	case errors.Is(err, app.ErrSyncConflict):
		httpdto.ErrorWithStatus(c, http.StatusConflict, "SYNC_REQUEST_CONFLICT", "ABORTED", "The requestId was already used for a different source.")
	case errors.Is(err, app.ErrRequestExpired):
		httpdto.ErrorWithStatus(c, http.StatusGone, "SYNC_REQUEST_EXPIRED", "FAILED_PRECONDITION", "The sync request record has expired; start a new synchronization.")
	case errors.Is(err, app.ErrSourceNotFound), errors.Is(err, domain.ErrSourceNotFound):
		httpdto.NotFound(c, "Current Nuclei POC source not found")
	case errors.Is(err, app.ErrTaskNotFound), errors.Is(err, domain.ErrSyncTaskNotFound):
		httpdto.NotFound(c, "Nuclei POC sync task not found")
	case errors.Is(err, app.ErrPOCNotFound), errors.Is(err, domain.ErrPOCNotFound):
		httpdto.NotFound(c, "Nuclei POC not found")
	default:
		httpdto.InternalError(c, "Nuclei POC operation failed")
	}
}

func parseCanonicalUUID(value string) (uuid.UUID, bool) {
	id, err := uuid.Parse(value)
	return id, err == nil && id != uuid.Nil && value == id.String()
}

func hasOnlyListQueryFields(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}
	allowed := map[string]struct{}{"pageSize": {}, "pageToken": {}, "filter": {}, "orderBy": {}}
	for key, values := range c.Request.URL.Query() {
		if _, ok := allowed[key]; !ok || len(values) > 1 {
			return false
		}
	}
	return true
}

func hasOnlyFilterOptionsQueryFields(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}
	values, ok := c.Request.URL.Query()["field"]
	return ok && len(values) == 1 && len(c.Request.URL.Query()) == 1
}

func safeErrorMessage(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	message := strings.TrimSpace(err.Error())
	if message == "" || len(message) > 240 {
		return fallback
	}
	return message
}

func isCanonicalTemplateIDSegment(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || strings.Contains(value, "/") {
		return false
	}
	return strings.IndexFunc(value, func(r rune) bool {
		return r == '\\' || r == '\x00' || r < 0x20 || r == 0x7f
	}) < 0
}

func sourceResponse(source *domain.Source) dto.SourceResponse {
	return dto.SourceResponse{Name: "nucleiPocSources/current", SourceType: string(source.SourceType), RepoURL: source.RepoURL, CommitSHA: source.CommitSHA, SyncedAt: source.SyncedAt, CreatedAt: source.CreatedAt.UTC(), UpdatedAt: source.UpdatedAt.UTC()}
}

func taskResponse(task *domain.SyncTask) dto.SyncTaskResponse {
	return dto.SyncTaskResponse{Name: "nucleiPocSyncTasks/" + task.ID.String(), RequestID: task.RequestID.String(), SourceType: string(task.SourceType), State: string(task.State), Phase: string(task.Phase), Counters: dto.SyncCounters{FilesSeen: task.Counters.FilesSeen, YAMLFilesSeen: task.Counters.YAMLFilesSeen, TemplatesValidated: task.Counters.TemplatesValidated, BytesRead: task.Counters.BytesRead}, CommitSHA: task.CommitSHA, CommittedPOCCount: task.CommittedPOCCount, FailureCode: task.FailureCode, FailureSummary: task.FailureSummary, Diagnostics: diagnosticsResponse(task.Diagnostics), CleanupStatus: string(task.CleanupStatus), CreatedAt: task.CreatedAt.UTC(), StartedAt: utcPtr(task.StartedAt), CompletedAt: utcPtr(task.CompletedAt), UpdatedAt: task.UpdatedAt.UTC()}
}

func diagnosticsResponse(value domain.Diagnostics) dto.Diagnostics {
	value = value.Safe()
	samples := make([]dto.DiagnosticSample, 0, len(value.Samples))
	for _, sample := range value.Samples {
		samples = append(samples, dto.DiagnosticSample{Category: sample.Category, RelativePath: sample.RelativePath, ReasonCode: sample.ReasonCode})
	}
	return dto.Diagnostics{Samples: samples, Total: value.Total, Truncated: value.Truncated}
}

func pocResponse(poc *domain.POC, includeContent bool) dto.NucleiPocResponse {
	response := dto.NucleiPocResponse{Name: "nucleiPocs/" + poc.TemplateID, TemplateID: poc.TemplateID, DisplayName: poc.DisplayName, Severity: poc.Severity, Tags: cloneStrings(poc.Tags), Author: poc.Author, Description: poc.Description, CVE: cloneStrings(poc.CVE), CWE: cloneStrings(poc.CWE), References: cloneStrings(poc.References), Remediation: poc.Remediation, RelativePath: poc.RelativePath, ContentSHA256: poc.ContentSHA256, IsEnabled: poc.IsEnabled, CreatedAt: poc.CreatedAt.UTC(), UpdatedAt: poc.UpdatedAt.UTC()}
	if includeContent {
		response.Content = poc.Content
	}
	return response
}

func cloneStrings(values []string) []string {
	// Collection fields are required JSON arrays; Go nil slices must not leak as null.
	if len(values) == 0 {
		return []string{}
	}
	return append([]string{}, values...)
}

func utcPtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}
