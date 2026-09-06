package domain

import (
	"time"

	"github.com/google/uuid"
)

type SyncTaskState string

const (
	SyncTaskValidatingSource    SyncTaskState = "VALIDATING_SOURCE"
	SyncTaskCloning             SyncTaskState = "CLONING"
	SyncTaskScanningFiles       SyncTaskState = "SCANNING_FILES"
	SyncTaskValidatingTemplates SyncTaskState = "VALIDATING_TEMPLATES"
	SyncTaskCommitting          SyncTaskState = "COMMITTING"
	SyncTaskCleaning            SyncTaskState = "CLEANING"
	SyncTaskSucceeded           SyncTaskState = "SUCCEEDED"
	SyncTaskFailed              SyncTaskState = "FAILED"
)

func (state SyncTaskState) Valid() bool {
	switch state {
	case SyncTaskValidatingSource, SyncTaskCloning, SyncTaskScanningFiles,
		SyncTaskValidatingTemplates, SyncTaskCommitting, SyncTaskCleaning,
		SyncTaskSucceeded, SyncTaskFailed:
		return true
	default:
		return false
	}
}

func (state SyncTaskState) Terminal() bool {
	return state == SyncTaskSucceeded || state == SyncTaskFailed
}

type SyncCounters struct {
	FilesSeen          *int64
	YAMLFilesSeen      *int64
	TemplatesValidated *int64
	BytesRead          *int64
}

type SyncTask struct {
	ID                 uuid.UUID
	RequestID          uuid.UUID
	RequestFingerprint string
	SourceType         SourceType
	RepoURL            string
	SourceID           uuid.UUID
	State              SyncTaskState
	Phase              SyncTaskState
	Counters           SyncCounters
	CommitSHA          string
	CommittedPOCCount  int64
	FailureCode        string
	FailureSummary     string
	Diagnostics        Diagnostics
	CleanupStatus      CleanupStatus
	WorkspaceKey       string
	CreatedAt          time.Time
	StartedAt          *time.Time
	CompletedAt        *time.Time
	UpdatedAt          time.Time
}

type RequestTombstone struct {
	RequestID          uuid.UUID
	RequestFingerprint string
	ReceivedAt         time.Time
	TerminalAt         *time.Time
	CreatedAt          time.Time
}
