package model

import (
	"time"

	"gorm.io/datatypes"
)

// ScanWorkflow mirrors the scan_workflow aggregate table. Stages retain their
// user-defined order in JSONB and are validated by the repository mapper.
type ScanWorkflow struct {
	ScanWorkflowID   string         `gorm:"column:scan_workflow_id;primaryKey;size:100"`
	DisplayName      string         `gorm:"column:display_name;size:300"`
	Description      string         `gorm:"column:description"`
	Stages           datatypes.JSON `gorm:"column:stages;type:jsonb"`
	IsBuiltin        bool           `gorm:"column:is_builtin"`
	DefinitionDigest *string        `gorm:"column:definition_digest;size:64"`
	RequestID        *string        `gorm:"column:request_id;type:uuid"`
	Version          int64          `gorm:"column:version"`
	CreateTime       time.Time      `gorm:"column:create_time;autoCreateTime"`
	UpdateTime       time.Time      `gorm:"column:update_time;autoUpdateTime"`
}

func (ScanWorkflow) TableName() string { return "scan_workflow" }
