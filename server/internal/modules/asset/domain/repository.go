package domain

import "context"

type TargetLookup interface {
	GetActiveByID(id int) (*TargetRef, error)
}

type WebsiteQueryStore interface {
	ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]Website, int64, error)
	ForEachByTargetID(ctx context.Context, targetID int, visit func(Website) error) error
	CountByTargetID(targetID int) (int64, error)
}

type WebsiteCommandStore interface {
	GetByID(id int) (*Website, error)
	BatchCreate(websites []Website) (int, error)
	Delete(id int) error
	BatchDelete(ids []int) (int64, error)
	BatchUpsert(websites []Website) (int64, error)
}

type EndpointQueryStore interface {
	ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]Endpoint, int64, error)
	ListFilterOptionsByTargetID(targetID int, field string) ([]FilterOption, error)
	GetByID(id int) (*Endpoint, error)
	ForEachByTargetID(ctx context.Context, targetID int, visit func(Endpoint) error) error
	CountByTargetID(targetID int) (int64, error)
}

type EndpointCommandStore interface {
	GetByID(id int) (*Endpoint, error)
	BatchCreate(endpoints []Endpoint) (int, error)
	Delete(id int) error
	BatchDelete(ids []int) (int64, error)
	BatchUpsert(endpoints []Endpoint) (int64, error)
}

type DirectoryQueryStore interface {
	ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]Directory, int64, error)
	ListFilterOptionsByTargetID(targetID int, field string) ([]FilterOption, error)
	ForEachByTargetID(ctx context.Context, targetID int, visit func(Directory) error) error
	CountByTargetID(targetID int) (int64, error)
}

type DirectoryCommandStore interface {
	BatchCreate(directories []Directory) (int, error)
	BatchDelete(ids []int) (int64, error)
	BatchUpsert(directories []Directory) (int64, error)
}

type SubdomainQueryStore interface {
	ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]Subdomain, int64, error)
	ForEachByTargetID(ctx context.Context, targetID int, visit func(Subdomain) error) error
	CountByTargetID(targetID int) (int64, error)
}

type SubdomainCommandStore interface {
	BatchCreate(subdomains []Subdomain) (int, error)
	BatchDelete(ids []int) (int64, error)
}

type HostPortQueryStore interface {
	GetIPAggregation(targetID int, page, pageSize int, filter, orderBy string) ([]IPAggregationRow, int64, error)
	GetHostsAndPortsByIP(targetID int, ip string, filter string) ([]string, []int, error)
	ForEachByTargetID(ctx context.Context, targetID int, visit func(HostPort) error) error
	ForEachByTargetIDAndIPs(ctx context.Context, targetID int, ips []string, visit func(HostPort) error) error
	CountByTargetID(targetID int) (int64, error)
}

type HostPortCommandStore interface {
	BatchUpsert(mappings []HostPort) (int64, error)
	DeleteByIPs(ips []string) (int64, error)
}

type ScreenshotQueryStore interface {
	ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]Screenshot, int64, error)
	ListSummariesByTargetAndURLs(targetID int, urls []string) ([]Screenshot, error)
	ListFilterOptionsByTargetID(targetID int, field string) ([]FilterOption, error)
	GetByID(id int) (*Screenshot, error)
}

type ScreenshotCommandStore interface {
	BatchDelete(ids []int) (int64, error)
	BatchUpsert(screenshots []Screenshot) (int64, error)
}
