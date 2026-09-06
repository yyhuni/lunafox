package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/application"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/loginvisual/repository/persistence"
	"gorm.io/gorm"
)

type SettingsRepository struct{ db *gorm.DB }

func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	if db == nil {
		panic("login visual settings database is required")
	}
	return &SettingsRepository{db: db}
}

func (repository *SettingsRepository) Get(ctx context.Context) (domain.Settings, error) {
	var settings model.LoginVisualSettings
	if err := repository.db.WithContext(ctx).Where("id = ?", 1).FirstOrCreate(&settings, model.LoginVisualSettings{ID: 1}).Error; err != nil {
		return domain.Settings{}, fmt.Errorf("read login visual settings: %w", err)
	}
	return repository.toSettings(ctx, settings)
}

func (repository *SettingsRepository) SaveDraft(ctx context.Context, media domain.Media) (domain.Settings, error) {
	var result domain.Settings
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		mediaModel := mediaToModel(media)
		if err := tx.Create(&mediaModel).Error; err != nil {
			return fmt.Errorf("create login visual media: %w", err)
		}
		var settings model.LoginVisualSettings
		if err := tx.Clauses(clauseForUpdate()).Where("id = ?", 1).FirstOrCreate(&settings, model.LoginVisualSettings{ID: 1}).Error; err != nil {
			return fmt.Errorf("lock login visual settings: %w", err)
		}
		if err := tx.Model(&settings).Updates(map[string]any{"draft_media_id": media.ID, "updated_at": time.Now().UTC()}).Error; err != nil {
			return fmt.Errorf("set login visual draft: %w", err)
		}
		settings.DraftMediaID = &media.ID
		var err error
		result, err = repository.toSettingsWithDB(ctx, tx, settings)
		return err
	})
	return result, err
}

func (repository *SettingsRepository) PublishDraft(ctx context.Context) (domain.Settings, error) {
	var result domain.Settings
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var settings model.LoginVisualSettings
		if err := tx.Clauses(clauseForUpdate()).Where("id = ?", 1).FirstOrCreate(&settings, model.LoginVisualSettings{ID: 1}).Error; err != nil {
			return fmt.Errorf("lock login visual settings: %w", err)
		}
		if settings.DraftMediaID == nil {
			return application.ErrNoDraft
		}
		if err := tx.Model(&settings).Updates(map[string]any{"published_media_id": *settings.DraftMediaID, "draft_media_id": nil, "updated_at": time.Now().UTC()}).Error; err != nil {
			return fmt.Errorf("publish login visual draft: %w", err)
		}
		settings.PublishedMediaID = settings.DraftMediaID
		settings.DraftMediaID = nil
		var err error
		result, err = repository.toSettingsWithDB(ctx, tx, settings)
		return err
	})
	return result, err
}

func (repository *SettingsRepository) RestoreDefault(ctx context.Context) (domain.Settings, error) {
	var result domain.Settings
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var settings model.LoginVisualSettings
		if err := tx.Clauses(clauseForUpdate()).Where("id = ?", 1).FirstOrCreate(&settings, model.LoginVisualSettings{ID: 1}).Error; err != nil {
			return fmt.Errorf("lock login visual settings: %w", err)
		}
		if err := tx.Model(&settings).Updates(map[string]any{"published_media_id": nil, "draft_media_id": nil, "updated_at": time.Now().UTC()}).Error; err != nil {
			return fmt.Errorf("restore default login visual: %w", err)
		}
		return nil
	})
	return result, err
}

func (repository *SettingsRepository) toSettings(ctx context.Context, settings model.LoginVisualSettings) (domain.Settings, error) {
	return repository.toSettingsWithDB(ctx, repository.db, settings)
}

func (repository *SettingsRepository) toSettingsWithDB(ctx context.Context, db *gorm.DB, settings model.LoginVisualSettings) (domain.Settings, error) {
	result := domain.Settings{}
	if settings.DraftMediaID != nil {
		media, err := readMedia(ctx, db, *settings.DraftMediaID)
		if err != nil {
			return domain.Settings{}, err
		}
		result.Draft = &media
	}
	if settings.PublishedMediaID != nil {
		media, err := readMedia(ctx, db, *settings.PublishedMediaID)
		if err != nil {
			return domain.Settings{}, err
		}
		result.Published = &media
	}
	return result, nil
}

func readMedia(ctx context.Context, db *gorm.DB, id string) (domain.Media, error) {
	var record model.LoginVisualMedia
	if err := db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Media{}, application.ErrMediaUnavailable
		}
		return domain.Media{}, fmt.Errorf("read login visual media: %w", err)
	}
	return modelToMedia(record), nil
}

func mediaToModel(media domain.Media) model.LoginVisualMedia {
	return model.LoginVisualMedia{ID: media.ID, Kind: string(media.Kind), ContentType: media.ContentType, SizeBytes: media.SizeBytes, DurationMS: media.Duration.Milliseconds(), StorageKey: media.StorageKey, PosterKey: media.PosterKey, CreatedAt: media.CreatedAt}
}
func modelToMedia(record model.LoginVisualMedia) domain.Media {
	return domain.Media{ID: record.ID, Kind: domain.MediaKind(record.Kind), ContentType: record.ContentType, SizeBytes: record.SizeBytes, Duration: time.Duration(record.DurationMS) * time.Millisecond, StorageKey: record.StorageKey, PosterKey: record.PosterKey, CreatedAt: record.CreatedAt.UTC()}
}

var _ application.SettingsStore = (*SettingsRepository)(nil)
