package repository

import (
	"fmt"
	"strings"
	"testing"

	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEndpointAndDirectoryWebsiteScopeIsAppliedToCountAndPageQueries(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		build       func(*gorm.DB) *gorm.DB
		applyOrder  func(*gorm.DB, string) *gorm.DB
		statusField string
	}{
		{
			name:        "endpoint",
			filter:      `websiteUrl=="https://api.acme.com/a" && statusCode=="200"`,
			statusField: "status_code::text =",
			build: func(db *gorm.DB) *gorm.DB {
				return db.Model(&model.Endpoint{}).
					Where("target_id = ?", 17).
					Scopes(applyEndpointListFilter(`websiteUrl=="https://api.acme.com/a" && statusCode=="200"`))
			},
			applyOrder: applyEndpointOrder,
		},
		{
			name:        "directory",
			filter:      `websiteUrl=="https://api.acme.com/a" && status=="200"`,
			statusField: "status::text =",
			build: func(db *gorm.DB) *gorm.DB {
				return db.Model(&model.Directory{}).
					Where("target_id = ?", 17).
					Scopes(applyDirectoryListFilter(`websiteUrl=="https://api.acme.com/a" && status=="200"`))
			},
			applyOrder: applyDirectoryOrder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newWebsiteScopeDryRunDB(t)

			var total int64
			countStatement := tt.build(db).Count(&total).Statement
			assertWebsiteScopeStatement(t, countStatement, tt.statusField, false)

			pageStatement := tt.build(db).
				Scopes(
					scope.WithPagination(2, 25),
					func(query *gorm.DB) *gorm.DB { return tt.applyOrder(query, "createdAt desc") },
				).
				Find(&[]model.Endpoint{}).Statement
			assertWebsiteScopeStatement(t, pageStatement, tt.statusField, true)
		})
	}
}

func newWebsiteScopeDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}
	return db
}

func assertWebsiteScopeStatement(t *testing.T, statement *gorm.Statement, statusField string, expectPagination bool) {
	t.Helper()
	if statement == nil {
		t.Fatal("expected generated statement")
	}

	query := strings.ToLower(statement.SQL.String())
	for _, expected := range []string{
		"target_id",
		"url ilike",
		"lower(substring(url from '^[^:]+://([^/' || chr(63) || '#]+)'))",
		"lower(substring(url from '^([^:]+)://'))",
		"coalesce(substring(url from '^[^:]+://[^/' || chr(63) || '#]+([^' || chr(63) || '#]*)'), '')",
		strings.ToLower(statusField),
	} {
		if !strings.Contains(query, expected) {
			t.Fatalf("generated SQL is missing %q: %s", expected, query)
		}
	}
	if expectPagination && (!strings.Contains(query, "order by created_at desc") || !strings.Contains(query, "limit") || !strings.Contains(query, "offset")) {
		t.Fatalf("page query must order and paginate after Website scope: %s", query)
	}
	if !expectPagination && strings.Contains(query, "limit") {
		t.Fatalf("count query must not paginate: %s", query)
	}

	arguments := fmt.Sprint(statement.Vars)
	for _, expected := range []string{"17", "api.acme.com", "https", "/a", "/a/%", "200"} {
		if !strings.Contains(arguments, expected) {
			t.Fatalf("generated SQL arguments are missing %q: %s", expected, arguments)
		}
	}
}
