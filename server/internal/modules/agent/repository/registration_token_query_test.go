package repository

import (
	"context"
	"testing"
	"time"

	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
)

func TestRegistrationTokenResourceReturnsCompleteScopedProjectionAfterExpiry(t *testing.T) {
	db := openRegistrationTokenTestDB(t)
	expiredAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	insertRegistrationToken(t, db, 1, "aaaaaaaa", expiredAt)
	insertRegistrationToken(t, db, 2, "bbbbbbbb", expiredAt)
	createdAt := time.Date(2025, 12, 31, 23, 0, 0, 0, time.UTC)
	agents := []model.Agent{
		{ID: 12, InstanceID: "token-1-second", DisplayName: "second", AuthenticationToken: "00000012", Status: "online", RegistrationTokenID: 1, CreatedAt: createdAt.Add(time.Minute)},
		{ID: 11, InstanceID: "token-1-first", DisplayName: "first", AuthenticationToken: "00000011", Status: "offline", RegistrationTokenID: 1, CreatedAt: createdAt},
		{ID: 21, InstanceID: "token-2", DisplayName: "other", AuthenticationToken: "00000021", Status: "online", RegistrationTokenID: 2, CreatedAt: createdAt},
	}
	for index := range agents {
		if err := db.Create(&agents[index]).Error; err != nil {
			t.Fatalf("insert Agent %d: %v", agents[index].ID, err)
		}
	}
	heartbeatAt := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	if err := db.Exec(`INSERT INTO agent_runtime_status (agent_id, observed_hostname, observed_source_ip, observed_ip_generation, last_heartbeat, health_state) VALUES (?, ?, ?, ?, ?, ?)`, 12, "node-12", "8.8.8.8", 2, heartbeatAt, "healthy").Error; err != nil {
		t.Fatalf("insert runtime status: %v", err)
	}

	repository := &registrationTokenRepository{db: db}
	resource, err := repository.GetResourceByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetResourceByID error: %v", err)
	}
	if resource.ID != 1 || !resource.ExpiresAt.Equal(expiredAt) || len(resource.Agents) != 2 {
		t.Fatalf("token resource = %#v", resource)
	}
	if resource.Agents[0].ID != 11 || resource.Agents[1].ID != 12 {
		t.Fatalf("token-scoped Agent order/identity = %#v", resource.Agents)
	}
	if resource.Agents[1].ObservedSourceIP != "8.8.8.8" || resource.Agents[1].LastHeartbeat == nil || !resource.Agents[1].LastHeartbeat.Equal(heartbeatAt) {
		t.Fatalf("current Agent projection = %#v", resource.Agents[1])
	}
	if resource.StateAt(expiredAt) != "expired" {
		t.Fatalf("exact-expiry state = %q", resource.StateAt(expiredAt))
	}
}
