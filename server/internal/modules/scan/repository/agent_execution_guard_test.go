package repository

import (
	"testing"

	agentrepo "github.com/yyhuni/lunafox/server/internal/modules/agent/repository"
	"gorm.io/gorm"
)

func savedPlanTestExecutionGuard(t *testing.T, db *gorm.DB) *agentrepo.AgentLicenseRepository {
	t.Helper()
	for _, statement := range []string{
		"CREATE TABLE IF NOT EXISTS agent (id INTEGER PRIMARY KEY, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)",
		"CREATE TABLE IF NOT EXISTS agent_license (singleton BOOLEAN PRIMARY KEY, signed_key TEXT NOT NULL)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if db.Migrator().HasTable("agent_runtime_status") {
		if err := db.Exec("INSERT INTO agent (id) SELECT agent_id FROM agent_runtime_status WHERE agent_id NOT IN (SELECT id FROM agent)").Error; err != nil {
			t.Fatal(err)
		}
	}
	return agentrepo.NewAgentLicenseRepository(db)
}
