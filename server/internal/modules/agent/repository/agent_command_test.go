package repository

import (
	"context"
	"testing"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAgentRepositoryCreateBackfillsID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE agent (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			api_key TEXT NOT NULL UNIQUE,
			status TEXT,
			hostname TEXT,
			ip_address TEXT,
			agent_version TEXT,
			worker_version TEXT,
			max_tasks INTEGER,
			cpu_threshold INTEGER,
			mem_threshold INTEGER,
			disk_threshold INTEGER,
			health_state TEXT,
			health_reason TEXT,
			health_message TEXT,
			health_since DATETIME,
			registration_token TEXT,
			connected_at DATETIME,
			last_heartbeat DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("create agent table failed: %v", err)
	}

	repo := &agentRepository{db: db}
	agent := agentdomain.NewRegisteredAgent(
		"abcd1234",
		"agent-test",
		"dev",
		"dev",
		"127.0.0.1",
		"deadbeef",
		agentdomain.AgentRegistrationOptions{},
	)

	if err := repo.Create(context.Background(), agent); err != nil {
		t.Fatalf("create agent failed: %v", err)
	}
	if agent.ID <= 0 {
		t.Fatalf("expected created agent ID to be backfilled, got %d", agent.ID)
	}
}
