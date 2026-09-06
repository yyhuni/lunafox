package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/geolocation"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (repository *agentLocationRepository) ReplaceLocationIfObservationMatches(
	ctx context.Context,
	snapshot agentdomain.AgentLocationSnapshot,
	generation int64,
) (bool, error) {
	if repository == nil || repository.db == nil {
		return false, errors.New("Agent location repository is required")
	}
	normalized, err := validateAgentLocationWrite(snapshot, generation)
	if err != nil {
		return false, err
	}
	replaced := false
	err = repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		matches, err := lockMatchingAgentObservation(tx, normalized.AgentID, normalized.SourceObservedIP, generation)
		if err != nil || !matches {
			return err
		}
		record := domainAgentLocationToModel(&normalized)
		result := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "agent_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"latitude",
				"longitude",
				"accuracy_radius_km",
				"source_observed_ip",
				"provider_key",
				"resolved_at",
				"forced_expired",
				"updated_at",
			}),
		}).Create(record)
		if result.Error != nil {
			return result.Error
		}
		replaced = result.RowsAffected == 1
		return nil
	})
	return replaced, err
}

func (repository *agentLocationRepository) MarkLocationExpiredIfObservationMatches(
	ctx context.Context,
	agentID int,
	sourceIP string,
	generation int64,
) (bool, error) {
	if repository == nil || repository.db == nil {
		return false, errors.New("Agent location repository is required")
	}
	normalizedSource, ok := geolocation.NormalizePublicIP(sourceIP)
	if agentID <= 0 || generation <= 0 || !ok {
		return false, errors.New("Agent identity, public source observation, and positive generation are required")
	}
	marked := false
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		matches, err := lockMatchingAgentObservation(tx, agentID, normalizedSource, generation)
		if err != nil || !matches {
			return err
		}
		result := tx.Model(&model.AgentLocation{}).
			Where("agent_id = ?", agentID).
			Updates(map[string]any{
				"forced_expired": true,
				"updated_at":     time.Now().UTC(),
			})
		if result.Error != nil {
			return result.Error
		}
		marked = result.RowsAffected == 1
		return nil
	})
	return marked, err
}

func lockMatchingAgentObservation(tx *gorm.DB, agentID int, sourceIP string, generation int64) (bool, error) {
	var runtimeStatus model.AgentRuntimeStatus
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("agent_id", "observed_source_ip", "observed_ip_generation").
		First(&runtimeStatus, "agent_id = ?", agentID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return runtimeStatus.ObservedSourceIP == sourceIP && runtimeStatus.ObservedIPGeneration == generation, nil
}

func validateAgentLocationWrite(snapshot agentdomain.AgentLocationSnapshot, generation int64) (agentdomain.AgentLocationSnapshot, error) {
	if snapshot.AgentID <= 0 || generation <= 0 {
		return agentdomain.AgentLocationSnapshot{}, errors.New("Agent identity and positive observation generation are required")
	}
	normalizedSource, ok := geolocation.NormalizePublicIP(snapshot.SourceObservedIP)
	if !ok || normalizedSource != snapshot.SourceObservedIP {
		return agentdomain.AgentLocationSnapshot{}, errors.New("normalized public source observation is required")
	}
	if !finiteCoordinate(snapshot.Latitude, -90, 90) || !finiteCoordinate(snapshot.Longitude, -180, 180) {
		return agentdomain.AgentLocationSnapshot{}, errors.New("complete in-range Agent location coordinates are required")
	}
	if snapshot.AccuracyRadiusKM != nil && (math.IsNaN(*snapshot.AccuracyRadiusKM) || math.IsInf(*snapshot.AccuracyRadiusKM, 0) || *snapshot.AccuracyRadiusKM < 0) {
		return agentdomain.AgentLocationSnapshot{}, errors.New("Agent location accuracy radius must be finite and non-negative")
	}
	if strings.TrimSpace(snapshot.ProviderKey) == "" || snapshot.ProviderKey != strings.TrimSpace(snapshot.ProviderKey) {
		return agentdomain.AgentLocationSnapshot{}, errors.New("Agent location provider key is required")
	}
	if snapshot.ResolvedAt.IsZero() || snapshot.UpdatedAt.IsZero() {
		return agentdomain.AgentLocationSnapshot{}, errors.New("Agent location resolution and update times are required")
	}
	if snapshot.ForcedExpired {
		return agentdomain.AgentLocationSnapshot{}, fmt.Errorf("successful Agent location replacement cannot be forced expired")
	}
	normalized := snapshot
	normalized.ResolvedAt = snapshot.ResolvedAt.UTC()
	normalized.UpdatedAt = snapshot.UpdatedAt.UTC()
	normalized.AccuracyRadiusKM = copyFloat64Ptr(snapshot.AccuracyRadiusKM)
	return normalized, nil
}

func finiteCoordinate(value, minimum, maximum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= minimum && value <= maximum
}
