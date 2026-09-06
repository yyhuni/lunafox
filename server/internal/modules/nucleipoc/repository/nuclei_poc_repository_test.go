package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	notificationdomain "github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	notificationrepo "github.com/yyhuni/lunafox/server/internal/modules/notification/repository"
	notificationmodel "github.com/yyhuni/lunafox/server/internal/modules/notification/repository/persistence"
	app "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/application"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRepositoryTest(t *testing.T) (*NucleiPOCRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&model.Source{}, &model.SyncTask{}, &model.CandidateImport{}, &model.POC{}, &model.RequestTombstone{}, &notificationmodel.Outbox{}); err != nil {
		t.Fatal(err)
	}
	producer := notificationrepo.NewProducerWriter(notificationrepo.NewOutboxRepository(db))
	return NewNucleiPOCRepository(db, producer), db
}

func createTestTask(t *testing.T, db *gorm.DB, sourceID uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()
	now := time.Now().UTC()
	taskID, requestID := uuid.New(), uuid.New()
	if err := db.Create(&model.Source{ID: sourceID, SourceType: "git", RepoURL: "https://example.com/templates.git", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SyncTask{ID: taskID, RequestID: requestID, RequestFingerprint: fingerprintDigest("request"), SourceType: "git", RepoURL: "https://example.com/templates.git", SourceID: sourceID, State: string(domain.SyncTaskCommitting), Phase: string(domain.SyncTaskCommitting), Diagnostics: []byte(`{"samples":[],"total":0,"truncated":false}`), CleanupStatus: string(domain.CleanupPending), CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.RequestTombstone{RequestID: requestID, RequestFingerprint: fingerprintDigest("request"), ReceivedAt: now, CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	return taskID, requestID
}

func testCandidate(taskID, sourceID uuid.UUID, id, name string) domain.CandidatePOC {
	now := time.Now().UTC()
	content := "id: " + id
	digest := sha256.Sum256([]byte(content))
	return domain.CandidatePOC{ID: uuid.New(), TaskID: taskID, SourceID: sourceID, TemplateID: id, DisplayName: name, Severity: "high", Tags: []string{"cve"}, RelativePath: "http/" + id + ".yaml", ContentSHA256: hex.EncodeToString(digest[:]), Content: content, CreatedAt: now}
}

func TestGetPOCNormalizesNullJSONCollections(t *testing.T) {
	repository, db := newRepositoryTest(t)
	now := time.Now().UTC()
	row := model.POC{
		ID:            uuid.New(),
		TemplateID:    "legacy-null",
		Severity:      "info",
		Tags:          []byte(`null`),
		CVE:           []byte(`null`),
		CWE:           []byte(`null`),
		References:    []byte(`null`),
		RelativePath:  "http/legacy-null.yaml",
		ContentSHA256: "digest",
		Content:       "id: legacy-null",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	poc, err := repository.GetPOC(context.Background(), "nucleiPocs/legacy-null")
	if err != nil {
		t.Fatal(err)
	}
	for field, values := range map[string][]string{
		"tags":       poc.Tags,
		"cve":        poc.CVE,
		"cwe":        poc.CWE,
		"references": poc.References,
	} {
		if values == nil {
			t.Errorf("%s is nil, want an empty slice", field)
		}
	}
}

func TestListFilterOptionsAggregatesCompleteCatalogTags(t *testing.T) {
	repository, db := newRepositoryTest(t)
	now := time.Now().UTC()
	for index, tags := range []string{`["CVE", "cve", " RCE "]`, `["rce", "http"]`, `null`, `not-json`} {
		if err := db.Create(&model.POC{ID: uuid.New(), SourceID: uuid.New(), TemplateID: fmt.Sprintf("filter-options-%d", index), Severity: "info", Tags: []byte(tags), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: fmt.Sprintf("http/%d.yaml", index), ContentSHA256: "digest", Content: "id: test", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
	}

	options, err := repository.ListFilterOptions(context.Background(), "tags")
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.FilterOption{{Value: "cve", Label: "cve", Count: 1}, {Value: "http", Label: "http", Count: 1}, {Value: "rce", Label: "rce", Count: 2}}
	if !reflect.DeepEqual(options, want) {
		t.Fatalf("options=%+v want=%+v", options, want)
	}
	if _, err := repository.ListFilterOptions(context.Background(), "severity"); err == nil {
		t.Fatal("expected unsupported field error")
	}
}

func TestListFilterOptionsReturnsAnEmptyResultForAnEmptyCatalog(t *testing.T) {
	repository, _ := newRepositoryTest(t)

	options, err := repository.ListFilterOptions(context.Background(), "tags")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(options, []domain.FilterOption{}) {
		t.Fatalf("options=%+v want an empty result", options)
	}
}

func TestPromoteCandidatesInheritsEnablementAndFreezesTask(t *testing.T) {
	repository, db := newRepositoryTest(t)
	ctx := context.Background()
	oldSourceID, oldTaskID := uuid.New(), uuid.New()
	now := time.Now().UTC().Add(-time.Hour)
	if err := db.Create(&model.Source{ID: oldSourceID, SourceType: "git", RepoURL: "https://old.example/templates.git", IsActive: true, CommitSHA: "old", SyncedAt: &now, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	old := testCandidate(oldTaskID, oldSourceID, "keep-id", "Old")
	if err := db.Create(&model.POC{ID: uuid.New(), SourceID: oldSourceID, TemplateID: old.TemplateID, DisplayName: old.DisplayName, Severity: old.Severity, Tags: []byte(`["old"]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: old.RelativePath, ContentSHA256: old.ContentSHA256, Content: old.Content, IsEnabled: false, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.POC{}).Where("template_id = ?", old.TemplateID).Update("is_enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	var persistedOld model.POC
	if err := db.Where("template_id = ?", old.TemplateID).First(&persistedOld).Error; err != nil || persistedOld.IsEnabled {
		t.Fatalf("failed to seed disabled old POC: %+v err=%v", persistedOld, err)
	}
	newSourceID := uuid.New()
	taskID, requestID := createTestTask(t, db, newSourceID)
	_ = requestID
	candidate := testCandidate(taskID, newSourceID, "keep-id", "New")
	if err := repository.StageCandidate(ctx, candidate); err != nil {
		t.Fatal(err)
	}
	newCandidate := testCandidate(taskID, newSourceID, "new-id", "New ID")
	if err := repository.StageCandidate(ctx, newCandidate); err != nil {
		t.Fatal(err)
	}
	commitSHA := strings.Repeat("a", 40)
	count, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{TaskID: taskID, Source: domain.Source{ID: newSourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"}, CommitSHA: commitSHA, SyncedAt: time.Now().UTC(), CleanupStatus: string(domain.CleanupClean)})
	if err != nil || count != 2 {
		t.Fatalf("promotion count=%d err=%v", count, err)
	}
	current, err := repository.GetCurrentSource(ctx)
	if err != nil || current.ID != newSourceID || current.CommitSHA != commitSHA {
		t.Fatalf("current source=%+v err=%v", current, err)
	}
	result, err := repository.ListPOCs(ctx, app.POCListQuery{PageSize: 10, OrderBy: "templateId asc"})
	if err != nil || result.TotalSize != 2 {
		t.Fatalf("list result=%+v err=%v", result, err)
	}
	for _, poc := range result.Results {
		if poc.TemplateID == "keep-id" && poc.IsEnabled {
			t.Fatalf("existing template enablement was not inherited: %+v", poc)
		}
		if poc.TemplateID == "new-id" && poc.IsEnabled {
			t.Fatal("new template must default disabled")
		}
	}
	task, err := repository.GetSyncTask(ctx, taskID)
	if err != nil || task.State != domain.SyncTaskSucceeded || task.CommittedPOCCount != 2 || task.CommitSHA != commitSHA {
		t.Fatalf("task=%+v err=%v", task, err)
	}
}

func TestPromoteCandidatesInheritsBothStatesAcrossContentAndPathChanges(t *testing.T) {
	repository, db := newRepositoryTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Add(-time.Hour)
	oldSourceID := uuid.New()
	if err := db.Create(&model.Source{ID: oldSourceID, SourceType: "git", RepoURL: "https://old.example/templates.git", IsActive: true, CommitSHA: strings.Repeat("a", 40), SyncedAt: &now, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	for _, seed := range []struct {
		id      string
		enabled bool
	}{
		{id: "keep-enabled", enabled: true},
		{id: "keep-disabled", enabled: false},
	} {
		candidate := testCandidate(uuid.New(), oldSourceID, seed.id, "Old "+seed.id)
		row := model.POC{ID: uuid.New(), SourceID: oldSourceID, TemplateID: candidate.TemplateID, DisplayName: candidate.DisplayName, Severity: candidate.Severity, Tags: []byte(`[]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: candidate.RelativePath, ContentSHA256: candidate.ContentSHA256, Content: candidate.Content, IsEnabled: seed.enabled, CreatedAt: now, UpdatedAt: now}
		if err := db.Select("*").Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}

	newSourceID := uuid.New()
	taskID, _ := createTestTask(t, db, newSourceID)
	for _, id := range []string{"keep-enabled", "keep-disabled"} {
		candidate := testCandidate(taskID, newSourceID, id, "Changed "+id)
		candidate.RelativePath = "moved/" + id + ".yaml"
		candidate.Content = "id: " + id + "\ninfo:\n  name: Changed"
		digest := sha256.Sum256([]byte(candidate.Content))
		candidate.ContentSHA256 = hex.EncodeToString(digest[:])
		if err := repository.StageCandidate(ctx, candidate); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{TaskID: taskID, Source: domain.Source{ID: newSourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"}, CommitSHA: strings.Repeat("b", 40), SyncedAt: time.Now().UTC(), CleanupStatus: string(domain.CleanupClean)}); err != nil {
		t.Fatalf("PromoteCandidates: %v", err)
	}
	for templateID, wantEnabled := range map[string]bool{"keep-enabled": true, "keep-disabled": false} {
		var row model.POC
		if err := db.Where("template_id = ?", templateID).First(&row).Error; err != nil {
			t.Fatalf("load %s: %v", templateID, err)
		}
		if row.IsEnabled != wantEnabled || !strings.HasPrefix(row.RelativePath, "moved/") || !strings.Contains(row.Content, "Changed") {
			t.Fatalf("promoted %s = %#v, want enabled=%t with changed content/path", templateID, row, wantEnabled)
		}
	}
}

func TestPromoteCandidatesTreatsReappearingTemplateAsNewAndPersistsFalse(t *testing.T) {
	repository, db := newRepositoryTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Add(-time.Hour)
	oldSourceID := uuid.New()
	if err := db.Create(&model.Source{ID: oldSourceID, SourceType: "git", RepoURL: "https://old.example/templates.git", IsActive: true, CommitSHA: strings.Repeat("a", 40), SyncedAt: &now, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	old := testCandidate(uuid.New(), oldSourceID, "reappear-id", "Previously enabled")
	oldRow := model.POC{ID: uuid.New(), SourceID: oldSourceID, TemplateID: old.TemplateID, DisplayName: old.DisplayName, Severity: old.Severity, Tags: []byte(`[]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: old.RelativePath, ContentSHA256: old.ContentSHA256, Content: old.Content, IsEnabled: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Select("*").Create(&oldRow).Error; err != nil {
		t.Fatal(err)
	}

	disappearanceSourceID := uuid.New()
	disappearanceTaskID, _ := createTestTask(t, db, disappearanceSourceID)
	if err := repository.StageCandidate(ctx, testCandidate(disappearanceTaskID, disappearanceSourceID, "survives-id", "Survives")); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{TaskID: disappearanceTaskID, Source: domain.Source{ID: disappearanceSourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"}, CommitSHA: strings.Repeat("b", 40), SyncedAt: time.Now().UTC(), CleanupStatus: string(domain.CleanupClean)}); err != nil {
		t.Fatalf("promote disappearance: %v", err)
	}
	var disappeared model.POC
	if err := db.Where("template_id = ?", "reappear-id").First(&disappeared).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("removed template remained in committed catalog: row=%#v err=%v", disappeared, err)
	}

	reappearanceSourceID := uuid.New()
	reappearanceTaskID, _ := createTestTask(t, db, reappearanceSourceID)
	if err := repository.StageCandidate(ctx, testCandidate(reappearanceTaskID, reappearanceSourceID, "reappear-id", "Reappeared")); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{TaskID: reappearanceTaskID, Source: domain.Source{ID: reappearanceSourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"}, CommitSHA: strings.Repeat("c", 40), SyncedAt: time.Now().UTC(), CleanupStatus: string(domain.CleanupClean)}); err != nil {
		t.Fatalf("promote reappearance: %v", err)
	}
	var reappeared model.POC
	if err := db.Where("template_id = ?", "reappear-id").First(&reappeared).Error; err != nil {
		t.Fatal(err)
	}
	if reappeared.IsEnabled {
		t.Fatalf("reappearing template restored historical state: %#v", reappeared)
	}
}

func TestPromotionRejectsEmptyAndPreservesOldCollection(t *testing.T) {
	repository, db := newRepositoryTest(t)
	ctx := context.Background()
	oldSourceID := uuid.New()
	now := time.Now().UTC()
	if err := db.Create(&model.Source{ID: oldSourceID, SourceType: "git", RepoURL: "https://old.example/templates.git", IsActive: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	old := testCandidate(uuid.New(), oldSourceID, "old-id", "Old")
	if err := db.Create(&model.POC{ID: uuid.New(), SourceID: oldSourceID, TemplateID: old.TemplateID, DisplayName: old.DisplayName, Severity: old.Severity, Tags: []byte(`[]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: old.RelativePath, ContentSHA256: old.ContentSHA256, Content: old.Content, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	newSourceID := uuid.New()
	taskID, _ := createTestTask(t, db, newSourceID)
	_, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{TaskID: taskID, Source: domain.Source{ID: newSourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"}, CommitSHA: "should-not-commit", SyncedAt: now, CleanupStatus: string(domain.CleanupClean)})
	if !errors.Is(err, domain.ErrEmptyCandidate) {
		t.Fatalf("err=%v, want empty candidate", err)
	}
	current, err := repository.GetCurrentSource(ctx)
	if err != nil || current.ID != oldSourceID {
		t.Fatalf("old source was not preserved: %+v err=%v", current, err)
	}
	result, err := repository.ListPOCs(ctx, app.POCListQuery{PageSize: 10})
	if err != nil || len(result.Results) != 1 || result.Results[0].TemplateID != "old-id" {
		t.Fatalf("old POCs were not preserved: %+v err=%v", result, err)
	}
}

func TestSetPOCActivationIsFullCatalogIdempotentAndCountsOnlyChanges(t *testing.T) {
	repository, db := newRepositoryTest(t)
	now := time.Now().UTC()
	sourceID := uuid.New()
	if err := db.Create(&model.Source{ID: sourceID, SourceType: "git", RepoURL: "https://example.com/templates.git", IsActive: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	for index, enabled := range []bool{false, true, false} {
		candidate := testCandidate(uuid.New(), sourceID, fmt.Sprintf("bulk-%d", index), "Bulk")
		if err := db.Select("*").Create(&model.POC{ID: uuid.New(), SourceID: sourceID, TemplateID: candidate.TemplateID, DisplayName: candidate.DisplayName, Severity: candidate.Severity, Tags: []byte(`[]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: candidate.RelativePath, ContentSHA256: candidate.ContentSHA256, Content: candidate.Content, IsEnabled: enabled, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
	}
	changed, err := repository.SetPOCActivation(context.Background(), true, nil)
	if err != nil || changed != 2 {
		t.Fatalf("enable all changed=%d err=%v, want 2", changed, err)
	}
	changed, err = repository.SetPOCActivation(context.Background(), true, nil)
	if err != nil || changed != 0 {
		t.Fatalf("repeated enable all changed=%d err=%v, want 0", changed, err)
	}
	changed, err = repository.SetPOCActivation(context.Background(), false, nil)
	if err != nil || changed != 3 {
		t.Fatalf("disable all changed=%d err=%v, want 3", changed, err)
	}
	var enabledCount int64
	if err := db.Model(&model.POC{}).Where("is_enabled = ?", true).Count(&enabledCount).Error; err != nil || enabledCount != 0 {
		t.Fatalf("catalog states after disable all count=%d err=%v", enabledCount, err)
	}
}

func TestListEnabledForExecutionOrdersAndRejectsDigestDrift(t *testing.T) {
	repository, db := newRepositoryTest(t)
	now := time.Now().UTC()
	makeRow := func(id, content string, enabled bool, digest string) model.POC {
		if digest == "" {
			hash := sha256.Sum256([]byte(content))
			digest = hex.EncodeToString(hash[:])
		}
		return model.POC{ID: uuid.New(), SourceID: uuid.New(), TemplateID: id, DisplayName: id, Severity: "high", Tags: []byte(`[]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: id + ".yaml", ContentSHA256: digest, Content: content, IsEnabled: enabled, CreatedAt: now, UpdatedAt: now}
	}
	zRow := makeRow("z-template", "id: z\n", true, "")
	if err := db.Select("*").Create(&zRow).Error; err != nil {
		t.Fatal(err)
	}
	aRow := makeRow("a-template", "id: a\n", true, "")
	if err := db.Select("*").Create(&aRow).Error; err != nil {
		t.Fatal(err)
	}
	disabledRow := makeRow("disabled", "id: disabled\n", false, "")
	if err := db.Select("*").Create(&disabledRow).Error; err != nil {
		t.Fatal(err)
	}
	got, err := repository.ListEnabledForExecution(context.Background())
	if err != nil || len(got) != 2 || got[0].TemplateID != "a-template" || got[1].TemplateID != "z-template" {
		t.Fatalf("enabled templates = %#v, err=%v", got, err)
	}
	if err := db.Model(&model.POC{}).Where("template_id = ?", "z-template").Update("content_sha256", strings.Repeat("a", 64)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ListEnabledForExecution(context.Background()); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("digest drift error = %v", err)
	}
}

func TestListEnabledForExecutionAllowsEmptyCatalog(t *testing.T) {
	repository, _ := newRepositoryTest(t)
	got, err := repository.ListEnabledForExecution(context.Background())
	if err != nil || len(got) != 0 {
		t.Fatalf("empty enabled catalog = %#v, err=%v", got, err)
	}
}

func TestSetPOCActivationSucceedsForEmptyCatalogAndRejectsActiveSyncWithoutWrites(t *testing.T) {
	repository, db := newRepositoryTest(t)
	changed, err := repository.SetPOCActivation(context.Background(), true, nil)
	if err != nil || changed != 0 {
		t.Fatalf("empty catalog changed=%d err=%v, want successful no-op", changed, err)
	}
	now := time.Now().UTC()
	sourceID := uuid.New()
	if err := db.Create(&model.Source{ID: sourceID, SourceType: "git", RepoURL: "https://example.com/templates.git", IsActive: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	candidate := testCandidate(uuid.New(), sourceID, "guarded", "Guarded")
	if err := db.Select("*").Create(&model.POC{ID: uuid.New(), SourceID: sourceID, TemplateID: candidate.TemplateID, DisplayName: candidate.DisplayName, Severity: candidate.Severity, Tags: []byte(`[]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: candidate.RelativePath, ContentSHA256: candidate.ContentSHA256, Content: candidate.Content, IsEnabled: false, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	taskID, _ := createTestTask(t, db, uuid.New())
	changed, err = repository.SetPOCActivation(context.Background(), true, nil)
	if !errors.Is(err, domain.ErrActiveSyncConflict) || changed != 0 {
		t.Fatalf("active sync changed=%d err=%v, want conflict and zero writes", changed, err)
	}
	var persisted model.POC
	if err := db.Where("template_id = ?", "guarded").First(&persisted).Error; err != nil || persisted.IsEnabled {
		t.Fatalf("active-sync conflict mutated catalog: row=%#v err=%v task=%s", persisted, err, taskID)
	}
}

func TestSetPOCActivationRollsBackEveryRowWhenOneWriteFails(t *testing.T) {
	repository, db := newRepositoryTest(t)
	now := time.Now().UTC()
	sourceID := uuid.New()
	if err := db.Create(&model.Source{ID: sourceID, SourceType: "git", RepoURL: "https://example.com/templates.git", IsActive: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	for _, templateID := range []string{"bulk-before-failure", "bulk-fail"} {
		candidate := testCandidate(uuid.New(), sourceID, templateID, "Rollback")
		if err := db.Select("*").Create(&model.POC{ID: uuid.New(), SourceID: sourceID, TemplateID: templateID, DisplayName: candidate.DisplayName, Severity: candidate.Severity, Tags: []byte(`[]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: candidate.RelativePath, ContentSHA256: candidate.ContentSHA256, Content: candidate.Content, IsEnabled: false, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec(`CREATE TRIGGER nuclei_poc_activation_failure
        BEFORE UPDATE OF is_enabled ON nuclei_poc
        WHEN OLD.template_id = 'bulk-fail'
        BEGIN SELECT RAISE(ABORT, 'injected activation failure'); END;`).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Exec("DROP TRIGGER IF EXISTS nuclei_poc_activation_failure") })

	if changed, err := repository.SetPOCActivation(context.Background(), true, nil); err == nil || changed != 0 {
		t.Fatalf("SetPOCActivation() changed=%d err=%v, want rollback error and zero result", changed, err)
	}
	var rows []model.POC
	if err := db.Order("template_id ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.IsEnabled {
			t.Fatalf("row %s remained enabled after rollback: %#v", row.TemplateID, row)
		}
	}
}

func TestCatalogMutationCommitOrdersPreserveLatestState(t *testing.T) {
	t.Run("single update commits before promotion and is inherited", func(t *testing.T) {
		repository, db := newRepositoryTest(t)
		ctx := context.Background()
		now := time.Now().UTC().Add(-time.Hour)
		oldSourceID := uuid.New()
		if err := db.Create(&model.Source{ID: oldSourceID, SourceType: "git", RepoURL: "https://old.example/templates.git", IsActive: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
		old := testCandidate(uuid.New(), oldSourceID, "ordered-id", "Ordered")
		if err := db.Select("*").Create(&model.POC{ID: uuid.New(), SourceID: oldSourceID, TemplateID: old.TemplateID, DisplayName: old.DisplayName, Severity: old.Severity, Tags: []byte(`[]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: old.RelativePath, ContentSHA256: old.ContentSHA256, Content: old.Content, IsEnabled: false, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := repository.UpdatePOCEnabled(ctx, "nucleiPocs/ordered-id", true); err != nil {
			t.Fatalf("single update before promotion: %v", err)
		}
		newSourceID := uuid.New()
		taskID, _ := createTestTask(t, db, newSourceID)
		candidate := testCandidate(taskID, newSourceID, "ordered-id", "Changed Ordered")
		if err := repository.StageCandidate(ctx, candidate); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{TaskID: taskID, Source: domain.Source{ID: newSourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"}, CommitSHA: strings.Repeat("d", 40), SyncedAt: time.Now().UTC(), CleanupStatus: string(domain.CleanupClean)}); err != nil {
			t.Fatal(err)
		}
		var row model.POC
		if err := db.Where("template_id = ?", "ordered-id").First(&row).Error; err != nil || !row.IsEnabled {
			t.Fatalf("promoted row=%#v err=%v, want enabled", row, err)
		}
	})

	t.Run("promotion commits before single update and update reads the new row", func(t *testing.T) {
		repository, db := newRepositoryTest(t)
		ctx := context.Background()
		sourceID := uuid.New()
		taskID, _ := createTestTask(t, db, sourceID)
		if err := repository.StageCandidate(ctx, testCandidate(taskID, sourceID, "newly-committed", "Newly committed")); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{TaskID: taskID, Source: domain.Source{ID: sourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"}, CommitSHA: strings.Repeat("e", 40), SyncedAt: time.Now().UTC(), CleanupStatus: string(domain.CleanupClean)}); err != nil {
			t.Fatal(err)
		}
		updated, err := repository.UpdatePOCEnabled(ctx, "nucleiPocs/newly-committed", true)
		if err != nil || updated == nil || !updated.IsEnabled {
			t.Fatalf("single update after promotion=%#v err=%v, want enabled new row", updated, err)
		}
	})
}

func TestPromotionRejectsMismatchedSourceAndInvalidCleanupBeforeMutation(t *testing.T) {
	repository, db := newRepositoryTest(t)
	ctx := context.Background()
	sourceID := uuid.New()
	taskID, _ := createTestTask(t, db, sourceID)
	candidate := testCandidate(taskID, sourceID, "template-id", "Template")
	if err := repository.StageCandidate(ctx, candidate); err != nil {
		t.Fatal(err)
	}
	_, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{
		TaskID:    taskID,
		Source:    domain.Source{ID: sourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://other.example/templates.git"},
		CommitSHA: strings.Repeat("b", 40), SyncedAt: time.Now().UTC(), CleanupStatus: "unknown",
	})
	if !errors.Is(err, domain.ErrInvalidPOC) {
		t.Fatalf("err=%v, want invalid POC", err)
	}
	var persisted model.POC
	if queryErr := db.First(&persisted).Error; !errors.Is(queryErr, gorm.ErrRecordNotFound) {
		t.Fatalf("promotion mutated catalog on validation failure: row=%+v err=%v", persisted, queryErr)
	}
}

func TestMarkTaskTerminalOnlyFreezesFailure(t *testing.T) {
	repository, db := newRepositoryTest(t)
	ctx := context.Background()
	taskID, _ := createTestTask(t, db, uuid.New())
	if err := repository.MarkTaskTerminal(ctx, taskID, domain.SyncTaskSucceeded, "", "", domain.Diagnostics{}, string(domain.CleanupClean), time.Now().UTC()); !errors.Is(err, domain.ErrInvalidPOC) {
		t.Fatalf("success terminal update err=%v, want invalid POC", err)
	}
	if err := repository.MarkTaskTerminal(ctx, taskID, domain.SyncTaskFailed, "TEMPLATE_INVALID", "ignored", domain.Diagnostics{}, string(domain.CleanupClean), time.Now().UTC()); err != nil {
		t.Fatalf("failure terminal update: %v", err)
	}
	task, err := repository.GetSyncTask(ctx, taskID)
	if err != nil || task.State != domain.SyncTaskFailed {
		t.Fatalf("task=%+v err=%v", task, err)
	}
}

func TestPromotionAndFailureWriteTaskScopedNucleiOutboxOccurrences(t *testing.T) {
	repository, db := newRepositoryTest(t)
	ctx := context.Background()
	newSourceID := uuid.New()
	taskID, _ := createTestTask(t, db, newSourceID)
	if err := repository.StageCandidate(ctx, testCandidate(taskID, newSourceID, "template-id", "Template")); err != nil {
		t.Fatal(err)
	}
	commitSHA := strings.Repeat("a", 40)
	if _, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{
		TaskID:    taskID,
		Source:    domain.Source{ID: newSourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"},
		CommitSHA: commitSHA, SyncedAt: time.Now().UTC(), CleanupStatus: string(domain.CleanupClean),
	}); err != nil {
		t.Fatalf("PromoteCandidates: %v", err)
	}
	var successOutbox notificationmodel.Outbox
	if err := db.Where("kind = ?", "nuclei-poc-sync-succeeded").First(&successOutbox).Error; err != nil {
		t.Fatalf("load success outbox: %v", err)
	}
	if successOutbox.EventID != "nuclei-poc-sync:"+taskID.String()+":succeeded" || !strings.Contains(string(successOutbox.Payload), commitSHA) {
		t.Fatalf("success outbox = %#v, want stable identity and full SHA", successOutbox)
	}
	if count, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{TaskID: taskID}); err != nil || count != 1 {
		t.Fatalf("replayed PromoteCandidates = count %d err %v, want idempotent success", count, err)
	}
	var successCount int64
	if err := db.Model(&notificationmodel.Outbox{}).Where("event_id = ?", successOutbox.EventID).Count(&successCount).Error; err != nil {
		t.Fatalf("count replayed success outbox: %v", err)
	}
	if successCount != 1 {
		t.Fatalf("replayed success outbox count = %d, want 1", successCount)
	}

	failureSourceID := uuid.New()
	failureTaskID, _ := createTestTask(t, db, failureSourceID)
	if err := repository.MarkTaskTerminal(ctx, failureTaskID, domain.SyncTaskFailed, "TEMPLATE_INVALID", "ignored", domain.Diagnostics{}, string(domain.CleanupClean), time.Now().UTC()); err != nil {
		t.Fatalf("MarkTaskTerminal: %v", err)
	}
	var failureOutbox notificationmodel.Outbox
	if err := db.Where("kind = ?", "nuclei-poc-sync-failed").First(&failureOutbox).Error; err != nil {
		t.Fatalf("load failure outbox: %v", err)
	}
	if failureOutbox.EventID != "nuclei-poc-sync:"+failureTaskID.String()+":failed" || strings.Contains(string(failureOutbox.Payload), "ignored") {
		t.Fatalf("failure outbox = %#v, want stable identity and sanitized summary", failureOutbox)
	}

	if err := repository.MarkTaskTerminal(ctx, failureTaskID, domain.SyncTaskFailed, "TEMPLATE_INVALID", "ignored", domain.Diagnostics{}, string(domain.CleanupClean), time.Now().UTC()); err != nil {
		t.Fatalf("replayed MarkTaskTerminal: %v", err)
	}
	var failureCount int64
	if err := db.Model(&notificationmodel.Outbox{}).Where("event_id = ?", failureOutbox.EventID).Count(&failureCount).Error; err != nil {
		t.Fatalf("count replayed failure outbox: %v", err)
	}
	if failureCount != 1 {
		t.Fatalf("replayed failure outbox count = %d, want 1", failureCount)
	}
}

func TestNucleiProducerFailureRollsBackSuccessAndFailureTerminalFacts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Source{}, &model.SyncTask{}, &model.CandidateImport{}, &model.POC{}, &model.RequestTombstone{}); err != nil {
		t.Fatal(err)
	}
	failingSink := &failingNucleiNotificationSink{err: errors.New("outbox unavailable")}
	repository := NewNucleiPOCRepository(db, failingSink)
	ctx := context.Background()
	now := time.Now().UTC()
	oldSourceID := uuid.New()
	if err := db.Create(&model.Source{ID: oldSourceID, SourceType: "git", RepoURL: "https://old.example/templates.git", IsActive: true, CommitSHA: strings.Repeat("a", 40), CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed active source: %v", err)
	}
	old := testCandidate(uuid.New(), oldSourceID, "retained-template", "Retained template")
	if err := db.Create(&model.POC{ID: uuid.New(), SourceID: oldSourceID, TemplateID: old.TemplateID, DisplayName: old.DisplayName, Severity: old.Severity, Tags: []byte(`[]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: old.RelativePath, ContentSHA256: old.ContentSHA256, Content: old.Content, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed active POC: %v", err)
	}
	sourceID := uuid.New()
	taskID, requestID := createTestTask(t, db, sourceID)
	if err := repository.StageCandidate(ctx, testCandidate(taskID, sourceID, "template-id", "Template")); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{
		TaskID:    taskID,
		Source:    domain.Source{ID: sourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"},
		CommitSHA: strings.Repeat("b", 40), SyncedAt: time.Now().UTC(), CleanupStatus: string(domain.CleanupClean),
	}); !strings.Contains(err.Error(), "outbox unavailable") {
		t.Fatalf("PromoteCandidates error = %v, want producer error", err)
	}
	task, err := repository.GetSyncTask(ctx, taskID)
	if err != nil || task.State.Terminal() || task.CommitSHA != "" || task.CommittedPOCCount != 0 || task.CompletedAt != nil {
		t.Fatalf("success producer failure task = %#v err=%v, want non-terminal rollback", task, err)
	}
	var rolledBackSource model.Source
	if err := db.Where("id = ?", sourceID).First(&rolledBackSource).Error; err != nil || rolledBackSource.IsActive {
		t.Fatalf("success producer failure source = %#v err=%v, want inactive staged source", rolledBackSource, err)
	}
	var preservedSource model.Source
	if err := db.Where("id = ?", oldSourceID).First(&preservedSource).Error; err != nil || !preservedSource.IsActive {
		t.Fatalf("success producer failure old source = %#v err=%v, want active collection preserved", preservedSource, err)
	}
	var preservedPOC model.POC
	if err := db.Where("source_id = ? AND template_id = ?", oldSourceID, old.TemplateID).First(&preservedPOC).Error; err != nil {
		t.Fatalf("success producer failure removed old POC collection: %v", err)
	}
	var replacementCount int64
	if err := db.Model(&model.POC{}).Where("source_id = ?", sourceID).Count(&replacementCount).Error; err != nil || replacementCount != 0 {
		t.Fatalf("success producer failure replacement POCs = %d err=%v, want 0", replacementCount, err)
	}
	var stagedCandidateCount int64
	if err := db.Model(&model.CandidateImport{}).Where("task_id = ?", taskID).Count(&stagedCandidateCount).Error; err != nil || stagedCandidateCount != 1 {
		t.Fatalf("success producer failure staged candidates = %d err=%v, want unchanged staged candidate", stagedCandidateCount, err)
	}
	var successTombstone model.RequestTombstone
	if err := db.Where("request_id = ?", requestID).First(&successTombstone).Error; err != nil || successTombstone.TerminalAt != nil {
		t.Fatalf("success producer failure tombstone = %#v err=%v, want non-terminal rollback", successTombstone, err)
	}

	failureTaskID, failureRequestID := createTestTask(t, db, uuid.New())
	if err := repository.MarkTaskTerminal(ctx, failureTaskID, domain.SyncTaskFailed, "TEMPLATE_INVALID", "ignored", domain.Diagnostics{}, string(domain.CleanupClean), time.Now().UTC()); !strings.Contains(err.Error(), "outbox unavailable") {
		t.Fatalf("MarkTaskTerminal error = %v, want producer error", err)
	}
	failureTask, err := repository.GetSyncTask(ctx, failureTaskID)
	if err != nil || failureTask.State.Terminal() || failureTask.FailureCode != "" || failureTask.FailureSummary != "" || failureTask.CompletedAt != nil {
		t.Fatalf("failure producer failure task = %#v err=%v, want non-terminal rollback", failureTask, err)
	}
	var failureTombstone model.RequestTombstone
	if err := db.Where("request_id = ?", failureRequestID).First(&failureTombstone).Error; err != nil || failureTombstone.TerminalAt != nil {
		t.Fatalf("failure producer failure tombstone = %#v err=%v, want non-terminal rollback", failureTombstone, err)
	}
}

func TestNucleiNonTerminalConflictExpiredAndMissingPathsDoNotWriteOccurrences(t *testing.T) {
	repository, db := newRepositoryTest(t)
	ctx := context.Background()
	input := app.CreateSyncInput{RequestID: uuid.New(), SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"}
	if _, err := repository.CreateOrReplaySyncTask(ctx, input, "git\x00https://example.com/templates.git", time.Now().UTC()); err != nil {
		t.Fatalf("CreateOrReplaySyncTask: %v", err)
	}
	second := app.CreateSyncInput{RequestID: uuid.New(), SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/other.git"}
	if _, err := repository.CreateOrReplaySyncTask(ctx, second, "git\x00https://example.com/other.git", time.Now().UTC()); !errors.Is(err, domain.ErrActiveSyncConflict) {
		t.Fatalf("second active task error = %v, want active conflict", err)
	}
	if _, err := repository.GetSyncTask(ctx, uuid.New()); !errors.Is(err, domain.ErrSyncTaskNotFound) {
		t.Fatalf("missing task error = %v, want not found", err)
	}

	expiredTaskID := uuid.New()
	if err := db.Create(&model.SyncTask{
		ID: expiredTaskID, RequestID: uuid.New(), RequestFingerprint: fingerprintDigest("expired"), SourceType: "git", RepoURL: "https://example.com/expired.git", SourceID: uuid.New(),
		State: string(domain.SyncTaskSucceeded), Phase: string(domain.SyncTaskSucceeded), Diagnostics: []byte(`{"samples":[],"total":0,"truncated":false}`), CleanupStatus: string(domain.CleanupClean), CompletedAt: pointerToTime(time.Now().UTC().Add(-31 * 24 * time.Hour)), CreatedAt: time.Now().UTC().Add(-31 * 24 * time.Hour), UpdatedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("seed expired task: %v", err)
	}
	if _, err := repository.DeleteExpiredTasks(ctx, time.Now().UTC().Add(-30*24*time.Hour), 10); err != nil {
		t.Fatalf("DeleteExpiredTasks: %v", err)
	}
	var outboxCount int64
	if err := db.Model(&notificationmodel.Outbox{}).Count(&outboxCount).Error; err != nil {
		t.Fatalf("count notification outbox: %v", err)
	}
	if outboxCount != 0 {
		t.Fatalf("non-terminal/conflict/expired/missing paths wrote %d notification occurrences, want 0", outboxCount)
	}
}

func TestRecoverInterruptedTaskWritesFailureOccurrenceAtomically(t *testing.T) {
	repository, db := newRepositoryTest(t)
	ctx := context.Background()
	sourceID := uuid.New()
	taskID, _ := createTestTask(t, db, sourceID)
	if _, err := repository.RecoverInterruptedTasks(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("RecoverInterruptedTasks: %v", err)
	}
	task, err := repository.GetSyncTask(ctx, taskID)
	if err != nil || task.State != domain.SyncTaskFailed {
		t.Fatalf("recovered task = %#v err=%v, want failed", task, err)
	}
	var outbox notificationmodel.Outbox
	if err := db.Where("event_id = ?", "nuclei-poc-sync:"+taskID.String()+":failed").First(&outbox).Error; err != nil {
		t.Fatalf("load recovered failure occurrence: %v", err)
	}
}

func TestNucleiPostCommitOutboxFailuresDoNotRewriteTerminalFacts(t *testing.T) {
	repository, db := newRepositoryTest(t)
	ctx := context.Background()
	sourceID := uuid.New()
	taskID, _ := createTestTask(t, db, sourceID)
	if err := repository.StageCandidate(ctx, testCandidate(taskID, sourceID, "template-id", "Template")); err != nil {
		t.Fatal(err)
	}
	commitSHA := strings.Repeat("c", 40)
	if _, err := repository.PromoteCandidates(ctx, app.CandidatePromotion{
		TaskID:        taskID,
		Source:        domain.Source{ID: sourceID, SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"},
		CommitSHA:     commitSHA,
		SyncedAt:      time.Now().UTC(),
		CleanupStatus: string(domain.CleanupClean),
	}); err != nil {
		t.Fatalf("PromoteCandidates: %v", err)
	}
	assertNucleiSuccessFacts(t, repository, db, taskID, sourceID, commitSHA)

	outbox := notificationrepo.NewOutboxRepository(db)
	transientWorker := notificationapp.NewOutboxWorker(
		outbox,
		nucleiOutboxMaterializerFake{err: errors.New("temporary projection store failure")},
		notificationapp.OutboxWorkerOptions{Owner: "nuclei-post-commit-test"},
	)
	if err := transientWorker.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce transient failure: %v", err)
	}
	var event notificationmodel.Outbox
	if err := db.Where("event_id = ?", "nuclei-poc-sync:"+taskID.String()+":succeeded").First(&event).Error; err != nil {
		t.Fatalf("load transient outbox event: %v", err)
	}
	if event.Status != "pending" || event.TerminalAt != nil || event.PublishedAt != nil {
		t.Fatalf("transient outbox event = %#v, want retryable pending state", event)
	}
	assertNucleiSuccessFacts(t, repository, db, taskID, sourceID, commitSHA)

	unsupportedWorker := notificationapp.NewOutboxWorker(
		outbox,
		nucleiOutboxMaterializerFake{err: &notificationapp.UnsupportedEventError{Code: "template_render_failed", Err: errors.New("invalid server template")}},
		notificationapp.OutboxWorkerOptions{Owner: "nuclei-post-commit-test"},
	)
	if err := unsupportedWorker.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce unsupported failure: %v", err)
	}
	if err := db.Where("id = ?", event.ID).First(&event).Error; err != nil {
		t.Fatalf("reload unsupported outbox event: %v", err)
	}
	if event.Status != "unsupported_terminal" || event.FailureCode != "template_render_failed" || event.TerminalAt == nil {
		t.Fatalf("unsupported outbox event = %#v, want terminal notification isolation", event)
	}
	assertNucleiSuccessFacts(t, repository, db, taskID, sourceID, commitSHA)
}

func assertNucleiSuccessFacts(t *testing.T, repository *NucleiPOCRepository, db *gorm.DB, taskID, sourceID uuid.UUID, commitSHA string) {
	t.Helper()
	task, err := repository.GetSyncTask(context.Background(), taskID)
	if err != nil || task.State != domain.SyncTaskSucceeded || task.CommitSHA != commitSHA || task.CommittedPOCCount != 1 {
		t.Fatalf("Nuclei task after post-commit notification failure = %#v err=%v, want unchanged success", task, err)
	}
	var source model.Source
	if err := db.Where("id = ?", sourceID).First(&source).Error; err != nil || !source.IsActive || source.CommitSHA != commitSHA {
		t.Fatalf("Nuclei source after post-commit notification failure = %#v err=%v, want unchanged active source", source, err)
	}
	var pocCount int64
	if err := db.Model(&model.POC{}).Where("source_id = ?", sourceID).Count(&pocCount).Error; err != nil || pocCount != 1 {
		t.Fatalf("Nuclei POC count after post-commit notification failure = %d err=%v, want unchanged catalog", pocCount, err)
	}
}

func pointerToTime(value time.Time) *time.Time {
	return &value
}

type failingNucleiNotificationSink struct{ err error }

func (sink *failingNucleiNotificationSink) WriteNucleiPOCSyncSucceeded(*gorm.DB, uuid.UUID, time.Time) error {
	return sink.err
}

func (sink *failingNucleiNotificationSink) WriteNucleiPOCSyncFailed(*gorm.DB, uuid.UUID, time.Time) error {
	return sink.err
}

type nucleiOutboxMaterializerFake struct{ err error }

func (fake nucleiOutboxMaterializerFake) Materialize(context.Context, notificationdomain.OutboxEvent) error {
	return fake.err
}

func TestCreateOrReplaySyncTaskAndTombstone(t *testing.T) {
	repository, _ := newRepositoryTest(t)
	ctx := context.Background()
	input := app.CreateSyncInput{RequestID: uuid.New(), SourceType: domain.SourceTypeGit, RepoURL: "https://example.com/templates.git"}
	task, err := repository.CreateOrReplaySyncTask(ctx, input, "git\x00https://example.com/templates.git", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	replay, err := repository.CreateOrReplaySyncTask(ctx, input, "git\x00https://example.com/templates.git", time.Now().UTC())
	if err != nil || replay.ID != task.ID {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	conflict := input
	conflict.SourceType = domain.SourceTypeCustom
	if _, err := repository.CreateOrReplaySyncTask(ctx, conflict, "custom\x00https://example.com/templates.git", time.Now().UTC()); !errors.Is(err, domain.ErrRequestReplayConflict) {
		t.Fatalf("err=%v, want replay conflict", err)
	}
}

func TestPOCCursorUsesRequestedPrimaryFieldAndAscendingTieBreak(t *testing.T) {
	row := model.POC{TemplateID: "z-template", DisplayName: "same", Severity: "low"}
	if !isAfterPOCCursor(row, "name desc", app.POCCursor{Value: "z-name", TemplateID: "a-template"}) {
		t.Fatal("row should be after a lower cursor value for descending order")
	}
	if isAfterPOCCursor(row, "name desc", app.POCCursor{Value: strings.ToLower(row.DisplayName), TemplateID: "zz-template"}) {
		t.Fatal("descending primary order must retain ascending templateId tie-break")
	}
}
