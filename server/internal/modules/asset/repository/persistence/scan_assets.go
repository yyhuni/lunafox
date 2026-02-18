package model

import "time"

// HostPort represents a host-port mapping.
type HostPort struct {
	ID        int             `gorm:"primaryKey;autoIncrement" json:"id"`
	TargetID  int             `gorm:"column:target_id;not null;index:idx_hpm_target;uniqueIndex:unique_target_host_ip_port,priority:1" json:"targetId"`
	Host      string          `gorm:"column:host;size:1000;not null;index:idx_hpm_host;uniqueIndex:unique_target_host_ip_port,priority:2" json:"host"`
	IP        string          `gorm:"column:ip;type:inet;not null;index:idx_hpm_ip;uniqueIndex:unique_target_host_ip_port,priority:3" json:"ip"`
	Port      int             `gorm:"column:port;not null;index:idx_hpm_port;uniqueIndex:unique_target_host_ip_port,priority:4" json:"port"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime;index:idx_hpm_created_at" json:"createdAt"`
	Target    *AssetTargetRef `gorm:"foreignKey:TargetID" json:"target,omitempty"`
}

func (HostPort) TableName() string {
	return "host_port_mapping"
}
