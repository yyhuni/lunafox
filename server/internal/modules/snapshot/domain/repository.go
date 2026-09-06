package domain

import "context"

type ScanRefLookup interface {
	GetScanRefByID(id int) (*ScanRef, error)
	GetTargetRefByScanID(scanID int) (*ScanTargetRef, error)
}

type WebsiteQueryStore interface {
	ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]WebsiteSnapshot, int64, error)
	ForEachByScanID(ctx context.Context, scanID int, visit func(WebsiteSnapshot) error) error
	CountByScanID(scanID int) (int64, error)
}

type WebsiteCommandStore interface {
	BatchCreate(snapshots []WebsiteSnapshot) (int64, error)
}

type EndpointQueryStore interface {
	ListByScanID(scanID int, page, pageSize int, filter string) ([]EndpointSnapshot, int64, error)
	ForEachByScanID(ctx context.Context, scanID int, visit func(EndpointSnapshot) error) error
	CountByScanID(scanID int) (int64, error)
}

type EndpointCommandStore interface {
	BatchCreate(snapshots []EndpointSnapshot) (int64, error)
}

type DirectoryQueryStore interface {
	ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]DirectorySnapshot, int64, error)
	ListFilterOptionsByScanID(scanID int, field string) ([]FilterOption, error)
	ForEachByScanID(ctx context.Context, scanID int, visit func(DirectorySnapshot) error) error
	CountByScanID(scanID int) (int64, error)
}

type DirectoryCommandStore interface {
	BatchCreate(snapshots []DirectorySnapshot) (int64, error)
}

type SubdomainQueryStore interface {
	ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]SubdomainSnapshot, int64, error)
	ForEachByScanID(ctx context.Context, scanID int, visit func(SubdomainSnapshot) error) error
	CountByScanID(scanID int) (int64, error)
}

type SubdomainCommandStore interface {
	BatchCreate(snapshots []SubdomainSnapshot) (int64, error)
}

type HostPortQueryStore interface {
	GetIPAggregation(scanID int, page, pageSize int, filter, orderBy string) ([]HostPortIPAggregationRow, int64, error)
	GetHostsAndPortsByIP(scanID int, ip string, filter string) ([]string, []int, error)
	ForEachByScanID(ctx context.Context, scanID int, visit func(HostPortSnapshot) error) error
	CountByScanID(scanID int) (int64, error)
}

type HostPortCommandStore interface {
	BatchCreate(snapshots []HostPortSnapshot) (int64, error)
}

type ScreenshotQueryStore interface {
	ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]ScreenshotSnapshot, int64, error)
	ListFilterOptionsByScanID(scanID int, field string) ([]FilterOption, error)
	FindByIDAndScanID(id int, scanID int) (*ScreenshotSnapshot, error)
}

type ScreenshotCommandStore interface {
	BatchUpsert(snapshots []ScreenshotSnapshot) (int64, error)
}

type VulnerabilitySnapshotRepository interface {
	ListByScanID(scanID int, page, pageSize int, filter, severity, ordering string) ([]VulnerabilitySnapshot, int64, error)
	List(page, pageSize int, filter, severity, ordering string) ([]VulnerabilitySnapshot, int64, error)
	GetByID(id int) (*VulnerabilitySnapshot, error)
	ForEachByScanID(ctx context.Context, scanID int, visit func(VulnerabilitySnapshot) error) error
	CountByScanID(scanID int) (int64, error)
	BatchCreate(snapshots []VulnerabilitySnapshot) (int64, error)
}
