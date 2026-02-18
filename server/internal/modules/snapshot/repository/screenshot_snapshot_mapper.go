package repository

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func screenshotSnapshotModelToDomain(item *model.ScreenshotSnapshot) *snapshotdomain.ScreenshotSnapshot {
	if item == nil {
		return nil
	}
	return &snapshotdomain.ScreenshotSnapshot{ID: item.ID, ScanID: item.ScanID, URL: item.URL, StatusCode: item.StatusCode, Image: append([]byte(nil), item.Image...), CreatedAt: timeutil.ToUTC(item.CreatedAt)}
}

func screenshotSnapshotDomainToModel(item *snapshotdomain.ScreenshotSnapshot) *model.ScreenshotSnapshot {
	if item == nil {
		return nil
	}
	return &model.ScreenshotSnapshot{ID: item.ID, ScanID: item.ScanID, URL: item.URL, StatusCode: item.StatusCode, Image: append([]byte(nil), item.Image...), CreatedAt: timeutil.ToUTC(item.CreatedAt)}
}

func screenshotSnapshotDomainListToModel(items []snapshotdomain.ScreenshotSnapshot) []model.ScreenshotSnapshot {
	results := make([]model.ScreenshotSnapshot, 0, len(items))
	for index := range items {
		results = append(results, *screenshotSnapshotDomainToModel(&items[index]))
	}
	return results
}

func screenshotSnapshotModelListToDomain(items []model.ScreenshotSnapshot) []snapshotdomain.ScreenshotSnapshot {
	results := make([]snapshotdomain.ScreenshotSnapshot, 0, len(items))
	for index := range items {
		results = append(results, *screenshotSnapshotModelToDomain(&items[index]))
	}
	return results
}
