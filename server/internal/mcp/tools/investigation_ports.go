package tools

import "context"

// OrganizationQuery is the bounded deployment-wide organization list shape.
// Organization is a business grouping only; it never narrows MCP trust scope.
type OrganizationQuery struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

// OrganizationReader exposes the read-only organization investigation surface.
type OrganizationReader interface {
	List(context.Context, OrganizationQuery) (Page[OrganizationRecord], error)
	Get(context.Context, int) (OrganizationRecord, error)
	ListTargets(context.Context, int, TargetQuery) (Page[OrganizationTargetRecord], error)
}

// TargetVulnerabilityReader is intentionally separate from the global reader
// so target scope cannot be accidentally omitted by a handler.
type TargetVulnerabilityReader interface {
	ListByTarget(context.Context, int, VulnerabilityQuery) (Page[VulnerabilityRecord], error)
}

// ScreenshotQuery is the public metadata-only screenshot list shape.
type ScreenshotQuery struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type ScreenshotReader interface {
	ListByTarget(context.Context, int, ScreenshotQuery) (Page[ScreenshotRecord], error)
	GetImage(context.Context, int, int) ([]byte, error)
}

type ScanWorkflowReader interface {
	List(context.Context, CatalogListQuery) (Page[ScanWorkflowRecord], error)
	Get(context.Context, string) (ScanWorkflowRecord, error)
}

type ScanWorkflowProfileReader interface {
	GetProfile(context.Context, string) (ScanWorkflowProfileRecord, error)
}

type EngineReader interface {
	List(context.Context, CatalogListQuery) (Page[EngineRecord], error)
	Get(context.Context, string) (EngineRecord, error)
}

type WordlistReader interface {
	List(context.Context, CatalogListQuery) (Page[WordlistRecord], error)
	Get(context.Context, int) (WordlistRecord, error)
}

type CatalogListQuery struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

type ServerLogReader interface {
	List(context.Context, LogQuery) (LogPage, error)
}

type AgentLogReader interface {
	List(context.Context, AgentLogQuery) (LogPage, error)
}

type LogQuery struct {
	PageSize  int
	PageToken string
	Direction string
}

type AgentLogQuery struct {
	AgentID   int
	Container string
	PageSize  int
	PageToken string
	Direction string
}

// VulnerabilityAction is the explicit review/unreview command boundary.
type VulnerabilityAction interface {
	SetReviewed(context.Context, VulnerabilityActionInput) (VulnerabilityActionRecord, error)
	BatchSetReviewed(context.Context, VulnerabilityBatchActionInput) (VulnerabilityBatchActionRecord, error)
}

type VulnerabilityActionInput struct {
	ID        int
	Name      string
	Action    string
	Reviewed  bool
	RequestID string
}

type VulnerabilityBatchActionInput struct {
	IDs       []int
	Names     []string
	Action    string
	Reviewed  bool
	RequestID string
}

// ScanStarter and OperationReader are deliberately narrow MCP ports. Their
// concrete implementation is supplied by bootstrap and may be backed by the
// normal Scan application service or a persistence-aware coordinator.
type ScanStarter interface {
	Start(context.Context, StartScanInput) (StartScanOutput, error)
}

type OperationReader interface {
	GetOperation(context.Context, string) (OperationRecord, error)
}
