package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type SubdomainCommandStore interface {
	BatchCreateContext(context.Context, []assetdomain.Subdomain) (int, error)
	BatchDelete(ids []int) (int64, error)
}

type SubdomainStore interface {
	SubdomainQueryStore
	SubdomainCommandStore
}
