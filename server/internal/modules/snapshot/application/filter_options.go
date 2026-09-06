package application

import (
	"context"
	"fmt"
	"strings"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var websiteSnapshotFilterOptionFields = map[string]struct{}{
	"statusCode":  {},
	"tech":        {},
	"webserver":   {},
	"contentType": {},
	"vhost":       {},
}

var directorySnapshotFilterOptionFields = map[string]struct{}{
	"status":      {},
	"contentType": {},
}

func normalizeWebsiteSnapshotFilterOptionField(field string) (string, error) {
	trimmed := strings.TrimSpace(field)
	if _, ok := websiteSnapshotFilterOptionFields[trimmed]; !ok {
		return "", fmt.Errorf("%w: unsupported filter option field", ErrUnsupportedWebsiteSnapshotFilter)
	}
	return trimmed, nil
}

func normalizeDirectorySnapshotFilterOptionField(field string) (string, error) {
	trimmed := strings.TrimSpace(field)
	if _, ok := directorySnapshotFilterOptionFields[trimmed]; !ok {
		return "", fmt.Errorf("%w: unsupported filter option field", ErrUnsupportedDirectorySnapshotFilter)
	}
	return trimmed, nil
}

func (service *WebsiteSnapshotQueryService) ListFilterOptionsByScan(ctx context.Context, scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}

	normalizedField, err := normalizeWebsiteSnapshotFilterOptionField(field)
	if err != nil {
		return nil, err
	}
	return service.store.ListFilterOptionsByScanID(scanID, normalizedField)
}

func (service *DirectorySnapshotQueryService) ListFilterOptionsByScan(ctx context.Context, scanID int, field string) ([]snapshotdomain.FilterOption, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}

	normalizedField, err := normalizeDirectorySnapshotFilterOptionField(field)
	if err != nil {
		return nil, err
	}
	return service.store.ListFilterOptionsByScanID(scanID, normalizedField)
}

func (service *HostPortSnapshotQueryService) ListPortOptionsByScan(ctx context.Context, scanID int) ([]snapshotdomain.FilterOption, error) {
	_ = ctx
	if _, err := service.scanLookup.GetScanRefByID(scanID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrSnapshotScanNotFound
		}
		return nil, err
	}
	return service.store.ListPortOptionsByScanID(scanID)
}
