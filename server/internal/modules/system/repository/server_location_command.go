package repository

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/geolocation"
	systemdomain "github.com/yyhuni/lunafox/server/internal/modules/system/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/system/repository/persistence"
	"gorm.io/gorm/clause"
)

func (repository *serverLocationRepository) Replace(ctx context.Context, snapshot systemdomain.ServerLocationSnapshot) error {
	if repository == nil || repository.db == nil {
		return errors.New("Server location repository is required")
	}
	normalized, err := validateServerLocationSuccess(snapshot)
	if err != nil {
		return err
	}
	record := persistence.ServerLocationSnapshot{
		SingletonID:      serverLocationSingletonID,
		ObservedEgressIP: normalized.ObservedEgressIP,
		Latitude:         normalized.Latitude,
		Longitude:        normalized.Longitude,
		AccuracyRadiusKM: copyServerLocationRadius(normalized.AccuracyRadiusKM),
		ProviderKey:      normalized.ProviderKey,
		ResolvedAt:       normalized.ResolvedAt,
		ForcedExpired:    false,
		UpdatedAt:        normalized.UpdatedAt,
	}
	return repository.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "singleton_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"observed_egress_ip",
			"latitude",
			"longitude",
			"accuracy_radius_km",
			"provider_key",
			"resolved_at",
			"forced_expired",
			"updated_at",
		}),
	}).Create(&record).Error
}

func (repository *serverLocationRepository) MarkExpired(ctx context.Context) error {
	if repository == nil || repository.db == nil {
		return errors.New("Server location repository is required")
	}
	return repository.db.WithContext(ctx).
		Model(&persistence.ServerLocationSnapshot{}).
		Where("singleton_id = ?", serverLocationSingletonID).
		Updates(map[string]any{"forced_expired": true, "updated_at": time.Now().UTC()}).Error
}

func validateServerLocationSuccess(snapshot systemdomain.ServerLocationSnapshot) (systemdomain.ServerLocationSnapshot, error) {
	normalizedIP, ok := geolocation.NormalizePublicIP(snapshot.ObservedEgressIP)
	if !ok || normalizedIP != snapshot.ObservedEgressIP {
		return systemdomain.ServerLocationSnapshot{}, errors.New("normalized public Server egress IP is required")
	}
	if !finiteServerLocationValue(snapshot.Latitude, -90, 90) || !finiteServerLocationValue(snapshot.Longitude, -180, 180) {
		return systemdomain.ServerLocationSnapshot{}, errors.New("complete in-range Server location coordinates are required")
	}
	if snapshot.AccuracyRadiusKM != nil && (math.IsNaN(*snapshot.AccuracyRadiusKM) || math.IsInf(*snapshot.AccuracyRadiusKM, 0) || *snapshot.AccuracyRadiusKM < 0) {
		return systemdomain.ServerLocationSnapshot{}, errors.New("Server location accuracy radius must be finite and non-negative")
	}
	if strings.TrimSpace(snapshot.ProviderKey) == "" || strings.TrimSpace(snapshot.ProviderKey) != snapshot.ProviderKey {
		return systemdomain.ServerLocationSnapshot{}, errors.New("Server location provider key is required")
	}
	if snapshot.ResolvedAt.IsZero() || snapshot.UpdatedAt.IsZero() {
		return systemdomain.ServerLocationSnapshot{}, errors.New("Server location resolution and update times are required")
	}
	if snapshot.ForcedExpired {
		return systemdomain.ServerLocationSnapshot{}, errors.New("successful Server location replacement cannot be forced expired")
	}
	normalized := snapshot
	normalized.ResolvedAt = snapshot.ResolvedAt.UTC()
	normalized.UpdatedAt = snapshot.UpdatedAt.UTC()
	normalized.AccuracyRadiusKM = copyServerLocationRadius(snapshot.AccuracyRadiusKM)
	return normalized, nil
}

func finiteServerLocationValue(value, minimum, maximum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= minimum && value <= maximum
}

func copyServerLocationRadius(radius *float64) *float64 {
	if radius == nil {
		return nil
	}
	copy := *radius
	return &copy
}
