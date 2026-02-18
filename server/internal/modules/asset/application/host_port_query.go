package application

import (
	"context"
	"database/sql"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type HostPortResponse struct {
	IP        string
	Hosts     []string
	Ports     []int
	CreatedAt time.Time
}

type HostPortQueryService struct {
	store        HostPortQueryStore
	targetLookup HostPortTargetLookup
}

func NewHostPortQueryService(store HostPortQueryStore, targetLookup HostPortTargetLookup) *HostPortQueryService {
	return &HostPortQueryService{store: store, targetLookup: targetLookup}
}

func (service *HostPortQueryService) ListByTarget(ctx context.Context, targetID, page, pageSize int, filter string) ([]HostPortResponse, int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrTargetNotFound
		}
		return nil, 0, err
	}

	ipRows, total, err := service.store.GetIPAggregation(targetID, page, pageSize, filter)
	if err != nil {
		return nil, 0, err
	}

	results := make([]HostPortResponse, 0, len(ipRows))
	for _, row := range ipRows {
		hosts, ports, err := service.store.GetHostsAndPortsByIP(targetID, row.IP, filter)
		if err != nil {
			return nil, 0, err
		}

		results = append(results, HostPortResponse{
			IP:        row.IP,
			Hosts:     hosts,
			Ports:     ports,
			CreatedAt: timeutil.ToUTC(row.CreatedAt),
		})
	}

	return results, total, nil
}

func (service *HostPortQueryService) StreamByTarget(ctx context.Context, targetID int) (*sql.Rows, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	return service.store.StreamByTargetID(targetID)
}

func (service *HostPortQueryService) StreamByTargetAndIPs(ctx context.Context, targetID int, ips []string) (*sql.Rows, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}

	return service.store.StreamByTargetIDAndIPs(targetID, ips)
}

func (service *HostPortQueryService) CountByTarget(ctx context.Context, targetID int) (int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	return service.store.CountByTargetID(targetID)
}

func (service *HostPortQueryService) ScanRow(ctx context.Context, rows *sql.Rows) (*assetdomain.HostPort, error) {
	_ = ctx
	return service.store.ScanRow(rows)
}
