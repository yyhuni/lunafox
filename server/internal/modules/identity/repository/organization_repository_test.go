package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOrganizationRepositoryCreateUpdateAndExists(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewOrganizationRepository(db)
	if repo == nil || repo.db != db {
		t.Fatalf("expected repository to hold provided db, got %+v", repo)
	}

	org := &model.Organization{
		Name:        "Acme",
		Description: "core team",
	}
	if err := repo.Create(org); err != nil {
		t.Fatalf("create organization failed: %v", err)
	}
	if org.ID == 0 {
		t.Fatalf("expected create to backfill id, got %+v", org)
	}

	org.Name = "Acme Labs"
	org.Description = "updated team"
	if err := repo.Update(org); err != nil {
		t.Fatalf("update organization failed: %v", err)
	}

	found, err := repo.GetActiveByID(org.ID)
	if err != nil {
		t.Fatalf("get active organization failed: %v", err)
	}
	if found.Name != "Acme Labs" || found.Description != "updated team" {
		t.Fatalf("unexpected active organization: %+v", found)
	}

	exists, err := repo.Exists(org.ID)
	if err != nil {
		t.Fatalf("exists check failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected organization %d to exist", org.ID)
	}

	existsByName, err := repo.ExistsByName("Acme Labs")
	if err != nil {
		t.Fatalf("exists by name failed: %v", err)
	}
	if !existsByName {
		t.Fatal("expected organization name to exist")
	}

	existsByName, err = repo.ExistsByName("Acme Labs", org.ID)
	if err != nil {
		t.Fatalf("exists by name with exclude id failed: %v", err)
	}
	if existsByName {
		t.Fatal("expected exclude id to ignore the current organization")
	}
}

func TestOrganizationRepositoryCreateAndUpdateReturnDBErrors(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	closeOrganizationRepositorySQLDB(t, db)

	repo := NewOrganizationRepository(db)
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{
			name: "create",
			call: func() error {
				return repo.Create(&model.Organization{Name: "Acme"})
			},
		},
		{
			name: "update",
			call: func() error {
				return repo.Update(&model.Organization{ID: 1, Name: "Acme"})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err == nil {
				t.Fatalf("expected %s to return database error", tc.name)
			}
		})
	}
}

func TestOrganizationRepositoryCreateContextMapsNamedUniqueConflict(t *testing.T) {
	named := &pq.Error{Code: "23505", Constraint: "idx_org_active_name_unique"}
	if !errors.Is(mapOrganizationWriteError(named), identitydomain.ErrOrganizationExists) {
		t.Fatal("expected named active-name conflict to map to ErrOrganizationExists")
	}
	other := &pq.Error{Code: "23505", Constraint: "idx_org_name"}
	if errors.Is(mapOrganizationWriteError(other), identitydomain.ErrOrganizationExists) {
		t.Fatal("unexpectedly mapped an unrelated unique conflict")
	}
}

func TestOrganizationRepositoryCreateContextRejectsCancellationBeforeWrite(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewOrganizationRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.CreateContext(ctx, &model.Organization{Name: "cancelled"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("create with cancelled context = %v, want context.Canceled", err)
	}
	var count int64
	if err := db.Model(&model.Organization{}).Where("name = ?", "cancelled").Count(&count).Error; err != nil {
		t.Fatalf("count cancelled organization: %v", err)
	}
	if count != 0 {
		t.Fatalf("cancelled create persisted %d rows", count)
	}
}

func TestOrganizationRepositoryPartialUniqueIndexAllowsSoftDeletedNameReuse(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	if err := db.Exec("CREATE UNIQUE INDEX idx_org_active_name_unique ON organization(name) WHERE deleted_at IS NULL").Error; err != nil {
		t.Fatalf("create partial unique index: %v", err)
	}
	repo := NewOrganizationRepository(db)
	first := &model.Organization{Name: "Reusable"}
	if err := repo.Create(first); err != nil {
		t.Fatalf("create first organization: %v", err)
	}
	if err := repo.Create(&model.Organization{Name: "Reusable"}); err == nil {
		t.Fatal("expected active duplicate to be rejected")
	}
	if err := repo.SoftDelete(first.ID); err != nil {
		t.Fatalf("soft delete first organization: %v", err)
	}
	second := &model.Organization{Name: "Reusable"}
	if err := repo.Create(second); err != nil {
		t.Fatalf("reuse soft-deleted organization name: %v", err)
	}
}

func TestOrganizationRepositoryFindAllCountsAndDeleteBehavior(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewOrganizationRepository(db)

	orgOld := &model.Organization{
		Name:        "Old Org",
		Description: "older",
		CreatedAt:   time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC),
	}
	orgNew := &model.Organization{
		Name:        "New Org",
		Description: "newer",
		CreatedAt:   time.Date(2026, 3, 12, 8, 0, 0, 0, time.UTC),
	}
	deletedAt := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	orgDeleted := &model.Organization{
		Name:        "Deleted Org",
		Description: "deleted",
		CreatedAt:   time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC),
		DeletedAt:   &deletedAt,
	}
	mustCreateOrganization(t, db, orgOld)
	mustCreateOrganization(t, db, orgNew)
	mustCreateOrganization(t, db, orgDeleted)

	activeTarget := &model.OrganizationTargetRef{
		Name:      "active.example.com",
		Type:      "domain",
		CreatedAt: time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC),
	}
	secondActiveTarget := &model.OrganizationTargetRef{
		Name:      "second.example.com",
		Type:      "domain",
		CreatedAt: time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
	}
	deletedTargetAt := time.Date(2026, 3, 16, 8, 0, 0, 0, time.UTC)
	deletedTarget := &model.OrganizationTargetRef{
		Name:      "deleted.example.com",
		Type:      "domain",
		CreatedAt: time.Date(2026, 3, 10, 11, 0, 0, 0, time.UTC),
		DeletedAt: &deletedTargetAt,
	}
	mustCreateTarget(t, db, activeTarget)
	mustCreateTarget(t, db, secondActiveTarget)
	mustCreateTarget(t, db, deletedTarget)

	mustLinkOrganizationTarget(t, db, orgOld.ID, activeTarget.ID)
	mustLinkOrganizationTarget(t, db, orgOld.ID, deletedTarget.ID)
	mustLinkOrganizationTarget(t, db, orgNew.ID, secondActiveTarget.ID)

	withCount, err := repo.FindByIDWithCount(orgOld.ID)
	if err != nil {
		t.Fatalf("find by id with count failed: %v", err)
	}
	if withCount.TargetCount != 1 {
		t.Fatalf("expected deleted targets to be excluded from count, got %+v", withCount)
	}

	list, total, err := repo.List(0, 0, "", "")
	if err != nil {
		t.Fatalf("find all failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 active organizations, got %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 listed organizations, got %d", len(list))
	}
	if list[0].ID != orgNew.ID || list[1].ID != orgOld.ID {
		t.Fatalf("expected organizations ordered by created_at desc, got %+v", list)
	}
	if list[0].TargetCount != 1 || list[1].TargetCount != 1 {
		t.Fatalf("unexpected target counts in list: %+v", list)
	}

	if err := repo.SoftDelete(orgOld.ID); err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}
	if _, err := repo.GetActiveByID(orgOld.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected soft-deleted organization to be hidden, got %v", err)
	}

	rowsAffected, err := repo.BatchSoftDelete([]int{orgNew.ID, orgDeleted.ID})
	if err != nil {
		t.Fatalf("batch soft delete failed: %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("expected only active organizations to be batch soft-deleted, got %d", rowsAffected)
	}

	exists, err := repo.Exists(orgDeleted.ID)
	if err != nil {
		t.Fatalf("exists after delete failed: %v", err)
	}
	if exists {
		t.Fatalf("expected deleted organization %d to be excluded from exists", orgDeleted.ID)
	}

	existsByName, err := repo.ExistsByName("Deleted Org")
	if err != nil {
		t.Fatalf("exists by name after delete failed: %v", err)
	}
	if existsByName {
		t.Fatal("expected deleted organization to be excluded from exists by name")
	}
}

func TestOrganizationRepositoryListOrdersByWhitelistedFields(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewOrganizationRepository(db)

	alphaOld := &model.Organization{Name: "Alpha", Description: "old alpha", CreatedAt: time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)}
	alphaNew := &model.Organization{Name: "Alpha", Description: "new alpha", CreatedAt: time.Date(2026, 3, 12, 8, 0, 0, 0, time.UTC)}
	beta := &model.Organization{Name: "Beta", Description: "beta", CreatedAt: time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC)}
	for _, org := range []*model.Organization{alphaOld, alphaNew, beta} {
		mustCreateOrganization(t, db, org)
	}

	orderedByName, total, err := repo.List(1, 10, "", "displayName")
	if err != nil {
		t.Fatalf("list organizations by displayName failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(orderedByName) != 3 || orderedByName[0].ID != alphaOld.ID || orderedByName[1].ID != alphaNew.ID || orderedByName[2].ID != beta.ID {
		t.Fatalf("expected displayName asc with id asc tie-breaker, got %+v", orderedByName)
	}

	orderedByCreated, _, err := repo.List(1, 10, "", "createdAt desc")
	if err != nil {
		t.Fatalf("list organizations by createdAt desc failed: %v", err)
	}
	if len(orderedByCreated) != 3 || orderedByCreated[0].ID != alphaNew.ID || orderedByCreated[1].ID != beta.ID || orderedByCreated[2].ID != alphaOld.ID {
		t.Fatalf("expected filtered organizations ordered by createdAt desc, got %+v", orderedByCreated)
	}
}

func TestOrganizationRepositoryLinkAndUnlinkTargets(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewOrganizationRepository(db)

	org := &model.Organization{Name: "Acme", Description: "core team"}
	firstTarget := &model.OrganizationTargetRef{Name: "first.example.com", Type: "domain"}
	secondTarget := &model.OrganizationTargetRef{Name: "second.example.com", Type: "domain"}
	mustCreateOrganization(t, db, org)
	mustCreateTarget(t, db, firstTarget)
	mustCreateTarget(t, db, secondTarget)

	if err := repo.BatchAddTargets(org.ID, nil); err != nil {
		t.Fatalf("batch add empty targets failed: %v", err)
	}
	if count := countOrganizationTargets(t, db, org.ID); count != 0 {
		t.Fatalf("expected no links after empty batch add, got %d", count)
	}

	if err := repo.BatchAddTargets(org.ID, []int{firstTarget.ID, firstTarget.ID, secondTarget.ID}); err != nil {
		t.Fatalf("batch add targets failed: %v", err)
	}
	if count := countOrganizationTargets(t, db, org.ID); count != 2 {
		t.Fatalf("expected duplicate links to be ignored, got %d", count)
	}

	rowsAffected, err := repo.UnlinkTargets(org.ID, nil)
	if err != nil {
		t.Fatalf("unlink empty targets failed: %v", err)
	}
	if rowsAffected != 0 {
		t.Fatalf("expected empty unlink to affect 0 rows, got %d", rowsAffected)
	}

	rowsAffected, err = repo.UnlinkTargets(org.ID, []int{firstTarget.ID})
	if err != nil {
		t.Fatalf("unlink targets failed: %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("expected one target to be unlinked, got %d", rowsAffected)
	}
	if count := countOrganizationTargets(t, db, org.ID); count != 1 {
		t.Fatalf("expected one remaining link after unlink, got %d", count)
	}

	if err := repo.BatchAddTargets(org.ID, []int{9999}); err == nil {
		t.Fatal("expected missing target to return error")
	}
	if err := db.Model(&model.OrganizationTargetRef{}).Where("id = ?", secondTarget.ID).Update("deleted_at", time.Now().UTC()).Error; err != nil {
		t.Fatalf("tombstone target: %v", err)
	}
	if err := repo.BatchAddTargets(org.ID, []int{secondTarget.ID}); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("tombstoned target relation error = %v, want ErrTargetNotFound", err)
	}
	if count := countOrganizationTargets(t, db, org.ID); count != 1 {
		t.Fatalf("tombstoned target must not create a relation, got %d links", count)
	}
}

func TestOrganizationRepositoryTargetFenceAllowsCommittedRelationshipBeforeLaterDelete(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewOrganizationRepository(db)
	organization := &model.Organization{Name: "Acme"}
	target := &model.OrganizationTargetRef{Name: "before-delete.example", Type: "domain"}
	mustCreateOrganization(t, db, organization)
	mustCreateTarget(t, db, target)

	if err := repo.BatchAddTargets(organization.ID, []int{target.ID}); err != nil {
		t.Fatalf("create active Target relationship: %v", err)
	}
	if err := db.Model(&model.OrganizationTargetRef{}).Where("id = ?", target.ID).Update("deleted_at", time.Now().UTC()).Error; err != nil {
		t.Fatalf("tombstone Target after relationship commit: %v", err)
	}
	if count := countOrganizationTargets(t, db, organization.ID); count != 1 {
		t.Fatalf("Target deletion must not retroactively roll back committed relationship, got %d links", count)
	}
}

func TestOrganizationRepositoryListTargetsByOrganizationID(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	repo := NewOrganizationRepository(db)

	org := &model.Organization{Name: "Acme", Description: "core team"}
	otherOrg := &model.Organization{Name: "Other", Description: "other team"}
	mustCreateOrganization(t, db, org)
	mustCreateOrganization(t, db, otherOrg)

	olderDomain := &model.OrganizationTargetRef{
		Name:      "older.example.com",
		Type:      "domain",
		CreatedAt: time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC),
	}
	newerDomain := &model.OrganizationTargetRef{
		Name:      "newer.example.com",
		Type:      "domain",
		CreatedAt: time.Date(2026, 3, 11, 9, 0, 0, 0, time.UTC),
	}
	ipTarget := &model.OrganizationTargetRef{
		Name:      "198.51.100.7",
		Type:      "ip",
		CreatedAt: time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC),
	}
	deletedAt := time.Date(2026, 3, 13, 9, 0, 0, 0, time.UTC)
	deletedDomain := &model.OrganizationTargetRef{
		Name:      "deleted.example.com",
		Type:      "domain",
		CreatedAt: time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC),
		DeletedAt: &deletedAt,
	}
	otherOrgTarget := &model.OrganizationTargetRef{
		Name:      "other.example.com",
		Type:      "domain",
		CreatedAt: time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC),
	}
	for _, target := range []*model.OrganizationTargetRef{
		olderDomain,
		newerDomain,
		ipTarget,
		deletedDomain,
		otherOrgTarget,
	} {
		mustCreateTarget(t, db, target)
	}

	mustLinkOrganizationTarget(t, db, org.ID, olderDomain.ID)
	mustLinkOrganizationTarget(t, db, org.ID, newerDomain.ID)
	mustLinkOrganizationTarget(t, db, org.ID, ipTarget.ID)
	mustLinkOrganizationTarget(t, db, org.ID, deletedDomain.ID)
	mustLinkOrganizationTarget(t, db, otherOrg.ID, otherOrgTarget.ID)

	domainTargets, total, err := repo.ListTargetsByOrganizationID(org.ID, 1, 1, "domain", "")
	if err != nil {
		t.Fatalf("find targets first page failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 active domain targets before pagination, got %d", total)
	}
	if len(domainTargets) != 1 || domainTargets[0].ID != newerDomain.ID {
		t.Fatalf("expected newest domain target on first page, got %+v", domainTargets)
	}

	domainTargets, total, err = repo.ListTargetsByOrganizationID(org.ID, 2, 1, "domain", "")
	if err != nil {
		t.Fatalf("find targets second page failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected stable total on second page, got %d", total)
	}
	if len(domainTargets) != 1 || domainTargets[0].ID != olderDomain.ID {
		t.Fatalf("expected older domain target on second page, got %+v", domainTargets)
	}

	allTargets, total, err := repo.ListTargetsByOrganizationID(org.ID, 0, 0, "", "")
	if err != nil {
		t.Fatalf("find all targets with default pagination failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected deleted and other-organization targets excluded from total, got %d", total)
	}
	if len(allTargets) != 3 {
		t.Fatalf("expected default pagination to return all active targets, got %d", len(allTargets))
	}
	if allTargets[0].ID != ipTarget.ID || allTargets[1].ID != newerDomain.ID || allTargets[2].ID != olderDomain.ID {
		t.Fatalf("expected targets ordered by created_at desc, got %+v", allTargets)
	}
}

func TestOrganizationRepositoryListTargetsByOrganizationIDReturnsDBError(t *testing.T) {
	db := newOrganizationRepositoryTestDB(t)
	closeOrganizationRepositorySQLDB(t, db)

	repo := NewOrganizationRepository(db)
	if _, _, err := repo.ListTargetsByOrganizationID(1, 1, 10, "", ""); err == nil {
		t.Fatal("expected find targets to return database error")
	}
}

func TestOrganizationPersistenceTableNames(t *testing.T) {
	if got := (model.OrganizationTargetRef{}).TableName(); got != "target" {
		t.Fatalf("unexpected organization target table name: %s", got)
	}
	if got := (model.Organization{}).TableName(); got != "organization" {
		t.Fatalf("unexpected organization table name: %s", got)
	}
}

func newOrganizationRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:identity-organization-%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db failed: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	for _, stmt := range []string{
		"PRAGMA foreign_keys = ON",
		`CREATE TABLE organization (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			deleted_at DATETIME
		)`,
		`CREATE TABLE target (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			last_scanned_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE organization_target (
			organization_id INTEGER NOT NULL,
			target_id INTEGER NOT NULL,
			PRIMARY KEY (organization_id, target_id),
			FOREIGN KEY (organization_id) REFERENCES organization(id),
			FOREIGN KEY (target_id) REFERENCES target(id)
		)`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec schema %q failed: %v", stmt, err)
		}
	}

	return db
}

func closeOrganizationRepositorySQLDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	if err := sqlDB.Close(); err != nil && !errors.Is(err, sql.ErrConnDone) {
		t.Fatalf("close sql db failed: %v", err)
	}
}

func mustCreateOrganization(t *testing.T, db *gorm.DB, org *model.Organization) {
	t.Helper()

	if err := db.Create(org).Error; err != nil {
		t.Fatalf("create organization seed failed: %v", err)
	}
}

func mustCreateTarget(t *testing.T, db *gorm.DB, target *model.OrganizationTargetRef) {
	t.Helper()

	if err := db.Create(target).Error; err != nil {
		t.Fatalf("create target seed failed: %v", err)
	}
}

func mustLinkOrganizationTarget(t *testing.T, db *gorm.DB, organizationID, targetID int) {
	t.Helper()

	if err := db.Exec(
		"INSERT INTO organization_target (organization_id, target_id) VALUES (?, ?)",
		organizationID,
		targetID,
	).Error; err != nil {
		t.Fatalf("link organization target failed: %v", err)
	}
}

func countOrganizationTargets(t *testing.T, db *gorm.DB, organizationID int) int64 {
	t.Helper()

	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM organization_target WHERE organization_id = ?",
		organizationID,
	).Scan(&count).Error; err != nil {
		t.Fatalf("count organization targets failed: %v", err)
	}
	return count
}
