package application

const (
	defaultScanPage     = 1
	defaultScanPageSize = 20
)

type ScanListFilter struct {
	Page     int
	PageSize int
	TargetID int
	Status   string
	Search   string
}

type ScanListQuery struct {
	Page     int
	PageSize int
	TargetID int
	Status   string
	Search   string
}

func (query *ScanListQuery) normalize() (page, pageSize, targetID int, status, search string) {
	if query == nil {
		return defaultScanPage, defaultScanPageSize, 0, "", ""
	}
	page = query.Page
	if page <= 0 {
		page = defaultScanPage
	}
	pageSize = query.PageSize
	if pageSize <= 0 {
		pageSize = defaultScanPageSize
	}
	return page, pageSize, query.TargetID, query.Status, query.Search
}
