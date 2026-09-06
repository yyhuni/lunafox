package dto

type Media struct {
	Kind        string `json:"kind"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	DurationMS  int64  `json:"durationMs,omitempty"`
}

type SettingsResponse struct {
	Draft      *Media `json:"draft,omitempty"`
	Published  *Media `json:"published,omitempty"`
	PreviewURL string `json:"previewUrl,omitempty"`
}

type PublicVisualResponse struct {
	Kind      string `json:"kind"`
	MediaURL  string `json:"mediaUrl,omitempty"`
	PosterURL string `json:"posterUrl,omitempty"`
}

type DiscoverabilityResponse struct {
	Unlocked bool `json:"unlocked"`
}
