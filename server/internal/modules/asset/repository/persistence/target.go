package model

import "time"

// Target is a local projection used for asset preload.
type AssetTargetRef struct {
	ID            int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string     `gorm:"column:name;size:300;index:idx_target_name" json:"name"`
	Type          string     `gorm:"column:type;size:20;default:'domain';index:idx_target_type" json:"type"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_target_created_at" json:"createdAt"`
	LastScannedAt *time.Time `gorm:"column:last_scanned_at" json:"lastScannedAt"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index:idx_target_deleted_at" json:"-"`
}

func (AssetTargetRef) TableName() string {
	return "target"
}
