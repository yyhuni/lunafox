package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
)

type CreateSyncInput struct {
	RequestID  uuid.UUID
	SourceType domain.SourceType
	RepoURL    string
}

// SetPOCActivationInput owns the target and optional selected scope. A nil
// Names list keeps the existing committed-catalog operation; a non-nil list is
// a selected command that must not be widened by a caller's query state.
type SetPOCActivationInput struct {
	Enabled bool
	Names   []string
}

// SetPOCActivationResult reports the requested target and only rows whose
// persisted value changed during the atomic collection assignment.
type SetPOCActivationResult struct {
	Enabled       bool
	AffectedCount int64
}

type POCListQuery struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
	Cursor    *POCCursor
	Offset    int
}

type POCCursor struct {
	Value      string
	TemplateID string
}

type POCListResult struct {
	Results       []domain.POC
	NextPageToken string
	TotalSize     int64
	HasMore       bool
	LastCursor    POCCursor
}

type CandidatePromotion struct {
	TaskID    uuid.UUID
	Source    domain.Source
	CommitSHA string
	SyncedAt  time.Time
	// CleanupStatus is recorded together with the collection replacement. A
	// residual workspace is a warning on an otherwise successful import, never
	// a reason to expose a partially promoted catalog.
	CleanupStatus string
}

// Store is the application boundary for persistence. Implementations own all
// SQL and transaction details; callers never receive persistence models.
type Store interface {
	GetCurrentSource(context.Context) (*domain.Source, error)
	CreateOrReplaySyncTask(context.Context, CreateSyncInput, string, time.Time) (*domain.SyncTask, error)
	GetSyncTask(context.Context, uuid.UUID) (*domain.SyncTask, error)
	// ClaimTask atomically transfers the initial task to the runner. A false
	// result means another process already claimed or terminalized the task.
	ClaimTask(context.Context, uuid.UUID, domain.SyncTaskState, time.Time) (bool, error)
	SetTaskWorkspaceKey(context.Context, uuid.UUID, string) error
	UpdateTaskProgress(context.Context, uuid.UUID, domain.SyncTaskState, domain.SyncCounters) error
	StageCandidate(context.Context, domain.CandidatePOC) error
	DeleteCandidatesForTask(context.Context, uuid.UUID) error
	PromoteCandidates(context.Context, CandidatePromotion) (int64, error)
	MarkTaskTerminal(context.Context, uuid.UUID, domain.SyncTaskState, string, string, domain.Diagnostics, string, time.Time) error
	ListPOCs(context.Context, POCListQuery) (*POCListResult, error)
	ListFilterOptions(context.Context, string) ([]domain.FilterOption, error)
	GetPOC(context.Context, string) (*domain.POC, error)
	UpdatePOCEnabled(context.Context, string, bool) (*domain.POC, error)
	SetPOCActivation(context.Context, bool, []string) (int64, error)
	DeleteExpiredTasks(context.Context, time.Time, int) (int64, error)
	DeleteExpiredTombstones(context.Context, time.Time, int) (int64, error)
	RecoverInterruptedTasks(context.Context, time.Time) (int64, error)
}

// ActiveWorkspaceKeyReader lets retention protect directories belonging to
// tasks that are still running. It is optional so small in-memory stores used
// by focused tests do not need to model cleanup bookkeeping.
type ActiveWorkspaceKeyReader interface {
	ActiveWorkspaceKeys(context.Context) ([]string, error)
}

type Workspace interface {
	Create(context.Context, uuid.UUID) (string, error)
	Remove(context.Context, string) error
	RemoveResiduals(context.Context) (int, error)
}

// ProtectedResidualWorkspace is implemented by the filesystem workspace. A
// retention pass must never remove a directory that an active task currently
// owns, even when the pass races with a runner.
type ProtectedResidualWorkspace interface {
	RemoveResidualsExcept(context.Context, map[string]struct{}) (int, error)
}

type GitRunner interface {
	Clone(context.Context, string, string) (commitSHA string, err error)
}

type SourceResolver interface {
	Validate(context.Context, domain.SourceType, string) (string, error)
}
