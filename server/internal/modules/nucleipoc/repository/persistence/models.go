package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Source struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	SourceType string     `gorm:"column:source_type;size:16;not null"`
	RepoURL    string     `gorm:"column:repo_url;type:text;not null"`
	IsActive   bool       `gorm:"column:is_active;not null;default:false"`
	CommitSHA  string     `gorm:"column:commit_sha;size:64;not null;default:''"`
	SyncedAt   *time.Time `gorm:"column:synced_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;not null"`
}

func (Source) TableName() string { return "nuclei_poc_source" }

type SyncTask struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey"`
	RequestID          uuid.UUID      `gorm:"column:request_id;type:uuid;not null;uniqueIndex"`
	RequestFingerprint string         `gorm:"column:request_fingerprint;size:64;not null"`
	SourceType         string         `gorm:"column:source_type;size:16;not null"`
	RepoURL            string         `gorm:"column:repo_url;type:text;not null"`
	SourceID           uuid.UUID      `gorm:"column:source_id;type:uuid;not null"`
	State              string         `gorm:"column:state;size:32;not null"`
	Phase              string         `gorm:"column:phase;size:32;not null"`
	FilesSeen          *int64         `gorm:"column:files_seen"`
	YAMLFilesSeen      *int64         `gorm:"column:yaml_files_seen"`
	TemplatesValidated *int64         `gorm:"column:templates_validated"`
	BytesRead          *int64         `gorm:"column:bytes_read"`
	CommitSHA          string         `gorm:"column:commit_sha;size:64;not null;default:''"`
	CommittedPOCCount  int64          `gorm:"column:committed_poc_count;not null;default:0"`
	FailureCode        string         `gorm:"column:failure_code;size:64;not null;default:''"`
	FailureSummary     string         `gorm:"column:failure_summary;size:500;not null;default:''"`
	Diagnostics        datatypes.JSON `gorm:"column:diagnostics;type:jsonb;not null"`
	CleanupStatus      string         `gorm:"column:cleanup_status;size:32;not null;default:pending"`
	WorkspaceKey       string         `gorm:"column:workspace_key;size:128;not null;default:''"`
	CreatedAt          time.Time      `gorm:"column:created_at;not null"`
	StartedAt          *time.Time     `gorm:"column:started_at"`
	CompletedAt        *time.Time     `gorm:"column:completed_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;not null"`
}

func (SyncTask) TableName() string { return "nuclei_poc_sync_task" }

type CandidateImport struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TaskID        uuid.UUID      `gorm:"column:task_id;type:uuid;not null;index"`
	SourceID      uuid.UUID      `gorm:"column:source_id;type:uuid;not null;index"`
	TemplateID    string         `gorm:"column:template_id;size:255;not null"`
	DisplayName   string         `gorm:"column:display_name;type:text;not null;default:''"`
	Severity      string         `gorm:"column:severity;size:16;not null;default:info"`
	Tags          datatypes.JSON `gorm:"column:tags;type:jsonb;not null"`
	Author        string         `gorm:"column:author;type:text;not null;default:''"`
	Description   string         `gorm:"column:description;type:text;not null;default:''"`
	CVE           datatypes.JSON `gorm:"column:cve;type:jsonb;not null"`
	CWE           datatypes.JSON `gorm:"column:cwe;type:jsonb;not null"`
	References    datatypes.JSON `gorm:"column:references;type:jsonb;not null"`
	Remediation   string         `gorm:"column:remediation;type:text;not null;default:''"`
	RelativePath  string         `gorm:"column:relative_path;type:text;not null"`
	ContentSHA256 string         `gorm:"column:content_sha256;size:64;not null"`
	Content       string         `gorm:"column:content;type:text;not null"`
	CreatedAt     time.Time      `gorm:"column:created_at;not null"`
}

func (CandidateImport) TableName() string { return "nuclei_poc_candidate_import" }

type POC struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey"`
	SourceID      uuid.UUID      `gorm:"column:source_id;type:uuid;not null;index"`
	TemplateID    string         `gorm:"column:template_id;size:255;not null;uniqueIndex"`
	DisplayName   string         `gorm:"column:display_name;type:text;not null;default:''"`
	Severity      string         `gorm:"column:severity;size:16;not null;default:info"`
	Tags          datatypes.JSON `gorm:"column:tags;type:jsonb;not null"`
	Author        string         `gorm:"column:author;type:text;not null;default:''"`
	Description   string         `gorm:"column:description;type:text;not null;default:''"`
	CVE           datatypes.JSON `gorm:"column:cve;type:jsonb;not null"`
	CWE           datatypes.JSON `gorm:"column:cwe;type:jsonb;not null"`
	References    datatypes.JSON `gorm:"column:references;type:jsonb;not null"`
	Remediation   string         `gorm:"column:remediation;type:text;not null;default:''"`
	RelativePath  string         `gorm:"column:relative_path;type:text;not null"`
	ContentSHA256 string         `gorm:"column:content_sha256;size:64;not null"`
	Content       string         `gorm:"column:content;type:text;not null"`
	// A newly materialized template has no operator history and starts disabled.
	// Promotion explicitly selects all columns so false is persisted verbatim.
	IsEnabled bool      `gorm:"column:is_enabled;not null;default:false"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

func (POC) TableName() string { return "nuclei_poc" }

type RequestTombstone struct {
	RequestID          uuid.UUID  `gorm:"column:request_id;type:uuid;primaryKey"`
	RequestFingerprint string     `gorm:"column:request_fingerprint;size:64;not null"`
	ReceivedAt         time.Time  `gorm:"column:received_at;not null"`
	TerminalAt         *time.Time `gorm:"column:terminal_at"`
	CreatedAt          time.Time  `gorm:"column:created_at;not null"`
}

func (RequestTombstone) TableName() string { return "nuclei_poc_sync_request_tombstone" }
