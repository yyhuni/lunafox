package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAgentRepositoryClusterAggregateUsesCompleteFleetAndStrictFreshness(t *testing.T) {
	db := openAgentClusterSummaryTestDatabase(t)
	generatedAt := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	agents := []struct {
		id       int
		status   string
		maxTasks int
	}{
		{id: 1, status: "online", maxTasks: 4},
		{id: 2, status: "online", maxTasks: 5},
		{id: 3, status: "online", maxTasks: 3},
		{id: 4, status: "offline", maxTasks: 2},
		{id: 5, status: "online", maxTasks: 1},
		{id: 6, status: "starting", maxTasks: -3},
		{id: 7, status: "online", maxTasks: 4},
		{id: 8, status: "online", maxTasks: 0},
	}
	for _, agent := range agents {
		if err := db.Exec("INSERT INTO agent (id, status, max_tasks) VALUES (?, ?, ?)", agent.id, agent.status, agent.maxTasks).Error; err != nil {
			t.Fatalf("insert Agent %d: %v", agent.id, err)
		}
	}
	runtimeRows := []struct {
		agentID     int
		healthState string
		heartbeat   time.Time
		used        int
	}{
		{agentID: 1, healthState: "healthy", heartbeat: generatedAt.Add(-time.Second), used: 6},
		{agentID: 2, healthState: "healthy", heartbeat: generatedAt.Add(-15 * time.Second), used: 1},
		{agentID: 3, healthState: "paused", heartbeat: generatedAt.Add(-time.Second), used: 1},
		{agentID: 4, healthState: "healthy", heartbeat: generatedAt.Add(time.Second), used: 1},
		{agentID: 5, healthState: "degraded", heartbeat: generatedAt.Add(-time.Second), used: 0},
		{agentID: 7, healthState: "healthy", heartbeat: generatedAt.Add(-14 * time.Second), used: -2},
		{agentID: 8, healthState: "healthy", heartbeat: generatedAt, used: 2},
	}
	for _, runtime := range runtimeRows {
		if err := db.Exec(
			"INSERT INTO agent_runtime_status (agent_id, health_state, last_heartbeat, task_slots_used) VALUES (?, ?, ?, ?)",
			runtime.agentID,
			runtime.healthState,
			runtime.heartbeat,
			runtime.used,
		).Error; err != nil {
			t.Fatalf("insert runtime %d: %v", runtime.agentID, err)
		}
	}
	if err := db.Exec(
		"INSERT INTO agent_location (agent_id, latitude, longitude, accuracy_radius_km, source_observed_ip, provider_key, resolved_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		1, 37.42, -122.08, nil, "8.8.8.8", "freeipapi", generatedAt.Add(-24*time.Hour),
	).Error; err != nil {
		t.Fatalf("insert current location: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO agent_location (agent_id, latitude, longitude, accuracy_radius_km, source_observed_ip, provider_key, resolved_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		2, -33.86, 151.21, 12.5, "1.1.1.1", "freeipapi", generatedAt.Add(-8*24*time.Hour),
	).Error; err != nil {
		t.Fatalf("insert expired location: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO agent_location (agent_id, latitude, longitude, accuracy_radius_km, source_observed_ip, provider_key, resolved_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		5, 91, 0, nil, "9.9.9.9", "freeipapi", generatedAt,
	).Error; err != nil {
		t.Fatalf("insert out-of-range location: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO agent_location (agent_id, latitude, longitude, accuracy_radius_km, source_observed_ip, provider_key, resolved_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		8, 0, 0, -1, "208.67.222.222", "freeipapi", generatedAt,
	).Error; err != nil {
		t.Fatalf("insert invalid-radius location: %v", err)
	}

	aggregate, err := (&agentRepository{db: db}).GetClusterAggregate(context.Background(), generatedAt)
	if err != nil {
		t.Fatalf("GetClusterAggregate: %v", err)
	}
	if aggregate.TotalNodes != 8 || aggregate.HealthyCount != 4 || aggregate.WarningCount != 1 || aggregate.OfflineCount != 1 || aggregate.UnknownCount != 2 {
		t.Fatalf("node buckets = %#v", aggregate)
	}
	if aggregate.StaleAgentCount != 3 {
		t.Fatalf("stale Agent count = %d, want 3", aggregate.StaleAgentCount)
	}
	if aggregate.ConfiguredSlots != 19 || aggregate.OccupiedSlots != 4 || aggregate.AvailableSlots != 4 || aggregate.UnavailableSlots != 11 || aggregate.OvercommittedSlots != 4 {
		t.Fatalf("capacity aggregate = %#v", aggregate)
	}
	if aggregate.ConfiguredSlots != aggregate.OccupiedSlots+aggregate.AvailableSlots+aggregate.UnavailableSlots {
		t.Fatalf("configured capacity is not conserved: %#v", aggregate)
	}
	if aggregate.PositionedCount != 2 || aggregate.UnpositionedCount != 6 {
		t.Fatalf("location coverage = %#v", aggregate)
	}
}

func TestAgentRepositoryClusterAggregateReturnsSuccessfulEmptyFleet(t *testing.T) {
	repository := &agentRepository{db: openAgentClusterSummaryTestDatabase(t)}
	aggregate, err := repository.GetClusterAggregate(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatalf("GetClusterAggregate: %v", err)
	}
	if aggregate.TotalNodes != 0 || aggregate.ConfiguredSlots != 0 || aggregate.PositionedCount != 0 || aggregate.UnpositionedCount != 0 {
		t.Fatalf("empty aggregate = %#v", aggregate)
	}
}

func TestAgentRepositoryClusterAggregateIncludesMoreThanOneThousandAgents(t *testing.T) {
	db := openAgentClusterSummaryTestDatabase(t)
	if err := db.Exec(`
WITH RECURSIVE sequence(id) AS (
    SELECT 1
    UNION ALL
    SELECT id + 1 FROM sequence WHERE id < 1001
)
INSERT INTO agent (id, status, max_tasks)
SELECT id, 'offline', 2 FROM sequence`).Error; err != nil {
		t.Fatalf("insert complete fleet: %v", err)
	}

	aggregate, err := (&agentRepository{db: db}).GetClusterAggregate(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatalf("GetClusterAggregate: %v", err)
	}
	if aggregate.TotalNodes != 1001 || aggregate.OfflineCount != 1001 || aggregate.StaleAgentCount != 1001 {
		t.Fatalf("complete-fleet counts = %#v", aggregate)
	}
	if aggregate.ConfiguredSlots != 2002 || aggregate.UnavailableSlots != 2002 || aggregate.OccupiedSlots != 0 || aggregate.AvailableSlots != 0 {
		t.Fatalf("complete-fleet capacity = %#v", aggregate)
	}
	if aggregate.PositionedCount != 0 || aggregate.UnpositionedCount != 1001 {
		t.Fatalf("complete-fleet coverage = %#v", aggregate)
	}
}

func openAgentClusterSummaryTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "agent-cluster-summary.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	for _, schema := range []string{
		"CREATE TABLE agent (id INTEGER PRIMARY KEY, status TEXT, max_tasks INTEGER)",
		"CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, health_state TEXT, last_heartbeat DATETIME, task_slots_used INTEGER)",
		"CREATE TABLE agent_location (agent_id INTEGER PRIMARY KEY, latitude REAL, longitude REAL, accuracy_radius_km REAL, source_observed_ip TEXT, provider_key TEXT, resolved_at DATETIME)",
	} {
		if err := db.Exec(schema).Error; err != nil {
			t.Fatalf("create Agent cluster summary schema: %v", err)
		}
	}
	return db
}
