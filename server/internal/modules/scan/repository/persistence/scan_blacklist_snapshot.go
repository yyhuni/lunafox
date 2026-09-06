package model

import "gorm.io/datatypes"

// ScanBlacklistSnapshot stores the immutable effective blacklist selected when
// one Scan was created. It is intentionally separate from Scan's hot row.
type ScanBlacklistSnapshot struct {
	ScanID   int            `gorm:"column:scan_id;primaryKey"`
	Patterns datatypes.JSON `gorm:"column:patterns;type:jsonb;not null"`
}

func (ScanBlacklistSnapshot) TableName() string {
	return "scan_blacklist_snapshot"
}
