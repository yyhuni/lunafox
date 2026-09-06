package repository

import (
	"context"
	"errors"

	systemdomain "github.com/yyhuni/lunafox/server/internal/modules/system/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/system/repository/persistence"
	"gorm.io/gorm"
)

func (repository *serverLocationRepository) Get(ctx context.Context) (*systemdomain.ServerLocationSnapshot, error) {
	if repository == nil || repository.db == nil {
		return nil, errors.New("Server location repository is required")
	}
	var record persistence.ServerLocationSnapshot
	err := repository.db.WithContext(ctx).First(&record, "singleton_id = ?", serverLocationSingletonID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return serverLocationModelToDomain(record), nil
}

func serverLocationModelToDomain(record persistence.ServerLocationSnapshot) *systemdomain.ServerLocationSnapshot {
	return &systemdomain.ServerLocationSnapshot{
		ObservedEgressIP: record.ObservedEgressIP,
		Latitude:         record.Latitude,
		Longitude:        record.Longitude,
		AccuracyRadiusKM: copyServerLocationRadius(record.AccuracyRadiusKM),
		ProviderKey:      record.ProviderKey,
		ResolvedAt:       record.ResolvedAt.UTC(),
		ForcedExpired:    record.ForcedExpired,
		UpdatedAt:        record.UpdatedAt.UTC(),
	}
}
