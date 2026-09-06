package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/geolocation"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Create creates a new agent.
func (r *agentRepository) Create(ctx context.Context, agent *agentdomain.Agent) error {
	if agent == nil {
		return fmt.Errorf("agent is required")
	}
	if agent.RegistrationTokenID <= 0 {
		return fmt.Errorf("agent registration token identity is required")
	}
	record := domainAgentToModel(agent)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		attribution := tx.Model(&model.RegistrationToken{}).
			Where("id = ? AND expires_at > CURRENT_TIMESTAMP", agent.RegistrationTokenID).
			Update("ever_attributed_at", gorm.Expr("COALESCE(ever_attributed_at, CURRENT_TIMESTAMP)"))
		if attribution.Error != nil {
			return attribution.Error
		}
		if attribution.RowsAffected != 1 {
			return agentdomain.ErrRegistrationTokenInvalid
		}
		return tx.Create(record).Error
	})
	if err != nil {
		return err
	}
	agent.ID = record.ID
	return nil
}

// Update updates an agent.
func (r *agentRepository) Update(ctx context.Context, agent *agentdomain.Agent) error {
	if err := r.db.WithContext(ctx).Save(domainAgentToModel(agent)).Error; err != nil {
		return err
	}
	status := domainAgentRuntimeStatusToModel(agent)
	if status == nil {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "agent_id"}},
		// Connection source and generation are owned exclusively by RecordConnection.
		DoUpdates: clause.AssignmentColumns([]string{"observed_hostname", "agent_version", "connected_at", "last_heartbeat", "health_state", "health_reason", "health_message", "health_since", "updated_at"}),
	}).Create(status).Error
}

// RecordConnection atomically records the ingress-owned connection address
// before the control stream can receive SessionReady. The public observation is
// derived here from the same candidate and never comes from Agent payloads.
func (r *agentRepository) RecordConnection(ctx context.Context, agentID int, connectionIP string, connectedAt time.Time) (agentdomain.AgentConnectionObservation, error) {
	if agentID <= 0 {
		return agentdomain.AgentConnectionObservation{}, fmt.Errorf("agent identity is required")
	}
	normalizedConnection, err := normalizeRepositoryConnectionIP(connectionIP)
	if err != nil {
		return agentdomain.AgentConnectionObservation{}, err
	}
	normalizedSource, _ := geolocation.NormalizePublicIP(normalizedConnection)
	if connectedAt.IsZero() {
		return agentdomain.AgentConnectionObservation{}, fmt.Errorf("connection time is required")
	}
	connectedAt = connectedAt.UTC()
	observation := agentdomain.AgentConnectionObservation{
		AgentID:      agentID,
		ConnectionIP: normalizedConnection,
		SourceIP:     normalizedSource,
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var identity model.Agent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&identity, agentID).Error; err != nil {
			return err
		}

		var runtimeStatus model.AgentRuntimeStatus
		// An absent runtime row is expected for the first connection. Find reports
		// it through RowsAffected so the normal initialization path is not logged
		// as a database error.
		statusResult := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("agent_id = ?", agentID).Find(&runtimeStatus)
		switch {
		case statusResult.Error != nil:
			return statusResult.Error
		case statusResult.RowsAffected == 0:
			observation.ConnectionChanged = normalizedConnection != ""
			observation.SourceChanged = normalizedSource != ""
			if observation.SourceChanged {
				observation.Generation = 1
			}
			runtimeStatus = model.AgentRuntimeStatus{
				AgentID:              agentID,
				ConnectionIP:         normalizedConnection,
				ObservedSourceIP:     normalizedSource,
				ObservedIPGeneration: observation.Generation,
				ConnectedAt:          &connectedAt,
				UpdatedAt:            connectedAt,
			}
			if err := tx.Create(&runtimeStatus).Error; err != nil {
				return err
			}
		default:
			observation.Generation = runtimeStatus.ObservedIPGeneration
			observation.SourceChanged = runtimeStatus.ObservedSourceIP != normalizedSource
			observation.ConnectionChanged = runtimeStatus.ConnectionIP != normalizedConnection
			if observation.SourceChanged {
				const maxGeneration = int64(^uint64(0) >> 1)
				if observation.Generation == maxGeneration {
					return fmt.Errorf("observed IP generation is exhausted for agent %d", agentID)
				}
				observation.Generation++
			}
			if err := tx.Model(&model.AgentRuntimeStatus{}).
				Where("agent_id = ?", agentID).
				Updates(map[string]any{
					"connection_ip":          normalizedConnection,
					"observed_source_ip":     normalizedSource,
					"observed_ip_generation": observation.Generation,
					"connected_at":           connectedAt,
					"updated_at":             connectedAt,
				}).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&model.Agent{}).
			Where("id = ?", agentID).
			Updates(map[string]any{"status": "online", "updated_at": connectedAt}).Error; err != nil {
			return err
		}
		if !observation.SourceChanged {
			return nil
		}
		return tx.Model(&model.AgentLocation{}).
			Where("agent_id = ?", agentID).
			Updates(map[string]any{"forced_expired": true, "updated_at": connectedAt}).Error
	})
	if err != nil {
		return agentdomain.AgentConnectionObservation{}, err
	}
	return observation, nil
}

// ClearConnectionIP removes only the current physical control-connection
// address. The caller serializes it with session replacement so a stale stream
// cannot erase a newer connection observation during the recovery window.
func (r *agentRepository) ClearConnectionIP(ctx context.Context, agentID int) error {
	if agentID <= 0 {
		return fmt.Errorf("agent identity is required")
	}
	return r.db.WithContext(ctx).Model(&model.AgentRuntimeStatus{}).
		Where("agent_id = ?", agentID).
		Updates(map[string]interface{}{
			"connection_ip": "",
			"updated_at":    time.Now().UTC(),
		}).Error
}

// normalizeRepositoryConnectionIP validates the server-owned candidate while
// allowing private transport addresses. Special-use public-looking ranges are
// rejected so they cannot be mistaken for a real current connection.
func normalizeRepositoryConnectionIP(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	address, err := netip.ParseAddr(trimmed)
	if err != nil {
		return "", fmt.Errorf("connection source is invalid: %w", err)
	}
	address = address.Unmap().WithZone("")
	if !address.IsValid() || address.IsUnspecified() || address.IsMulticast() {
		return "", fmt.Errorf("connection source is not a usable address")
	}
	normalized := address.String()
	if _, ok := geolocation.NormalizePublicIP(normalized); ok || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || netip.MustParsePrefix("100.64.0.0/10").Contains(address) {
		return normalized, nil
	}
	return "", fmt.Errorf("connection source is a reserved or unsupported address")
}

// UpdateStatus updates agent status.
func (r *agentRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Agent{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     status,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
		if status != "offline" {
			return nil
		}
		return tx.Model(&model.AgentRuntimeStatus{}).
			Where("agent_id = ?", id).
			Updates(map[string]interface{}{"connection_ip": "", "updated_at": now}).Error
	})
}

// MarkOfflineFromOnline is the authoritative monitor transition. The status
// predicate makes repeated stale checks harmless, and the candidate write is
// kept in the same transaction so the transition never commits without its
// durable outbox handoff.
func (r *agentRepository) MarkOfflineFromOnline(ctx context.Context, id int) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("agent identity is required")
	}
	now := time.Now().UTC()
	transitioned := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Agent{}).
			Where("id = ? AND status = ?", id, "online").
			Updates(map[string]interface{}{"status": "offline", "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := tx.Model(&model.AgentRuntimeStatus{}).
			Where("agent_id = ?", id).
			Updates(map[string]interface{}{"connection_ip": "", "updated_at": now}).Error; err != nil {
			return err
		}
		if r.offlineNotificationSink != nil {
			if err := r.offlineNotificationSink.WriteAgentOffline(tx, id, now); err != nil {
				return err
			}
		}
		transitioned = true
		return nil
	})
	return transitioned, err
}

// Delete deletes an agent.
func (r *agentRepository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&model.Agent{}, id).Error
}

// UpdateHeartbeat updates agent last heartbeat time and runtime fields.
func (r *agentRepository) UpdateHeartbeat(ctx context.Context, id int, update agentdomain.AgentHeartbeatUpdate) error {
	now := update.LastHeartbeat
	majorsJSON, err := json.Marshal(update.SupportedEngineAPIMajors)
	if err != nil {
		return err
	}
	status := &model.AgentRuntimeStatus{
		AgentID:                  id,
		SessionID:                strings.TrimSpace(update.SessionID),
		SessionEpoch:             update.SessionEpoch,
		ObservedHostname:         strings.TrimSpace(update.ObservedHostname),
		AgentVersion:             strings.TrimSpace(update.AgentVersion),
		OperatingSystem:          strings.TrimSpace(update.OperatingSystem),
		Architecture:             strings.TrimSpace(update.Architecture),
		ContainerRuntimeReady:    update.ContainerRuntimeReady,
		SupportedEngineAPIMajors: majorsJSON,
		LastHeartbeat:            &now,
		CPUUsage:                 update.CPU,
		MemUsage:                 update.Mem,
		DiskUsage:                update.Disk,
		RunningTasks:             update.RunningTasks,
		TaskSlotsUsed:            update.TaskSlotsUsed,
		UptimeSeconds:            update.Uptime,
		HealthState:              update.HealthState,
		HealthReason:             update.HealthReason,
		HealthMessage:            update.HealthMessage,
		HealthSince:              update.HealthSince,
		UpdatedAt:                now,
	}
	if update.HasHealth {
		status.HealthState = update.HealthState
		status.HealthReason = update.HealthReason
		status.HealthMessage = update.HealthMessage
		status.HealthSince = update.HealthSince
	}
	if status.HealthState == "" {
		status.HealthState = "healthy"
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "agent_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"session_id",
				"session_epoch",
				"observed_hostname",
				"agent_version",
				"operating_system",
				"architecture",
				"container_runtime_ready",
				"supported_engine_api_majors",
				"last_heartbeat",
				"cpu_usage",
				"mem_usage",
				"disk_usage",
				"running_tasks",
				"task_slots_used",
				"uptime_seconds",
				"health_state",
				"health_reason",
				"health_message",
				"health_since",
				"updated_at",
			}),
			Where: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "agent_runtime_status.session_epoch < excluded.session_epoch OR (agent_runtime_status.session_epoch = excluded.session_epoch AND agent_runtime_status.session_id = excluded.session_id)"}}},
		}).Create(status)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return agentdomain.ErrStaleAgentHeartbeat
		}
		return tx.Model(&model.Agent{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{"status": "online", "updated_at": now}).Error
	})
}
