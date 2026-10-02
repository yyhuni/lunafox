package repository

import (
	"context"
	"errors"
	"fmt"
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
	websites, websiteTotal, websiteCapped, err := NewWebsiteRepository(db).SearchGlobalWebsites(context.Background(), query)
	if err != nil {
		t.Fatalf("search websites: %v", err)
	}
	if len(websites) != 2 || websites[0].ID != 10 || websites[1].ID != 20 {
		t.Fatalf("only active current Website rows should match, got %+v", websites)
	}
	if websiteTotal != 2 || websiteCapped {
		t.Fatalf("Website total must be the exact match count, got total=%d capped=%v", websiteTotal, websiteCapped)
	}

	endpoints, endpointTotal, endpointCapped, err := NewEndpointRepository(db).SearchGlobalEndpoints(context.Background(), query)
	if err != nil {
		t.Fatalf("search endpoints: %v", err)
	}
	if len(endpoints) != 1 || endpoints[0].ID != 40 {
		t.Fatalf("Endpoint search must only read the Endpoint table, got %+v", endpoints)
	}
	if endpointTotal != 1 || endpointCapped {
		t.Fatalf("Endpoint total must be the exact match count, got total=%d capped=%v", endpointTotal, endpointCapped)
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

	containsURLAST, err := assetapp.ParseGlobalAssetSearchQuery(`url="example"`)
	if err != nil {
		t.Fatalf("parse URL contains: %v", err)
	}
	containsURL, _, _, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: containsURLAST, PageSize: 10})
	if err != nil || len(containsURL) != 3 || containsURL[0].ID != 3 || containsURL[1].ID != 2 || containsURL[2].ID != 1 {
		t.Fatalf("URL contains search must be case-insensitive and ordered: items=%+v err=%v", containsURL, err)
	}

	exactURLAST, err := assetapp.ParseGlobalAssetSearchQuery(`url=="https://Example.test/Admin"`)
	if err != nil {
		t.Fatalf("parse exact URL: %v", err)
	}
	exactURL, _, _, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: exactURLAST, PageSize: 2})
	if err != nil || len(exactURL) != 1 || exactURL[0].ID != 3 {
		t.Fatalf("URL double-equals search must use exact bytes: items=%+v err=%v", exactURL, err)
	}

	exactAST, err := assetapp.ParseGlobalAssetSearchQuery(`title=="Admin"`)
	if err != nil {
		t.Fatalf("parse exact: %v", err)
	}
	exact, _, _, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: exactAST, PageSize: 10})
	if err != nil || len(exact) != 1 || exact[0].ID != 3 {
		t.Fatalf("exact text search must remain case-sensitive: items=%+v err=%v", exact, err)
	}

	hostContainsAST, err := assetapp.ParseGlobalAssetSearchQuery(`host="example.test"`)
	if err != nil {
		t.Fatalf("parse host contains: %v", err)
	}
	allHosts, _, _, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: hostContainsAST, PageSize: 10})
	if err != nil || len(allHosts) != 3 || allHosts[0].ID != 3 || allHosts[1].ID != 2 || allHosts[2].ID != 1 {
		t.Fatalf("host contains search must retain descending keyset order: items=%+v err=%v", allHosts, err)
	}
	shortHostAST, err := assetapp.ParseGlobalAssetSearchQuery(`host="ex"`)
	if err != nil {
		t.Fatalf("parse two-character host contains: %v", err)
	}
	shortHosts, _, _, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: shortHostAST, PageSize: 10})
	if err != nil || len(shortHosts) != 3 {
		t.Fatalf("two-character host contains search must execute: items=%+v err=%v", shortHosts, err)
	}
	shortTitleAST, err := assetapp.ParseGlobalAssetSearchQuery(`title="Ad"`)
	if err != nil {
		t.Fatalf("parse two-character title contains: %v", err)
	}
	shortTitles, _, _, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: shortTitleAST, PageSize: 10})
	if err != nil || len(shortTitles) != 2 {
		t.Fatalf("two-character title contains search must execute: items=%+v err=%v", shortTitles, err)
	}
	cursor := &assetapp.GlobalAssetSearchCursor{CreatedAt: now, ID: 2}
	afterCursor, cursorTotal, cursorCapped, err := repo.SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: hostContainsAST, PageSize: 10, Cursor: cursor})
	if err != nil || len(afterCursor) != 1 || afterCursor[0].ID != 1 {
		t.Fatalf("cursor should retain only rows after same-timestamp ID 2: items=%+v err=%v", afterCursor, err)
	}
	if cursorTotal != 3 || cursorCapped {
		t.Fatalf("match count must ignore the cursor predicate, got total=%d capped=%v", cursorTotal, cursorCapped)
	}
}

func TestGlobalAssetSearchRepositoryFiltersByScreenshotExistence(t *testing.T) {
	db := newAssetRepositoryDB(t)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	if err := db.Create(&model.AssetTargetRef{ID: 2, Name: "second.example", Type: assetdomain.TargetTypeDomain}).Error; err != nil {
		t.Fatalf("seed second target: %v", err)
	}
	if err := db.Create([]model.Website{
		{ID: 1, TargetID: 1, URL: "https://shot.example.test/with-shot", Host: "shot.example.test", CreatedAt: now, Tech: pq.StringArray{}},
		{ID: 2, TargetID: 1, URL: "https://shot.example.test/without-shot", Host: "shot.example.test", CreatedAt: now.Add(-time.Second), Tech: pq.StringArray{}},
		{ID: 3, TargetID: 2, URL: "https://shot.example.test/with-shot", Host: "shot.example.test", CreatedAt: now.Add(-2 * time.Second), Tech: pq.StringArray{}},
	}).Error; err != nil {
		t.Fatalf("seed websites: %v", err)
	}
	// The screenshot belongs to target 1 only: website 3 shares the URL but a
	// different target, so existence must stay scoped to (target_id, url).
	if err := db.Create(&model.Screenshot{ID: 51, TargetID: 1, URL: "https://shot.example.test/with-shot", Image: []byte("png"), CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed screenshot: %v", err)
	}
	if err := db.Create(&model.Endpoint{ID: 40, TargetID: 1, URL: "https://shot.example.test/with-shot", Host: "shot.example.test", CreatedAt: now, Tech: pq.StringArray{}}).Error; err != nil {
		t.Fatalf("seed endpoint: %v", err)
	}

	trueAST, err := assetapp.ParseGlobalAssetSearchQuery(`url="shot.example.test" && hasScreenshot=="true"`)
	if err != nil {
		t.Fatalf("parse hasScreenshot true query: %v", err)
	}
	withShot, withShotTotal, withShotCapped, err := NewWebsiteRepository(db).SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: trueAST, PageSize: 10})
	if err != nil || len(withShot) != 1 || withShot[0].ID != 1 || withShotTotal != 1 || withShotCapped {
		t.Fatalf("hasScreenshot true must match only the (target,url) with a screenshot: items=%+v total=%d capped=%v err=%v", withShot, withShotTotal, withShotCapped, err)
	}

	falseAST, err := assetapp.ParseGlobalAssetSearchQuery(`url="shot.example.test" && hasScreenshot="false"`)
	if err != nil {
		t.Fatalf("parse hasScreenshot false query: %v", err)
	}
	withoutShot, withoutShotTotal, withoutShotCapped, err := NewWebsiteRepository(db).SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: falseAST, PageSize: 10})
	if err != nil || len(withoutShot) != 2 || withoutShot[0].ID != 2 || withoutShot[1].ID != 3 || withoutShotTotal != 2 || withoutShotCapped {
		t.Fatalf("hasScreenshot false must match assets without a screenshot: items=%+v total=%d capped=%v err=%v", withoutShot, withoutShotTotal, withoutShotCapped, err)
	}

	endpointAST, err := assetapp.ParseGlobalAssetSearchQuery(`hasScreenshot=="true"`)
	if err != nil {
		t.Fatalf("parse endpoint boolean query: %v", err)
	}
	endpoints, _, _, err := NewEndpointRepository(db).SearchGlobalEndpoints(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: endpointAST, PageSize: 10})
	if err != nil || len(endpoints) != 1 || endpoints[0].ID != 40 {
		t.Fatalf("hasScreenshot must share semantics on the Endpoint table: items=%+v err=%v", endpoints, err)
	}
}

func TestGlobalAssetSearchHasScreenshotSQLUsesExistenceProbe(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry-run db: %v", err)
	}
	ast, err := assetapp.ParseGlobalAssetSearchQuery(`host="api" && hasScreenshot=="false"`)
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
	statement := query.Order("asset.created_at DESC").Order("asset.id DESC").Limit(11).Find(&[]model.Website{}).Statement
	if statement == nil {
		t.Fatal("expected SQL statement")
	}
	sql := strings.ToLower(statement.SQL.String())
	for _, required := range []string{
		"not exists (select 1 from screenshot as shot",
		"shot.target_id = asset.target_id",
		"shot.url = asset.url",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("hasScreenshot SQL is missing %q: %s", required, sql)
		}
	}
	if strings.Contains(sql, "shot.image") {
		t.Fatalf("hasScreenshot probe must never read the image column: %s", sql)
	}
}

func TestGlobalAssetSearchRepositoryPreservesPercentBytesForURLPredicates(t *testing.T) {
	db := newAssetRepositoryDB(t)
	if err := db.Create([]model.Website{
		{ID: 1, TargetID: 1, URL: "https://example.test/100%25-complete", Tech: pq.StringArray{}},
		{ID: 2, TargetID: 1, URL: "https://example.test/100x25-complete", Tech: pq.StringArray{}},
	}).Error; err != nil {
		t.Fatalf("seed websites: %v", err)
	}
	ast, err := assetapp.ParseGlobalAssetSearchQuery(`url="100%25"`)
	if err != nil {
		t.Fatalf("parse percent URL contains query: %v", err)
	}
	items, _, _, err := NewWebsiteRepository(db).SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: ast, PageSize: 10})
	if err != nil || len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("URL contains wildcard bytes must remain literal: items=%+v err=%v", items, err)
	}

	exactAST, err := assetapp.ParseGlobalAssetSearchQuery(`url=="https://example.test/100%25-complete"`)
	if err != nil {
		t.Fatalf("parse exact percent URL query: %v", err)
	}
	exactItems, _, _, err := NewWebsiteRepository(db).SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: exactAST, PageSize: 10})
	if err != nil || len(exactItems) != 1 || exactItems[0].ID != 1 {
		t.Fatalf("exact percent URL must remain literal: items=%+v err=%v", exactItems, err)
	}
}

func TestGlobalAssetSearchRepositoryAllowsTwoCharacterURLContains(t *testing.T) {
	db := newAssetRepositoryDB(t)
	if err := db.Create(&model.Website{ID: 1, TargetID: 1, URL: "https://JD.com", Tech: pq.StringArray{}}).Error; err != nil {
		t.Fatalf("seed website: %v", err)
	}
	for _, rawQuery := range []string{"jd", `url="jd"`} {
		ast, err := assetapp.ParseGlobalAssetSearchQuery(rawQuery)
		if err != nil {
			t.Fatalf("parse two-character URL query %q: %v", rawQuery, err)
		}
		items, _, _, err := NewWebsiteRepository(db).SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: ast, PageSize: 10})
		if err != nil || len(items) != 1 || items[0].ID != 1 {
			t.Fatalf("two-character URL contains query %q must match case-insensitively: items=%+v err=%v", rawQuery, items, err)
		}
	}
}

func TestGlobalAssetSearchRepositoryCapsMatchCountAtReleaseCap(t *testing.T) {
	db := newAssetRepositoryDB(t)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	rows := make([]model.Website, 0, assetapp.GlobalAssetSearchTotalSizeCap+2)
	for id := 1; id <= assetapp.GlobalAssetSearchTotalSizeCap+1; id++ {
		rows = append(rows, model.Website{ID: id, TargetID: 1, URL: fmt.Sprintf("https://bulk.example.test/page-%d", id), Host: "bulk.example.test", CreatedAt: now.Add(-time.Duration(id) * time.Second), Tech: pq.StringArray{}})
	}
	if err := db.CreateInBatches(rows, 500).Error; err != nil {
		t.Fatalf("seed websites: %v", err)
	}
	ast, err := assetapp.ParseGlobalAssetSearchQuery(`url="bulk.example.test"`)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}

	pageSize := 10
	// The repository fetches pageSize+1 rows so the service can detect the next page.
	items, totalSize, totalSizeCapped, err := NewWebsiteRepository(db).SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: ast, PageSize: pageSize})
	if err != nil {
		t.Fatalf("search websites: %v", err)
	}
	if len(items) != pageSize+1 {
		t.Fatalf("page must fetch pageSize+1 for next-page detection, got %d items", len(items))
	}
	if totalSize != assetapp.GlobalAssetSearchTotalSizeCap || !totalSizeCapped {
		t.Fatalf("cap-exceeding match count must report cap+capped, got total=%d capped=%v", totalSize, totalSizeCapped)
	}

	cursor := &assetapp.GlobalAssetSearchCursor{CreatedAt: items[len(items)-1].CreatedAt, ID: items[len(items)-1].ID}
	_, pagedTotal, pagedCapped, err := NewWebsiteRepository(db).SearchGlobalWebsites(context.Background(), assetapp.GlobalAssetSearchStoreQuery{AST: ast, PageSize: pageSize, Cursor: cursor})
	if err != nil {
		t.Fatalf("search next page: %v", err)
	}
	if pagedTotal != assetapp.GlobalAssetSearchTotalSizeCap || !pagedCapped {
		t.Fatalf("capped count must stay whole-query on later pages, got total=%d capped=%v", pagedTotal, pagedCapped)
	}
}

func TestGlobalAssetSearchCountSQLIsCappedAndCursorFree(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry-run db: %v", err)
	}
	ast, err := assetapp.ParseGlobalAssetSearchQuery(`url="api" && statusCode="200"`)
	if err != nil {
		t.Fatalf("parse AST: %v", err)
	}
	matches := db.Table("website AS asset").
		Select("1").
		Joins("JOIN target AS active_target ON active_target.id = asset.target_id AND active_target.deleted_at IS NULL")
	matches, err = applyGlobalAssetSearchPredicates(matches, ast, "postgres")
	if err != nil {
		t.Fatalf("apply predicates: %v", err)
	}
	matches = matches.Limit(assetapp.GlobalAssetSearchTotalSizeCap + 1)
	statement := db.Table("(?) AS capped_matches", matches).Select("COUNT(*)").Scan(&countSink{}).Statement
	if statement == nil {
		t.Fatal("expected count SQL statement")
	}
	sql := strings.ToLower(statement.SQL.String())
	for _, required := range []string{
		"select count(*) from",
		"from website as asset",
		"asset.url ilike",
		"asset.status_code =",
		"limit 10001",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("capped count SQL is missing %q: %s", required, sql)
		}
	}
	for _, forbidden := range []string{"offset", "(asset.created_at, asset.id) <", "union"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("capped count SQL must not contain %q: %s", forbidden, sql)
		}
	}
}

type countSink struct {
	Count int64
}

func TestGlobalAssetSearchSQLHasNoCrossTableOrOffsetCountPath(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry-run db: %v", err)
	}
	ast, err := assetapp.ParseGlobalAssetSearchQuery(`url="api" && host="api" && title=="Admin" && statusCode="200" && tech="nginx"`)
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
		"asset.url ilike",
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
