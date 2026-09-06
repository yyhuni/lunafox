package repository

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAgentRegistrationTransactionSupportsConcurrentInterleavedTokens(t *testing.T) {
	db := openRegistrationTokenTestDB(t)
	insertRegistrationToken(t, db, 1, "aaaaaaaa", time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC))
	insertRegistrationToken(t, db, 2, "bbbbbbbb", time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC))

	repository := &agentRepository{db: db}
	start := make(chan struct{})
	errorsByAgent := make(chan error, 12)
	var group sync.WaitGroup
	for index := 0; index < 12; index++ {
		index := index
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			tokenID := 1
			if index%2 == 1 {
				tokenID = 2
			}
			agent := agentdomain.NewRegisteredAgent(
				tokenID,
				fmt.Sprintf("instance-%02d", index),
				fmt.Sprintf("node-%02d", index),
				"1.0.0",
				fmt.Sprintf("%08x", index+1),
				agentdomain.AgentRegistrationOptions{},
			)
			errorsByAgent <- repository.Create(context.Background(), agent)
		}()
	}
	close(start)
	group.Wait()
	close(errorsByAgent)
	for err := range errorsByAgent {
		if err != nil {
			t.Fatalf("concurrent registration failed: %v", err)
		}
	}

	for _, tokenID := range []int{1, 2} {
		var count int64
		if err := db.Model(&model.Agent{}).Where("registration_token_id = ?", tokenID).Count(&count).Error; err != nil {
			t.Fatalf("count token %d agents: %v", tokenID, err)
		}
		if count != 6 {
			t.Fatalf("token %d agent count = %d, want 6", tokenID, count)
		}
		var token model.RegistrationToken
		if err := db.First(&token, tokenID).Error; err != nil {
			t.Fatalf("read token %d: %v", tokenID, err)
		}
		if token.EverAttributedAt == nil {
			t.Fatalf("token %d did not retain attribution marker", tokenID)
		}
	}
}

func TestAgentRegistrationTransactionRollsBackMarkerAndRejectsExactExpiry(t *testing.T) {
	db := openRegistrationTokenTestDB(t)
	insertRegistrationToken(t, db, 1, "aaaaaaaa", time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC))
	insertRegistrationToken(t, db, 2, "bbbbbbbb", time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC))
	if err := db.Create(&model.Agent{
		InstanceID: "existing", DisplayName: "existing", AuthenticationToken: "deadbeef",
		Status: "offline", RegistrationTokenID: 1,
	}).Error; err != nil {
		t.Fatalf("insert existing Agent: %v", err)
	}

	repository := &agentRepository{db: db}
	conflict := agentdomain.NewRegisteredAgent(2, "conflict", "conflict", "1.0.0", "deadbeef", agentdomain.AgentRegistrationOptions{})
	if err := repository.Create(context.Background(), conflict); err == nil {
		t.Fatal("expected Agent insert conflict")
	}
	var rolledBack model.RegistrationToken
	if err := db.First(&rolledBack, 2).Error; err != nil {
		t.Fatalf("read rolled-back token: %v", err)
	}
	if rolledBack.EverAttributedAt != nil {
		t.Fatalf("failed Agent insert retained attribution marker: %s", rolledBack.EverAttributedAt)
	}

	if err := db.Exec(`INSERT INTO registration_token (id, token, expires_at, created_at) VALUES (3, 'cccccccc', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatalf("insert exactly expired token: %v", err)
	}
	expired := agentdomain.NewRegisteredAgent(3, "expired", "expired", "1.0.0", "feedbeef", agentdomain.AgentRegistrationOptions{})
	if err := repository.Create(context.Background(), expired); !errors.Is(err, agentdomain.ErrRegistrationTokenInvalid) {
		t.Fatalf("exact-expiry registration error = %v", err)
	}
	var count int64
	if err := db.Model(&model.Agent{}).Where("registration_token_id = 3").Count(&count).Error; err != nil {
		t.Fatalf("count exact-expiry Agents: %v", err)
	}
	if count != 0 {
		t.Fatalf("exact-expiry token created %d Agents", count)
	}
}

func TestRegistrationTokenCleanupRechecksAttributionAndRetainsMarkerAfterAgentDeletion(t *testing.T) {
	db := openRegistrationTokenTestDB(t)
	expiredAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	insertRegistrationToken(t, db, 1, "aaaaaaaa", time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC))
	insertRegistrationToken(t, db, 2, "bbbbbbbb", expiredAt)
	insertRegistrationToken(t, db, 3, "cccccccc", expiredAt)
	insertRegistrationToken(t, db, 4, "dddddddd", expiredAt)
	insertRegistrationToken(t, db, 5, "eeeeeeee", expiredAt.Add(time.Minute))

	agentRepository := &agentRepository{db: db}
	agent := agentdomain.NewRegisteredAgent(2, "attributed", "attributed", "1.0.0", "deadbeef", agentdomain.AgentRegistrationOptions{})
	// Registration uses a future expiry first; the cleanup scenario then ages it.
	if err := db.Model(&model.RegistrationToken{}).Where("id = 2").Update("expires_at", time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)).Error; err != nil {
		t.Fatalf("make token valid: %v", err)
	}
	if err := agentRepository.Create(context.Background(), agent); err != nil {
		t.Fatalf("register attributed Agent: %v", err)
	}
	if err := db.Delete(&model.Agent{}, agent.ID).Error; err != nil {
		t.Fatalf("delete attributed Agent: %v", err)
	}
	if err := db.Model(&model.RegistrationToken{}).Where("id = 2").Update("expires_at", expiredAt).Error; err != nil {
		t.Fatalf("expire attributed token: %v", err)
	}
	// Simulate a late attribution marker becoming visible immediately before the
	// cleanup statement evaluates its predicates.
	lateAttribution := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if err := db.Model(&model.RegistrationToken{}).Where("id = 3").Update("ever_attributed_at", lateAttribution).Error; err != nil {
		t.Fatalf("mark late attribution: %v", err)
	}

	tokenRepository := &registrationTokenRepository{db: db}
	if err := tokenRepository.DeleteNeverAttributedBefore(context.Background(), expiredAt); err != nil {
		t.Fatalf("cleanup tokens: %v", err)
	}
	for _, retainedID := range []int{2, 3} {
		var count int64
		if err := db.Model(&model.RegistrationToken{}).Where("id = ?", retainedID).Count(&count).Error; err != nil {
			t.Fatalf("count retained token %d: %v", retainedID, err)
		}
		if count != 1 {
			t.Fatalf("attributed token %d was removed", retainedID)
		}
	}
	var deletedCount int64
	if err := db.Model(&model.RegistrationToken{}).Where("id = 4").Count(&deletedCount).Error; err != nil {
		t.Fatalf("count eligible never-attributed token: %v", err)
	}
	if deletedCount != 0 {
		t.Fatal("eligible never-attributed token was not deleted")
	}
	var youngCount int64
	if err := db.Model(&model.RegistrationToken{}).Where("id = 5").Count(&youngCount).Error; err != nil {
		t.Fatalf("count retained young token: %v", err)
	}
	if youngCount != 1 {
		t.Fatal("token younger than the 24-hour post-expiry minimum was deleted")
	}
}

func openRegistrationTokenTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "registration-token.db") + "?_busy_timeout=10000&_journal_mode=WAL&_foreign_keys=on"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open registration-token database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("read sql database: %v", err)
	}
	// Concurrent callers still overlap at the application boundary while SQLite
	// serializes the write transactions deterministically for this contract test.
	sqlDB.SetMaxOpenConns(1)
	if err := db.Exec(`
		CREATE TABLE registration_token (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			ever_attributed_at DATETIME,
			created_at DATETIME
		);
		CREATE TABLE agent (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			instance_id TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			authentication_token TEXT NOT NULL UNIQUE,
			status TEXT,
			max_tasks INTEGER,
			cpu_threshold INTEGER,
			mem_threshold INTEGER,
			disk_threshold INTEGER,
			registration_token_id INTEGER NOT NULL REFERENCES registration_token(id) ON DELETE RESTRICT,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE agent_runtime_status (
			agent_id INTEGER PRIMARY KEY,
			observed_hostname TEXT,
			observed_source_ip TEXT NOT NULL DEFAULT '',
			observed_ip_generation INTEGER NOT NULL DEFAULT 0,
			last_heartbeat DATETIME,
			health_state TEXT
		);
	`).Error; err != nil {
		t.Fatalf("create registration-token schema: %v", err)
	}
	return db
}

func insertRegistrationToken(t *testing.T, db *gorm.DB, id int, secret string, expiresAt time.Time) {
	t.Helper()
	if err := db.Create(&model.RegistrationToken{ID: id, Token: secret, ExpiresAt: expiresAt, CreatedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatalf("insert registration token %d: %v", id, err)
	}
}
