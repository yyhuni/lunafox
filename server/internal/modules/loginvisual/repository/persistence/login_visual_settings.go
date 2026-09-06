package persistence

import "time"

// LoginVisualSettings stores the singleton draft and published pointers.
type LoginVisualSettings struct {
	ID               int       `gorm:"primaryKey;column:id"`
	DraftMediaID     *string   `gorm:"column:draft_media_id"`
	PublishedMediaID *string   `gorm:"column:published_media_id"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (LoginVisualSettings) TableName() string { return "login_visual_settings" }
