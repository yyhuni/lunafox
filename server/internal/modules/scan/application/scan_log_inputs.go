package application

const (
	defaultScanLogLimit = 200
	maxScanLogLimit     = 1000
)

type ScanLogCreateItem struct {
	Level   string
	Content string
}

type ScanLogListQuery struct {
	AfterID int64
	Limit   int
}

func (query *ScanLogListQuery) normalize() (afterID int64, limit int) {
	if query == nil {
		return 0, defaultScanLogLimit
	}
	afterID = query.AfterID
	if afterID < 0 {
		afterID = 0
	}
	limit = query.Limit
	if limit <= 0 {
		limit = defaultScanLogLimit
	}
	if limit > maxScanLogLimit {
		limit = maxScanLogLimit
	}
	return afterID, limit
}

type ScanLogBulkCreateRequest struct {
	ScanID int
	Items  []ScanLogCreateItem
}

func (request *ScanLogBulkCreateRequest) normalize() (scanID int, items []ScanLogCreateItem) {
	if request == nil {
		return 0, nil
	}
	return request.ScanID, request.Items
}
