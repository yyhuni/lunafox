package persistence

import "time"

// LoginVisualMedia stores the metadata for one managed login-page visual.
type LoginVisualMedia struct {
	ID          string    `gorm:"primaryKey;column:id"`
	Kind        string    `gorm:"column:kind"`
	ContentType string    `gorm:"column:content_type"`
	SizeBytes   int64     `gorm:"column:size_bytes"`
	DurationMS  int64     `gorm:"column:duration_ms"`
	StorageKey  string    `gorm:"column:storage_key"`
	PosterKey   string    `gorm:"column:poster_key"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (LoginVisualMedia) TableName() string { return "login_visual_media" }
