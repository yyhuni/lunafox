package results

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	DirectoryURLMaxBytes         = 2000
	DirectoryContentTypeMaxBytes = 1024
)

// Directory is one complete FFUF observation. Every field is required on the
// wire; zero is an observed value for all numeric fields, not an unknown value.
type Directory struct {
	URL           string `json:"url"`
	Status        int    `json:"status"`
	ContentLength int64  `json:"contentLength"`
	ContentType   string `json:"contentType"`
	Duration      int64  `json:"duration"`
}

func validateDirectory(item Directory) error {
	if _, err := ValidateObservedAssetURL(item.URL); err != nil {
		return fmt.Errorf("directory url is invalid: %w", err)
	}
	if item.Status < 0 || item.Status > 999 {
		return fmt.Errorf("directory status must be between 0 and 999")
	}
	if item.ContentLength < 0 {
		return fmt.Errorf("directory contentLength must be non-negative")
	}
	if !utf8.ValidString(item.ContentType) || strings.Contains(item.ContentType, "\x00") || len([]byte(item.ContentType)) > DirectoryContentTypeMaxBytes {
		return fmt.Errorf("directory contentType must be valid UTF-8, NUL-free, and at most %d bytes", DirectoryContentTypeMaxBytes)
	}
	if item.Duration < 0 {
		return fmt.Errorf("directory duration must be non-negative")
	}
	return nil
}
