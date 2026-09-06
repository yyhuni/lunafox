package model

import "time"

// OrganizationTargetRef is a local projection used by identity context queries.
type OrganizationTargetRef struct {
	ID            int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string     `gorm:"column:name;size:300;index:idx_target_name" json:"name"`
	Type          string     `gorm:"column:type;size:20;default:'domain';index:idx_target_type" json:"type"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_target_created_at" json:"createdAt"`
	LastScannedAt *time.Time `gorm:"column:last_scanned_at" json:"lastScannedAt"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index:idx_target_deleted_at" json:"-"`
}

func (OrganizationTargetRef) TableName() string {
	return "target"
}

// Organization represents an organization.
type Organization struct {
	ID          int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"column:name;size:300;index:idx_org_name" json:"name"`
	Description string     `gorm:"column:description;size:1000" json:"description"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_org_created_at" json:"createdAt"`
	DeletedAt   *time.Time `gorm:"column:deleted_at;index:idx_org_deleted_at" json:"-"`

	Targets []OrganizationTargetRef `gorm:"many2many:organization_target;" json:"targets,omitempty"`
}

func (Organization) TableName() string {
	return "organization"
}
