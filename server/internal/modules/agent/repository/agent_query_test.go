package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAgentRepositoryListQueryUsesRuntimeJoinFilterAndCreatedAtTieBreaker(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	stmt := applyAgentOrder(
		applyAgentFilter(
			db.Model(&model.Agent{}).
				Joins("LEFT JOIN agent_runtime_status ON agent_runtime_status.agent_id = agent.id"),
			`(displayName="edge" || observedHostname="edge" || connectionIp="edge") && status=="online" && healthState=="healthy"`,
		),
		"createdAt desc",
	).Find(&[]model.Agent{}).Statement

	sql := stmt.SQL.String()
	for _, required := range []string{
		"LEFT JOIN agent_runtime_status ON agent_runtime_status.agent_id = agent.id",
		"agent.display_name ILIKE ?",
		"agent_runtime_status.observed_hostname ILIKE ?",
		"agent_runtime_status.connection_ip ILIKE ?",
		"agent.status = ?",
		"agent_runtime_status.health_state = ?",
		"ORDER BY agent.created_at DESC,agent.id DESC",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("agent list SQL must contain %q, got %s", required, sql)
		}
	}
}

func TestAgentRepositoryFindByAuthenticationTokenMapsMissingAgentToDomainError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:agent-auth-token-not-found?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`CREATE TABLE agent (id INTEGER PRIMARY KEY, authentication_token TEXT NOT NULL)`).Error; err != nil {
		t.Fatalf("create agent table: %v", err)
	}

	repository := &agentRepository{db: db}
	agent, err := repository.FindByAuthenticationToken(context.Background(), "missing")
	if agent != nil {
		t.Fatalf("missing token returned agent: %#v", agent)
	}
	if !errors.Is(err, agentdomain.ErrAgentNotFound) {
		t.Fatalf("missing token error = %v, want ErrAgentNotFound", err)
	}
}
