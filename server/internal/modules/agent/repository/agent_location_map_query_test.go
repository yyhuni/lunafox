package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAgentRepositoryLocationMapReturnsEveryValidPositionWithoutSampling(t *testing.T) {
	db := openAgentLocationMapTestDatabase(t)
	resolvedAt := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	if err := db.Exec(`
WITH RECURSIVE sequence(id) AS (
    SELECT 1
    UNION ALL
    SELECT id + 1 FROM sequence WHERE id < 1007
)
INSERT INTO agent (id, display_name, status)
SELECT id, 'agent-' || id, CASE WHEN id = 1001 THEN 'offline' ELSE 'online' END FROM sequence`).Error; err != nil {
		t.Fatalf("insert Agents: %v", err)
	}
	if err := db.Exec(`
WITH RECURSIVE sequence(id) AS (
    SELECT 1
    UNION ALL
    SELECT id + 1 FROM sequence WHERE id < 1001
)
INSERT INTO agent_location (
    agent_id, latitude, longitude, accuracy_radius_km, source_observed_ip,
    provider_key, resolved_at, forced_expired, updated_at
)
SELECT
    id, (id % 160) - 80, (id % 340) - 170, NULL, '8.8.8.8',
    'freeipapi', ?, CASE WHEN id = 2 THEN 1 ELSE 0 END, ?
FROM sequence`, resolvedAt, resolvedAt).Error; err != nil {
		t.Fatalf("insert valid locations: %v", err)
	}
	for _, runtime := range []struct {
		agentID int
		used    int
	}{
		{agentID: 1, used: 0},
		{agentID: 2, used: 3},
		{agentID: 1001, used: 1},
	} {
		if err := db.Exec("INSERT INTO agent_runtime_status (agent_id, health_state, task_slots_used) VALUES (?, 'healthy', ?)", runtime.agentID, runtime.used).Error; err != nil {
			t.Fatalf("insert runtime %d: %v", runtime.agentID, err)
		}
	}
	invalidLocations := []struct {
		agentID  int
		latitude float64
		radius   any
		source   string
		provider string
		resolved any
	}{
		{agentID: 1002, latitude: 91, radius: nil, source: "8.8.8.8", provider: "freeipapi", resolved: resolvedAt},
		{agentID: 1003, latitude: 1, radius: -1.0, source: "8.8.8.8", provider: "freeipapi", resolved: resolvedAt},
		{agentID: 1004, latitude: 1, radius: nil, source: "", provider: "freeipapi", resolved: resolvedAt},
		{agentID: 1005, latitude: 1, radius: nil, source: "8.8.8.8", provider: "", resolved: resolvedAt},
		{agentID: 1006, latitude: 1, radius: nil, source: "8.8.8.8", provider: "freeipapi", resolved: nil},
	}
	for _, location := range invalidLocations {
		if err := db.Exec(`
INSERT INTO agent_location (
    agent_id, latitude, longitude, accuracy_radius_km, source_observed_ip,
    provider_key, resolved_at, forced_expired, updated_at
) VALUES (?, ?, 1, ?, ?, ?, ?, 0, ?)`,
			location.agentID,
			location.latitude,
			location.radius,
			location.source,
			location.provider,
			location.resolved,
			resolvedAt,
		).Error; err != nil {
			t.Fatalf("insert invalid location %d: %v", location.agentID, err)
		}
	}

	records, err := (&agentRepository{db: db}).ListLocationMapAgents(context.Background())
	if err != nil {
		t.Fatalf("ListLocationMapAgents: %v", err)
	}
	if len(records) != 1001 {
		t.Fatalf("positioned Agent count = %d, want 1001", len(records))
	}
	if records[0].AgentID != 1 || records[len(records)-1].AgentID != 1001 {
		t.Fatalf("map query order/range = first %d last %d", records[0].AgentID, records[len(records)-1].AgentID)
	}
	if records[0].TaskSlotsUsed == nil || *records[0].TaskSlotsUsed != 0 {
		t.Fatalf("zero task usage was not preserved: %#v", records[0].TaskSlotsUsed)
	}
	if records[1].TaskSlotsUsed == nil || *records[1].TaskSlotsUsed != 3 || !records[1].Location.ForcedExpired {
		t.Fatalf("active/expired projection = %#v", records[1])
	}
	if records[2].TaskSlotsUsed != nil {
		t.Fatalf("missing runtime observation became zero: %#v", records[2].TaskSlotsUsed)
	}
	last := records[len(records)-1]
	if last.Status != "offline" || last.TaskSlotsUsed == nil || *last.TaskSlotsUsed != 1 {
		t.Fatalf("offline positioned Agent was lost or rewritten: %#v", last)
	}
	for _, record := range records {
		if record.AgentID > 1001 {
			t.Fatalf("invalid or unknown point entered map: %#v", record)
		}
	}
}

func openAgentLocationMapTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "agent-location-map.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	for index, schema := range []string{
		"CREATE TABLE agent (id INTEGER PRIMARY KEY, display_name TEXT, status TEXT)",
		"CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, health_state TEXT, task_slots_used INTEGER)",
		"CREATE TABLE agent_location (agent_id INTEGER PRIMARY KEY, latitude REAL, longitude REAL, accuracy_radius_km REAL, source_observed_ip TEXT, provider_key TEXT, resolved_at DATETIME, forced_expired BOOLEAN, updated_at DATETIME)",
	} {
		if err := db.Exec(schema).Error; err != nil {
			t.Fatalf("create map schema %d: %v", index, err)
		}
	}
	return db
}
