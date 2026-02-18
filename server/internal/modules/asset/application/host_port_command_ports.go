package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type HostPortCommandStore interface {
	BulkUpsert(mappings []assetdomain.HostPort) (int64, error)
	DeleteByIPs(ips []string) (int64, error)
}

type HostPortStore interface {
	HostPortQueryStore
	HostPortCommandStore
}
