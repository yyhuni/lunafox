package domain

import "time"

// Fact is the immutable, shared materialization of one supported occurrence.
type Fact struct {
	ID                   int64
	EventID              string
	Kind                 Kind
	PayloadVersion       int
	Subject              string
	Category             Category
	Priority             Priority
	OccurredAt           time.Time
	Payload              []byte
	AudienceFrozenAt     *time.Time
	DestinationsFrozenAt *time.Time
	CreatedAt            time.Time
}

// Recipient freezes an active user's locale at fact materialization time,
// independently of later account changes.
type Recipient struct {
	ID        int64
	FactID    int64
	UserID    int
	Locale    Locale
	Category  Category
	CreatedAt time.Time
}

// InboxItem is a per-user projection with frozen display text and read state.
type InboxItem struct {
	ID          int64
	FactID      int64
	RecipientID int64
	UserID      int
	Name        string
	Kind        Kind
	Category    Category
	Priority    Priority
	Subject     string
	Locale      Locale
	Title       string
	Message     string
	OccurredAt  time.Time
	CreatedAt   time.Time
	ReadAt      *time.Time
}

// RenderSnapshot is frozen at projection creation and reused by inbox reads
// and every delivery attempt.
type RenderSnapshot struct {
	Locale          Locale
	TemplateVersion int
	Title           string
	Message         string
	ProviderPayload []byte
}
