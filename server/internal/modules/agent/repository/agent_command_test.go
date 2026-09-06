package repository

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type agentOfflineNotificationSinkStub struct {
	calls int
	err   error
}

func (sink *agentOfflineNotificationSinkStub) WriteAgentOffline(_ *gorm.DB, _ int, _ time.Time) error {
	sink.calls++
	return sink.err
}

func TestAgentRepositoryMarkOfflineFromOnlineIsAtomicAndIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "agent-offline.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`CREATE TABLE agent (id INTEGER PRIMARY KEY, status TEXT NOT NULL, updated_at DATETIME)`).Error; err != nil {
		t.Fatalf("create agent table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, connection_ip TEXT NOT NULL DEFAULT '', updated_at DATETIME)`).Error; err != nil {
		t.Fatalf("create runtime status table: %v", err)
	}
	if err := db.Exec(`INSERT INTO agent (id, status) VALUES (7, 'online'), (8, 'online')`).Error; err != nil {
		t.Fatalf("seed agents: %v", err)
	}
	if err := db.Exec(`INSERT INTO agent_runtime_status (agent_id, connection_ip) VALUES (7, '172.20.0.7'), (8, '172.20.0.8')`).Error; err != nil {
		t.Fatalf("seed runtime status: %v", err)
	}

	sink := &agentOfflineNotificationSinkStub{}
	repository := &agentRepository{db: db, offlineNotificationSink: sink}
	transitioned, err := repository.MarkOfflineFromOnline(context.Background(), 7)
	if err != nil || !transitioned {
		t.Fatalf("first MarkOfflineFromOnline() = (%v, %v), want (true, nil)", transitioned, err)
	}
	transitioned, err = repository.MarkOfflineFromOnline(context.Background(), 7)
	if err != nil || transitioned {
		t.Fatalf("repeated MarkOfflineFromOnline() = (%v, %v), want (false, nil)", transitioned, err)
	}
	if sink.calls != 1 {
		t.Fatalf("notification sink calls = %d, want 1", sink.calls)
	}
	var clearedConnectionIP string
	if err := db.Raw(`SELECT connection_ip FROM agent_runtime_status WHERE agent_id = 7`).Scan(&clearedConnectionIP).Error; err != nil {
		t.Fatalf("read cleared connection address: %v", err)
	}
	if clearedConnectionIP != "" {
		t.Fatalf("offline Agent retained connection address %q", clearedConnectionIP)
	}

	sink.err = errors.New("outbox unavailable")
	if _, err := repository.MarkOfflineFromOnline(context.Background(), 8); !errors.Is(err, sink.err) {
		t.Fatalf("sink error = %v, want %v", err, sink.err)
	}
	var status string
	if err := db.Raw(`SELECT status FROM agent WHERE id = 8`).Scan(&status).Error; err != nil {
		t.Fatalf("read rolled-back agent status: %v", err)
	}
	if status != "online" {
		t.Fatalf("agent status after outbox failure = %q, want online", status)
	}
	var retainedConnectionIP string
	if err := db.Raw(`SELECT connection_ip FROM agent_runtime_status WHERE agent_id = 8`).Scan(&retainedConnectionIP).Error; err != nil {
		t.Fatalf("read rolled-back connection address: %v", err)
	}
	if retainedConnectionIP != "172.20.0.8" {
		t.Fatalf("outbox rollback changed connection address to %q", retainedConnectionIP)
	}
}

func TestAgentRepositoryUpdateHeartbeatPersistsExecutionSnapshotAndTaskCounts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "agent-heartbeat.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`CREATE TABLE agent (id INTEGER PRIMARY KEY, status TEXT, updated_at DATETIME)`).Error; err != nil {
		t.Fatalf("create agent table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE agent_runtime_status (
		agent_id INTEGER PRIMARY KEY, session_id TEXT, session_epoch INTEGER,
			observed_hostname TEXT, connection_ip TEXT NOT NULL DEFAULT '', observed_source_ip TEXT NOT NULL DEFAULT '',
			observed_ip_generation INTEGER NOT NULL DEFAULT 0, agent_version TEXT,
		operating_system TEXT, architecture TEXT, container_runtime_ready BOOLEAN,
		supported_engine_api_majors JSON, connected_at DATETIME, last_heartbeat DATETIME,
		health_state TEXT, health_reason TEXT, health_message TEXT, health_since DATETIME,
		cpu_usage REAL, mem_usage REAL, disk_usage REAL, running_tasks INTEGER,
		task_slots_used INTEGER, uptime_seconds INTEGER, updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create runtime status table: %v", err)
	}
	if err := db.Exec(`INSERT INTO agent (id, status) VALUES (7, 'offline')`).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := db.Exec(`INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch, connection_ip) VALUES (7, '', 0, '172.20.0.7')`).Error; err != nil {
		t.Fatalf("seed runtime status: %v", err)
	}
	repository := &agentRepository{db: db}
	err = repository.UpdateHeartbeat(context.Background(), 7, agentdomain.AgentHeartbeatUpdate{
		LastHeartbeat:            time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC),
		SessionID:                "session-7",
		SessionEpoch:             4,
		AgentVersion:             "2.0.0",
		OperatingSystem:          "linux",
		Architecture:             "amd64",
		ContainerRuntimeReady:    true,
		SupportedEngineAPIMajors: []uint32{2},
		CPU:                      1.5,
		Mem:                      2.5,
		Disk:                     3.5,
		RunningTasks:             2,
		TaskSlotsUsed:            3,
		Uptime:                   99,
	})
	if err != nil {
		t.Fatalf("UpdateHeartbeat() error = %v", err)
	}
	var status model.AgentRuntimeStatus
	if err := db.First(&status, "agent_id = ?", 7).Error; err != nil {
		t.Fatalf("read runtime status: %v", err)
	}
	var majors []uint32
	if err := json.Unmarshal(status.SupportedEngineAPIMajors, &majors); err != nil {
		t.Fatalf("decode persisted majors: %v", err)
	}
	if status.SessionID != "session-7" || status.SessionEpoch != 4 || status.OperatingSystem != "linux" || status.Architecture != "amd64" || !status.ContainerRuntimeReady || len(majors) != 1 || majors[0] != 2 || status.RunningTasks != 2 || status.TaskSlotsUsed != 3 || status.UptimeSeconds != 99 || status.ConnectionIP != "172.20.0.7" {
		t.Fatalf("persisted execution snapshot = %#v majors=%v", status, majors)
	}
	for _, stale := range []agentdomain.AgentHeartbeatUpdate{
		{LastHeartbeat: time.Now().UTC(), SessionID: "session-old", SessionEpoch: 3, OperatingSystem: "darwin", Architecture: "arm64", SupportedEngineAPIMajors: []uint32{99}},
		{LastHeartbeat: time.Now().UTC(), SessionID: "session-other", SessionEpoch: 4, OperatingSystem: "darwin", Architecture: "arm64", SupportedEngineAPIMajors: []uint32{99}},
	} {
		if err := repository.UpdateHeartbeat(context.Background(), 7, stale); !errors.Is(err, agentdomain.ErrStaleAgentHeartbeat) {
			t.Fatalf("stale UpdateHeartbeat() error = %v, want ErrStaleAgentHeartbeat", err)
		}
	}
	var after model.AgentRuntimeStatus
	if err := db.First(&after, "agent_id = ?", 7).Error; err != nil {
		t.Fatalf("read runtime status after stale writes: %v", err)
	}
	if after.SessionID != "session-7" || after.SessionEpoch != 4 || after.OperatingSystem != "linux" {
		t.Fatalf("stale heartbeat overwrote current snapshot: %#v", after)
	}
}

func TestAgentRepositoryCreateBackfillsID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:agent_repo_create?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
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
			instance_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			authentication_token TEXT NOT NULL UNIQUE,
			status TEXT,
			max_tasks INTEGER,
			cpu_threshold INTEGER,
			mem_threshold INTEGER,
			disk_threshold INTEGER,
			registration_token_id INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);
		INSERT INTO registration_token (id, token, expires_at, created_at)
		VALUES (1, 'abcd1234', '2099-01-01 00:00:00', CURRENT_TIMESTAMP);
	`).Error; err != nil {
		t.Fatalf("create agent table failed: %v", err)
	}

	repo := &agentRepository{db: db}
	agent := agentdomain.NewRegisteredAgent(
		1,
		"agt_123456",
		"agent-test",
		"dev",
		"deadbeef",
		agentdomain.AgentRegistrationOptions{},
	)

	if err := repo.Create(context.Background(), agent); err != nil {
		t.Fatalf("create agent failed: %v", err)
	}
	if agent.ID <= 0 {
		t.Fatalf("expected created agent ID to be backfilled, got %d", agent.ID)
	}
	if agent.InstanceID != "agt_123456" {
		t.Fatalf("expected instance id preserved, got %q", agent.InstanceID)
	}
}
