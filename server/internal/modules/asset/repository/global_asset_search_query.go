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
// performs the full bounded page in one short transaction. SET LOCAL is scoped
// to that transaction so a pooled PostgreSQL connection cannot leak the search
// timeout into a later unrelated request.
func executeGlobalAssetSearchQuery(db *gorm.DB, ctx context.Context, table string, query assetapp.GlobalAssetSearchStoreQuery, destination any) error {
	if db == nil {
		return errors.New("global asset search database unavailable")
	}
	if ctx == nil {
		return context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if table != "website" && table != "endpoint" {
		return fmt.Errorf("unsupported global asset search table %q", table)
	}
	if query.PageSize < 1 {
		return fmt.Errorf("invalid global asset search page size")
	}

	transaction := db.WithContext(ctx).Begin()
	if transaction.Error != nil {
		return mapGlobalAssetSearchRepositoryError(transaction.Error)
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback().Error
		}
	}()

	if transaction.Dialector.Name() == "postgres" {
		if err := transaction.Exec("SET LOCAL statement_timeout = '" + globalAssetSearchStatementTimeout + "'").Error; err != nil {
			return mapGlobalAssetSearchRepositoryError(err)
		}
	}

	search := transaction.Table(table + " AS asset").
		Select("asset.*").
		Joins("JOIN target AS active_target ON active_target.id = asset.target_id AND active_target.deleted_at IS NULL")
	var err error
	search, err = applyGlobalAssetSearchPredicates(search, query.AST, transaction.Dialector.Name())
	if err != nil {
		return err
	}
	if query.Cursor != nil {
		search = search.Where("(asset.created_at, asset.id) < (?, ?)", query.Cursor.CreatedAt.UTC(), query.Cursor.ID)
	}
	if err := search.Order("asset.created_at DESC").Order("asset.id DESC").Limit(query.PageSize + 1).Find(destination).Error; err != nil {
		return mapGlobalAssetSearchRepositoryError(err)
	}
	if err := transaction.Commit().Error; err != nil {
		return mapGlobalAssetSearchRepositoryError(err)
	}
	committed = true
	return nil
}

func applyGlobalAssetSearchPredicates(db *gorm.DB, ast assetapp.GlobalAssetSearchAST, dialect string) (*gorm.DB, error) {
	if ast.Mode == assetapp.GlobalAssetSearchModePlainURL {
		return applyGlobalAssetSearchTextPredicate(db, "asset.url", assetapp.GlobalAssetSearchOperatorExact, ast.PlainURL, dialect), nil
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
