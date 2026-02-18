package model

import "time"

// ScanInputTarget represents a scan input target entry.
type ScanInputTarget struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID    int       `gorm:"column:scan_id;not null;index:idx_scan_input_target_scan" json:"scanId"`
	Value     string    `gorm:"column:value;size:2000" json:"value"`
	InputType string    `gorm:"column:input_type;size:10;index:idx_scan_input_target_type" json:"inputType"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`

	Scan *Scan `gorm:"foreignKey:ScanID" json:"scan,omitempty"`
}

func (ScanInputTarget) TableName() string {
	return "scan_input_target"
}

const (
	InputTypeDomain = "domain"
	InputTypeIP     = "ip"
	InputTypeCIDR   = "cidr"
	InputTypeURL    = "url"
)
