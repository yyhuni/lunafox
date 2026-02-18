package application

const (
	defaultSnapshotPage     = 1
	defaultSnapshotPageSize = 20
)

type SnapshotListQuery struct {
	Page     int
	PageSize int
	Filter   string
}

func (query *SnapshotListQuery) normalize() (page, pageSize int, filter string) {
	if query == nil {
		return defaultSnapshotPage, defaultSnapshotPageSize, ""
	}
	page = query.Page
	if page <= 0 {
		page = defaultSnapshotPage
	}
	pageSize = query.PageSize
	if pageSize <= 0 {
		pageSize = defaultSnapshotPageSize
	}
	return page, pageSize, query.Filter
}
