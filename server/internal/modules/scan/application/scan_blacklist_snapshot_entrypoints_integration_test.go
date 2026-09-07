package application

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	blacklistrepo "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository"
	blacklistmodel "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository/persistence"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// blacklistSnapshotEntryPointStore models the production wiring boundary: the
// Scan repository owns its transaction and asks the Policy repository through
// that transaction context before it writes one immutable snapshot.
type blacklistSnapshotEntryPointStore struct {
	repository  *scanrepo.ScanRepository
	policies    *blacklistrepo.PolicyRepository
	afterCommit func(int) error
	commits     int
}

func (store *blacklistSnapshotEntryPointStore) CreateWithScanTasksAndPlans(ctx context.Context, scan *CreateScan, finalize ScanCreateTaskFinalizer) error {
	if err := store.repository.CreateWithScanTasksAndPlans(ctx, scan, store.policies.ReadEffectivePatternsForScan, finalize); err != nil {
		return err
	}
	store.commits++
	if store.afterCommit != nil {
		return store.afterCommit(store.commits)
	}
	return nil
}

func TestScanCreateEntryPointsFreezeTheirOwnCommittedBlacklist(t *testing.T) {
	t.Run("quick", func(t *testing.T) {
		db, scanRepository, policies := setupBlacklistSnapshotEntryPointDB(t, 1, 2)
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"global.before.example"})
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeTarget, intPointer(1), []string{"local.one.example"})
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeTarget, intPointer(2), []string{"local.two.example"})
		store := &blacklistSnapshotEntryPointStore{repository: scanRepository, policies: policies}
		store.afterCommit = func(commits int) error {
			if commits != 1 {
				return nil
			}
			return replaceBlacklistSnapshotEntryPointGlobalPolicy(policies, []string{"global.after.example"})
		}
		service, configuration := newBlacklistSnapshotEntryPointService(t, store, entryPointTargetRefs(1, 2), []TargetRef{
			{ID: 1, Name: "one.example", Type: "domain"},
			{ID: 2, Name: "two.example", Type: "domain"},
		}, nil)

		result, err := service.CreateQuick(context.Background(), &CreateQuickInput{
			Targets:       []string{"one.example", "two.example"},
			ScanWorkflow:  "scanWorkflows/subdomain_discovery",
			Configuration: configuration,
			InputSource:   InputSourceScanSnapshot,
			TriggerType:   ScanTriggerTypeManual,
		})
		if err != nil {
			t.Fatalf("CreateQuick failed: %v", err)
		}
		if len(result.Scans) != 2 {
			t.Fatalf("quick scans = %#v", result.Scans)
		}
		assertBlacklistSnapshotEntryPointPatterns(t, scanRepository, result.Scans[0].ID, []string{"global.before.example", "local.one.example"})
		assertBlacklistSnapshotEntryPointPatterns(t, scanRepository, result.Scans[1].ID, []string{"global.after.example", "local.two.example"})
	})

	t.Run("batch create", func(t *testing.T) {
		db, scanRepository, policies := setupBlacklistSnapshotEntryPointDB(t, 3, 4)
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"global.before.example"})
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeTarget, intPointer(3), []string{"local.three.example"})
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeTarget, intPointer(4), []string{"local.four.example"})
		store := &blacklistSnapshotEntryPointStore{repository: scanRepository, policies: policies}
		store.afterCommit = func(commits int) error {
			if commits != 1 {
				return nil
			}
			return replaceBlacklistSnapshotEntryPointGlobalPolicy(policies, []string{"global.after.example"})
		}
		service, configuration := newBlacklistSnapshotEntryPointService(t, store, entryPointTargetRefs(3, 4), nil, nil)

		result, err := service.CreateBatch(context.Background(), &CreateBatchInput{
			Requests:      []CreateBatchItem{{TargetID: 3}, {TargetID: 4}},
			ScanWorkflow:  "scanWorkflows/subdomain_discovery",
			Configuration: configuration,
			InputSource:   InputSourceScanSnapshot,
			TriggerType:   ScanTriggerTypeManual,
		})
		if err != nil {
			t.Fatalf("CreateBatch failed: %v", err)
		}
		if len(result.Scans) != 2 {
			t.Fatalf("batch scans = %#v", result.Scans)
		}
		assertBlacklistSnapshotEntryPointPatterns(t, scanRepository, result.Scans[0].ID, []string{"global.before.example", "local.three.example"})
		assertBlacklistSnapshotEntryPointPatterns(t, scanRepository, result.Scans[1].ID, []string{"global.after.example", "local.four.example"})
	})

	t.Run("organization expansion", func(t *testing.T) {
		db, scanRepository, policies := setupBlacklistSnapshotEntryPointDB(t, 5, 6)
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"global.before.example"})
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeTarget, intPointer(5), []string{"local.five.example"})
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeTarget, intPointer(6), []string{"local.six.example"})
		store := &blacklistSnapshotEntryPointStore{repository: scanRepository, policies: policies}
		store.afterCommit = func(commits int) error {
			if commits != 1 {
				return nil
			}
			return replaceBlacklistSnapshotEntryPointGlobalPolicy(policies, []string{"global.after.example"})
		}
		service, configuration := newBlacklistSnapshotEntryPointService(t, store, entryPointTargetRefs(5, 6), nil, []TargetRef{
			{ID: 5, Name: "five.example", Type: "domain"},
			{ID: 6, Name: "six.example", Type: "domain"},
		})

		result, err := service.CreateBatch(context.Background(), &CreateBatchInput{
			Requests:      []CreateBatchItem{{OrganizationID: 12}},
			ScanWorkflow:  "scanWorkflows/subdomain_discovery",
			Configuration: configuration,
			InputSource:   InputSourceScanSnapshot,
			TriggerType:   ScanTriggerTypeManual,
		})
		if err != nil {
			t.Fatalf("CreateBatch organization expansion failed: %v", err)
		}
		if len(result.Scans) != 2 {
			t.Fatalf("organization scans = %#v", result.Scans)
		}
		assertBlacklistSnapshotEntryPointPatterns(t, scanRepository, result.Scans[0].ID, []string{"global.before.example", "local.five.example"})
		assertBlacklistSnapshotEntryPointPatterns(t, scanRepository, result.Scans[1].ID, []string{"global.after.example", "local.six.example"})
	})

	t.Run("later policy failure keeps earlier committed scan", func(t *testing.T) {
		db, scanRepository, policies := setupBlacklistSnapshotEntryPointDB(t, 7, 8)
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"global.before.example"})
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeTarget, intPointer(7), []string{"local.seven.example"})
		seedBlacklistSnapshotEntryPointPolicy(t, db, blacklistdomain.ScopeTarget, intPointer(8), []string{"local.eight.example"})
		store := &blacklistSnapshotEntryPointStore{repository: scanRepository, policies: policies}
		store.afterCommit = func(commits int) error {
			if commits != 1 {
				return nil
			}
			return db.Exec(`UPDATE blacklist_policy SET patterns = '["Example.COM"]' WHERE scope = 'global'`).Error
		}
		service, configuration := newBlacklistSnapshotEntryPointService(t, store, entryPointTargetRefs(7, 8), nil, nil)

		result, err := service.CreateBatch(context.Background(), &CreateBatchInput{
			Requests:      []CreateBatchItem{{TargetID: 7}, {TargetID: 8}},
			ScanWorkflow:  "scanWorkflows/subdomain_discovery",
			Configuration: configuration,
			InputSource:   InputSourceScanSnapshot,
			TriggerType:   ScanTriggerTypeManual,
		})
		if !errors.Is(err, blacklistrepo.ErrPolicyDataIntegrity) {
			t.Fatalf("CreateBatch error = %v, want policy data-integrity failure", err)
		}
		if result == nil || result.CreatedCount != 1 || len(result.Scans) != 1 || result.Scans[0].TargetID != 7 {
			t.Fatalf("partial batch result = %#v", result)
		}
		assertBlacklistSnapshotEntryPointPatterns(t, scanRepository, result.Scans[0].ID, []string{"global.before.example", "local.seven.example"})
		var scanCount int64
		if err := db.Table("scan").Count(&scanCount).Error; err != nil {
			t.Fatalf("count scans: %v", err)
		}
		if scanCount != 1 {
			t.Fatalf("later failed batch item rolled back or committed unexpectedly: scans=%d", scanCount)
		}
	})
}

func setupBlacklistSnapshotEntryPointDB(t *testing.T, targetIDs ...int) (*gorm.DB, *scanrepo.ScanRepository, *blacklistrepo.PolicyRepository) {
	t.Helper()
	db := setupIntegrationDB(t)
	for _, targetID := range targetIDs {
		if targetID == 1 {
			continue
		}
		if err := db.Exec(`INSERT INTO target (id, name, type) VALUES (?, ?, 'domain')`, targetID, fmt.Sprintf("target-%d.example", targetID)).Error; err != nil {
			t.Fatalf("insert target %d: %v", targetID, err)
		}
	}
	if err := db.AutoMigrate(&blacklistmodel.Policy{}); err != nil {
		t.Fatalf("create blacklist policy table: %v", err)
	}
	return db, scanrepo.NewScanRepository(db), blacklistrepo.NewPolicyRepository(db)
}

func seedBlacklistSnapshotEntryPointPolicy(t *testing.T, db *gorm.DB, scope blacklistdomain.Scope, targetID *int, patterns []string) {
	t.Helper()
	canonical, err := blacklistdomain.CanonicalizePolicyPatterns(patterns)
	if err != nil {
		t.Fatalf("canonicalize policy: %v", err)
	}
	payload, err := blacklistdomain.CanonicalPatternsJSON(canonical)
	if err != nil {
		t.Fatalf("encode policy: %v", err)
	}
	if err := db.Create(&blacklistmodel.Policy{
		Scope:     string(scope),
		TargetID:  targetID,
		Patterns:  datatypes.JSON(append([]byte(nil), payload...)),
		UpdatedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}
}

func replaceBlacklistSnapshotEntryPointGlobalPolicy(policies *blacklistrepo.PolicyRepository, patterns []string) error {
	current, err := policies.GetGlobal(context.Background())
	if err != nil {
		return err
	}
	etag, err := blacklistdomain.ETag(current.Patterns)
	if err != nil {
		return err
	}
	_, changed, err := policies.ReplacePatterns(context.Background(), blacklistdomain.ScopeGlobal, nil, etag, patterns)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("expected global policy update to change content")
	}
	return nil
}

func newBlacklistSnapshotEntryPointService(t *testing.T, store ScanCreateCommandStore, targets map[int]TargetRef, quickTargets, organizationTargets []TargetRef) (*ScanCreateService, map[string]any) {
	t.Helper()
	workflowReader, packageReader := scanCreateReaderStubs()
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(
		store,
		func(_ context.Context, targetID int) (*TargetRef, error) {
			target, ok := targets[targetID]
			if !ok {
				return nil, fmt.Errorf("target %d not found", targetID)
			}
			return &target, nil
		},
		func(context.Context, []string) (*QuickTargetResolution, error) {
			return &QuickTargetResolution{Targets: append([]TargetRef(nil), quickTargets...)}, nil
		},
		workflowReader,
		packageReader,
		func(context.Context, int) ([]TargetRef, error) {
			return append([]TargetRef(nil), organizationTargets...), nil
		},
	), packageReader)
	definition := scanCreateTestEngineDefinition("engine.lunafox.subdomain_discovery", []string{engineexecution.TargetTypeDomain})
	return service, completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{"subdomain_discovery": definition.Execution})
}

func entryPointTargetRefs(targetIDs ...int) map[int]TargetRef {
	targets := make(map[int]TargetRef, len(targetIDs))
	for _, targetID := range targetIDs {
		targets[targetID] = TargetRef{ID: targetID, Name: fmt.Sprintf("target-%d.example", targetID), Type: "domain"}
	}
	return targets
}

func assertBlacklistSnapshotEntryPointPatterns(t *testing.T, repository *scanrepo.ScanRepository, scanID int, want []string) {
	t.Helper()
	got, err := repository.LoadBlacklistSnapshot(context.Background(), scanID)
	if err != nil {
		t.Fatalf("load scan %d blacklist snapshot: %v", scanID, err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("scan %d blacklist snapshot = %#v, want %#v", scanID, got, want)
	}
}
