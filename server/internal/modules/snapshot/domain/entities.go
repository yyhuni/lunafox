package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

// ScanRef is the minimal scan projection required by snapshot context.
type ScanRef struct {
	ID       int
	TargetID int
}

// ScanTargetRef is the minimal target projection required by snapshot context.
type ScanTargetRef struct {
	ID        int
	Name      string
	Type      string
	CreatedAt time.Time
}

type WebsiteSnapshot struct {
	ID              int
	ScanID          int
	URL             string
	Host            string
	Title           string
	StatusCode      *int
	ContentLength   *int
	Location        string
	Webserver       string
	ContentType     string
	Tech            []string
	ResponseBody    string
	Vhost           *bool
	ResponseHeaders string
	CreatedAt       time.Time
}

type EndpointSnapshot struct {
	ID                       int
	ScanID                   int
	URL                      string
	Host                     string
	Title                    string
	StatusCode               *int
	ContentLength            *int
	Location                 string
	Webserver                string
	ContentType              string
	Tech                     []string
	ResponseBody             string
	ResponseBodyTruncated    bool
	Vhost                    *bool
	ResponseHeaders          string
	ResponseHeadersTruncated bool
	CreatedAt                time.Time
}

type DirectorySnapshot struct {
	ID            int
	ScanID        int
	URL           string
	Status        *int
	ContentLength *int64
	ContentType   string
	Duration      *int64
	CreatedAt     time.Time
}

type SubdomainSnapshot struct {
	ID        int
	ScanID    int
	DNSName   string
	CreatedAt time.Time
}

type HostPortSnapshot struct {
	ID        int
	ScanID    int
	Host      string
	IP        string
	Port      int
	CreatedAt time.Time
}

// HostPortInputEvidence is the complete finalized HostPort projection used by
// the hostPorts input product. It is not a persisted business field.
type HostPortInputEvidence struct {
	Host string
	IP   string
	Port int
}

type HostPortIPAggregationRow struct {
	IP        string
	CreatedAt time.Time
}

type FilterOption struct {
	Value string
	Label string
	Count int64
}

type ScreenshotSnapshot struct {
	ID         int
	ScanID     int
	URL        string
	StatusCode *int16
	Image      []byte
	CreatedAt  time.Time
}

type VulnerabilitySnapshot struct {
	ID          int
	ScanID      int
	URL         string
	VulnType    string
	Severity    string
	Source      string
	CVSSScore   *decimal.Decimal
	Description string
	RawOutput   datatypes.JSON
	CreatedAt   time.Time
}
