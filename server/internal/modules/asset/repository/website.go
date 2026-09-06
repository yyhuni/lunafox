package repository

import (
	contractresults "github.com/yyhuni/lunafox/contracts/results"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
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
	"url":         {Column: "url", Exact: true},
	"statusCode":  {Column: "status_code", IsNumeric: true},
	"tech":        {Column: "tech", IsArray: true},
	"webserver":   {Column: "webserver"},
	"contentType": {Column: "content_type"},
	"vhost":       {Column: "vhost", NeedsCast: true},
}

var websiteFilterMappingNormalized = scope.NormalizeFilterMapping(WebsiteFilterMapping)

// ExtractHostFromURL derives the Host from a raw observed URL without parsing
// or rebuilding its payload portion.
func ExtractHostFromURL(rawURL string) string {
	host, err := contractresults.DeriveObservedAssetURLHost(rawURL)
	if err != nil {
		return ""
	}
	return host
}
