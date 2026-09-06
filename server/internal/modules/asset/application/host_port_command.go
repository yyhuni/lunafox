package application

import (
	"context"
	"net"
	"strings"

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
	targetLookup AssetCommandTargetLookup
}

func NewHostPortCommandService(store HostPortCommandStore, targetLookup AssetCommandTargetLookup) *HostPortCommandService {
	return &HostPortCommandService{store: store, targetLookup: targetLookup}
}

func (service *HostPortCommandService) BatchUpsert(ctx context.Context, targetID int, items []HostPortItem) (int64, error) {
	if ctx == nil {
		return 0, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if _, err := service.targetLookup.GetActiveByIDContext(ctx, targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	mappings := make([]assetdomain.HostPort, 0, len(items))
	for _, item := range items {
		// Host-port assets are currently IPv4-only; skip IPv6 without failing mixed result batches.
		if !isIPv4HostPortAddress(item.IP) {
			continue
		}
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

	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return service.store.BatchUpsertContext(ctx, mappings)
}

func isIPv4HostPortAddress(value string) bool {
	parsed := net.ParseIP(strings.TrimSpace(value))
	return parsed != nil && parsed.To4() != nil
}

func (service *HostPortCommandService) BatchDeleteByIPs(ctx context.Context, ips []string) (int64, error) {
	_ = ctx

	if len(ips) == 0 {
		return 0, nil
	}

	return service.store.DeleteByIPs(ips)
}
