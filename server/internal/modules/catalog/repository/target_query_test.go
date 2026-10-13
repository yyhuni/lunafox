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

func TestApplyTargetFilterMapsScannedPresence(t *testing.T) {
	db := openTargetDryRunDB(t)

	scanned := applyTargetFilter(db.Model(&model.Target{}), `displayName="example" && scanned=="true"`).Find(&[]model.Target{}).Statement
	scannedSQL := scanned.SQL.String()
	if !strings.Contains(scannedSQL, "name ILIKE ?") || !strings.Contains(scannedSQL, "last_scanned_at IS NOT NULL") {
		t.Fatalf("expected name search and scanned presence, got SQL: %s", scannedSQL)
	}

	unscanned := applyTargetFilter(db.Model(&model.Target{}), `scanned=="false" && type=="domain"`).Find(&[]model.Target{}).Statement
	unscannedSQL := unscanned.SQL.String()
	if !strings.Contains(unscannedSQL, "last_scanned_at IS NULL") || strings.Contains(unscannedSQL, "last_scanned_at IS NOT NULL") || !strings.Contains(unscannedSQL, "type = ?") {
		t.Fatalf("expected unscanned and exact type predicates, got SQL: %s", unscannedSQL)
	}

	both := applyTargetFilter(db.Model(&model.Target{}), `(scanned=="true" || scanned=="false") && displayName="example"`).Find(&[]model.Target{}).Statement
	bothSQL := both.SQL.String()
	if strings.Contains(bothSQL, "last_scanned_at") || !strings.Contains(bothSQL, "name ILIKE ?") {
		t.Fatalf("selecting both scanned states must not add a presence predicate, got SQL: %s", bothSQL)
	}
}

func TestApplyTargetFilterRejectsUnsupportedScannedFilter(t *testing.T) {
	db := openTargetDryRunDB(t)

	for _, filter := range []string{
		`scanned="true"`,
		`scanned!="true"`,
		`scanned=="yes"`,
		`displayName="example" || scanned=="true"`,
		`scanned=="true" && scanned=="false"`,
	} {
		stmt := applyTargetFilter(db.Model(&model.Target{}), filter).Find(&[]model.Target{}).Statement
		if sql := stmt.SQL.String(); !strings.Contains(sql, "1 = 0") {
			t.Fatalf("filter %q must fail closed, got SQL: %s", filter, sql)
		}
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
