package persistence

import (
	"time"

	"gorm.io/datatypes"
)

// Policy is the direct five-column persistence model for blacklist_policy.
type Policy struct {
	ID        int            `gorm:"primaryKey;autoIncrement;column:id"`
	Scope     string         `gorm:"column:scope;not null"`
	TargetID  *int           `gorm:"column:target_id"`
	Patterns  datatypes.JSON `gorm:"column:patterns;type:jsonb;not null"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null"`
}

func (Policy) TableName() string {
	return "blacklist_policy"
}
