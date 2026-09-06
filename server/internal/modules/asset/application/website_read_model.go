package application

import "time"

// WebsiteScreenshotSummary is the metadata projection shown beside a Website.
// Image bytes remain owned by the dedicated Screenshot blob endpoint.
type WebsiteScreenshotSummary struct {
	ID         int
	Name       string
	URL        string
	StatusCode *int16
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// WebsiteReadModel combines the persisted Website with read-time evidence.
type WebsiteReadModel struct {
	Website    Website
	Screenshot *WebsiteScreenshotSummary
}
