package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type HostPortItem struct {
	Host string
	IP   string
	Port int
}

type HostPortCommandService struct {
	store        HostPortCommandStore
	targetLookup HostPortTargetLookup
}

func NewHostPortCommandService(store HostPortCommandStore, targetLookup HostPortTargetLookup) *HostPortCommandService {
	return &HostPortCommandService{store: store, targetLookup: targetLookup}
}

func (service *HostPortCommandService) BulkUpsert(ctx context.Context, targetID int, items []HostPortItem) (int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	mappings := make([]assetdomain.HostPort, 0, len(items))
	for _, item := range items {
		mappings = append(mappings, assetdomain.HostPort{
			TargetID: targetID,
			Host:     item.Host,
			IP:       item.IP,
			Port:     item.Port,
		})
	}

	if len(mappings) == 0 {
		return 0, nil
	}

	return service.store.BulkUpsert(mappings)
}

func (service *HostPortCommandService) BulkDeleteByIPs(ctx context.Context, ips []string) (int64, error) {
	_ = ctx

	if len(ips) == 0 {
		return 0, nil
	}

	return service.store.DeleteByIPs(ips)
}
