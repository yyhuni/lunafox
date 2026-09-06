package repository

import (
	"strings"
	"testing"

	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestParseExactTagFilter(t *testing.T) {
	value, ok := parseExactTagFilter(`tags=="fuzz"`)
	if !ok || value != "fuzz" {
		t.Fatalf("expected exact tag filter, got value=%q ok=%v", value, ok)
	}

	value, ok = parseExactTagFilter(`tags="subdomain"`)
	if !ok || value != "subdomain" {
		t.Fatalf("expected tag filter, got value=%q ok=%v", value, ok)
	}

	if _, ok := parseExactTagFilter(`displayName="fuzz"`); ok {
		t.Fatal("non-tag filter must not parse as tag filter")
	}
}

func TestApplyWordlistFilterUsesJSONBExactTagContainment(t *testing.T) {
	db := openWordlistDryRunDB(t)

	stmt := applyWordlistFilter(db.Model(&model.Wordlist{}), `tags=="custom-fuzz"`).Find(&[]model.Wordlist{}).Statement

	sql := stmt.SQL.String()
	if !strings.Contains(sql, "tags @> ?::jsonb") {
		t.Fatalf("expected JSONB containment filter, got SQL: %s", sql)
	}
	if len(stmt.Vars) != 1 || stmt.Vars[0] != `["custom-fuzz"]` {
		t.Fatalf("expected encoded tag array var, got %#v", stmt.Vars)
	}
}

func TestApplyWordlistFilterCombinesExactTagWithSearchFields(t *testing.T) {
	db := openWordlistDryRunDB(t)

	stmt := applyWordlistFilter(db.Model(&model.Wordlist{}), `tags=="custom" && fileName="admin.txt" && fileSize=="1024" && fileHash="abc"`).Find(&[]model.Wordlist{}).Statement

	sql := stmt.SQL.String()
	for _, fragment := range []string{
		"tags @> ?::jsonb",
		"file_name ILIKE ?",
		"file_size::text = ?",
		"file_hash ILIKE ?",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL fragment %q in %s", fragment, sql)
		}
	}
	if len(stmt.Vars) != 4 {
		t.Fatalf("expected four filter variables, got %#v", stmt.Vars)
	}
}

func TestApplyWordlistFilterRejectsFuzzyTagFilter(t *testing.T) {
	db := openWordlistDryRunDB(t)

	stmt := applyWordlistFilter(db.Model(&model.Wordlist{}), `tags="custom"`).Find(&[]model.Wordlist{}).Statement

	if sql := stmt.SQL.String(); !strings.Contains(sql, "1 = 0") {
		t.Fatalf("fuzzy tag filters must fail closed to avoid non-exact JSONB matching, got SQL: %s", sql)
	}
}

func TestApplyWordlistFilterAcceptsGroupedFacetOrWithSearch(t *testing.T) {
	db := openWordlistDryRunDB(t)

	stmt := applyWordlistFilter(db.Model(&model.Wordlist{}), `(tags=="fuzz" || tags=="subdomain") && fileName="admin.txt"`).Find(&[]model.Wordlist{}).Statement

	sql := stmt.SQL.String()
	if !strings.Contains(sql, "tags @> ?::jsonb") || !strings.Contains(sql, "file_name ILIKE ?") {
		t.Fatalf("expected grouped tag OR and fileName search SQL, got: %s", sql)
	}
	if strings.Contains(sql, "1 = 0") {
		t.Fatalf("grouped ordinary filter must not fail closed, got: %s", sql)
	}
}

func TestApplyWordlistOrderIncludesStableTieBreaker(t *testing.T) {
	db := openWordlistDryRunDB(t)

	stmt := applyWordlistOrder(db.Model(&model.Wordlist{}), "updatedAt desc").Find(&[]model.Wordlist{}).Statement
	sql := stmt.SQL.String()

	if !strings.Contains(sql, "updated_at DESC") || !strings.Contains(sql, "id DESC") {
		t.Fatalf("expected updatedAt desc order with stable id tie-breaker, got SQL: %s", sql)
	}
}

func openWordlistDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry-run db failed: %v", err)
	}
	return db
}
