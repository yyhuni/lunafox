package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type SubdomainCommandStore interface {
	BulkCreate(subdomains []assetdomain.Subdomain) (int, error)
	BulkDelete(ids []int) (int64, error)
}

type SubdomainStore interface {
	SubdomainQueryStore
	SubdomainCommandStore
}
