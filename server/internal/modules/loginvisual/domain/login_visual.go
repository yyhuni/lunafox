package domain

import "time"

type MediaKind string

const (
	MediaKindImage MediaKind = "image"
	MediaKindVideo MediaKind = "video"
)

type Media struct {
	ID          string
	Kind        MediaKind
	ContentType string
	SizeBytes   int64
	Duration    time.Duration
	StorageKey  string
	PosterKey   string
	CreatedAt   time.Time
}

type Settings struct {
	Draft     *Media
	Published *Media
}

type PublicVisual struct {
	Kind      MediaKind
	MediaURL  string
	PosterURL string
	HasVisual bool
}
