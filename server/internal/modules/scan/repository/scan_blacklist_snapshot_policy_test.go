package repository

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	blacklistrepo "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository"
	blacklistmodel "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository/persistence"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestScanRepositoryCreateWithScanTasksAndPlansFreezesEffectivePolicyUnion(t *testing.T) {
	db := openScanBlacklistSnapshotPolicyDB(t)
	targetID := 7
	seedScanBlacklistPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"*.example.com", "192.0.2.0/24"})
	seedScanBlacklistPolicy(t, db, blacklistdomain.ScopeTarget, &targetID, []string{"192.0.2.0/24", "example.com"})
	policyStore := blacklistrepo.NewPolicyRepository(db)
	scanRepository := NewScanRepository(db)

	first := createScanWithEffectivePolicy(t, scanRepository, policyStore, targetID)
	firstPatterns, err := scanRepository.LoadBlacklistSnapshot(context.Background(), first.ID)
	if err != nil {
		t.Fatalf("load first snapshot: %v", err)
	}
	wantFirst := []string{"*.example.com", "192.0.2.0/24", "example.com"}
	if !reflect.DeepEqual(firstPatterns, wantFirst) {
		t.Fatalf("first snapshot = %#v, want %#v", firstPatterns, wantFirst)
	}

	global, err := policyStore.GetGlobal(context.Background())
	if err != nil {
		t.Fatalf("read global policy: %v", err)
	}
	etag, err := blacklistdomain.ETag(global.Patterns)
	if err != nil {
		t.Fatalf("derive global etag: %v", err)
	}
	if _, changed, err := policyStore.ReplacePatterns(context.Background(), blacklistdomain.ScopeGlobal, nil, etag, []string{"after.example.com"}); err != nil || !changed {
		t.Fatalf("replace global policy: changed=%t err=%v", changed, err)
	}

	second := createScanWithEffectivePolicy(t, scanRepository, policyStore, targetID)
	secondPatterns, err := scanRepository.LoadBlacklistSnapshot(context.Background(), second.ID)
	if err != nil {
		t.Fatalf("load second snapshot: %v", err)
	}
	wantSecond := []string{"192.0.2.0/24", "after.example.com", "example.com"}
	if !reflect.DeepEqual(secondPatterns, wantSecond) {
		t.Fatalf("second snapshot = %#v, want %#v", secondPatterns, wantSecond)
	}
	frozenFirst, err := scanRepository.LoadBlacklistSnapshot(context.Background(), first.ID)
	if err != nil || !reflect.DeepEqual(frozenFirst, wantFirst) {
		t.Fatalf("first snapshot changed after policy update: %#v, %v", frozenFirst, err)
	}
}

func TestScanRepositoryCreateWithScanTasksAndPlansPersistsValidEmptyEffectivePolicy(t *testing.T) {
	db := openScanBlacklistSnapshotPolicyDB(t)
	targetID := 8
	seedScanBlacklistPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{})
	seedScanBlacklistPolicy(t, db, blacklistdomain.ScopeTarget, &targetID, []string{})
	policyStore := blacklistrepo.NewPolicyRepository(db)
	scanRepository := NewScanRepository(db)

	scan := createScanWithEffectivePolicy(t, scanRepository, policyStore, targetID)
	patterns, err := scanRepository.LoadBlacklistSnapshot(context.Background(), scan.ID)
	if err != nil {
		t.Fatalf("load empty snapshot: %v", err)
	}
	if patterns == nil || len(patterns) != 0 {
		t.Fatalf("empty snapshot patterns = %#v", patterns)
	}
	var encoded string
	if err := db.Table("scan_blacklist_snapshot").Select("patterns").Where("scan_id = ?", scan.ID).Scan(&encoded).Error; err != nil {
		t.Fatalf("read encoded empty snapshot: %v", err)
	}
	if encoded != "[]" {
		t.Fatalf("encoded empty snapshot = %q, want []", encoded)
	}
}

func TestScanRepositoryCreateWithScanTasksAndPlansFailsClosedForMissingOrCorruptPolicy(t *testing.T) {
	t.Run("missing target policy", func(t *testing.T) {
		db := openScanBlacklistSnapshotPolicyDB(t)
		seedScanBlacklistPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"example.com"})
		scanRepository := NewScanRepository(db)
		policyStore := blacklistrepo.NewPolicyRepository(db)
		scan := &ScanCreateRecord{TargetID: 9, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
		err := scanRepository.CreateWithScanTasksAndPlans(context.Background(), scan, policyStore.ReadEffectivePatternsForScan, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil })
		if !errors.Is(err, blacklistrepo.ErrPolicyNotFound) {
			t.Fatalf("missing policy error = %v", err)
		}
		assertNoScanOrBlacklistSnapshot(t, db)
	})

	t.Run("corrupt global policy", func(t *testing.T) {
		db := openScanBlacklistSnapshotPolicyDB(t)
		targetID := 10
		if err := db.Create(&blacklistmodel.Policy{
			Scope:     string(blacklistdomain.ScopeGlobal),
			Patterns:  datatypes.JSON([]byte(`["Example.COM"]`)),
			UpdatedAt: time.Now().UTC(),
		}).Error; err != nil {
			t.Fatalf("seed corrupt global policy: %v", err)
		}
		seedScanBlacklistPolicy(t, db, blacklistdomain.ScopeTarget, &targetID, []string{})
		scanRepository := NewScanRepository(db)
		policyStore := blacklistrepo.NewPolicyRepository(db)
		scan := &ScanCreateRecord{TargetID: targetID, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
		err := scanRepository.CreateWithScanTasksAndPlans(context.Background(), scan, policyStore.ReadEffectivePatternsForScan, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil })
		if !errors.Is(err, blacklistrepo.ErrPolicyDataIntegrity) {
			t.Fatalf("corrupt policy error = %v", err)
		}
		assertNoScanOrBlacklistSnapshot(t, db)
	})
}

func TestScanRepositoryCreateWithScanTasksAndPlansRejectsOversizedEffectivePolicy(t *testing.T) {
	db := openScanBlacklistSnapshotPolicyDB(t)
	scanRepository := NewScanRepository(db)
	scan := &ScanCreateRecord{TargetID: 11, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
	patterns := make([]string, blacklistdomain.MaxEffectivePatterns+1)
	for index := range patterns {
		patterns[index] = "host" + string(rune('a'+(index%26))) + ".example.com"
	}

	err := scanRepository.CreateWithScanTasksAndPlans(context.Background(), scan, func(context.Context, int) ([]string, error) {
		return patterns, nil
	}, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil })
	if !errors.Is(err, blacklistdomain.ErrEffectiveLimitExceeded) {
		t.Fatalf("oversized effective policy error = %v", err)
	}
	assertNoScanOrBlacklistSnapshot(t, db)
}

func TestScanBlacklistSnapshotDoesNotEnterCreateProjectionOrSavedPlan(t *testing.T) {
	db := openScanBlacklistSnapshotPolicyDB(t)
	targetID := 1
	seedScanBlacklistPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"blacklisted.example"})
	seedScanBlacklistPolicy(t, db, blacklistdomain.ScopeTarget, &targetID, []string{})
	policyStore := blacklistrepo.NewPolicyRepository(db)
	scanRepository := NewScanRepository(db)
	scan := &ScanCreateRecord{
		TargetID:       targetID,
		ScanWorkflowID: "default",
		InputSource:    scandomain.InputSourceScanSnapshot,
		TriggerType:    scandomain.ScanTriggerTypeManual,
		Status:         "pending",
		ScanTasks: []CreateScanTaskRecord{{
			StageOrder: 1,
			StageID:    "discovery",
			StepOrder:  1,
			StepID:     "subdomains",
			EngineID:   "engine.lunafox.subdomain_discovery",
			Status:     "pending",
		}},
	}
	if err := scanRepository.CreateWithScanTasksAndPlans(context.Background(), scan, policyStore.ReadEffectivePatternsForScan, func(scanID, taskID int, task *scandomain.CreateScanTask) error {
		task.ResolvedExecutionPlan = mustSavedPlanForTask(t, "scans/"+strconv.Itoa(scanID)+"/tasks/"+strconv.Itoa(taskID), "scans/"+strconv.Itoa(scanID), task.EngineID, task.StageID, task.StepID)
		return nil
	}); err != nil {
		t.Fatalf("create Scan with saved plan: %v", err)
	}

	for _, projection := range []any{scandomain.CreateScan{}, scandomain.CreateScanTask{}, scandomain.QueryScan{}, scandomain.ScanTaskRecord{}} {
		typeOfProjection := reflect.TypeOf(projection)
		if _, exists := typeOfProjection.FieldByName("BlacklistSnapshot"); exists {
			t.Fatalf("%s exposes a blacklist snapshot field", typeOfProjection.Name())
		}
	}
	var savedPlan struct {
		Plan []byte `gorm:"column:resolved_execution_plan"`
	}
	if err := db.Table("scan_task").Select("resolved_execution_plan").Where("scan_id = ?", scan.ID).Take(&savedPlan).Error; err != nil {
		t.Fatalf("load saved plan: %v", err)
	}
	if bytes.Contains(savedPlan.Plan, []byte("blacklisted.example")) {
		t.Fatalf("saved plan contains private blacklist snapshot bytes")
	}
}

func openScanBlacklistSnapshotPolicyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("create scan tables: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	if err := db.Exec(createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("create scan task table: %v", err)
	}
	if err := db.AutoMigrate(&blacklistmodel.Policy{}); err != nil {
		t.Fatalf("create policy table: %v", err)
	}
	return db
}

func seedScanBlacklistPolicy(t *testing.T, db *gorm.DB, scope blacklistdomain.Scope, targetID *int, patterns []string) {
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

func createScanWithEffectivePolicy(t *testing.T, scanRepository *ScanRepository, policyStore *blacklistrepo.PolicyRepository, targetID int) *ScanCreateRecord {
	t.Helper()
	scan := &ScanCreateRecord{TargetID: targetID, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
	if err := scanRepository.CreateWithScanTasksAndPlans(context.Background(), scan, policyStore.ReadEffectivePatternsForScan, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil }); err != nil {
		t.Fatalf("create Scan: %v", err)
	}
	return scan
}
