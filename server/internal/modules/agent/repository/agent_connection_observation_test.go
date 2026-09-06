package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestAgentRepositoryRecordConnectionFencesSourceChanges(t *testing.T) {
	db := openAgentConnectionTestDatabase(t)
	repository := &agentRepository{db: db}
	ctx := context.Background()
	baseTime := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)

	first, err := repository.RecordConnection(ctx, 7, "8.8.8.8", baseTime)
	if err != nil {
		t.Fatalf("RecordConnection(first): %v", err)
	}
	if first.ConnectionIP != "8.8.8.8" || first.SourceIP != "8.8.8.8" || first.Generation != 1 || !first.SourceChanged {
		t.Fatalf("first observation = %#v", first)
	}
	insertAgentLocationSnapshot(t, db, 7, "8.8.8.8", baseTime)

	same, err := repository.RecordConnection(ctx, 7, "8.8.8.8", baseTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("RecordConnection(same): %v", err)
	}
	if same.ConnectionIP != "8.8.8.8" || same.Generation != 1 || same.SourceChanged || same.ConnectionChanged {
		t.Fatalf("same-IP reconnect = %#v", same)
	}
	assertAgentLocationProvenance(t, db, 7, "8.8.8.8", false, baseTime)

	changed, err := repository.RecordConnection(ctx, 7, "1.1.1.1", baseTime.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("RecordConnection(changed): %v", err)
	}
	if changed.ConnectionIP != "1.1.1.1" || changed.Generation != 2 || !changed.SourceChanged || !changed.ConnectionChanged {
		t.Fatalf("changed observation = %#v", changed)
	}
	assertAgentLocationProvenance(t, db, 7, "8.8.8.8", true, baseTime)

	unknown, err := repository.RecordConnection(ctx, 7, "", baseTime.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("RecordConnection(unknown): %v", err)
	}
	if unknown.ConnectionIP != "" || unknown.SourceIP != "" || unknown.Generation != 3 || !unknown.SourceChanged || !unknown.ConnectionChanged {
		t.Fatalf("public-to-unknown observation = %#v", unknown)
	}

	returned, err := repository.RecordConnection(ctx, 7, "8.8.8.8", baseTime.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("RecordConnection(returned): %v", err)
	}
	if returned.ConnectionIP != "8.8.8.8" || returned.Generation != 4 || !returned.SourceChanged || !returned.ConnectionChanged {
		t.Fatalf("A-to-B-to-A observation = %#v", returned)
	}

	var status model.AgentRuntimeStatus
	if err := db.First(&status, "agent_id = ?", 7).Error; err != nil {
		t.Fatalf("read runtime status: %v", err)
	}
	if status.ConnectionIP != "8.8.8.8" || status.ObservedSourceIP != "8.8.8.8" || status.ObservedIPGeneration != 4 || status.ConnectedAt == nil || !status.ConnectedAt.Equal(baseTime.Add(4*time.Minute)) {
		t.Fatalf("persisted runtime observation = %#v", status)
	}
	var agent model.Agent
	if err := db.First(&agent, 7).Error; err != nil {
		t.Fatalf("read agent: %v", err)
	}
	if agent.Status != "online" || !agent.UpdatedAt.Equal(baseTime.Add(4*time.Minute)) {
		t.Fatalf("persisted Agent connection state = %#v", agent)
	}
}

func TestAgentRepositoryFirstConnectionDoesNotLogMissingRuntimeStatus(t *testing.T) {
	trace := &recordConnectionTrace{}
	db := openAgentConnectionTestDatabaseWithLogger(t, trace)
	repository := &agentRepository{db: db}

	if _, err := repository.RecordConnection(context.Background(), 7, "8.8.8.8", time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("RecordConnection(first): %v", err)
	}
	if trace.recordNotFound != 0 {
		t.Fatalf("first connection emitted %d record-not-found query logs", trace.recordNotFound)
	}

	var status model.AgentRuntimeStatus
	if err := db.First(&status, "agent_id = ?", 7).Error; err != nil {
		t.Fatalf("read created runtime status: %v", err)
	}
}

func TestAgentRepositoryHeartbeatAndGenericUpdateCannotOverwriteObservedSource(t *testing.T) {
	db := openAgentConnectionTestDatabase(t)
	repository := &agentRepository{db: db}
	ctx := context.Background()
	now := time.Date(2026, 8, 4, 11, 0, 0, 0, time.UTC)
	if _, err := repository.RecordConnection(ctx, 7, "8.8.8.8", now); err != nil {
		t.Fatalf("RecordConnection: %v", err)
	}

	if err := repository.UpdateHeartbeat(ctx, 7, agentdomain.AgentHeartbeatUpdate{
		LastHeartbeat: now.Add(time.Second),
		SessionID:     "session-7",
		SessionEpoch:  1,
	}); err != nil {
		t.Fatalf("UpdateHeartbeat: %v", err)
	}
	staleAgent := &agentdomain.Agent{
		ID:                   7,
		InstanceID:           "agt-7",
		DisplayName:          "Agent 7",
		AuthenticationToken:  "deadbeef",
		Status:               "online",
		RegistrationTokenID:  1,
		ObservedHostname:     "node-7",
		ObservedSourceIP:     "1.1.1.1",
		ObservedIPGeneration: 99,
		AgentVersion:         "dev",
		CreatedAt:            now.Add(-time.Hour),
		UpdatedAt:            now.Add(2 * time.Second),
	}
	if err := repository.Update(ctx, staleAgent); err != nil {
		t.Fatalf("generic Update: %v", err)
	}

	var status model.AgentRuntimeStatus
	if err := db.First(&status, "agent_id = ?", 7).Error; err != nil {
		t.Fatalf("read runtime status: %v", err)
	}
	if status.ConnectionIP != "8.8.8.8" || status.ObservedSourceIP != "8.8.8.8" || status.ObservedIPGeneration != 1 {
		t.Fatalf("non-ingress write overwrote source observation: %#v", status)
	}
}

func TestAgentRepositoryPrivateConnectionDoesNotCreatePublicObservation(t *testing.T) {
	db := openAgentConnectionTestDatabase(t)
	repository := &agentRepository{db: db}
	ctx := context.Background()
	baseTime := time.Date(2026, 8, 4, 12, 30, 0, 0, time.UTC)

	first, err := repository.RecordConnection(ctx, 7, "172.20.0.5", baseTime)
	if err != nil {
		t.Fatalf("RecordConnection(first private): %v", err)
	}
	if first.ConnectionIP != "172.20.0.5" || first.SourceIP != "" || first.Generation != 0 || first.SourceChanged || !first.ConnectionChanged {
		t.Fatalf("first private observation = %#v", first)
	}

	changed, err := repository.RecordConnection(ctx, 7, "172.20.0.6", baseTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("RecordConnection(changed private): %v", err)
	}
	if changed.ConnectionIP != "172.20.0.6" || changed.SourceIP != "" || changed.Generation != 0 || changed.SourceChanged || !changed.ConnectionChanged {
		t.Fatalf("private connection change must not create a GeoIP observation: %#v", changed)
	}

	var status model.AgentRuntimeStatus
	if err := db.First(&status, "agent_id = ?", 7).Error; err != nil {
		t.Fatalf("read runtime status: %v", err)
	}
	if status.ConnectionIP != "172.20.0.6" || status.ObservedSourceIP != "" || status.ObservedIPGeneration != 0 {
		t.Fatalf("persisted private connection state = %#v", status)
	}
}

func TestAgentRepositoryClearConnectionIPPreservesGeoIPProvenance(t *testing.T) {
	db := openAgentConnectionTestDatabase(t)
	repository := &agentRepository{db: db}
	ctx := context.Background()
	baseTime := time.Date(2026, 8, 4, 13, 0, 0, 0, time.UTC)

	if _, err := repository.RecordConnection(ctx, 7, "8.8.8.8", baseTime); err != nil {
		t.Fatalf("RecordConnection: %v", err)
	}
	if err := repository.ClearConnectionIP(ctx, 7); err != nil {
		t.Fatalf("ClearConnectionIP: %v", err)
	}

	var status model.AgentRuntimeStatus
	if err := db.First(&status, "agent_id = ?", 7).Error; err != nil {
		t.Fatalf("read runtime status: %v", err)
	}
	if status.ConnectionIP != "" || status.ObservedSourceIP != "8.8.8.8" || status.ObservedIPGeneration != 1 {
		t.Fatalf("detached runtime observation = %#v", status)
	}
}

func openAgentConnectionTestDatabase(t *testing.T) *gorm.DB {
	return openAgentConnectionTestDatabaseWithLogger(t, nil)
}

func openAgentConnectionTestDatabaseWithLogger(t *testing.T, sqlLogger gormlogger.Interface) *gorm.DB {
	t.Helper()
	config := &gorm.Config{}
	if sqlLogger != nil {
		config.Logger = sqlLogger
	}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), config)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE agent (
			id INTEGER PRIMARY KEY, instance_id TEXT NOT NULL, display_name TEXT NOT NULL,
			authentication_token TEXT NOT NULL, status TEXT NOT NULL, max_tasks INTEGER,
			cpu_threshold INTEGER, mem_threshold INTEGER, disk_threshold INTEGER,
			registration_token_id INTEGER NOT NULL, created_at DATETIME, updated_at DATETIME
		);
		CREATE TABLE agent_runtime_status (
			agent_id INTEGER PRIMARY KEY, session_id TEXT, session_epoch INTEGER DEFAULT 0,
			observed_hostname TEXT, connection_ip TEXT NOT NULL DEFAULT '', observed_source_ip TEXT NOT NULL DEFAULT '',
			observed_ip_generation INTEGER NOT NULL DEFAULT 0, agent_version TEXT,
			operating_system TEXT, architecture TEXT, container_runtime_ready BOOLEAN NOT NULL DEFAULT FALSE,
			supported_engine_api_majors JSON NOT NULL DEFAULT '[]', connected_at DATETIME,
			last_heartbeat DATETIME, health_state TEXT DEFAULT 'healthy', health_reason TEXT,
			health_message TEXT, health_since DATETIME, cpu_usage REAL DEFAULT 0,
			mem_usage REAL DEFAULT 0, disk_usage REAL DEFAULT 0, running_tasks INTEGER DEFAULT 0,
			task_slots_used INTEGER DEFAULT 0, uptime_seconds INTEGER DEFAULT 0, updated_at DATETIME
		);
		CREATE TABLE agent_location (
			agent_id INTEGER PRIMARY KEY, latitude REAL NOT NULL, longitude REAL NOT NULL,
			accuracy_radius_km REAL, source_observed_ip TEXT NOT NULL, provider_key TEXT NOT NULL,
			resolved_at DATETIME NOT NULL, forced_expired BOOLEAN NOT NULL DEFAULT FALSE,
			updated_at DATETIME NOT NULL
		);
		INSERT INTO agent (
			id, instance_id, display_name, authentication_token, status, max_tasks,
			cpu_threshold, mem_threshold, disk_threshold, registration_token_id, created_at, updated_at
		) VALUES (7, 'agt-7', 'Agent 7', 'deadbeef', 'offline', 5, 85, 85, 90, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

type recordConnectionTrace struct {
	recordNotFound int
}

func (trace *recordConnectionTrace) LogMode(gormlogger.LogLevel) gormlogger.Interface {
	return trace
}

func (*recordConnectionTrace) Info(context.Context, string, ...interface{}) {}

func (*recordConnectionTrace) Warn(context.Context, string, ...interface{}) {}

func (*recordConnectionTrace) Error(context.Context, string, ...interface{}) {}

func (trace *recordConnectionTrace) Trace(_ context.Context, _ time.Time, _ func() (string, int64), err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		trace.recordNotFound++
	}
}

func insertAgentLocationSnapshot(t *testing.T, db *gorm.DB, agentID int, sourceIP string, resolvedAt time.Time) {
	t.Helper()
	if err := db.Create(&model.AgentLocation{
		AgentID:          agentID,
		Latitude:         35.0,
		Longitude:        139.0,
		SourceObservedIP: sourceIP,
		ProviderKey:      "freeipapi",
		ResolvedAt:       resolvedAt,
		UpdatedAt:        resolvedAt,
	}).Error; err != nil {
		t.Fatalf("create Agent location: %v", err)
	}
}

func assertAgentLocationProvenance(t *testing.T, db *gorm.DB, agentID int, sourceIP string, forcedExpired bool, resolvedAt time.Time) {
	t.Helper()
	var location model.AgentLocation
	if err := db.First(&location, "agent_id = ?", agentID).Error; err != nil {
		t.Fatalf("read Agent location: %v", err)
	}
	if location.SourceObservedIP != sourceIP || location.ForcedExpired != forcedExpired || !location.ResolvedAt.Equal(resolvedAt) || location.Latitude != 35 || location.Longitude != 139 {
		t.Fatalf("Agent location provenance changed unexpectedly: %#v", location)
	}
}
