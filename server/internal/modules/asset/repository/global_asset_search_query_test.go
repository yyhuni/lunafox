package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGlobalAssetSearchRepositoriesOnlyReadSelectedCurrentTableForActiveTargets(t *testing.T) {
	db := newAssetRepositoryDB(t)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	deletedAt := now.Add(-time.Hour)
	if err := db.Create([]model.AssetTargetRef{
		{ID: 2, Name: "second.example", Type: assetdomain.TargetTypeDomain},
		{ID: 3, Name: "deleted.example", Type: assetdomain.TargetTypeDomain, DeletedAt: &deletedAt},
	}).Error; err != nil {
		t.Fatalf("seed targets: %v", err)
	}
	if err := db.Create([]model.Website{
		{ID: 10, TargetID: 1, URL: "https://shared.example.test/login", Host: "one.example.test", Title: "One", CreatedAt: now, Tech: pq.StringArray{}},
		{ID: 20, TargetID: 2, URL: "https://shared.example.test/login", Host: "two.example.test", Title: "Two", CreatedAt: now.Add(-time.Second), Tech: pq.StringArray{}},
		{ID: 30, TargetID: 3, URL: "https://deleted.example.test/login", Host: "deleted.example.test", Title: "Deleted", CreatedAt: now.Add(-2 * time.Second), Tech: pq.StringArray{}},
	}).Error; err != nil {
		t.Fatalf("seed websites: %v", err)
	}
	if err := db.Create([]model.Endpoint{{ID: 40, TargetID: 1, URL: "https://shared.example.test/login", Host: "one.example.test", CreatedAt: now, Tech: pq.StringArray{}}}).Error; err != nil {
		t.Fatalf("seed endpoint: %v", err)
	}
	if err := db.Exec(`CREATE TABLE website_snapshot (id INTEGER PRIMARY KEY, url TEXT NOT NULL)`).Error; err != nil {
		t.Fatalf("create snapshot fixture: %v", err)
	}
	if err := db.Exec(`INSERT INTO website_snapshot (id, url) VALUES (99, 'https://snapshot-only.example.test/login')`).Error; err != nil {
		t.Fatalf("seed snapshot fixture: %v", err)
	}

	ast, err := assetapp.ParseGlobalAssetSearchQuery("https://shared.example.test/login")
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	query := assetapp.GlobalAssetSearchStoreQuery{AST: ast, PageSize: 10}
	websites, err := NewWebsiteRepository(db).SearchGlobalWebsites(context.Background(), query)
	if err != nil {
		t.Fatalf("search websites: %v", err)
	}
	if len(websites) != 2 || websites[0].ID != 10 || websites[1].ID != 20 {
		t.Fatalf("only active current Website rows should match, got %+v", websites)
	}

	endpoints, err := NewEndpointRepository(db).SearchGlobalEndpoints(context.Background(), query)
	if err != nil {
		t.Fatalf("search endpoints: %v", err)
	}
	if len(endpoints) != 1 || endpoints[0].ID != 40 {
		t.Fatalf("Endpoint search must only read the Endpoint table, got %+v", endpoints)
	}
}

func TestGlobalAssetSearchRepositoryUsesStableKeysetAndFieldSemantics(t *testing.T) {
	db := newAssetRepositoryDB(t)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	status := 200
	if err := db.Create([]model.Website{
		{ID: 3, TargetID: 1, URL: "https://Example.test/Admin", Host: "API.Example.test", Title: "Admin", StatusCode: &status, CreatedAt: now, Tech: pq.StringArray{}},
		{ID: 2, TargetID: 1, URL: "https://example.test/admin", Host: "api.example.test", Title: "admin", StatusCode: &status, CreatedAt: now, Tech: pq.StringArray{}},
		{ID: 1, TargetID: 1, URL: "https://example.test/other", Host: "other.example.test", Title: "Other", StatusCode: &status, CreatedAt: now.Add(-time.Second), Tech: pq.StringArray{}},
	}).Error; err != nil {
		t.Fatalf("seed websites: %v", err)
	}
	repo := NewWebsiteRepository(db)

	exactURLAST, err := assetapp.ParseGlobalAssetSearchQuery(`url="https://Example.test/Admin"`)
	if err != nil {
		t.Fatalf("parse exact URL: %v", err)
	}
	exactURL, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: exactURLAST, PageSize: 2})
	if err != nil || len(exactURL) != 1 || exactURL[0].ID != 3 {
		t.Fatalf("URL search must use exact bytes: items=%+v err=%v", exactURL, err)
	}

	exactAST, err := assetapp.ParseGlobalAssetSearchQuery(`title=="Admin"`)
	if err != nil {
		t.Fatalf("parse exact: %v", err)
	}
	exact, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: exactAST, PageSize: 10})
	if err != nil || len(exact) != 1 || exact[0].ID != 3 {
		t.Fatalf("exact text search must remain case-sensitive: items=%+v err=%v", exact, err)
	}

	hostContainsAST, err := assetapp.ParseGlobalAssetSearchQuery(`host="example.test"`)
	if err != nil {
		t.Fatalf("parse host contains: %v", err)
	}
	allHosts, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: hostContainsAST, PageSize: 10})
	if err != nil || len(allHosts) != 3 || allHosts[0].ID != 3 || allHosts[1].ID != 2 || allHosts[2].ID != 1 {
		t.Fatalf("host contains search must retain descending keyset order: items=%+v err=%v", allHosts, err)
	}
	cursor := &assetapp.GlobalAssetSearchCursor{CreatedAt: now, ID: 2}
	afterCursor, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: hostContainsAST, PageSize: 10, Cursor: cursor})
	if err != nil || len(afterCursor) != 1 || afterCursor[0].ID != 1 {
		t.Fatalf("cursor should retain only rows after same-timestamp ID 2: items=%+v err=%v", afterCursor, err)
	}
}

func TestGlobalAssetSearchRepositoryPreservesPercentBytesForExactURL(t *testing.T) {
	db := newAssetRepositoryDB(t)
	if err := db.Create([]model.Website{
		{ID: 1, TargetID: 1, URL: "https://example.test/100%25-complete", Tech: pq.StringArray{}},
		{ID: 2, TargetID: 1, URL: "https://example.test/100x25-complete", Tech: pq.StringArray{}},
	}).Error; err != nil {
		t.Fatalf("seed websites: %v", err)
	}
	ast, err := assetapp.ParseGlobalAssetSearchQuery(`url="https://example.test/100%25-complete"`)
	if err != nil {
		t.Fatalf("parse exact percent URL query: %v", err)
	}
	items, err := NewWebsiteRepository(db).SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: ast, PageSize: 10})
	if err != nil || len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("exact percent URL must remain literal: items=%+v err=%v", items, err)
	}
}

func TestGlobalAssetSearchSQLHasNoCrossTableOrOffsetCountPath(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry-run db: %v", err)
	}
	ast, err := assetapp.ParseGlobalAssetSearchQuery(`host="api" && title=="Admin" && statusCode="200" && tech="nginx"`)
	if err != nil {
		t.Fatalf("parse AST: %v", err)
	}
	query := db.Table("website AS asset").
		Select("asset.*").
		Joins("JOIN target AS active_target ON active_target.id = asset.target_id AND active_target.deleted_at IS NULL")
	query, err = applyGlobalAssetSearchPredicates(query, ast, "postgres")
	if err != nil {
		t.Fatalf("apply predicates: %v", err)
	}
	statement := query.
		Where("(asset.created_at, asset.id) < (?, ?)", time.Now().UTC(), 17).
		Order("asset.created_at DESC").
		Order("asset.id DESC").
		Limit(11).
		Find(&[]model.Website{}).Statement
	if statement == nil {
		t.Fatal("expected SQL statement")
	}
	sql := strings.ToLower(statement.SQL.String())
	for _, required := range []string{
		"from website as asset",
		"join target as active_target",
		"active_target.deleted_at is null",
		"asset.host ilike",
		"asset.title =",
		"asset.status_code =",
		"asset.tech @>",
		"(asset.created_at, asset.id) <",
		"order by asset.created_at desc,asset.id desc",
		"limit 11",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("generated search SQL is missing %q: %s", required, sql)
		}
	}
	for _, forbidden := range []string{"union", "offset", "count(", "vulnerability"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("generated search SQL must not contain %q: %s", forbidden, sql)
		}
	}
}

func TestMapGlobalAssetSearchRepositoryErrorMapsPostgresCancellation(t *testing.T) {
	if !errors.Is(mapGlobalAssetSearchRepositoryError(errors.New("ERROR: canceling statement due to statement timeout (SQLSTATE 57014)")), assetapp.ErrGlobalAssetSearchTimeout) {
		t.Fatal("statement timeout must not cross the repository boundary")
	}
}
