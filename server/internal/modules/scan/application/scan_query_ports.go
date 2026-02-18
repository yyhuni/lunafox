package application

type ScanQueryStore interface {
	FindAll(page, pageSize int, targetID int, status, search string) ([]QueryScan, int64, error)
	GetQueryByID(id int) (*QueryScan, error)
	GetStatistics() (*QueryStatistics, error)
}
