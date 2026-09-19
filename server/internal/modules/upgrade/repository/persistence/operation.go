package persistence

import "time"

// Operation is the database projection for a user-visible system upgrade.
// JSON columns contain bounded summaries only; secrets and command output are
// intentionally not represented by this model.
type Operation struct {
	ID                        string     `gorm:"column:id;type:uuid;primaryKey"`
	RequestID                 string     `gorm:"column:request_id;type:uuid;not null;uniqueIndex"`
	OperatorID                int        `gorm:"column:operator_id;not null;index"`
	ManifestID                string     `gorm:"column:manifest_id;size:200;not null"`
	ManifestDigest            string     `gorm:"column:manifest_digest;size:71;not null"`
	ReleaseVersion            string     `gorm:"column:release_version;size:64;not null"`
	CompatibilityRange        string     `gorm:"column:compatibility_range;size:128;not null"`
	MaintenanceWindowMinutes  int        `gorm:"column:maintenance_window_minutes;not null"`
	Status                    string     `gorm:"column:status;size:32;not null;index"`
	MigrationStatus           string     `gorm:"column:migration_status;size:32;not null"`
	MigrationType             string     `gorm:"column:migration_type;size:32;not null"`
	MigrationID               string     `gorm:"column:migration_id;size:200;not null;default:''"`
	MigrationChecksum         string     `gorm:"column:migration_checksum;size:71;not null;default:''"`
	CancelledScanCount        int        `gorm:"column:cancelled_scan_count;not null;default:0"`
	CancelledTaskCount        int        `gorm:"column:cancelled_task_count;not null;default:0"`
	AgentDesiredVersion       string     `gorm:"column:agent_desired_version;size:64;not null;default:''"`
	AgentTargetDigest         string     `gorm:"column:agent_target_digest;size:71;not null;default:''"`
	AgentExpectedCount        int        `gorm:"column:agent_expected_count;not null;default:0"`
	AgentReadyCount           int        `gorm:"column:agent_ready_count;not null;default:0"`
	AgentMissingCount         int        `gorm:"column:agent_missing_count;not null;default:0"`
	AgentUnhealthyCount       int        `gorm:"column:agent_unhealthy_count;not null;default:0"`
	AgentExpectations         []byte     `gorm:"column:agent_expectations;type:jsonb;not null"`
	AgentVerificationDeadline *time.Time `gorm:"column:agent_verification_deadline"`
	ObservedDigests           []byte     `gorm:"column:observed_digests;type:jsonb;not null"`
	StageTimes                []byte     `gorm:"column:stage_times;type:jsonb;not null"`
	ProgressEvents            []byte     `gorm:"column:progress_events;type:jsonb;not null"`
	Diagnostic                string     `gorm:"column:diagnostic;type:text;not null;default:''"`
	CreatedAt                 time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt                 time.Time  `gorm:"column:updated_at;not null"`
	CompletedAt               *time.Time `gorm:"column:completed_at"`
}

func (Operation) TableName() string { return "upgrade_operation" }
