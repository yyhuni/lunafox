package model

import (
	"time"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Scan is a local projection used by snapshot relations.
type SnapshotScanRef struct {
	ID int `gorm:"primaryKey;autoIncrement" json:"id"`
}

func (SnapshotScanRef) TableName() string {
	return "scan"
}

// DirectorySnapshot represents a directory snapshot.
type DirectorySnapshot struct {
	ID            int              `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID        int              `gorm:"column:scan_id;not null;index:idx_directory_snap_scan;uniqueIndex:unique_directory_per_scan_snapshot,priority:1" json:"scanId"`
	URL           string           `gorm:"column:url;size:2000;index:idx_directory_snap_url;uniqueIndex:unique_directory_per_scan_snapshot,priority:2" json:"url"`
	Status        *int             `gorm:"column:status;index:idx_directory_snap_status" json:"status"`
	ContentLength *int             `gorm:"column:content_length" json:"contentLength"`
	ContentType   string           `gorm:"column:content_type;size:200;index:idx_directory_snap_content_type" json:"contentType"`
	Duration      *int             `gorm:"column:duration" json:"duration"`
	CreatedAt     time.Time        `gorm:"column:created_at;autoCreateTime;index:idx_directory_snap_created_at" json:"createdAt"`
	Scan          *SnapshotScanRef `gorm:"foreignKey:ScanID" json:"scan,omitempty"`
}

func (DirectorySnapshot) TableName() string {
	return "directory_snapshot"
}

// EndpointSnapshot represents an endpoint snapshot.
type EndpointSnapshot struct {
	ID                int              `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID            int              `gorm:"column:scan_id;not null;index:idx_endpoint_snap_scan;uniqueIndex:unique_endpoint_per_scan_snapshot,priority:1" json:"scanId"`
	URL               string           `gorm:"column:url;type:text;index:idx_endpoint_snap_url;uniqueIndex:unique_endpoint_per_scan_snapshot,priority:2" json:"url"`
	Host              string           `gorm:"column:host;size:253;index:idx_endpoint_snap_host" json:"host"`
	Title             string           `gorm:"column:title;type:text;index:idx_endpoint_snap_title" json:"title"`
	StatusCode        *int             `gorm:"column:status_code;index:idx_endpoint_snap_status_code" json:"statusCode"`
	ContentLength     *int             `gorm:"column:content_length" json:"contentLength"`
	Location          string           `gorm:"column:location;type:text" json:"location"`
	Webserver         string           `gorm:"column:webserver;type:text;index:idx_endpoint_snap_webserver" json:"webserver"`
	ContentType       string           `gorm:"column:content_type;type:text" json:"contentType"`
	Tech              pq.StringArray   `gorm:"column:tech;type:varchar(100)[]" json:"tech"`
	ResponseBody      string           `gorm:"column:response_body;type:text" json:"responseBody"`
	Vhost             *bool            `gorm:"column:vhost" json:"vhost"`
	ResponseHeaders   string           `gorm:"column:response_headers;type:text" json:"responseHeaders"`
	CreatedAt         time.Time        `gorm:"column:created_at;autoCreateTime;index:idx_endpoint_snap_created_at" json:"createdAt"`
	Scan              *SnapshotScanRef `gorm:"foreignKey:ScanID" json:"scan,omitempty"`
}

func (EndpointSnapshot) TableName() string {
	return "endpoint_snapshot"
}

func (e *EndpointSnapshot) BeforeCreate(tx *gorm.DB) error {
	if e.Tech == nil {
		e.Tech = []string{}
	}
	return nil
}

// HostPortSnapshot represents a host-port snapshot.
type HostPortSnapshot struct {
	ID        int              `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID    int              `gorm:"column:scan_id;not null;index:idx_hpm_snap_scan;uniqueIndex:unique_scan_host_ip_port_snapshot,priority:1" json:"scanId"`
	Host      string           `gorm:"column:host;size:1000;not null;index:idx_hpm_snap_host;uniqueIndex:unique_scan_host_ip_port_snapshot,priority:2" json:"host"`
	IP        string           `gorm:"column:ip;type:inet;not null;index:idx_hpm_snap_ip;uniqueIndex:unique_scan_host_ip_port_snapshot,priority:3" json:"ip"`
	Port      int              `gorm:"column:port;not null;index:idx_hpm_snap_port;uniqueIndex:unique_scan_host_ip_port_snapshot,priority:4" json:"port"`
	CreatedAt time.Time        `gorm:"column:created_at;autoCreateTime;index:idx_hpm_snap_created_at" json:"createdAt"`
	Scan      *SnapshotScanRef `gorm:"foreignKey:ScanID" json:"scan,omitempty"`
}

func (HostPortSnapshot) TableName() string {
	return "host_port_mapping_snapshot"
}

// ScreenshotSnapshot represents a screenshot snapshot.
type ScreenshotSnapshot struct {
	ID         int              `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID     int              `gorm:"column:scan_id;not null;index:idx_screenshot_snap_scan;uniqueIndex:unique_screenshot_per_scan_snapshot,priority:1" json:"scanId"`
	URL        string           `gorm:"column:url;type:text;uniqueIndex:unique_screenshot_per_scan_snapshot,priority:2" json:"url"`
	StatusCode *int16           `gorm:"column:status_code" json:"statusCode"`
	Image      []byte           `gorm:"column:image;type:bytea" json:"-"`
	CreatedAt  time.Time        `gorm:"column:created_at;autoCreateTime;index:idx_screenshot_snap_created_at" json:"createdAt"`
	Scan       *SnapshotScanRef `gorm:"foreignKey:ScanID" json:"scan,omitempty"`
}

func (ScreenshotSnapshot) TableName() string {
	return "screenshot_snapshot"
}

// SubdomainSnapshot represents a subdomain snapshot.
type SubdomainSnapshot struct {
	ID        int              `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID    int              `gorm:"column:scan_id;not null;index:idx_subdomain_snap_scan;uniqueIndex:unique_subdomain_per_scan_snapshot,priority:1" json:"scanId"`
	Name      string           `gorm:"column:name;size:1000;index:idx_subdomain_snap_name;uniqueIndex:unique_subdomain_per_scan_snapshot,priority:2" json:"name"`
	CreatedAt time.Time        `gorm:"column:created_at;autoCreateTime;index:idx_subdomain_snap_created_at" json:"createdAt"`
	Scan      *SnapshotScanRef `gorm:"foreignKey:ScanID" json:"scan,omitempty"`
}

func (SubdomainSnapshot) TableName() string {
	return "subdomain_snapshot"
}

// WebsiteSnapshot represents a website snapshot.
type WebsiteSnapshot struct {
	ID              int              `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID          int              `gorm:"column:scan_id;not null;index:idx_website_snap_scan;uniqueIndex:unique_website_per_scan_snapshot,priority:1" json:"scanId"`
	URL             string           `gorm:"column:url;type:text;index:idx_website_snap_url;uniqueIndex:unique_website_per_scan_snapshot,priority:2" json:"url"`
	Host            string           `gorm:"column:host;size:253;index:idx_website_snap_host" json:"host"`
	Title           string           `gorm:"column:title;type:text;index:idx_website_snap_title" json:"title"`
	StatusCode      *int             `gorm:"column:status_code" json:"statusCode"`
	ContentLength   *int             `gorm:"column:content_length" json:"contentLength"`
	Location        string           `gorm:"column:location;type:text" json:"location"`
	Webserver       string           `gorm:"column:webserver;type:text" json:"webserver"`
	ContentType     string           `gorm:"column:content_type;type:text" json:"contentType"`
	Tech            pq.StringArray   `gorm:"column:tech;type:varchar(100)[]" json:"tech"`
	ResponseBody    string           `gorm:"column:response_body;type:text" json:"responseBody"`
	Vhost           *bool            `gorm:"column:vhost" json:"vhost"`
	ResponseHeaders string           `gorm:"column:response_headers;type:text" json:"responseHeaders"`
	CreatedAt       time.Time        `gorm:"column:created_at;autoCreateTime;index:idx_website_snap_created_at" json:"createdAt"`
	Scan            *SnapshotScanRef `gorm:"foreignKey:ScanID" json:"scan,omitempty"`
}

func (WebsiteSnapshot) TableName() string {
	return "website_snapshot"
}

func (w *WebsiteSnapshot) BeforeCreate(tx *gorm.DB) error {
	if w.Tech == nil {
		w.Tech = []string{}
	}
	return nil
}

// VulnerabilitySnapshot represents a vulnerability snapshot.
type VulnerabilitySnapshot struct {
	ID          int              `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID      int              `gorm:"column:scan_id;not null;index:idx_vuln_snap_scan" json:"scanId"`
	URL         string           `gorm:"column:url;size:2000;index:idx_vuln_snap_url" json:"url"`
	VulnType    string           `gorm:"column:vuln_type;size:100;index:idx_vuln_snap_type" json:"vulnType"`
	Severity    string           `gorm:"column:severity;size:20;default:'unknown';index:idx_vuln_snap_severity" json:"severity"`
	Source      string           `gorm:"column:source;size:50;index:idx_vuln_snap_source" json:"source"`
	CVSSScore   *decimal.Decimal `gorm:"column:cvss_score;type:decimal(3,1);default:0.0" json:"cvssScore"`
	Description string           `gorm:"column:description;type:text" json:"description"`
	RawOutput   datatypes.JSON   `gorm:"column:raw_output;type:jsonb" json:"rawOutput"`
	CreatedAt   time.Time        `gorm:"column:created_at;autoCreateTime;index:idx_vuln_snap_created_at" json:"createdAt"`
	Scan        *SnapshotScanRef `gorm:"foreignKey:ScanID" json:"scan,omitempty"`
}

func (VulnerabilitySnapshot) TableName() string {
	return "vulnerability_snapshot"
}

func (v *VulnerabilitySnapshot) BeforeCreate(tx *gorm.DB) error {
	if v.RawOutput == nil {
		v.RawOutput = datatypes.JSON([]byte("{}"))
	}
	return nil
}
