package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

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

func TestAgentManagementReadsProjectGlobalEligibilityBeforeFilteringAndPagination(t *testing.T) {
	db := openRegistrationTokenTestDB(t)
	if err := db.Exec("CREATE TABLE agent_location (agent_id INTEGER PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	insertRegistrationToken(t, db, 1, "aaaaaaaa", now.Add(time.Hour))
	for id := 1; id <= 4; id++ {
		status := "offline"
		if id >= 3 {
			status = "online"
		}
		if err := db.Create(&model.Agent{
			ID: id, InstanceID: fmt.Sprintf("eligibility-%d", id), DisplayName: fmt.Sprintf("Agent %d", id),
			AuthenticationToken: fmt.Sprintf("token-%d", id), Status: status, RegistrationTokenID: 1, CreatedAt: now,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	repository := NewAgentRepository(db)
	agents, total, err := repository.List(context.Background(), 1, 1, `status=="online"`, "createdAt desc")
	if err != nil || total != 2 || len(agents) != 1 {
		t.Fatalf("list=%#v total=%d err=%v", agents, total, err)
	}
	if agents[0].ID != 4 || agents[0].ExecutionEligibility != agentdomain.AgentExecutionLicenseRestricted || agents[0].Status != "online" {
		t.Fatalf("newest online Agent must retain connection and global entitlement: %#v", agents[0])
	}
	earlier, err := repository.GetByID(context.Background(), 3)
	if err != nil || earlier.ExecutionEligibility != agentdomain.AgentExecutionEligible {
		t.Fatalf("third identity detail=%#v err=%v", earlier, err)
	}
	if err := db.Exec("INSERT INTO agent_license (singleton, signed_key) VALUES (TRUE, 'corrupt')").Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := repository.List(context.Background(), 1, 1, "", "createdAt desc"); err == nil {
		t.Fatal("management projection hid corrupt entitlement")
	}
	// Authentication remains independent of the management entitlement projection:
	// withholding new work must not reject an existing control connection.
	if agent, err := repository.FindByAuthenticationToken(context.Background(), "token-4"); err != nil || agent.ID != 4 {
		t.Fatalf("authentication was coupled to entitlement: agent=%#v err=%v", agent, err)
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
