package repository

import (
	"strings"
	"testing"

	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApplyTargetFilterUsesDisplayNameContainsAndTypeExactMatch(t *testing.T) {
	db := openTargetDryRunDB(t)

	stmt := applyTargetFilter(db.Model(&model.Target{}), `displayName="example" && (type=="domain" || type=="ip")`).Find(&[]model.Target{}).Statement

	sql := stmt.SQL.String()
	if !strings.Contains(sql, "name ILIKE ?") || !strings.Contains(sql, "type = ?") {
		t.Fatalf("expected displayName contains and exact type predicates, got SQL: %s", sql)
	}
	if len(stmt.Vars) != 3 {
		t.Fatalf("expected search and two type variables, got %#v", stmt.Vars)
	}
}

func TestApplyTargetFilterRejectsFuzzyTypeFilter(t *testing.T) {
	db := openTargetDryRunDB(t)

	stmt := applyTargetFilter(db.Model(&model.Target{}), `type="domain"`).Find(&[]model.Target{}).Statement

	if sql := stmt.SQL.String(); !strings.Contains(sql, "1 = 0") {
		t.Fatalf("fuzzy type filters must fail closed, got SQL: %s", sql)
	}
}

func TestApplyTargetOrderIncludesStableTieBreakerAndNullsLast(t *testing.T) {
	db := openTargetDryRunDB(t)

	for _, testCase := range []struct {
		orderBy   string
		fragments []string
	}{
		{orderBy: "", fragments: []string{"created_at DESC", "id DESC"}},
		{orderBy: "displayName", fragments: []string{"name ASC", "id ASC"}},
		{orderBy: "displayName desc", fragments: []string{"name DESC", "id DESC"}},
		{orderBy: "createdAt", fragments: []string{"created_at ASC", "id ASC"}},
		{orderBy: "createdAt desc", fragments: []string{"created_at DESC", "id DESC"}},
		{orderBy: "lastScannedAt", fragments: []string{"last_scanned_at ASC NULLS LAST", "id ASC"}},
		{orderBy: "lastScannedAt desc", fragments: []string{"last_scanned_at DESC NULLS LAST", "id DESC"}},
	} {
		stmt := applyTargetOrder(db.Model(&model.Target{}), testCase.orderBy).Find(&[]model.Target{}).Statement
		sql := stmt.SQL.String()
		for _, fragment := range testCase.fragments {
			if !strings.Contains(sql, fragment) {
				t.Fatalf("orderBy %q expected SQL fragment %q in %s", testCase.orderBy, fragment, sql)
			}
		}
	}
}

func openTargetDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry-run db failed: %v", err)
	}
	return db
}
