package persistence

import "time"

// LoginVisualDiscovery stores the account-owned discoverability unlock state.
type LoginVisualDiscovery struct {
	UserID     int       `gorm:"primaryKey;column:user_id"`
	UnlockedAt time.Time `gorm:"column:unlocked_at"`
}

func (LoginVisualDiscovery) TableName() string { return "login_visual_discovery" }
