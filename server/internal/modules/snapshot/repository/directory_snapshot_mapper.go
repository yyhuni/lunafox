package repository

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func directorySnapshotModelToDomain(item *model.DirectorySnapshot) *snapshotdomain.DirectorySnapshot {
	if item == nil {
		return nil
	}
	return &snapshotdomain.DirectorySnapshot{
		ID:            item.ID,
		ScanID:        item.ScanID,
		URL:           item.URL,
		Status:        item.Status,
		ContentLength: item.ContentLength,
		ContentType:   item.ContentType,
		Duration:      item.Duration,
		CreatedAt:     timeutil.ToUTC(item.CreatedAt),
	}
}

func directorySnapshotDomainToModel(item *snapshotdomain.DirectorySnapshot) *model.DirectorySnapshot {
	if item == nil {
		return nil
	}
	return &model.DirectorySnapshot{
		ID:            item.ID,
		ScanID:        item.ScanID,
		URL:           item.URL,
		Status:        item.Status,
		ContentLength: item.ContentLength,
		ContentType:   item.ContentType,
		Duration:      item.Duration,
		CreatedAt:     timeutil.ToUTC(item.CreatedAt),
	}
}

func directorySnapshotDomainListToModel(items []snapshotdomain.DirectorySnapshot) []model.DirectorySnapshot {
	results := make([]model.DirectorySnapshot, 0, len(items))
	for index := range items {
		results = append(results, *directorySnapshotDomainToModel(&items[index]))
	}
	return results
}

func directorySnapshotModelListToDomain(items []model.DirectorySnapshot) []snapshotdomain.DirectorySnapshot {
	results := make([]snapshotdomain.DirectorySnapshot, 0, len(items))
	for index := range items {
		results = append(results, *directorySnapshotModelToDomain(&items[index]))
	}
	return results
}
