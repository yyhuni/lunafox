package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openAgentQuotaPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LUNAFOX_AGENT_QUOTA_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set LUNAFOX_AGENT_QUOTA_POSTGRES_DSN with search_path=agent_quota_contract")
	}
	if !strings.Contains(dsn, "search_path=agent_quota_contract") {
		t.Fatal("quota tests require isolated search_path=agent_quota_contract")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS agent_quota_contract").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.RegistrationToken{}, &model.Agent{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE agent, registration_token RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Exec("DROP SCHEMA agent_quota_contract CASCADE").Error })
	insertRegistrationToken(t, db, 1, "aaaaaaaa", time.Now().Add(time.Hour))
	insertRegistrationToken(t, db, 2, "bbbbbbbb", time.Now().Add(time.Hour))
	return db
}

func quotaAgent(index, token int) *agentdomain.Agent {
	return agentdomain.NewRegisteredAgent(token, fmt.Sprintf("quota-%d", index), "node", "1.0.0", fmt.Sprintf("%08x", index), agentdomain.AgentRegistrationOptions{})
}

func TestAgentQuotaPostgresConcurrentLastSlot(t *testing.T) {
	for _, differentTokens := range []bool{false, true} {
		t.Run(fmt.Sprintf("different-tokens-%t", differentTokens), func(t *testing.T) {
			db := openAgentQuotaPostgresDB(t)
			for i := 1; i <= 2; i++ {
				if err := NewAgentRepository(db).Create(context.Background(), quotaAgent(i, 1)); err != nil {
					t.Fatal(err)
				}
			}
			start := make(chan struct{})
			results := make(chan error, 6)
			for i := 3; i < 9; i++ {
				go func(index int) {
					<-start
					token := 1
					if differentTokens {
						token = 1 + index%2
					}
					results <- NewAgentRepository(db.Session(&gorm.Session{NewDB: true})).Create(context.Background(), quotaAgent(index, token))
				}(i)
			}
			close(start)
			successes := 0
			for i := 0; i < 6; i++ {
				if err := <-results; err == nil {
					successes++
				} else if !errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
					t.Fatal(err)
				}
			}
			var count int64
			db.Model(&model.Agent{}).Count(&count)
			if successes != 1 || count != 3 {
				t.Fatalf("successes=%d count=%d", successes, count)
			}
		})
	}
}

func TestAgentQuotaPostgresDeletionRollbackAndLegacy(t *testing.T) {
	db := openAgentQuotaPostgresDB(t)
	repo := NewAgentRepository(db)
	// Direct fixtures represent an already-over-limit deployment before upgrade.
	for i := 1; i <= 5; i++ {
		if err := db.Create(domainAgentToModel(quotaAgent(i, 1))).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Create(context.Background(), quotaAgent(6, 2)); !errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
		t.Fatal(err)
	}
	var token model.RegistrationToken
	db.First(&token, 2)
	if token.EverAttributedAt != nil {
		t.Fatal("rejected registration consumed attribution")
	}
	if err := repo.UpdateStatus(context.Background(), 1, "online"); err != nil {
		t.Fatal(err)
	}
	rollback := errors.New("rollback deletion")
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.Agent{}, "id > 2").Error; err != nil {
			return err
		}
		return rollback
	}); !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), quotaAgent(6, 2)); !errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
		t.Fatal(err)
	}
	if err := db.Delete(&model.Agent{}, "id > 3").Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), quotaAgent(6, 2)); !errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
		t.Fatal(err)
	}
	if err := db.Delete(&model.Agent{}, 3).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), quotaAgent(6, 2)); err != nil {
		t.Fatal(err)
	}
}

func TestAgentQuotaPostgresCancellationAndInsertFailureReleaseLock(t *testing.T) {
	db := openAgentQuotaPostgresDB(t)
	repo := NewAgentRepository(db)
	if err := repo.Create(context.Background(), quotaAgent(1, 1)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), quotaAgent(1, 2)); err == nil {
		t.Fatal("expected unique conflict")
	}
	var token model.RegistrationToken
	db.First(&token, 2)
	if token.EverAttributedAt != nil {
		t.Fatal("failed insert retained marker")
	}
	holder := db.Begin()
	defer holder.Rollback()
	if err := holder.Exec("SELECT pg_advisory_xact_lock(?)", agentQuotaAdvisoryLock).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := repo.Create(ctx, quotaAgent(2, 2)); err == nil {
		t.Fatal("expected cancelled lock wait")
	}
	if err := holder.Commit().Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), quotaAgent(2, 2)); err != nil {
		t.Fatal(err)
	}
}

func TestAgentQuotaPostgresCountFailureDoesNotCreateIdentity(t *testing.T) {
	db := openAgentQuotaPostgresDB(t)
	if err := db.Exec("ALTER TABLE agent RENAME TO agent_unavailable").Error; err != nil {
		t.Fatal(err)
	}
	if err := NewAgentRepository(db).Create(context.Background(), quotaAgent(1, 1)); err == nil || errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
		t.Fatalf("unexpected count error: %v", err)
	}
	if err := db.Exec("ALTER TABLE agent_unavailable RENAME TO agent").Error; err != nil {
		t.Fatal(err)
	}
	if err := NewAgentRepository(db).Create(context.Background(), quotaAgent(1, 1)); err != nil {
		t.Fatal(err)
	}
}
