package application

import (
	"context"
	"fmt"
	"strings"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var websiteFilterOptionFields = map[string]struct{}{
	"statusCode":  {},
	"tech":        {},
	"webserver":   {},
	"contentType": {},
	"vhost":       {},
}

var directoryFilterOptionFields = map[string]struct{}{
	"status":      {},
	"contentType": {},
}

func normalizeWebsiteFilterOptionField(field string) (string, error) {
	trimmed := strings.TrimSpace(field)
	if _, ok := websiteFilterOptionFields[trimmed]; !ok {
		return "", fmt.Errorf("%w: unsupported filter option field", ErrUnsupportedWebsiteFilter)
	}
	return trimmed, nil
}

func normalizeDirectoryFilterOptionField(field string) (string, error) {
	trimmed := strings.TrimSpace(field)
	if _, ok := directoryFilterOptionFields[trimmed]; !ok {
		return "", fmt.Errorf("%w: unsupported filter option field", ErrUnsupportedDirectoryFilter)
	}
	return trimmed, nil
}

func (service *WebsiteQueryService) ListFilterOptionsByTarget(ctx context.Context, targetID int, field string) ([]assetdomain.FilterOption, error) {
	_ = ctx
	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	normalizedField, err := normalizeWebsiteFilterOptionField(field)
	if err != nil {
		return nil, err
	}
	return service.store.ListFilterOptionsByTargetID(targetID, normalizedField)
}

func (service *DirectoryQueryService) ListFilterOptionsByTarget(ctx context.Context, targetID int, field string) ([]assetdomain.FilterOption, error) {
	_ = ctx
	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	normalizedField, err := normalizeDirectoryFilterOptionField(field)
	if err != nil {
		return nil, err
	}
	return service.store.ListFilterOptionsByTargetID(targetID, normalizedField)
}

func (service *HostPortQueryService) ListPortOptionsByTarget(ctx context.Context, targetID int) ([]assetdomain.FilterOption, error) {
	_ = ctx
	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}
	return service.store.ListPortOptionsByTargetID(targetID)
}
