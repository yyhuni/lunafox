package catalogwiring_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	catalogwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/catalog"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBatchCreateTargetsSharesTransactionAndAssociatesExistingTargets(t *testing.T) {
	db := openTargetBatchTransactionDB(t)
	targetRepo := catalogrepo.NewTargetRepository(db)
	organizationRepo := identityrepo.NewOrganizationRepository(db)
	targetStore := catalogwiring.NewCatalogTargetCommandStoreAdapter(targetRepo)
	organizationStore := catalogwiring.NewCatalogOrganizationTargetBindingStoreAdapter(organizationRepo)
	service := catalogapp.NewTargetCommandService(
		targetStore,
		organizationStore,
		catalogwiring.NewCatalogTransactionCoordinator(db),
	)

	organization := newOrganization(t, db, "Platform")
	existing := &catalogdomain.Target{Name: "existing.example.com", Type: catalogdomain.TargetTypeDomain}
	if err := targetRepo.Create(existing); err != nil {
		t.Fatalf("seed existing target: %v", err)
	}

	result, err := service.BatchCreateTargetsContext(context.Background(), []string{
		" existing.example.com ",
		"new.example.com",
		"new.example.com",
		"***",
	}, &organization.ID)
	if err != nil {
		t.Fatalf("batch create: %v", err)
	}
	if result.CreatedCount != 1 || result.FailedCount != 1 || !result.AssociationCompleted {
		t.Fatalf("unexpected batch result: %+v", result)
	}
	if got := countRows(t, db, "target"); got != 2 {
		t.Fatalf("target rows = %d, want 2", got)
	}
	if got := countRows(t, db, "blacklist_policy"); got != 2 {
		t.Fatalf("blacklist policies = %d, want 2", got)
	}
	if got := countOrganizationTargets(t, db, organization.ID); got != 2 {
		t.Fatalf("organization relationships = %d, want 2", got)
	}
}

func TestBatchCreateTargetsRollsBackWhenAssociationFails(t *testing.T) {
	db := openTargetBatchTransactionDB(t)
	targetRepo := catalogrepo.NewTargetRepository(db)
	organizationRepo := identityrepo.NewOrganizationRepository(db)
	organization := newOrganization(t, db, "Rollback")
	baseBinding := catalogwiring.NewCatalogOrganizationTargetBindingStoreAdapter(organizationRepo)
	service := catalogapp.NewTargetCommandService(
		catalogwiring.NewCatalogTargetCommandStoreAdapter(targetRepo),
		failingBindingStore{delegate: baseBinding},
		catalogwiring.NewCatalogTransactionCoordinator(db),
	)

	_, err := service.BatchCreateTargetsContext(context.Background(), []string{"rollback.example.com"}, &organization.ID)
	if !errors.Is(err, catalogapp.ErrTargetOrgBindingFail) {
		t.Fatalf("association error = %v, want ErrTargetOrgBindingFail", err)
	}
	if got := countRows(t, db, "target"); got != 0 {
		t.Fatalf("association failure left %d targets", got)
	}
	if got := countRows(t, db, "blacklist_policy"); got != 0 {
		t.Fatalf("association failure left %d policies", got)
	}
	if got := countOrganizationTargets(t, db, organization.ID); got != 0 {
		t.Fatalf("association failure left %d relationships", got)
	}
}

func TestBatchCreateTargetsRollsBackWhenContextCancelsBeforeCommit(t *testing.T) {
	db := openTargetBatchTransactionDB(t)
	targetRepo := catalogrepo.NewTargetRepository(db)
	organizationRepo := identityrepo.NewOrganizationRepository(db)
	organization := newOrganization(t, db, "Cancellation")
	ctx, cancel := context.WithCancel(context.Background())
	service := catalogapp.NewTargetCommandService(
		catalogwiring.NewCatalogTargetCommandStoreAdapter(targetRepo),
		catalogwiring.NewCatalogOrganizationTargetBindingStoreAdapter(organizationRepo),
		cancellingTransactionCoordinator{db: db, cancel: cancel},
	)

	_, err := service.BatchCreateTargetsContext(ctx, []string{"cancel.example.com"}, &organization.ID)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v, want context.Canceled", err)
	}
	if got := countRows(t, db, "target"); got != 0 {
		t.Fatalf("cancelled batch left %d targets", got)
	}
	if got := countRows(t, db, "blacklist_policy"); got != 0 {
		t.Fatalf("cancelled batch left %d policies", got)
	}
	if got := countOrganizationTargets(t, db, organization.ID); got != 0 {
		t.Fatalf("cancelled batch left %d relationships", got)
	}
}

type failingBindingStore struct {
	delegate catalogapp.OrganizationTargetBindingStore
}

func (store failingBindingStore) ExistsByID(id int) (bool, error) {
	return store.delegate.ExistsByID(id)
}

func (store failingBindingStore) BatchAddTargets(int, []int) error {
	return errors.New("simulated association failure")
}

func (store failingBindingStore) ExistsByIDContext(ctx context.Context, id int) (bool, error) {
	return store.delegate.(catalogapp.OrganizationTargetBindingContextStore).ExistsByIDContext(ctx, id)
}

func (store failingBindingStore) BatchAddTargetsContext(ctx context.Context, organizationID int, targetIDs []int) error {
	delegate := store.delegate.(catalogapp.OrganizationTargetBindingContextStore)
	if err := delegate.BatchAddTargetsContext(ctx, organizationID, targetIDs); err != nil {
		return err
	}
	return errors.New("simulated association failure")
}

type cancellingTransactionCoordinator struct {
	db     *gorm.DB
	cancel context.CancelFunc
}

func (coordinator cancellingTransactionCoordinator) WithinTransaction(ctx context.Context, callback func(context.Context) error) error {
	return coordinator.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txContext := dbtx.WithTransaction(ctx, tx)
		if err := callback(txContext); err != nil {
			return err
		}
		coordinator.cancel()
		return txContext.Err()
	})
}

func openTargetBatchTransactionDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:catalog-batch-%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE organization (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, deleted_at DATETIME)`,
		`CREATE UNIQUE INDEX idx_org_active_name_unique ON organization(name) WHERE deleted_at IS NULL`,
		`CREATE TABLE target (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, type TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, last_scanned_at DATETIME, deleted_at DATETIME)`,
		`CREATE UNIQUE INDEX idx_target_active_identity_unique ON target(type, name) WHERE deleted_at IS NULL`,
		`CREATE TABLE organization_target (organization_id INTEGER NOT NULL, target_id INTEGER NOT NULL, PRIMARY KEY (organization_id, target_id))`,
		`CREATE TABLE blacklist_policy (id INTEGER PRIMARY KEY AUTOINCREMENT, scope TEXT NOT NULL, target_id INTEGER, patterns JSON NOT NULL, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, UNIQUE(scope, target_id))`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create test schema: %v", err)
		}
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	return db
}

func newOrganization(t *testing.T, db *gorm.DB, name string) *struct {
	ID int
} {
	t.Helper()
	var id int
	if err := db.Raw("INSERT INTO organization (name) VALUES (?) RETURNING id", name).Scan(&id).Error; err != nil {
		t.Fatalf("seed organization: %v", err)
	}
	return &struct{ ID int }{ID: id}
}

func countRows(t *testing.T, db *gorm.DB, table string) int {
	t.Helper()
	var count int
	if err := db.Raw("SELECT COUNT(*) FROM " + table).Scan(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func countOrganizationTargets(t *testing.T, db *gorm.DB, organizationID int) int {
	t.Helper()
	var count int
	if err := db.Raw("SELECT COUNT(*) FROM organization_target WHERE organization_id = ?", organizationID).Scan(&count).Error; err != nil {
		t.Fatalf("count organization targets: %v", err)
	}
	return count
}
