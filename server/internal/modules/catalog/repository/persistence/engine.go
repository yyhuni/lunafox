package model

import (
	"time"

	"gorm.io/datatypes"
)

// Engine is the persistence projection of one successfully installed engine.
type Engine struct {
	ID             int            `gorm:"primaryKey;autoIncrement"`
	EngineID       string         `gorm:"column:engine_id;size:255;uniqueIndex"`
	Publisher      string         `gorm:"column:publisher;size:255"`
	PackageVersion string         `gorm:"column:package_version;size:255"`
	ArtifactRef    string         `gorm:"column:artifact_ref;size:1000;uniqueIndex"`
	PackageDigest  string         `gorm:"column:package_digest;size:71;uniqueIndex"`
	Manifest       datatypes.JSON `gorm:"column:manifest;type:jsonb"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

func (Engine) TableName() string { return "engine" }
