package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
	"net/url"
)

// WebsiteRepository handles website database operations
type WebsiteRepository struct {
	db *gorm.DB
}

// NewWebsiteRepository creates a new website repository
func NewWebsiteRepository(db *gorm.DB) *WebsiteRepository {
	return &WebsiteRepository{db: db}
}

// WebsiteFilterMapping defines field mapping for website filtering
var WebsiteFilterMapping = scope.FilterMapping{
	"url":    {Column: "url"},
	"host":   {Column: "host"},
	"title":  {Column: "title"},
	"status": {Column: "status_code", IsNumeric: true},
	"tech":   {Column: "tech", IsArray: true},
}

var websiteFilterMappingNormalized = scope.NormalizeFilterMapping(WebsiteFilterMapping)

// ExtractHostFromURL extracts host from URL
func ExtractHostFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
