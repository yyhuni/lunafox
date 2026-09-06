package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type WordlistQueryStore interface {
	List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Wordlist, int64, error)
	ListAll() ([]catalogdomain.Wordlist, error)
	ListTagSummaries(page, pageSize int, filter string) ([]catalogdomain.WordlistTagSummary, int64, error)
	GetByID(id int) (*catalogdomain.Wordlist, error)
	GetByIDContext(ctx context.Context, id int) (*catalogdomain.Wordlist, error)
	Update(wordlist *catalogdomain.Wordlist) error
}

// WordlistQueryStoreContext is the request-aware extension used by MCP.
// Keeping it optional preserves compatibility with small legacy test stores.
type WordlistQueryStoreContext interface {
	ListContext(context.Context, int, int, string, string) ([]catalogdomain.Wordlist, int64, error)
	ListAllContext(context.Context) ([]catalogdomain.Wordlist, error)
}
