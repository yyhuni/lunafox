package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type HostPortCommandStore interface {
	BatchUpsertContext(context.Context, []assetdomain.HostPort) (int64, error)
	DeleteByIPs(ips []string) (int64, error)
}

type HostPortStore interface {
	HostPortQueryStore
	HostPortCommandStore
}
