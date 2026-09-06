package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	notificationprovider "github.com/yyhuni/lunafox/server/internal/modules/notification/provider"
	notificationrepo "github.com/yyhuni/lunafox/server/internal/modules/notification/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// verifyNotificationFreshPostgresPipeline runs as part of the guarded empty
// baseline test. It crosses real PostgreSQL persistence and actual provider
// adapters so a fresh-install schema cannot silently drift from the worker
// lifecycle it is meant to support.
func verifyNotificationFreshPostgresPipeline(t *testing.T, ctx context.Context, sqlDB *sql.DB, dsn string) {
	t.Helper()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open GORM PostgreSQL for notification pipeline: %v", err)
	}

	var userID int
	if err := sqlDB.QueryRowContext(ctx, `SELECT id FROM auth_user WHERE username = 'admin'`).Scan(&userID); err != nil {
		t.Fatalf("load fresh notification recipient: %v", err)
	}
	targetID, scanID := seedNotificationFreshPostgresScan(t, ctx, sqlDB)
	agentID := seedNotificationFreshPostgresAgent(t, ctx, sqlDB)
	assertNotificationFreshPostgresFeishuDestination(t, ctx, sqlDB)

	providerTransport := &notificationFreshPostgresProviderTransport{t: t}
	providerClient := &http.Client{Transport: providerTransport}

	outbox := notificationrepo.NewOutboxRepository(db)
	destinations := notificationrepo.NewDestinationRepository(db)
	if _, err := destinations.Update(ctx, domain.Destination{
		Provider:      domain.ProviderDiscord,
		Credential:    "https://discord.com/api/webhooks/fresh-postgres-discord/fresh-postgres-token",
		Enabled:       true,
		Subscriptions: []domain.Kind{domain.KindScanSucceeded, domain.KindAgentOffline},
	}); err != nil {
		t.Fatalf("configure fresh Discord destination: %v", err)
	}
	if _, err := destinations.Update(ctx, domain.Destination{
		Provider:      domain.ProviderWeCom,
		Credential:    "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=fresh-postgres-wecom",
		Enabled:       true,
		Subscriptions: []domain.Kind{domain.KindVulnerabilityObserved},
	}); err != nil {
		t.Fatalf("configure fresh WeCom destination: %v", err)
	}

	producer := notificationrepo.NewProducerWriter(outbox)
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("scan").Where("id = ? AND status = ?", scanID, "running").Updates(map[string]any{
			"status":     "succeeded",
			"stopped_at": time.Now().UTC(),
		}).Error; err != nil {
			return err
		}
		if err := producer.WriteScanTerminal(tx, scanID, "succeeded", "", "", time.Now().UTC()); err != nil {
			return err
		}
		if err := producer.WriteVulnerabilityObservations(tx, scanID, []domain.VulnerabilityObservedPayload{{
			ScanID:     scanID,
			TargetID:   targetID,
			TargetName: "notification-fresh-postgres.example",
			URL:        "https://notification-fresh-postgres.example/login",
			VulnType:   "sql-injection",
			Severity:   "high",
		}}, time.Now().UTC()); err != nil {
			return err
		}
		if err := tx.Table("agent").Where("id = ? AND status = ?", agentID, "online").Update("status", "offline").Error; err != nil {
			return err
		}
		return producer.WriteAgentOffline(tx, agentID, time.Now().UTC())
	}); err != nil {
		t.Fatalf("write fresh notification producer occurrences: %v", err)
	}

	hub := notificationapp.NewRealtimeHub()
	refreshHints, unsubscribe := hub.Subscribe(userID)
	defer unsubscribe()
	inbox := notificationrepo.NewInboxRepository(db)
	deliveryStore := notificationrepo.NewDeliveryRepository(db)
	materializer := notificationapp.NewMaterializer(
		notificationrepo.NewTransactionCoordinator(db),
		notificationrepo.NewFactRepository(db),
		notificationrepo.NewAudienceRepository(db),
		destinations,
		notificationapp.NewTemplates(),
		hub,
	)
	outboxWorker := notificationapp.NewOutboxWorker(outbox, materializer, notificationapp.OutboxWorkerOptions{Owner: "fresh-postgres-outbox"})
	if err := outboxWorker.RunOnce(ctx); err != nil {
		t.Fatalf("materialize fresh notification outbox: %v", err)
	}
	select {
	case <-refreshHints:
	case <-time.After(time.Second):
		t.Fatal("fresh notification materialization did not emit an SSE refresh hint")
	}

	inboxService := notificationapp.NewInboxService(inbox, inbox)
	page, err := inboxService.List(ctx, userID, notificationapp.InboxListInput{PageSize: 10})
	if err != nil {
		t.Fatalf("list fresh notification inbox: %v", err)
	}
	if len(page.Results) != 3 || page.TotalSize != 3 {
		t.Fatalf("fresh notification inbox = %d results / %d total, want three", len(page.Results), page.TotalSize)
	}
	unread, err := inboxService.UnreadCount(ctx, userID)
	if err != nil {
		t.Fatalf("count fresh notification inbox unread state: %v", err)
	}
	if unread != 3 {
		t.Fatalf("fresh notification unread count = %d, want 3", unread)
	}
	assertNotificationFreshPostgresCount(t, ctx, sqlDB, "notification_outbox", 3)
	assertNotificationFreshPostgresCount(t, ctx, sqlDB, "notification_fact", 3)
	assertNotificationFreshPostgresCount(t, ctx, sqlDB, "notification_delivery", 3)
	assertNotificationFreshPostgresProviderDeliveryCount(t, ctx, sqlDB, domain.ProviderFeishu, 0)

	deliveryWorker := notificationapp.NewDeliveryWorker(
		deliveryStore,
		destinations,
		[]notificationapp.ProviderAdapter{
			notificationprovider.NewDiscordAdapter(providerClient),
			notificationprovider.NewWeComAdapter(providerClient),
			notificationprovider.NewFeishuAdapter(providerClient),
		},
		notificationapp.DeliveryWorkerOptions{Owner: "fresh-postgres-delivery"},
	)
	if err := deliveryWorker.RunOnce(ctx); err != nil {
		t.Fatalf("run initial fresh notification delivery batch: %v", err)
	}

	var agentDeliveryID int64
	if err := sqlDB.QueryRowContext(ctx, `SELECT id FROM notification_delivery WHERE event_id LIKE 'agent:%'`).Scan(&agentDeliveryID); err != nil {
		t.Fatalf("load fresh Agent retry delivery: %v", err)
	}
	for attempt := 2; attempt <= 6; attempt++ {
		if _, err := sqlDB.ExecContext(ctx, `UPDATE notification_delivery SET next_attempt_at = CURRENT_TIMESTAMP - INTERVAL '1 second' WHERE id = $1`, agentDeliveryID); err != nil {
			t.Fatalf("make Agent retry attempt %d due: %v", attempt, err)
		}
		if err := deliveryWorker.RunOnce(ctx); err != nil {
			t.Fatalf("run Agent retry attempt %d: %v", attempt, err)
		}
	}

	var delivered, failedTerminal, attempts int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_delivery WHERE status = 'delivered'`).Scan(&delivered); err != nil {
		t.Fatalf("count delivered fresh notifications: %v", err)
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_delivery WHERE status = 'failed_terminal'`).Scan(&failedTerminal); err != nil {
		t.Fatalf("count terminally failed fresh notifications: %v", err)
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT attempt_count FROM notification_delivery WHERE id = $1`, agentDeliveryID).Scan(&attempts); err != nil {
		t.Fatalf("read fresh Agent delivery attempt count: %v", err)
	}
	if delivered != 2 || failedTerminal != 1 || attempts != 6 {
		t.Fatalf("fresh delivery lifecycle = delivered %d failed %d attempts %d, want 2/1/6", delivered, failedTerminal, attempts)
	}
	if providerTransport.discordRequests != 7 || providerTransport.wecomRequests != 1 || providerTransport.feishuRequests != 0 {
		t.Fatalf("fresh provider calls = Discord %d WeCom %d Feishu %d, want 7/1/0", providerTransport.discordRequests, providerTransport.wecomRequests, providerTransport.feishuRequests)
	}

	var scanFactID int64
	if err := sqlDB.QueryRowContext(ctx, `SELECT id FROM notification_fact WHERE event_id = $1`, fmt.Sprintf("scan:%d:succeeded", scanID)).Scan(&scanFactID); err != nil {
		t.Fatalf("load fresh Scan notification fact: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, `UPDATE notification_outbox SET published_at = CURRENT_TIMESTAMP - INTERVAL '91 days' WHERE event_id = $1`, fmt.Sprintf("scan:%d:succeeded", scanID)); err != nil {
		t.Fatalf("age fresh Scan outbox for retention: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, `UPDATE notification_inbox SET created_at = CURRENT_TIMESTAMP - INTERVAL '91 days' WHERE fact_id = $1`, scanFactID); err != nil {
		t.Fatalf("age fresh Scan inbox for retention: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, `UPDATE notification_delivery SET terminal_at = CURRENT_TIMESTAMP - INTERVAL '91 days' WHERE event_id = $1`, fmt.Sprintf("scan:%d:succeeded", scanID)); err != nil {
		t.Fatalf("age fresh Scan delivery for retention: %v", err)
	}
	retention := notificationapp.NewRetentionJob(outbox, inbox, deliveryStore, notificationapp.RetentionJobOptions{BatchSize: 10})
	retained, err := retention.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run fresh notification retention: %v", err)
	}
	if retained.PublishedOutboxRows != 1 || retained.ExpiredInboxRows != 1 || retained.TerminalDeliveries != 1 {
		t.Fatalf("fresh retention result = %#v, want one row from each lifecycle", retained)
	}
	assertNotificationFreshPostgresCount(t, ctx, sqlDB, "notification_fact", 3)
}

func (transport *notificationFreshPostgresProviderTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		transport.t.Errorf("read notification provider request: %v", err)
		return notificationFreshPostgresProviderResponse(http.StatusInternalServerError, ""), nil
	}

	switch request.URL.Host {
	case "discord.com":
		if request.URL.Path != "/api/webhooks/fresh-postgres-discord/fresh-postgres-token" {
			transport.t.Errorf("Discord request path = %q", request.URL.Path)
			return notificationFreshPostgresProviderResponse(http.StatusNotFound, ""), nil
		}
		transport.discordRequests++
		if strings.Contains(string(body), "Agent offline") {
			return notificationFreshPostgresProviderResponse(http.StatusInternalServerError, ""), nil
		}
		return notificationFreshPostgresProviderResponse(http.StatusNoContent, ""), nil
	case "qyapi.weixin.qq.com":
		if request.URL.Path != "/cgi-bin/webhook/send" || request.URL.Query().Get("key") != "fresh-postgres-wecom" {
			transport.t.Errorf("WeCom request URL = %q", request.URL.String())
			return notificationFreshPostgresProviderResponse(http.StatusNotFound, ""), nil
		}
		transport.wecomRequests++
		return notificationFreshPostgresProviderResponse(http.StatusOK, `{"errcode":0,"errmsg":"ok"}`), nil
	case "open.feishu.cn":
		transport.feishuRequests++
		transport.t.Errorf("unexpected Feishu delivery request to %q", request.URL.String())
		return notificationFreshPostgresProviderResponse(http.StatusInternalServerError, ""), nil
	default:
		transport.t.Errorf("unexpected notification provider host %q", request.URL.Host)
		return notificationFreshPostgresProviderResponse(http.StatusNotFound, ""), nil
	}
}

func notificationFreshPostgresProviderResponse(status int, body string) *http.Response {
	header := make(http.Header)
	if body != "" {
		header.Set("Content-Type", "application/json")
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type notificationFreshPostgresProviderTransport struct {
	t               *testing.T
	discordRequests int
	wecomRequests   int
	feishuRequests  int
}

func assertNotificationFreshPostgresFeishuDestination(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	var credential string
	var enabled bool
	if err := db.QueryRowContext(ctx, `SELECT credential, enabled FROM notification_destination WHERE provider = 'feishu'`).Scan(&credential, &enabled); err != nil {
		t.Fatalf("read fresh Feishu destination: %v", err)
	}
	if credential != "" || enabled {
		t.Fatalf("fresh Feishu destination = credential %q enabled %t, want empty disabled", credential, enabled)
	}
	var subscriptions int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM notification_destination_subscription subscription
		JOIN notification_destination destination ON destination.id = subscription.destination_id
		WHERE destination.provider = 'feishu'
	`).Scan(&subscriptions); err != nil {
		t.Fatalf("count fresh Feishu subscriptions: %v", err)
	}
	if subscriptions != 0 {
		t.Fatalf("fresh Feishu destination subscriptions = %d, want 0", subscriptions)
	}
}

func assertNotificationFreshPostgresProviderDeliveryCount(t *testing.T, ctx context.Context, db *sql.DB, provider domain.Provider, want int) {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_delivery WHERE provider = $1`, provider).Scan(&count); err != nil {
		t.Fatalf("count fresh %s notification deliveries: %v", provider, err)
	}
	if count != want {
		t.Fatalf("fresh %s notification deliveries = %d, want %d", provider, count, want)
	}
}

func seedNotificationFreshPostgresScan(t *testing.T, ctx context.Context, db *sql.DB) (int, int) {
	t.Helper()
	var targetID, scanID int
	if err := db.QueryRowContext(ctx, `INSERT INTO target (name, type) VALUES ('notification-fresh-postgres.example', 'domain') RETURNING id`).Scan(&targetID); err != nil {
		t.Fatalf("seed fresh notification target: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
		INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status)
		VALUES ($1, 'notification-fresh-postgres', 'scan_snapshot', 'manual', 'running')
		RETURNING id
	`, targetID).Scan(&scanID); err != nil {
		t.Fatalf("seed fresh notification Scan: %v", err)
	}
	return targetID, scanID
}

func seedNotificationFreshPostgresAgent(t *testing.T, ctx context.Context, db *sql.DB) int {
	t.Helper()
	var registrationTokenID, agentID int
	if err := db.QueryRowContext(ctx, `INSERT INTO registration_token (token) VALUES ('ntfpg001') RETURNING id`).Scan(&registrationTokenID); err != nil {
		t.Fatalf("seed fresh notification registration token: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
		INSERT INTO agent (instance_id, display_name, authentication_token, status, registration_token_id)
		VALUES ('notification-fresh-postgres-agent', 'fresh-notification-agent', 'ntfpg002', 'online', $1)
		RETURNING id
	`, registrationTokenID).Scan(&agentID); err != nil {
		t.Fatalf("seed fresh notification Agent: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO agent_runtime_status (agent_id, last_heartbeat) VALUES ($1, CURRENT_TIMESTAMP)`, agentID); err != nil {
		t.Fatalf("seed fresh notification Agent runtime: %v", err)
	}
	return agentID
}

func assertNotificationFreshPostgresCount(t *testing.T, ctx context.Context, db *sql.DB, table string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&count); err != nil {
		t.Fatalf("count fresh notification %s rows: %v", table, err)
	}
	if count != want {
		t.Fatalf("fresh notification %s rows = %d, want %d", table, count, want)
	}
}
