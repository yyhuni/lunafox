package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// EndpointRepository handles endpoint database operations
type EndpointRepository struct {
	db *gorm.DB
}

// NewEndpointRepository creates a new endpoint repository
func NewEndpointRepository(db *gorm.DB) *EndpointRepository {
	return &EndpointRepository{db: db}
}

// EndpointFilterMapping defines field mapping for endpoint filtering
var EndpointFilterMapping = scope.FilterMapping{
	"url":         {Column: "url", Exact: true},
	"statusCode":  {Column: "status_code", IsNumeric: true},
	"tech":        {Column: "tech", IsArray: true},
	"webserver":   {Column: "webserver"},
	"contentType": {Column: "content_type"},
	"vhost":       {Column: "vhost", NeedsCast: true},
}

var endpointFilterMappingNormalized = scope.NormalizeFilterMapping(EndpointFilterMapping)
