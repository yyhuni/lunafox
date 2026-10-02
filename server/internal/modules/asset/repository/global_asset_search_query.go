package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	"gorm.io/gorm"
)

const globalAssetSearchStatementTimeout = "5s"

// executeGlobalAssetSearchQuery keeps the query table fixed by its caller and
// performs the full bounded page plus the capped match count in one short
// transaction. SET LOCAL is scoped to that transaction so a pooled PostgreSQL
// connection cannot leak the search timeout into a later unrelated request.
func executeGlobalAssetSearchQuery(db *gorm.DB, ctx context.Context, table string, query assetapp.GlobalAssetSearchStoreQuery, destination any) (int64, bool, error) {
	if db == nil {
		return 0, false, errors.New("global asset search database unavailable")
	}
	if ctx == nil {
		return 0, false, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return 0, false, err
	}
	if table != "website" && table != "endpoint" {
		return 0, false, fmt.Errorf("unsupported global asset search table %q", table)
	}
	if query.PageSize < 1 {
		return 0, false, fmt.Errorf("invalid global asset search page size")
	}

	transaction := db.WithContext(ctx).Begin()
	if transaction.Error != nil {
		return 0, false, mapGlobalAssetSearchRepositoryError(transaction.Error)
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback().Error
		}
	}()

	if transaction.Dialector.Name() == "postgres" {
		if err := transaction.Exec("SET LOCAL statement_timeout = '" + globalAssetSearchStatementTimeout + "'").Error; err != nil {
			return 0, false, mapGlobalAssetSearchRepositoryError(err)
		}
	}

	search := transaction.Table(table + " AS asset").
		Select("asset.*").
		Joins("JOIN target AS active_target ON active_target.id = asset.target_id AND active_target.deleted_at IS NULL")
	var err error
	search, err = applyGlobalAssetSearchPredicates(search, query.AST, transaction.Dialector.Name())
	if err != nil {
		return 0, false, err
	}
	if query.Cursor != nil {
		search = search.Where("(asset.created_at, asset.id) < (?, ?)", query.Cursor.CreatedAt.UTC(), query.Cursor.ID)
	}
	if err := search.Order("asset.created_at DESC").Order("asset.id DESC").Limit(query.PageSize + 1).Find(destination).Error; err != nil {
		return 0, false, mapGlobalAssetSearchRepositoryError(err)
	}
	totalSize, totalSizeCapped, err := countGlobalAssetSearchMatches(transaction, table, query)
	if err != nil {
		return 0, false, err
	}
	if err := transaction.Commit().Error; err != nil {
		return 0, false, mapGlobalAssetSearchRepositoryError(err)
	}
	committed = true
	return totalSize, totalSizeCapped, nil
}

// countGlobalAssetSearchMatches counts the whole match set with a hard LIMIT of
// cap+1 rows, so a trigram-wide match on a million-row table costs the same as
// a narrow one. The cursor predicate is intentionally absent: the count is a
// property of the query, not of the current page.
func countGlobalAssetSearchMatches(transaction *gorm.DB, table string, query assetapp.GlobalAssetSearchStoreQuery) (int64, bool, error) {
	matches := transaction.Table(table + " AS asset").
		Select("1").
		Joins("JOIN target AS active_target ON active_target.id = asset.target_id AND active_target.deleted_at IS NULL")
	// Dialector is embedded in gorm.Config, so its Name method is promoted to
	// *gorm.DB; spelling out transaction.Dialector.Name() trips staticcheck
	// QF1008 on newly linted lines.
	matches, err := applyGlobalAssetSearchPredicates(matches, query.AST, transaction.Name())
	if err != nil {
		return 0, false, err
	}
	matches = matches.Limit(assetapp.GlobalAssetSearchTotalSizeCap + 1)

	var count int64
	if err := transaction.Table("(?) AS capped_matches", matches).Select("COUNT(*)").Scan(&count).Error; err != nil {
		return 0, false, mapGlobalAssetSearchRepositoryError(err)
	}
	if count > assetapp.GlobalAssetSearchTotalSizeCap {
		return assetapp.GlobalAssetSearchTotalSizeCap, true, nil
	}
	return count, false, nil
}

func applyGlobalAssetSearchPredicates(db *gorm.DB, ast assetapp.GlobalAssetSearchAST, dialect string) (*gorm.DB, error) {
	if ast.Mode == assetapp.GlobalAssetSearchModePlainURL {
		return applyGlobalAssetSearchTextPredicate(db, "asset.url", assetapp.GlobalAssetSearchOperatorContains, ast.PlainURL, dialect), nil
	}
	if ast.Mode != assetapp.GlobalAssetSearchModeStructured || len(ast.Conditions) == 0 {
		return nil, fmt.Errorf("invalid typed global asset search AST")
	}
	for _, condition := range ast.Conditions {
		switch condition.Field {
		case assetapp.GlobalAssetSearchFieldURL:
			db = applyGlobalAssetSearchTextPredicate(db, "asset.url", condition.Operator, condition.Text, dialect)
		case assetapp.GlobalAssetSearchFieldHost:
			db = applyGlobalAssetSearchTextPredicate(db, "asset.host", condition.Operator, condition.Text, dialect)
		case assetapp.GlobalAssetSearchFieldTitle:
			db = applyGlobalAssetSearchTextPredicate(db, "asset.title", condition.Operator, condition.Text, dialect)
		case assetapp.GlobalAssetSearchFieldStatusCode:
			if condition.StatusCode == nil {
				return nil, fmt.Errorf("statusCode condition has no integer value")
			}
			db = db.Where("asset.status_code = ?", *condition.StatusCode)
		case assetapp.GlobalAssetSearchFieldTech:
			if dialect != "postgres" {
				return nil, fmt.Errorf("global tech search requires PostgreSQL")
			}
			// @> is an exact element-array containment predicate backed by the
			// existing tech GIN index; it never turns an array into substring text.
			db = db.Where("asset.tech @> ?::varchar(100)[]", pq.Array([]string{condition.Text}))
		case assetapp.GlobalAssetSearchFieldHasScreenshot:
			if condition.HasScreenshot == nil {
				return nil, fmt.Errorf("hasScreenshot condition has no boolean value")
			}
			// The probe rides the unique_screenshot_per_target(target_id, url)
			// index and never reads the image column; current-state screenshots
			// are unique per (target_id, url), so no new index is required.
			existence := "EXISTS"
			if !*condition.HasScreenshot {
				existence = "NOT EXISTS"
			}
			db = db.Where(existence + " (SELECT 1 FROM screenshot AS shot WHERE shot.target_id = asset.target_id AND shot.url = asset.url)")
		default:
			return nil, fmt.Errorf("unsupported global asset search field %q", condition.Field)
		}
	}
	return db, nil
}

func applyGlobalAssetSearchTextPredicate(db *gorm.DB, column string, operator assetapp.GlobalAssetSearchOperator, value, dialect string) *gorm.DB {
	if operator == assetapp.GlobalAssetSearchOperatorExact {
		return db.Where(column+" = ?", value)
	}
	pattern := globalAssetSearchContainsPattern(value)
	if dialect == "postgres" {
		return db.Where(column+" ILIKE ? ESCAPE '\\'", pattern)
	}
	// SQLite is used only by focused repository tests. PostgreSQL production
	// search uses ILIKE above so the configured trigram operator classes apply.
	return db.Where("LOWER("+column+") LIKE LOWER(?) ESCAPE '\\'", pattern)
}

func globalAssetSearchContainsPattern(value string) string {
	escaped := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(value)
	return "%" + escaped + "%"
}

func mapGlobalAssetSearchRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	var postgresErr *pgconn.PgError
	if errors.As(err, &postgresErr) && postgresErr.Code == "57014" {
		return assetapp.ErrGlobalAssetSearchTimeout
	}
	if strings.Contains(err.Error(), "SQLSTATE 57014") || strings.Contains(strings.ToLower(err.Error()), "statement timeout") {
		return assetapp.ErrGlobalAssetSearchTimeout
	}
	return err
}
