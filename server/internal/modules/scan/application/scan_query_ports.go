package application

import "context"

type ScanQueryStore interface {
	List(page, pageSize int, targetID int, status, filter, orderBy string) ([]QueryScan, int64, error)
	GetDetailByID(id int) (*QueryScan, error)
	GetGlobalStatsSummary() (*QueryStatistics, error)
}

// ScanQueryStoreContext carries MCP and other request-scoped cancellation into
// production query repositories without widening legacy test stores.
type ScanQueryStoreContext interface {
	ListContext(context.Context, int, int, int, string, string, string) ([]QueryScan, int64, error)
	GetDetailByIDContext(context.Context, int) (*QueryScan, error)
}
