package domain

import "database/sql"

type TargetLookup interface {
	GetActiveByID(id int) (*TargetRef, error)
}

type WebsiteQueryStore interface {
	FindByTargetID(targetID int, page, pageSize int, filter string) ([]Website, int64, error)
	StreamByTargetID(targetID int) (*sql.Rows, error)
	CountByTargetID(targetID int) (int64, error)
	ScanRow(rows *sql.Rows) (*Website, error)
}

type WebsiteCommandStore interface {
	GetByID(id int) (*Website, error)
	BulkCreate(websites []Website) (int, error)
	Delete(id int) error
	BulkDelete(ids []int) (int64, error)
	BulkUpsert(websites []Website) (int64, error)
}

type EndpointQueryStore interface {
	FindByTargetID(targetID int, page, pageSize int, filter string) ([]Endpoint, int64, error)
	GetByID(id int) (*Endpoint, error)
	StreamByTargetID(targetID int) (*sql.Rows, error)
	CountByTargetID(targetID int) (int64, error)
	ScanRow(rows *sql.Rows) (*Endpoint, error)
}

type EndpointCommandStore interface {
	GetByID(id int) (*Endpoint, error)
	BulkCreate(endpoints []Endpoint) (int, error)
	Delete(id int) error
	BulkDelete(ids []int) (int64, error)
	BulkUpsert(endpoints []Endpoint) (int64, error)
}

type DirectoryQueryStore interface {
	FindByTargetID(targetID int, page, pageSize int, filter string) ([]Directory, int64, error)
	StreamByTargetID(targetID int) (*sql.Rows, error)
	CountByTargetID(targetID int) (int64, error)
	ScanRow(rows *sql.Rows) (*Directory, error)
}

type DirectoryCommandStore interface {
	BulkCreate(directories []Directory) (int, error)
	BulkDelete(ids []int) (int64, error)
	BulkUpsert(directories []Directory) (int64, error)
}

type SubdomainQueryStore interface {
	FindByTargetID(targetID int, page, pageSize int, filter string) ([]Subdomain, int64, error)
	StreamByTargetID(targetID int) (*sql.Rows, error)
	CountByTargetID(targetID int) (int64, error)
	ScanRow(rows *sql.Rows) (*Subdomain, error)
}

type SubdomainCommandStore interface {
	BulkCreate(subdomains []Subdomain) (int, error)
	BulkDelete(ids []int) (int64, error)
}

type HostPortQueryStore interface {
	GetIPAggregation(targetID int, page, pageSize int, filter string) ([]IPAggregationRow, int64, error)
	GetHostsAndPortsByIP(targetID int, ip string, filter string) ([]string, []int, error)
	StreamByTargetID(targetID int) (*sql.Rows, error)
	StreamByTargetIDAndIPs(targetID int, ips []string) (*sql.Rows, error)
	CountByTargetID(targetID int) (int64, error)
	ScanRow(rows *sql.Rows) (*HostPort, error)
}

type HostPortCommandStore interface {
	BulkUpsert(mappings []HostPort) (int64, error)
	DeleteByIPs(ips []string) (int64, error)
}

type ScreenshotQueryStore interface {
	FindByTargetID(targetID int, page, pageSize int, filter string) ([]Screenshot, int64, error)
	GetByID(id int) (*Screenshot, error)
}

type ScreenshotCommandStore interface {
	BulkDelete(ids []int) (int64, error)
	BulkUpsert(screenshots []Screenshot) (int64, error)
}
