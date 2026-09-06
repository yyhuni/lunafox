// Package persistence contains the database-shaped notification records.
package persistence

import (
	"time"

	"gorm.io/datatypes"
)

// Outbox stores an immutable producer envelope and its worker state.
type Outbox struct {
	ID             int64          `gorm:"primaryKey;autoIncrement"`
	EventID        string         `gorm:"column:event_id;size:255;not null;uniqueIndex"`
	Kind           string         `gorm:"column:kind;size:64;not null"`
	PayloadVersion int            `gorm:"column:payload_version;not null"`
	SubjectName    string         `gorm:"column:subject_name;size:500;not null"`
	OccurredAt     time.Time      `gorm:"column:occurred_at;not null"`
	Priority       string         `gorm:"column:priority;size:16;not null"`
	Payload        datatypes.JSON `gorm:"column:payload;type:jsonb;not null"`
	Status         string         `gorm:"column:status;size:32;not null"`
	AvailableAt    time.Time      `gorm:"column:available_at;not null"`
	LeaseOwner     *string        `gorm:"column:lease_owner;size:128"`
	LeaseExpiresAt *time.Time     `gorm:"column:lease_expires_at"`
	FailureCode    string         `gorm:"column:failure_code;size:128;not null"`
	TerminalAt     *time.Time     `gorm:"column:terminal_at"`
	PublishedAt    *time.Time     `gorm:"column:published_at"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

func (Outbox) TableName() string { return "notification_outbox" }

// Fact is the shared canonical notification fact.
type Fact struct {
	ID                   int64          `gorm:"primaryKey;autoIncrement"`
	EventID              string         `gorm:"column:event_id;size:255;not null;uniqueIndex"`
	Kind                 string         `gorm:"column:kind;size:64;not null"`
	PayloadVersion       int            `gorm:"column:payload_version;not null"`
	SubjectName          string         `gorm:"column:subject_name;size:500;not null"`
	Category             string         `gorm:"column:category;size:32;not null"`
	Priority             string         `gorm:"column:priority;size:16;not null"`
	OccurredAt           time.Time      `gorm:"column:occurred_at;not null"`
	Payload              datatypes.JSON `gorm:"column:payload;type:jsonb;not null"`
	AudienceFrozenAt     *time.Time     `gorm:"column:audience_frozen_at"`
	DestinationsFrozenAt *time.Time     `gorm:"column:destinations_frozen_at"`
	CreatedAt            time.Time      `gorm:"column:created_at;autoCreateTime"`
}

func (Fact) TableName() string { return "notification_fact" }

// Recipient freezes audience and locale for fixed inbox delivery.
type Recipient struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	FactID    int64     `gorm:"column:fact_id;not null;uniqueIndex:idx_notification_recipient_fact_user,priority:1"`
	UserID    int       `gorm:"column:user_id;not null;uniqueIndex:idx_notification_recipient_fact_user,priority:2"`
	Locale    string    `gorm:"column:locale;size:2;not null"`
	Category  string    `gorm:"column:category;size:32;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Recipient) TableName() string { return "notification_recipient" }

// Inbox stores one user's frozen render snapshot and read state.
type Inbox struct {
	ID          int64      `gorm:"primaryKey;autoIncrement"`
	FactID      int64      `gorm:"column:fact_id;not null;uniqueIndex:idx_notification_inbox_fact_user,priority:1"`
	RecipientID int64      `gorm:"column:recipient_id;not null"`
	UserID      int        `gorm:"column:user_id;not null;uniqueIndex:idx_notification_inbox_fact_user,priority:2"`
	Name        string     `gorm:"column:name;size:500;not null;uniqueIndex"`
	Kind        string     `gorm:"column:kind;size:64;not null"`
	Category    string     `gorm:"column:category;size:32;not null"`
	Priority    string     `gorm:"column:priority;size:16;not null"`
	SubjectName string     `gorm:"column:subject_name;size:500;not null"`
	Locale      string     `gorm:"column:locale;size:2;not null"`
	Title       string     `gorm:"column:title;size:500;not null"`
	Message     string     `gorm:"column:message;type:text;not null"`
	OccurredAt  time.Time  `gorm:"column:occurred_at;not null"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	ReadAt      *time.Time `gorm:"column:read_at"`
}

func (Inbox) TableName() string { return "notification_inbox" }

// Destination is the installation-owned external delivery configuration.
type Destination struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	Provider   string    `gorm:"column:provider;size:16;not null;uniqueIndex"`
	Credential string    `gorm:"column:credential;type:text;not null"`
	Enabled    bool      `gorm:"column:enabled;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Destination) TableName() string { return "notification_destination" }

// DestinationSubscription is one exact canonical kind opt-in.
type DestinationSubscription struct {
	DestinationID int64     `gorm:"column:destination_id;primaryKey"`
	Kind          string    `gorm:"column:kind;size:64;primaryKey"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (DestinationSubscription) TableName() string { return "notification_destination_subscription" }

// Delivery stores a provider-independent render snapshot and lifecycle.
type Delivery struct {
	ID               int64          `gorm:"primaryKey;autoIncrement"`
	EventID          string         `gorm:"column:event_id;size:255;not null;uniqueIndex:idx_notification_delivery_event_destination,priority:1"`
	FactID           int64          `gorm:"column:fact_id;not null"`
	DestinationID    int64          `gorm:"column:destination_id;not null;uniqueIndex:idx_notification_delivery_event_destination,priority:2"`
	Provider         string         `gorm:"column:provider;size:16;not null"`
	Status           string         `gorm:"column:status;size:32;not null"`
	AttemptCount     int            `gorm:"column:attempt_count;not null"`
	FirstAttemptAt   *time.Time     `gorm:"column:first_attempt_at"`
	NextAttemptAt    time.Time      `gorm:"column:next_attempt_at;not null"`
	LeaseOwner       *string        `gorm:"column:lease_owner;size:128"`
	LeaseExpiresAt   *time.Time     `gorm:"column:lease_expires_at"`
	Locale           string         `gorm:"column:locale;size:2;not null"`
	TemplateVersion  int            `gorm:"column:template_version;not null"`
	Title            string         `gorm:"column:title;size:500;not null"`
	Message          string         `gorm:"column:message;type:text;not null"`
	ProviderSnapshot datatypes.JSON `gorm:"column:provider_snapshot;type:jsonb;not null"`
	TerminalAt       *time.Time     `gorm:"column:terminal_at"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

func (Delivery) TableName() string { return "notification_delivery" }

// DeliveryAttempt preserves redacted provider attempt outcomes.
type DeliveryAttempt struct {
	ID                int64     `gorm:"primaryKey;autoIncrement"`
	DeliveryID        int64     `gorm:"column:delivery_id;not null;uniqueIndex:idx_notification_delivery_attempt_number,priority:1"`
	AttemptNumber     int       `gorm:"column:attempt_number;not null;uniqueIndex:idx_notification_delivery_attempt_number,priority:2"`
	AttemptedAt       time.Time `gorm:"column:attempted_at;not null"`
	CompletedAt       time.Time `gorm:"column:completed_at;not null"`
	HTTPStatus        *int      `gorm:"column:http_status"`
	ErrorClass        string    `gorm:"column:error_class;size:128;not null"`
	ProviderRequestID string    `gorm:"column:provider_request_id;size:500;not null"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (DeliveryAttempt) TableName() string { return "notification_delivery_attempt" }
