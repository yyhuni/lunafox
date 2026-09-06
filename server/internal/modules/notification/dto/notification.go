// Package dto defines the notification HTTP boundary shapes.
package dto

import "time"

// InboxListQuery uses AIP-style pageSize/pageToken pagination only.
type InboxListQuery struct {
	PageSize  int    `form:"pageSize"`
	PageToken string `form:"pageToken"`
}

// NotificationResponse is a frozen user-owned inbox projection.
type NotificationResponse struct {
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	Category   string     `json:"category"`
	Priority   string     `json:"priority"`
	Subject    string     `json:"subject"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	OccurredAt time.Time  `json:"occurredAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	ReadAt     *time.Time `json:"readAt,omitempty"`
}

// NotificationListResponse is the current-user durable inbox list shape.
type NotificationListResponse struct {
	Results       []NotificationResponse `json:"results"`
	NextPageToken string                 `json:"nextPageToken"`
	TotalSize     int64                  `json:"totalSize"`
}

// UnreadCountResponse is the authoritative unread-state response.
type UnreadCountResponse struct {
	UnreadCount int64 `json:"unreadCount"`
}

// LocaleResponse returns the persisted notification locale only.
type LocaleResponse struct {
	Locale string `json:"locale"`
}

// LocaleUpdateRequest changes the current user's persisted locale.
type LocaleUpdateRequest struct {
	Locale string `json:"locale" binding:"required"`
}

// DestinationResponse returns a complete authorized installation credential.
type DestinationResponse struct {
	Provider              string   `json:"provider"`
	Credential            string   `json:"credential"`
	Enabled               bool     `json:"enabled"`
	Subscriptions         []string `json:"subscriptions"`
	RequiresWebhookUpdate bool     `json:"requiresWebhookUpdate"`
}

// DestinationListResponse lists the fixed external destination settings.
type DestinationListResponse struct {
	Results        []DestinationResponse `json:"results"`
	SupportedKinds []string              `json:"supportedKinds"`
}

// DestinationUpdateRequest updates one path-selected provider only.
type DestinationUpdateRequest struct {
	Credential    string   `json:"credential"`
	Enabled       *bool    `json:"enabled"`
	Subscriptions []string `json:"subscriptions"`
}

// DestinationTestDeliveryRequest uses an unsaved credential only for the
// one-shot command; it carries no enablement or subscription state.
type DestinationTestDeliveryRequest struct {
	Credential string `json:"credential"`
}

// DestinationTestDeliveryResponse exposes a fixed, credential-safe terminal
// category. Adapter details must never cross this HTTP boundary.
type DestinationTestDeliveryResponse struct {
	Result string `json:"result"`
}
