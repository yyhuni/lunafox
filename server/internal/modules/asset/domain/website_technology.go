package domain

// WebsiteTechnology is the minimal current-only Website projection update.
// URL and Host identify a missing row; an existing row receives only Tech.
type WebsiteTechnology struct {
	URL  string
	Host string
	Tech []string
}
