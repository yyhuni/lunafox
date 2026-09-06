package adapters

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogdto "github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	identitydto "github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
	securityapp "github.com/yyhuni/lunafox/server/internal/modules/security/application"
	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type organizationReader struct {
	facade *identityapp.OrganizationFacade
}

// NewOrganizationReader projects the deployment-wide Identity read surface.
func NewOrganizationReader(facade *identityapp.OrganizationFacade) tools.OrganizationReader {
	return &organizationReader{facade: facade}
}

func (reader *organizationReader) List(ctx context.Context, query tools.OrganizationQuery) (tools.Page[tools.OrganizationRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.OrganizationRecord]{}, mcpErrors.ErrInternal
	}
	shape := struct{ Filter, OrderBy string }{strings.TrimSpace(query.Filter), strings.TrimSpace(query.OrderBy)}
	cursor, err := decodeListCursor(query.PageToken, "organizations", query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.OrganizationRecord]{}, err
	}
	result, err := reader.facade.ListOrganizationsContext(ctx, &identitydto.OrganizationListQuery{
		PaginationQuery: httpdto.PaginationQuery{PageSize: query.PageSize, PageToken: cursor.InnerToken},
		Filter:          shape.Filter,
		OrderBy:         shape.OrderBy,
	})
	if err != nil {
		if errors.Is(err, identityapp.ErrUnsupportedOrganizationFilter) || errors.Is(err, identityapp.ErrUnsupportedOrganizationOrderBy) || errors.Is(err, identityapp.ErrInvalidOrganizationPageToken) {
			return tools.Page[tools.OrganizationRecord]{}, invalidReaderInput()
		}
		return tools.Page[tools.OrganizationRecord]{}, mapInvestigationReaderError(err, identityapp.ErrOrganizationNotFound)
	}
	if result == nil {
		return tools.Page[tools.OrganizationRecord]{}, mcpErrors.ErrCommandFailed
	}
	items := make([]tools.OrganizationRecord, 0, len(result.Organizations))
	for _, item := range result.Organizations {
		items = append(items, organizationRecord(item))
	}
	next, err := wrapInnerCursor("organizations", query.PageSize, shape, cursor.Page+1, result.NextPageToken)
	if err != nil {
		return tools.Page[tools.OrganizationRecord]{}, err
	}
	return tools.Page[tools.OrganizationRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

func (reader *organizationReader) Get(ctx context.Context, id int) (tools.OrganizationRecord, error) {
	if reader.facade == nil {
		return tools.OrganizationRecord{}, mcpErrors.ErrInternal
	}
	item, err := reader.facade.GetOrganizationByIDContext(ctx, id)
	if err != nil {
		return tools.OrganizationRecord{}, mapInvestigationReaderError(err, identityapp.ErrOrganizationNotFound)
	}
	if item == nil {
		return tools.OrganizationRecord{}, mcpErrors.ErrNotFound
	}
	return organizationRecord(identitydomain.OrganizationWithTargetCount{
		Organization: identitydomain.Organization{
			ID: item.ID, Name: item.Name, Description: item.Description, CreatedAt: item.CreatedAt, DeletedAt: item.DeletedAt,
		}, TargetCount: item.TargetCount,
	}), nil
}

func (reader *organizationReader) ListTargets(ctx context.Context, organizationID int, query tools.TargetQuery) (tools.Page[tools.OrganizationTargetRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.OrganizationTargetRecord]{}, mcpErrors.ErrInternal
	}
	if organizationID <= 0 {
		return tools.Page[tools.OrganizationTargetRecord]{}, mcpErrors.ErrInvalidInput
	}
	if query.Type != "" && query.Type != "domain" && query.Type != "ip" && query.Type != "cidr" {
		return tools.Page[tools.OrganizationTargetRecord]{}, mcpErrors.ErrInvalidInput
	}
	// Identity's organization-target query accepts a name filter and a
	// separate target type argument. Do not synthesize type==... into the
	// filter: that field is intentionally not part of the Identity filter
	// allowlist and would make an otherwise valid typed query fail.
	filter := strings.TrimSpace(query.Filter)
	shape := struct {
		OrganizationID  int
		Filter, OrderBy string
		Type            string
	}{organizationID, filter, query.OrderBy, query.Type}
	cursor, err := decodeListCursor(query.PageToken, "organizationTargets", query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.OrganizationTargetRecord]{}, err
	}
	innerToken := organizationTargetPageToken(cursor.Page)
	targets, total, err := reader.facade.ListOrganizationTargetsContext(ctx, organizationID, &identitydto.TargetListQuery{
		PaginationQuery: httpdto.PaginationQuery{PageSize: query.PageSize, PageToken: innerToken},
		Type:            query.Type,
		Filter:          filter,
	})
	if err != nil {
		return tools.Page[tools.OrganizationTargetRecord]{}, mapInvestigationReaderError(err, identityapp.ErrOrganizationNotFound, identityapp.ErrTargetNotFound)
	}
	items := make([]tools.OrganizationTargetRecord, 0, len(targets))
	for _, item := range targets {
		items = append(items, organizationTargetRecord(item))
	}
	next := ""
	if int64(cursor.Page*query.PageSize) < total {
		next, err = encodeNextCursor("organizationTargets", query.PageSize, shape, cursor.Page+1, organizationTargetPageToken(cursor.Page+1))
		if err != nil {
			return tools.Page[tools.OrganizationTargetRecord]{}, err
		}
	}
	return tools.Page[tools.OrganizationTargetRecord]{Items: items, NextPageToken: next, TotalSize: total}, nil
}

func organizationTargetPageToken(page int) string {
	if page <= 1 {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("page:%d", page)))
}

type targetVulnerabilityReader struct {
	facade *securityapp.VulnerabilityFacade
}

// NewTargetVulnerabilityReader creates a mandatory target-scoped vulnerability port.
func NewTargetVulnerabilityReader(facade *securityapp.VulnerabilityFacade) tools.TargetVulnerabilityReader {
	return &targetVulnerabilityReader{facade: facade}
}

func (reader *targetVulnerabilityReader) ListByTarget(ctx context.Context, targetID int, query tools.VulnerabilityQuery) (tools.Page[tools.VulnerabilityRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.VulnerabilityRecord]{}, mcpErrors.ErrInternal
	}
	filter := vulnerabilityFilter(query)
	shape := struct {
		TargetID        int
		Filter, OrderBy string
	}{targetID, filter, query.OrderBy}
	cursor, err := decodeListCursor(query.PageToken, "targetVulnerabilities", query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.VulnerabilityRecord]{}, err
	}
	result, err := reader.facade.ListByTargetContext(ctx, targetID, securityapp.VulnerabilityListQueryInput{
		PageSize: query.PageSize, PageToken: cursor.InnerToken, Filter: filter, OrderBy: query.OrderBy,
	})
	if err != nil {
		if errors.Is(err, securityapp.ErrUnsupportedVulnerabilityFilter) || errors.Is(err, securityapp.ErrUnsupportedVulnerabilityOrderBy) || errors.Is(err, securityapp.ErrInvalidVulnerabilityPageToken) || errors.Is(err, securityapp.ErrInvalidVulnerabilityInput) {
			return tools.Page[tools.VulnerabilityRecord]{}, invalidReaderInput()
		}
		return tools.Page[tools.VulnerabilityRecord]{}, mapInvestigationReaderError(err, securityapp.ErrVulnerabilityNotFound, securityapp.ErrTargetNotFound)
	}
	if result == nil {
		return tools.Page[tools.VulnerabilityRecord]{}, mcpErrors.ErrCommandFailed
	}
	items := make([]tools.VulnerabilityRecord, 0, len(result.Vulnerabilities))
	for _, value := range result.Vulnerabilities {
		mapped := vulnerabilityRecord(value)
		mapped.RawOutput = nil
		items = append(items, mapped)
	}
	next, err := wrapInnerCursor("targetVulnerabilities", query.PageSize, shape, cursor.Page+1, result.NextPageToken)
	if err != nil {
		return tools.Page[tools.VulnerabilityRecord]{}, err
	}
	return tools.Page[tools.VulnerabilityRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

type screenshotReader struct{ facade *assetapp.ScreenshotFacade }

// NewScreenshotReader creates the metadata/image split screenshot port.
func NewScreenshotReader(facade *assetapp.ScreenshotFacade) tools.ScreenshotReader {
	return &screenshotReader{facade: facade}
}

func (reader *screenshotReader) ListByTarget(ctx context.Context, targetID int, query tools.ScreenshotQuery) (tools.Page[tools.ScreenshotRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.ScreenshotRecord]{}, mcpErrors.ErrInternal
	}
	shape := struct {
		TargetID        int
		Filter, OrderBy string
	}{targetID, strings.TrimSpace(query.Filter), strings.TrimSpace(query.OrderBy)}
	cursor, err := decodeListCursor(query.PageToken, "targetScreenshots", query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.ScreenshotRecord]{}, err
	}
	result, err := reader.facade.ListByTargetContext(ctx, targetID, assetapp.ScreenshotListQueryInput{
		PageSize: query.PageSize, PageToken: cursor.InnerToken, Filter: shape.Filter, OrderBy: shape.OrderBy,
	})
	if err != nil {
		if errors.Is(err, assetapp.ErrUnsupportedScreenshotFilter) || errors.Is(err, assetapp.ErrUnsupportedScreenshotOrderBy) || errors.Is(err, assetapp.ErrInvalidScreenshotPageToken) {
			return tools.Page[tools.ScreenshotRecord]{}, invalidReaderInput()
		}
		return tools.Page[tools.ScreenshotRecord]{}, mapInvestigationReaderError(err, assetapp.ErrScreenshotNotFound, assetapp.ErrTargetNotFound)
	}
	if result == nil {
		return tools.Page[tools.ScreenshotRecord]{}, mcpErrors.ErrCommandFailed
	}
	items := make([]tools.ScreenshotRecord, 0, len(result.Screenshots))
	for _, value := range result.Screenshots {
		items = append(items, screenshotRecord(value))
	}
	next, err := wrapInnerCursor("targetScreenshots", query.PageSize, shape, cursor.Page+1, result.NextPageToken)
	if err != nil {
		return tools.Page[tools.ScreenshotRecord]{}, err
	}
	return tools.Page[tools.ScreenshotRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

func (reader *screenshotReader) GetImage(ctx context.Context, targetID, id int) ([]byte, error) {
	if reader.facade == nil {
		return nil, mcpErrors.ErrInternal
	}
	item, err := reader.facade.GetByIDForTargetContext(ctx, targetID, id)
	if err != nil {
		return nil, mapInvestigationReaderError(err, assetapp.ErrScreenshotNotFound, assetapp.ErrTargetNotFound)
	}
	if item == nil || len(item.Image) == 0 {
		return nil, mcpErrors.ErrNotFound
	}
	if len(item.Image) > tools.MaxImageBytes {
		return nil, mcpErrors.ErrResultTooLarge
	}
	return append([]byte(nil), item.Image...), nil
}

type workflowReader struct {
	service *catalogapp.ScanWorkflowManagementService
}
type profileReader struct {
	service *catalogapp.ScanWorkflowProfileService
}
type engineReader struct {
	facade *catalogapp.EngineCatalogFacade
}
type wordlistReader struct{ facade *catalogapp.WordlistFacade }

func NewScanWorkflowReader(service *catalogapp.ScanWorkflowManagementService) tools.ScanWorkflowReader {
	return &workflowReader{service: service}
}

func NewScanWorkflowProfileReader(service *catalogapp.ScanWorkflowProfileService) tools.ScanWorkflowProfileReader {
	return &profileReader{service: service}
}

func NewEngineReader(facade *catalogapp.EngineCatalogFacade) tools.EngineReader {
	return &engineReader{facade: facade}
}

func NewWordlistReader(facade *catalogapp.WordlistFacade) tools.WordlistReader {
	return &wordlistReader{facade: facade}
}

func (reader *workflowReader) List(ctx context.Context, query tools.CatalogListQuery) (tools.Page[tools.ScanWorkflowRecord], error) {
	if reader.service == nil {
		return tools.Page[tools.ScanWorkflowRecord]{}, mcpErrors.ErrInternal
	}
	shape := struct{ Filter string }{strings.TrimSpace(query.Filter)}
	cursor, err := decodeListCursor(query.PageToken, "scanWorkflows", query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.ScanWorkflowRecord]{}, err
	}
	result, err := reader.service.ListScanWorkflows(ctx, catalogapp.ListManagedScanWorkflowsInput{PageSize: query.PageSize, PageToken: cursor.InnerToken, Filter: shape.Filter})
	if err != nil {
		return tools.Page[tools.ScanWorkflowRecord]{}, mapCatalogError(err)
	}
	if result == nil {
		return tools.Page[tools.ScanWorkflowRecord]{}, mcpErrors.ErrCommandFailed
	}
	items := make([]tools.ScanWorkflowRecord, 0, len(result.Results))
	for _, item := range result.Results {
		items = append(items, workflowRecord(item))
	}
	next, err := wrapInnerCursor("scanWorkflows", query.PageSize, shape, cursor.Page+1, result.NextPageToken)
	if err != nil {
		return tools.Page[tools.ScanWorkflowRecord]{}, err
	}
	return tools.Page[tools.ScanWorkflowRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

func (reader *workflowReader) Get(ctx context.Context, name string) (tools.ScanWorkflowRecord, error) {
	if reader.service == nil {
		return tools.ScanWorkflowRecord{}, mcpErrors.ErrInternal
	}
	id, err := resourcenames.ParseScanWorkflow(name)
	if err != nil || resourcenames.ScanWorkflow(id) != strings.TrimSpace(name) {
		return tools.ScanWorkflowRecord{}, mcpErrors.ErrInvalidInput
	}
	item, err := reader.service.GetScanWorkflow(ctx, id)
	if err != nil {
		return tools.ScanWorkflowRecord{}, mapCatalogError(err)
	}
	if item == nil {
		return tools.ScanWorkflowRecord{}, mcpErrors.ErrNotFound
	}
	return workflowRecord(*item), nil
}

func (reader *profileReader) GetProfile(ctx context.Context, name string) (tools.ScanWorkflowProfileRecord, error) {
	if reader.service == nil {
		return tools.ScanWorkflowProfileRecord{}, mcpErrors.ErrInternal
	}
	id, err := resourcenames.ParseScanWorkflow(name)
	if err != nil || resourcenames.ScanWorkflow(id) != strings.TrimSpace(name) {
		return tools.ScanWorkflowProfileRecord{}, mcpErrors.ErrInvalidInput
	}
	item, err := reader.service.GetScanWorkflowProfile(ctx, id)
	if err != nil {
		return tools.ScanWorkflowProfileRecord{}, mapCatalogError(err)
	}
	if item == nil {
		return tools.ScanWorkflowProfileRecord{}, mcpErrors.ErrNotFound
	}
	return tools.ScanWorkflowProfileRecord{Name: resourcenames.ScanWorkflow(id) + "/profile", ScanWorkflow: resourcenames.ScanWorkflow(id), Configuration: item.Configuration}, nil
}

func (reader *engineReader) List(ctx context.Context, query tools.CatalogListQuery) (tools.Page[tools.EngineRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.EngineRecord]{}, mcpErrors.ErrInternal
	}
	if strings.TrimSpace(query.Filter) != "" || strings.TrimSpace(query.OrderBy) != "" {
		return tools.Page[tools.EngineRecord]{}, mcpErrors.ErrInvalidInput
	}
	cursor, err := decodeListCursor(query.PageToken, "engines", query.PageSize, struct{}{})
	if err != nil {
		return tools.Page[tools.EngineRecord]{}, err
	}
	items, err := reader.facade.ListEnginesContext(ctx)
	if err != nil {
		return tools.Page[tools.EngineRecord]{}, mapCatalogError(err)
	}
	total := len(items)
	start := (cursor.Page - 1) * query.PageSize
	if start >= total {
		return tools.Page[tools.EngineRecord]{Items: []tools.EngineRecord{}, TotalSize: int64(total)}, nil
	}
	end := start + query.PageSize
	if end > total {
		end = total
	}
	result := make([]tools.EngineRecord, 0, end-start)
	for _, item := range items[start:end] {
		result = append(result, engineRecord(item))
	}
	next := ""
	if end < total {
		next, err = encodeNextCursor("engines", query.PageSize, struct{}{}, cursor.Page+1, "")
		if err != nil {
			return tools.Page[tools.EngineRecord]{}, err
		}
	}
	return tools.Page[tools.EngineRecord]{Items: result, NextPageToken: next, TotalSize: int64(total)}, nil
}

func (reader *engineReader) Get(ctx context.Context, name string) (tools.EngineRecord, error) {
	if reader.facade == nil {
		return tools.EngineRecord{}, mcpErrors.ErrInternal
	}
	id, err := resourcenames.ParseEngine(name)
	if err != nil || resourcenames.Engine(id) != strings.TrimSpace(name) {
		return tools.EngineRecord{}, mcpErrors.ErrInvalidInput
	}
	item, err := reader.facade.GetEngineByIDContext(ctx, id)
	if err != nil {
		return tools.EngineRecord{}, mapCatalogError(err)
	}
	if item == nil {
		return tools.EngineRecord{}, mcpErrors.ErrNotFound
	}
	return engineRecord(*item), nil
}

func (reader *wordlistReader) List(ctx context.Context, query tools.CatalogListQuery) (tools.Page[tools.WordlistRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.WordlistRecord]{}, mcpErrors.ErrInternal
	}
	shape := struct{ Filter, OrderBy string }{strings.TrimSpace(query.Filter), strings.TrimSpace(query.OrderBy)}
	cursor, err := decodeListCursor(query.PageToken, "wordlists", query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.WordlistRecord]{}, err
	}
	result, err := reader.facade.ListContext(ctx, &catalogdto.WordlistListQuery{PaginationQuery: httpdto.PaginationQuery{PageSize: query.PageSize, PageToken: cursor.InnerToken}, Filter: shape.Filter, OrderBy: shape.OrderBy})
	if err != nil {
		return tools.Page[tools.WordlistRecord]{}, mapCatalogError(err)
	}
	if result == nil {
		return tools.Page[tools.WordlistRecord]{}, mcpErrors.ErrCommandFailed
	}
	items := make([]tools.WordlistRecord, 0, len(result.Wordlists))
	for _, item := range result.Wordlists {
		items = append(items, wordlistRecord(item))
	}
	next, err := wrapInnerCursor("wordlists", query.PageSize, shape, cursor.Page+1, result.NextPageToken)
	if err != nil {
		return tools.Page[tools.WordlistRecord]{}, err
	}
	return tools.Page[tools.WordlistRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

func (reader *wordlistReader) Get(ctx context.Context, id int) (tools.WordlistRecord, error) {
	if reader.facade == nil {
		return tools.WordlistRecord{}, mcpErrors.ErrInternal
	}
	item, err := reader.facade.GetByIDContext(ctx, id)
	if err != nil {
		return tools.WordlistRecord{}, mapCatalogError(err)
	}
	if item == nil {
		return tools.WordlistRecord{}, mcpErrors.ErrNotFound
	}
	return wordlistRecord(*item), nil
}

type serverLogReader struct{ service *systemapp.ServerLogService }
type agentLogReader struct {
	service *agentapp.LokiLogQueryService
	lookup  *agentapp.AgentFacade
}

func NewServerLogReader(service *systemapp.ServerLogService) tools.ServerLogReader {
	return &serverLogReader{service: service}
}

func NewAgentLogReader(service *agentapp.LokiLogQueryService, lookup *agentapp.AgentFacade) tools.AgentLogReader {
	return &agentLogReader{service: service, lookup: lookup}
}

func (reader *serverLogReader) List(ctx context.Context, query tools.LogQuery) (tools.LogPage, error) {
	if reader.service == nil {
		return tools.LogPage{}, mcpErrors.ErrInternal
	}
	result, err := reader.service.Query(ctx, systemapp.ServerLogQueryInput{Limit: query.PageSize, Cursor: query.PageToken, Direction: query.Direction})
	if err != nil {
		return tools.LogPage{}, mapLogError(err)
	}
	return serverLogPage(result), nil
}

func (reader *agentLogReader) List(ctx context.Context, query tools.AgentLogQuery) (tools.LogPage, error) {
	if reader.service == nil || reader.lookup == nil {
		return tools.LogPage{}, mcpErrors.ErrInternal
	}
	container := strings.TrimSpace(query.Container)
	if query.AgentID <= 0 || !validAgentContainer(container) {
		return tools.LogPage{}, mcpErrors.ErrInvalidInput
	}
	// The current deployment has one canonical Agent runtime container. The
	// database does not store arbitrary container names, so accepting anything
	// else would turn the Loki query into an unbound selector.
	if container != "lunafox-agent" {
		return tools.LogPage{}, mcpErrors.ErrInvalidInput
	}
	agent, err := reader.lookup.GetAgent(ctx, query.AgentID)
	if err != nil {
		return tools.LogPage{}, mapInvestigationReaderError(err, agentapp.ErrAgentNotFound)
	}
	// A nil,nil lookup result is not a valid agent reference. Treat it as a
	// missing resource instead of allowing the unbound log query to proceed.
	if agent == nil {
		return tools.LogPage{}, mcpErrors.ErrNotFound
	}
	result, err := reader.service.Query(ctx, agentapp.LokiLogQueryInput{AgentID: query.AgentID, Container: container, Limit: query.PageSize, Cursor: query.PageToken, Direction: query.Direction})
	if err != nil {
		return tools.LogPage{}, mapLogError(err)
	}
	return agentLogPage(result), nil
}

var agentContainerPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

func validAgentContainer(value string) bool {
	return agentContainerPattern.MatchString(strings.TrimSpace(value))
}

func serverLogPage(result systemapp.ServerLogQueryResult) tools.LogPage {
	items := make([]tools.LogRecord, 0, len(result.Logs))
	for _, item := range result.Logs {
		items = append(items, tools.LogRecord{ID: item.ID, TS: item.TS, TSNs: item.TSNs, Stream: item.Stream, Line: item.Line, Truncated: item.Truncated})
	}
	return tools.LogPage{Items: items, NextPageToken: result.NextCursor, PreviousPageToken: result.PreviousCursor, HasOlder: result.HasOlder, HasNewer: result.HasNewer, CaughtUp: result.CaughtUp, Gap: result.Gap, GapReason: result.GapReason}
}

func agentLogPage(result agentapp.LokiLogQueryResult) tools.LogPage {
	items := make([]tools.LogRecord, 0, len(result.Logs))
	for _, item := range result.Logs {
		items = append(items, tools.LogRecord{ID: item.ID, TS: item.TS, TSNs: item.TSNs, Stream: item.Stream, Line: item.Line, Truncated: item.Truncated})
	}
	return tools.LogPage{Items: items, NextPageToken: result.NextCursor, PreviousPageToken: result.PreviousCursor, HasOlder: result.HasOlder, HasNewer: result.HasNewer, CaughtUp: result.CaughtUp, Gap: result.Gap, GapReason: result.GapReason}
}

func organizationRecord(value identitydomain.OrganizationWithTargetCount) tools.OrganizationRecord {
	return tools.OrganizationRecord{ID: value.ID, Name: fmt.Sprintf("organizations/%d", value.ID), DisplayName: value.Name, Description: value.Description, CreatedAt: value.CreatedAt.UTC(), TargetCount: value.TargetCount}
}

func organizationTargetRecord(value identitydomain.OrganizationTargetRef) tools.OrganizationTargetRecord {
	return tools.OrganizationTargetRecord{ID: value.ID, Name: resourcenames.Target(value.ID), Type: value.Type, CreatedAt: value.CreatedAt.UTC(), LastScannedAt: utcPtr(value.LastScannedAt), DeletedAt: utcPtr(value.DeletedAt)}
}

func screenshotRecord(value assetdomain.Screenshot) tools.ScreenshotRecord {
	return tools.ScreenshotRecord{ID: value.ID, Name: fmt.Sprintf("targets/%d/screenshots/%d", value.TargetID, value.ID), TargetID: value.TargetID, URL: value.URL, StatusCode: value.StatusCode, CreatedAt: value.CreatedAt.UTC(), UpdatedAt: value.UpdatedAt.UTC()}
}

func workflowRecord(value catalogapp.ManagedScanWorkflow) tools.ScanWorkflowRecord {
	return tools.ScanWorkflowRecord{Name: resourcenames.ScanWorkflow(value.ScanWorkflowID), DisplayName: value.DisplayName, Description: value.Description, Stages: value.Stages, IsBuiltin: value.IsBuiltin, IsExecutable: value.IsExecutable, ETag: value.ETag, CreateTime: value.CreateTime.UTC(), UpdateTime: value.UpdateTime.UTC()}
}

func engineRecord(value catalogapp.EngineCatalogItem) tools.EngineRecord {
	record := tools.EngineRecord{Name: resourcenames.Engine(value.EngineID), EngineID: value.EngineID, ManifestVersion: value.ManifestVersion, Publisher: value.Publisher, PackageVersion: value.PackageVersion, ArtifactRef: value.ArtifactRef, PackageDigest: value.PackageDigest, EngineAPIMajor: value.EngineAPIMajor, SupportedTargetTypes: append([]string(nil), value.SupportedTargetTypes...), ExecutionResources: append([]string(nil), value.ExecutionResources...)}
	record.ConfigSections = make([]tools.EngineConfigSectionRecord, 0, len(value.ConfigSections))
	for _, section := range value.ConfigSections {
		mapped := tools.EngineConfigSectionRecord{ID: section.ID, DefaultEnabled: section.DefaultEnabled, RequiredEnabled: section.RequiredEnabled, Params: make([]tools.EngineConfigParamRecord, 0, len(section.Params))}
		for _, param := range section.Params {
			item := tools.EngineConfigParamRecord{Key: param.Key, Type: param.Type, Default: param.Default, Minimum: param.Minimum, Maximum: param.Maximum, MinLength: param.MinLength, MaxLength: param.MaxLength, MinItems: param.MinItems, MaxItems: param.MaxItems, Pattern: param.Pattern, Enum: append([]string(nil), param.Enum...)}
			if param.Resource != nil {
				item.Resource = &tools.EngineConfigParamResourceRecord{Kind: param.Resource.Kind}
			}
			mapped.Params = append(mapped.Params, item)
		}
		record.ConfigSections = append(record.ConfigSections, mapped)
	}
	return record
}

func wordlistRecord(value catalogdomain.Wordlist) tools.WordlistRecord {
	return tools.WordlistRecord{ID: value.ID, Name: resourcenames.Wordlist(value.ID), FileName: value.FileName, Description: value.Description, Tags: append([]string(nil), value.Tags...), FileSize: value.FileSize, LineCount: value.LineCount, FileHash: value.FileHash, CreatedAt: value.CreatedAt.UTC(), UpdatedAt: value.UpdatedAt.UTC()}
}

func wrapInnerCursor(resource string, pageSize int, shape any, page int, inner string) (string, error) {
	if inner == "" {
		return "", nil
	}
	return encodeNextCursor(resource, pageSize, shape, page, inner)
}

func mapCatalogError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if dberrors.IsRecordNotFound(err) || errors.Is(err, catalogapp.ErrTargetNotFound) || errors.Is(err, catalogapp.ErrWordlistNotFound) || errors.Is(err, catalogapp.ErrEngineNotFound) || errors.Is(err, catalogdomain.ErrScanWorkflowNotFound) {
		return mcpErrors.ErrNotFound
	}
	if errors.Is(err, catalogapp.ErrInvalidScanWorkflowPageToken) ||
		errors.Is(err, catalogapp.ErrInvalidScanWorkflowUpdate) ||
		errors.Is(err, catalogapp.ErrUnsupportedWordlistFilter) ||
		errors.Is(err, catalogapp.ErrUnsupportedWordlistOrderBy) ||
		errors.Is(err, catalogapp.ErrInvalidWordlistPageToken) {
		return mcpErrors.ErrInvalidInput
	}
	return mcpErrors.ErrCommandFailed
}

func mapLogError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, systemapp.ErrServerLogQueryTimeout) || errors.Is(err, agentapp.ErrLokiQueryTimeout) {
		return mcpErrors.ErrDeadlineExceeded
	}
	if errors.Is(err, systemapp.ErrServerLogCursorInvalid) || errors.Is(err, systemapp.ErrServerLogCursorQueryMismatch) || errors.Is(err, agentapp.ErrLogCursorInvalid) || errors.Is(err, agentapp.ErrLogCursorQueryMismatch) {
		return mcpErrors.ErrInvalidInput
	}
	if errors.Is(err, agentapp.ErrLokiContainerNotFound) {
		return mcpErrors.ErrNotFound
	}
	return mcpErrors.ErrCommandFailed
}
