package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/notification/repository/persistence"
	nucleipocdomain "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	nucleipocrepo "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/repository"
	nucleipocmodel "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMaterializationRepositoriesFreezeAudienceAndDestinationFanout(t *testing.T) {
	db := newNotificationProjectionDB(t)
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	insertNotificationUser(t, db, 1, "zh", true)
	destinations := NewDestinationRepository(db)
	if _, err := destinations.Update(context.Background(), domain.Destination{
		Provider:      domain.ProviderDiscord,
		Credential:    "https://discord.com/api/webhooks/materializer/token",
		Enabled:       true,
		Subscriptions: []domain.Kind{domain.KindScanSucceeded},
	}); err != nil {
		t.Fatalf("configure Discord destination: %v", err)
	}

	occurrence, err := domain.NewScanSucceededOccurrence(11, 3, "example.test", now)
	if err != nil {
		t.Fatalf("NewScanSucceededOccurrence: %v", err)
	}
	materializer := notificationapp.NewMaterializer(
		NewTransactionCoordinator(db),
		NewFactRepository(db),
		NewAudienceRepository(db),
		destinations,
		notificationapp.NewTemplates(),
		nil,
	)
	event := domain.OutboxEvent{ID: 1, Occurrence: occurrence}
	if err := materializer.Materialize(context.Background(), event); err != nil {
		t.Fatalf("first Materialize: %v", err)
	}

	var firstInbox model.Inbox
	if err := db.Where("user_id = ?", 1).First(&firstInbox).Error; err != nil {
		t.Fatalf("load frozen inbox: %v", err)
	}
	if firstInbox.Locale != string(domain.LocaleChinese) || firstInbox.Title != "扫描完成" {
		t.Fatalf("first inbox snapshot = locale %q title %q, want zh snapshot", firstInbox.Locale, firstInbox.Title)
	}
	assertNotificationProjectionCounts(t, db, 1, 1, 1, 1)

	insertNotificationUser(t, db, 2, "en", true)
	if err := db.Table("auth_user").Where("id = ?", 1).Update("locale", "en").Error; err != nil {
		t.Fatalf("change original user locale: %v", err)
	}
	if _, err := destinations.Update(context.Background(), domain.Destination{
		Provider:      domain.ProviderWeCom,
		Credential:    "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=materializer-token",
		Enabled:       true,
		Subscriptions: []domain.Kind{domain.KindScanSucceeded},
	}); err != nil {
		t.Fatalf("configure later WeCom destination: %v", err)
	}
	if err := materializer.Materialize(context.Background(), event); err != nil {
		t.Fatalf("replay Materialize: %v", err)
	}
	assertNotificationProjectionCounts(t, db, 1, 1, 1, 1)

	var replayedInbox model.Inbox
	if err := db.Where("id = ?", firstInbox.ID).First(&replayedInbox).Error; err != nil {
		t.Fatalf("load replayed inbox: %v", err)
	}
	if replayedInbox.Locale != string(domain.LocaleChinese) || replayedInbox.Title != firstInbox.Title || replayedInbox.Message != firstInbox.Message {
		t.Fatalf("replay changed frozen inbox snapshot: before=%#v after=%#v", firstInbox, replayedInbox)
	}
	var fact model.Fact
	if err := db.Where("event_id = ?", occurrence.EventID).First(&fact).Error; err != nil {
		t.Fatalf("load fact: %v", err)
	}
	if fact.AudienceFrozenAt == nil || fact.DestinationsFrozenAt == nil {
		t.Fatalf("fact freeze markers = audience %v destination %v, want both set", fact.AudienceFrozenAt, fact.DestinationsFrozenAt)
	}
}

func TestFixedInboxDeliveryProjectsEveryActiveUserAndPreservesExternalFanout(t *testing.T) {
	db := newNotificationProjectionDB(t)
	ctx := context.Background()
	now := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	insertNotificationUser(t, db, 1, "en", true)
	insertNotificationUser(t, db, 2, "zh", true)
	insertNotificationUser(t, db, 3, "en", false)

	destinations := NewDestinationRepository(db)
	materializer := notificationapp.NewMaterializer(
		NewTransactionCoordinator(db),
		NewFactRepository(db),
		NewAudienceRepository(db),
		destinations,
		notificationapp.NewTemplates(),
		nil,
	)
	scanSucceeded, err := domain.NewScanSucceededOccurrence(11, 3, "example.test", now)
	if err != nil {
		t.Fatalf("NewScanSucceededOccurrence: %v", err)
	}
	vulnerabilityObserved, err := domain.NewVulnerabilityObservedOccurrence(domain.VulnerabilityObservedPayload{
		ScanID:     12,
		TargetID:   4,
		TargetName: "example.test",
		URL:        "https://example.test",
		VulnType:   "xss",
		Severity:   "high",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("NewVulnerabilityObservedOccurrence: %v", err)
	}
	agentOffline, err := domain.NewAgentOfflineOccurrence(5, "agent-a", now.Add(-time.Minute), now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("NewAgentOfflineOccurrence: %v", err)
	}
	for index, occurrence := range []domain.Occurrence{scanSucceeded, vulnerabilityObserved, agentOffline} {
		if err := materializer.Materialize(ctx, domain.OutboxEvent{ID: int64(index + 1), Occurrence: occurrence}); err != nil {
			t.Fatalf("materialize fixed-inbox occurrence %q: %v", occurrence.Kind, err)
		}
	}

	assertNotificationProjectionCounts(t, db, 3, 6, 6, 0)
	var inactiveInboxCount int64
	if err := db.Model(&model.Inbox{}).Where("user_id = ?", 3).Count(&inactiveInboxCount).Error; err != nil {
		t.Fatalf("count inactive user inbox projections: %v", err)
	}
	if inactiveInboxCount != 0 {
		t.Fatalf("inactive user inbox projections = %d, want 0", inactiveInboxCount)
	}

	if _, err := destinations.Update(ctx, domain.Destination{
		Provider:      domain.ProviderDiscord,
		Credential:    "https://discord.com/api/webhooks/materializer/token",
		Enabled:       true,
		Subscriptions: []domain.Kind{domain.KindScanFailed},
	}); err != nil {
		t.Fatalf("configure Discord destination: %v", err)
	}
	scanFailed, err := domain.NewScanFailedOccurrence(13, 5, "example.test", "network_error", "connection reset", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("NewScanFailedOccurrence: %v", err)
	}
	if err := materializer.Materialize(ctx, domain.OutboxEvent{ID: 4, Occurrence: scanFailed}); err != nil {
		t.Fatalf("materialize externally configured occurrence: %v", err)
	}
	assertNotificationProjectionCounts(t, db, 4, 8, 8, 1)

	insertNotificationUser(t, db, 4, "en", true)
	if err := materializer.Materialize(ctx, domain.OutboxEvent{ID: 1, Occurrence: scanSucceeded}); err != nil {
		t.Fatalf("replay frozen audience: %v", err)
	}
	assertNotificationProjectionCounts(t, db, 4, 8, 8, 1)
}

func TestNucleiMaterializationFreezesInboxOnlyAudienceWithoutDeliveries(t *testing.T) {
	db := newNotificationProjectionDB(t)
	ctx := context.Background()
	now := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	insertNotificationUser(t, db, 1, "en", true)
	insertNotificationUser(t, db, 2, "zh", true)

	destinations := NewDestinationRepository(db)
	if _, err := destinations.Update(ctx, domain.Destination{
		Provider:      domain.ProviderDiscord,
		Credential:    "https://discord.com/api/webhooks/materializer/token",
		Enabled:       true,
		Subscriptions: []domain.Kind{domain.KindScanSucceeded},
	}); err != nil {
		t.Fatalf("configure Discord destination: %v", err)
	}
	materializer := notificationapp.NewMaterializer(
		NewTransactionCoordinator(db),
		NewFactRepository(db),
		NewAudienceRepository(db),
		destinations,
		notificationapp.NewTemplates(),
		nil,
	)
	taskID := uuid.MustParse("6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12")
	occurrence, err := domain.NewNucleiPOCSyncSucceededOccurrence(taskID, "git", "0123456789abcdef0123456789abcdef01234567", 184, now)
	if err != nil {
		t.Fatalf("NewNucleiPOCSyncSucceededOccurrence: %v", err)
	}
	if err := materializer.Materialize(ctx, domain.OutboxEvent{ID: 1, Occurrence: occurrence}); err != nil {
		t.Fatalf("Materialize nuclei occurrence: %v", err)
	}
	assertNotificationProjectionCounts(t, db, 1, 2, 2, 0)

	var fact model.Fact
	if err := db.Where("event_id = ?", occurrence.EventID).First(&fact).Error; err != nil {
		t.Fatalf("load nuclei fact: %v", err)
	}
	if fact.DestinationsFrozenAt == nil {
		t.Fatal("inbox-only Nuclei fact did not freeze an empty destination set")
	}
	var inboxes []model.Inbox
	if err := db.Order("user_id ASC").Find(&inboxes).Error; err != nil {
		t.Fatalf("load nuclei inbox snapshots: %v", err)
	}
	if len(inboxes) != 2 || inboxes[0].Locale != "en" || inboxes[1].Locale != "zh" || !strings.Contains(inboxes[0].Message, "0123456789ab") {
		t.Fatalf("nuclei inbox snapshots = %#v, want active-user locale-frozen displays", inboxes)
	}
}

func TestNucleiTaskRetentionDoesNotShortenFrozenInboxHistory(t *testing.T) {
	db := newNotificationProjectionDB(t)
	if err := db.AutoMigrate(&nucleipocmodel.Source{}, &nucleipocmodel.SyncTask{}, &nucleipocmodel.CandidateImport{}, &nucleipocmodel.POC{}); err != nil {
		t.Fatalf("migrate nuclei retention records: %v", err)
	}
	now := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	insertNotificationUser(t, db, 1, "en", true)
	taskID := uuid.MustParse("6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12")
	sourceID := uuid.New()
	completedAt := now.Add(-31 * 24 * time.Hour)
	if err := db.Create(&nucleipocmodel.Source{ID: sourceID, SourceType: "git", RepoURL: "https://example.com/templates.git", CreatedAt: completedAt, UpdatedAt: completedAt}).Error; err != nil {
		t.Fatalf("create expired source: %v", err)
	}
	if err := db.Create(&nucleipocmodel.SyncTask{
		ID: taskID, RequestID: uuid.New(), RequestFingerprint: "retention", SourceType: "git", RepoURL: "https://example.com/templates.git", SourceID: sourceID,
		State: string(nucleipocdomain.SyncTaskSucceeded), Phase: string(nucleipocdomain.SyncTaskSucceeded), CommitSHA: "0123456789abcdef0123456789abcdef01234567", CommittedPOCCount: 184,
		Diagnostics: []byte(`{"samples":[],"total":0,"truncated":false}`), CleanupStatus: string(nucleipocdomain.CleanupClean), CompletedAt: &completedAt, CreatedAt: completedAt, UpdatedAt: completedAt,
	}).Error; err != nil {
		t.Fatalf("create expired task: %v", err)
	}

	occurrence, err := domain.NewNucleiPOCSyncSucceededOccurrence(taskID, "git", "0123456789abcdef0123456789abcdef01234567", 184, completedAt)
	if err != nil {
		t.Fatalf("NewNucleiPOCSyncSucceededOccurrence: %v", err)
	}
	materializer := notificationapp.NewMaterializer(
		NewTransactionCoordinator(db),
		NewFactRepository(db),
		NewAudienceRepository(db),
		NewDestinationRepository(db),
		notificationapp.NewTemplates(),
		nil,
	)
	if err := materializer.Materialize(context.Background(), domain.OutboxEvent{ID: 1, Occurrence: occurrence}); err != nil {
		t.Fatalf("Materialize nuclei occurrence: %v", err)
	}
	// Materialization normally timestamps an inbox projection at projection
	// time. Backfill its creation time to model an already-committed 31-day-old
	// notification independently from the task row before exercising retention.
	if err := db.Model(&model.Inbox{}).Where("user_id = ?", 1).Update("created_at", completedAt).Error; err != nil {
		t.Fatalf("backfill inbox creation time: %v", err)
	}
	nuclei := nucleipocrepo.NewNucleiPOCRepository(db)
	if _, err := nuclei.DeleteExpiredTasks(context.Background(), now.Add(-30*24*time.Hour), 10); err != nil {
		t.Fatalf("DeleteExpiredTasks: %v", err)
	}
	if _, err := nuclei.GetSyncTask(context.Background(), taskID); err == nil {
		t.Fatal("expired Nuclei task is still readable")
	}
	inboxRepository := NewInboxRepository(db)
	inbox, err := inboxRepository.List(context.Background(), 1, 10, "", now)
	if err != nil || len(inbox.Results) != 1 || inbox.Results[0].Title != "Nuclei POC sync completed" {
		t.Fatalf("frozen inbox after task expiry = %#v err=%v, want readable notification snapshot", inbox, err)
	}
	if inbox.Results[0].CreatedAt.UTC() != completedAt {
		t.Fatalf("frozen inbox creation time = %s, want 31-day-old %s", inbox.Results[0].CreatedAt, completedAt)
	}
	if unread, err := inboxRepository.UnreadCount(context.Background(), 1, now); err != nil || unread != 1 {
		t.Fatalf("unread count after 30-day task expiry = %d err=%v, want retained inbox row", unread, err)
	}
	if deleted, err := inboxRepository.DeleteExpiredInbox(context.Background(), domain.RetentionBatch{Limit: 10, Before: now.Add(-domain.NotificationRetention)}); err != nil || deleted != 0 {
		t.Fatalf("90-day inbox retention deleted %d rows err=%v, want 31-day-old row retained", deleted, err)
	}
}

func TestUpdateLocaleIsIdempotentAndCorrectsPreviouslyInitializedValue(t *testing.T) {
	db := newNotificationProjectionDB(t)
	insertNotificationUser(t, db, 1, "en", true)
	repository := NewInboxRepository(db)

	if err := repository.UpdateLocale(context.Background(), 1, domain.LocaleChinese); err != nil {
		t.Fatalf("update locale to zh: %v", err)
	}
	var first struct {
		Locale              string     `gorm:"column:locale"`
		LocaleInitializedAt *time.Time `gorm:"column:locale_initialized_at"`
	}
	if err := db.Table("auth_user").Select("locale, locale_initialized_at").Where("id = ?", 1).Take(&first).Error; err != nil {
		t.Fatalf("read first locale update: %v", err)
	}
	if first.Locale != string(domain.LocaleChinese) || first.LocaleInitializedAt == nil {
		t.Fatalf("first locale state = %#v, want zh with initialization timestamp", first)
	}

	if err := repository.UpdateLocale(context.Background(), 1, domain.LocaleChinese); err != nil {
		t.Fatalf("repeat locale update: %v", err)
	}
	var repeated struct {
		Locale              string     `gorm:"column:locale"`
		LocaleInitializedAt *time.Time `gorm:"column:locale_initialized_at"`
	}
	if err := db.Table("auth_user").Select("locale, locale_initialized_at").Where("id = ?", 1).Take(&repeated).Error; err != nil {
		t.Fatalf("read repeated locale update: %v", err)
	}
	if repeated.Locale != first.Locale || repeated.LocaleInitializedAt == nil || !repeated.LocaleInitializedAt.Equal(*first.LocaleInitializedAt) {
		t.Fatalf("repeat locale state = %#v, want unchanged %#v", repeated, first)
	}

	if err := repository.UpdateLocale(context.Background(), 1, domain.LocaleEnglish); err != nil {
		t.Fatalf("correct locale back to en: %v", err)
	}
	var corrected struct {
		Locale string `gorm:"column:locale"`
	}
	if err := db.Table("auth_user").Select("locale").Where("id = ?", 1).Take(&corrected).Error; err != nil {
		t.Fatalf("read corrected locale: %v", err)
	}
	if corrected.Locale != string(domain.LocaleEnglish) {
		t.Fatalf("corrected locale = %q, want en", corrected.Locale)
	}
}

func newNotificationProjectionDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE auth_user (
	        id INTEGER PRIMARY KEY,
	        locale TEXT NOT NULL,
	        locale_initialized_at DATETIME,
	        is_active BOOLEAN NOT NULL
    )`).Error; err != nil {
		t.Fatalf("create auth_user: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Fact{},
		&model.Recipient{},
		&model.Inbox{},
		&model.Destination{},
		&model.DestinationSubscription{},
		&model.Delivery{},
	); err != nil {
		t.Fatalf("migrate notification projection tables: %v", err)
	}
	return db
}

func insertNotificationUser(t *testing.T, db *gorm.DB, id int, locale string, active bool) {
	t.Helper()
	if err := db.Exec("INSERT INTO auth_user (id, locale, is_active) VALUES (?, ?, ?)", id, locale, active).Error; err != nil {
		t.Fatalf("insert notification user %d: %v", id, err)
	}
}

func assertNotificationProjectionCounts(t *testing.T, db *gorm.DB, facts, recipients, inbox, deliveries int64) {
	t.Helper()
	for _, assertion := range []struct {
		name   string
		model  any
		wanted int64
	}{
		{name: "facts", model: &model.Fact{}, wanted: facts},
		{name: "recipients", model: &model.Recipient{}, wanted: recipients},
		{name: "inbox", model: &model.Inbox{}, wanted: inbox},
		{name: "deliveries", model: &model.Delivery{}, wanted: deliveries},
	} {
		var actual int64
		if err := db.Model(assertion.model).Count(&actual).Error; err != nil {
			t.Fatalf("count %s: %v", assertion.name, err)
		}
		if actual != assertion.wanted {
			t.Fatalf("%s count = %d, want %d", assertion.name, actual, assertion.wanted)
		}
	}
}
