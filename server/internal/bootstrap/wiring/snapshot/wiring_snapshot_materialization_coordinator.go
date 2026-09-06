package snapshotwiring

import (
	"context"

	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
)

type snapshotMaterializationCoordinator struct {
	db       *gorm.DB
	scanRepo *scanrepo.ScanRepository
}

func newSnapshotMaterializationCoordinator(db *gorm.DB, scanRepo *scanrepo.ScanRepository) *snapshotMaterializationCoordinator {
	return &snapshotMaterializationCoordinator{db: db, scanRepo: scanRepo}
}

func (coordinator *snapshotMaterializationCoordinator) Materialize(ctx context.Context, scanID int, targetID int, persist func(context.Context) error) error {
	return coordinator.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txContext := dbtx.WithTransaction(ctx, tx)
		if err := persist(txContext); err != nil {
			return err
		}
		return coordinator.scanRepo.RefreshScanResultSummary(txContext, scanID, targetID)
	})
}
