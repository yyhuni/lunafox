package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// EndpointSnapshotRepository handles endpoint snapshot database operations
type EndpointSnapshotRepository struct {
	db *gorm.DB
}

// NewEndpointSnapshotRepository creates a new endpoint snapshot repository
func NewEndpointSnapshotRepository(db *gorm.DB) *EndpointSnapshotRepository {
	return &EndpointSnapshotRepository{db: db}
}

// EndpointSnapshotFilterMapping defines field mapping for endpoint snapshot filtering
var EndpointSnapshotFilterMapping = scope.FilterMapping{
	"url":         {Column: "url", Exact: true},
	"statusCode":  {Column: "status_code", IsNumeric: true},
	"tech":        {Column: "tech", IsArray: true},
	"webserver":   {Column: "webserver"},
	"contentType": {Column: "content_type"},
	"vhost":       {Column: "vhost", NeedsCast: true},
}

var endpointSnapshotFilterMappingNormalized = scope.NormalizeFilterMapping(EndpointSnapshotFilterMapping)
