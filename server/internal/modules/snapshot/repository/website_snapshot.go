package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// WebsiteSnapshotRepository handles website snapshot database operations
type WebsiteSnapshotRepository struct {
	db *gorm.DB
}

// NewWebsiteSnapshotRepository creates a new website snapshot repository
func NewWebsiteSnapshotRepository(db *gorm.DB) *WebsiteSnapshotRepository {
	return &WebsiteSnapshotRepository{db: db}
}

// WebsiteSnapshotFilterMapping defines field mapping for website snapshot filtering
var WebsiteSnapshotFilterMapping = scope.FilterMapping{
	"url":       {Column: "url"},
	"host":      {Column: "host"},
	"title":     {Column: "title"},
	"status":    {Column: "status_code", IsNumeric: true},
	"webserver": {Column: "webserver"},
	"tech":      {Column: "tech", IsArray: true},
}

var websiteSnapshotFilterMappingNormalized = scope.NormalizeFilterMapping(WebsiteSnapshotFilterMapping)
