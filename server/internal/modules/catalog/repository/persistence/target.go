package model

import "time"

// Organization is a local projection used for target preload.
type TargetOrganizationRef struct {
	ID          int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"column:name;size:300;index:idx_org_name" json:"name"`
	Description string     `gorm:"column:description;size:1000" json:"description"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_org_created_at" json:"createdAt"`
	DeletedAt   *time.Time `gorm:"column:deleted_at;index:idx_org_deleted_at" json:"-"`
}

func (TargetOrganizationRef) TableName() string {
	return "organization"
}

// Target represents a scan target.
type Target struct {
	ID            int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string     `gorm:"column:name;size:300;index:idx_target_name" json:"name"`
	Type          string     `gorm:"column:type;size:20;default:'domain';index:idx_target_type" json:"type"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_target_created_at" json:"createdAt"`
	LastScannedAt *time.Time `gorm:"column:last_scanned_at" json:"lastScannedAt"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index:idx_target_deleted_at" json:"-"`

	Organizations []TargetOrganizationRef `gorm:"many2many:organization_target;" json:"organizations,omitempty"`
}

func (Target) TableName() string {
	return "target"
}

const (
	TargetTypeDomain = "domain"
	TargetTypeIP     = "ip"
	TargetTypeCIDR   = "cidr"
)
