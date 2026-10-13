package repository

import (
	"context"
	"database/sql"
	"errors"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"gorm.io/gorm"
)

type globalAssetExportSnapshot struct {
	tx    *gorm.DB
	table string
	ast   assetapp.GlobalAssetSearchAST
}

// GlobalAssetExportSnapshotStore opens Website and Endpoint snapshots over the
// same database connection used by the asset repositories.
type GlobalAssetExportSnapshotStore struct{ db *gorm.DB }

func NewGlobalAssetExportSnapshotStore(db *gorm.DB) *GlobalAssetExportSnapshotStore {
	return &GlobalAssetExportSnapshotStore{db: db}
}

func (store *GlobalAssetExportSnapshotStore) WithSnapshot(ctx context.Context, ast assetapp.GlobalAssetSearchAST, assetType assetapp.GlobalAssetSearchAssetType, visit func(assetapp.GlobalAssetExportSnapshot) error) error {
	return withGlobalAssetExportSnapshot(store.db, ctx, ast, assetType, visit)
}

func (snapshot *globalAssetExportSnapshot) ForEachWebsite(ctx context.Context, visit func(assetdomain.Website) error) error {
	if snapshot == nil || snapshot.table != "website" {
		return errors.New("website snapshot is unavailable")
	}
	return snapshot.eachWebsite(ctx, func(row model.Website) error { return visit(*websiteModelToDomain(&row)) })
}

func (snapshot *globalAssetExportSnapshot) ForEachEndpoint(ctx context.Context, visit func(assetdomain.Endpoint) error) error {
	if snapshot == nil || snapshot.table != "endpoint" {
		return errors.New("endpoint snapshot is unavailable")
	}
	return snapshot.eachEndpoint(ctx, func(row model.Endpoint) error { return visit(*endpointModelToDomain(&row)) })
}

func (snapshot *globalAssetExportSnapshot) eachWebsite(ctx context.Context, visit func(model.Website) error) error {
	query, err := snapshot.query(ctx)
	if err != nil {
		return err
	}
	rows, err := query.Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var row model.Website
		if err := snapshot.tx.ScanRows(rows, &row); err != nil {
			return err
		}
		if err := visit(row); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (snapshot *globalAssetExportSnapshot) eachEndpoint(ctx context.Context, visit func(model.Endpoint) error) error {
	query, err := snapshot.query(ctx)
	if err != nil {
		return err
	}
	rows, err := query.Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var row model.Endpoint
		if err := snapshot.tx.ScanRows(rows, &row); err != nil {
			return err
		}
		if err := visit(row); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (snapshot *globalAssetExportSnapshot) query(ctx context.Context) (*gorm.DB, error) {
	query := snapshot.tx.WithContext(ctx).Table(snapshot.table + " AS asset").Select("asset.*").
		Joins("JOIN target AS active_target ON active_target.id = asset.target_id AND active_target.deleted_at IS NULL")
	var err error
	query, err = applyGlobalAssetSearchPredicates(query, snapshot.ast, snapshot.tx.Name())
	if err != nil {
		return nil, err
	}
	return query.Order("asset.created_at DESC").Order("asset.id DESC"), nil
}

// WithSnapshot keeps the database transaction open for both encoding passes.
func (r *WebsiteRepository) WithSnapshot(ctx context.Context, ast assetapp.GlobalAssetSearchAST, assetType assetapp.GlobalAssetSearchAssetType, visit func(assetapp.GlobalAssetExportSnapshot) error) error {
	return withGlobalAssetExportSnapshot(r.db, ctx, ast, assetType, visit)
}

func (r *EndpointRepository) WithSnapshot(ctx context.Context, ast assetapp.GlobalAssetSearchAST, assetType assetapp.GlobalAssetSearchAssetType, visit func(assetapp.GlobalAssetExportSnapshot) error) error {
	return withGlobalAssetExportSnapshot(r.db, ctx, ast, assetType, visit)
}

func withGlobalAssetExportSnapshot(db *gorm.DB, ctx context.Context, ast assetapp.GlobalAssetSearchAST, assetType assetapp.GlobalAssetSearchAssetType, visit func(assetapp.GlobalAssetExportSnapshot) error) error {
	if db == nil || ctx == nil || visit == nil {
		return errors.New("global asset export snapshot unavailable")
	}
	table := string(assetType)
	if table != "website" && table != "endpoint" {
		return assetapp.ErrInvalidGlobalAssetExport
	}
	tx := db.WithContext(ctx).Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if tx.Error != nil {
		return tx.Error
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback().Error
		}
	}()
	// PostgreSQL accepts the explicit transaction declaration and enforces both
	// snapshot isolation and read-only mode. SQLite fixtures already receive
	// the same TxOptions from GORM but do not implement this PostgreSQL syntax.
	if tx.Name() == "postgres" {
		if err := tx.Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY").Error; err != nil {
			return err
		}
	}
	snapshot := &globalAssetExportSnapshot{tx: tx, table: table, ast: ast}
	if err := visit(snapshot); err != nil {
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	committed = true
	return nil
}
