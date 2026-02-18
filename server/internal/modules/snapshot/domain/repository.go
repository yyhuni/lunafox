package domain

import "database/sql"

type ScanRefLookup interface {
	GetScanRefByID(id int) (*ScanRef, error)
	GetTargetRefByScanID(scanID int) (*ScanTargetRef, error)
}

type WebsiteQueryStore interface {
	FindByScanID(scanID int, page, pageSize int, filter string) ([]WebsiteSnapshot, int64, error)
	StreamByScanID(scanID int) (*sql.Rows, error)
	CountByScanID(scanID int) (int64, error)
	ScanRow(rows *sql.Rows) (*WebsiteSnapshot, error)
}

type WebsiteCommandStore interface {
	BulkCreate(snapshots []WebsiteSnapshot) (int64, error)
}

type EndpointQueryStore interface {
	FindByScanID(scanID int, page, pageSize int, filter string) ([]EndpointSnapshot, int64, error)
	StreamByScanID(scanID int) (*sql.Rows, error)
	CountByScanID(scanID int) (int64, error)
	ScanRow(rows *sql.Rows) (*EndpointSnapshot, error)
}

type EndpointCommandStore interface {
	BulkCreate(snapshots []EndpointSnapshot) (int64, error)
}

type DirectoryQueryStore interface {
	FindByScanID(scanID int, page, pageSize int, filter string) ([]DirectorySnapshot, int64, error)
	StreamByScanID(scanID int) (*sql.Rows, error)
	CountByScanID(scanID int) (int64, error)
	ScanRow(rows *sql.Rows) (*DirectorySnapshot, error)
}

type DirectoryCommandStore interface {
	BulkCreate(snapshots []DirectorySnapshot) (int64, error)
}

type SubdomainQueryStore interface {
	FindByScanID(scanID int, page, pageSize int, filter string) ([]SubdomainSnapshot, int64, error)
	StreamByScanID(scanID int) (*sql.Rows, error)
	CountByScanID(scanID int) (int64, error)
	ScanRow(rows *sql.Rows) (*SubdomainSnapshot, error)
}

type SubdomainCommandStore interface {
	BulkCreate(snapshots []SubdomainSnapshot) (int64, error)
}

type HostPortQueryStore interface {
	FindByScanID(scanID int, page, pageSize int, filter string) ([]HostPortSnapshot, int64, error)
	StreamByScanID(scanID int) (*sql.Rows, error)
	CountByScanID(scanID int) (int64, error)
	ScanRow(rows *sql.Rows) (*HostPortSnapshot, error)
}

type HostPortCommandStore interface {
	BulkCreate(snapshots []HostPortSnapshot) (int64, error)
}

type ScreenshotQueryStore interface {
	FindByScanID(scanID int, page, pageSize int, filter string) ([]ScreenshotSnapshot, int64, error)
	FindByIDAndScanID(id int, scanID int) (*ScreenshotSnapshot, error)
}

type ScreenshotCommandStore interface {
	BulkUpsert(snapshots []ScreenshotSnapshot) (int64, error)
}

type VulnerabilitySnapshotRepository interface {
	FindByScanID(scanID int, page, pageSize int, filter, severity, ordering string) ([]VulnerabilitySnapshot, int64, error)
	FindAll(page, pageSize int, filter, severity, ordering string) ([]VulnerabilitySnapshot, int64, error)
	GetByID(id int) (*VulnerabilitySnapshot, error)
	StreamByScanID(scanID int) (*sql.Rows, error)
	CountByScanID(scanID int) (int64, error)
	ScanRow(rows *sql.Rows) (*VulnerabilitySnapshot, error)
	BulkCreate(snapshots []VulnerabilitySnapshot) (int64, error)
}
