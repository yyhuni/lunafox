package model

import (
	"strings"
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestGlobalAssetSearchPersistenceIndexesUseTrigramSearchIndexes(t *testing.T) {
	assertGlobalAssetSearchIndexes(t, Website{}, map[string]string{
		"idx_website_host_trgm":             "host gin_trgm_ops",
		"idx_website_title_trgm":            "title gin_trgm_ops",
		"idx_website_response_headers_trgm": "response_headers gin_trgm_ops",
	})
	assertGlobalAssetSearchIndexes(t, Endpoint{}, map[string]string{
		"idx_endpoint_host_trgm":             "host gin_trgm_ops",
		"idx_endpoint_title_trgm":            "title gin_trgm_ops",
		"idx_endpoint_response_headers_trgm": "response_headers gin_trgm_ops",
	})
}

func assertGlobalAssetSearchIndexes(t *testing.T, value any, expected map[string]string) {
	t.Helper()
	parsed, err := schema.Parse(value, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	indexes := parsed.ParseIndexes()
	for name, expression := range expected {
		index, ok := indexes[name]
		if !ok {
			t.Fatalf("missing search index %q", name)
		}
		if index.Type != "gin" || len(index.Fields) != 1 || index.Fields[0].Expression != expression {
			t.Fatalf("index %q must be GIN %q, got %+v", name, expression, index)
		}
	}
	for name, index := range indexes {
		if index.Type == "gin" && len(index.Fields) == 1 && strings.Contains(index.Fields[0].Expression, "response_body") {
			t.Fatalf("response body must not gain a GIN search index: %s", name)
		}
	}
}
