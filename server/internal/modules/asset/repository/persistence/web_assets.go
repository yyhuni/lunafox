package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Directory represents a directory asset.
type Directory struct {
	ID            int             `gorm:"primaryKey;autoIncrement" json:"id"`
	TargetID      int             `gorm:"column:target_id;not null;index:idx_directory_target;uniqueIndex:unique_directory_url_target,priority:1" json:"targetId"`
	URL           string          `gorm:"column:url;size:2000;not null;index:idx_directory_url;uniqueIndex:unique_directory_url_target,priority:2" json:"url"`
	Status        *int            `gorm:"column:status;index:idx_directory_status" json:"status"`
	ContentLength *int            `gorm:"column:content_length" json:"contentLength"`
	ContentType   string          `gorm:"column:content_type;size:200" json:"contentType"`
	Duration      *int            `gorm:"column:duration" json:"duration"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime;index:idx_directory_created_at" json:"createdAt"`
	Target        *AssetTargetRef `gorm:"foreignKey:TargetID" json:"target,omitempty"`
}

func (Directory) TableName() string {
	return "directory"
}

// Subdomain represents a subdomain asset.
type Subdomain struct {
	ID        int             `gorm:"primaryKey;autoIncrement" json:"id"`
	TargetID  int             `gorm:"column:target_id;not null;index:idx_subdomain_target;uniqueIndex:unique_subdomain_name_target,priority:2" json:"targetId"`
	Name      string          `gorm:"column:name;size:1000;index:idx_subdomain_name;uniqueIndex:unique_subdomain_name_target,priority:1" json:"name"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime;index:idx_subdomain_created_at" json:"createdAt"`
	Target    *AssetTargetRef `gorm:"foreignKey:TargetID" json:"target,omitempty"`
}

func (Subdomain) TableName() string {
	return "subdomain"
}

// Website represents a website asset.
type Website struct {
	ID              int             `gorm:"primaryKey;autoIncrement" json:"id"`
	TargetID        int             `gorm:"column:target_id;not null;index:idx_website_target;uniqueIndex:unique_website_url_target,priority:2" json:"targetId"`
	URL             string          `gorm:"column:url;type:text;index:idx_website_url;uniqueIndex:unique_website_url_target,priority:1" json:"url"`
	Host            string          `gorm:"column:host;size:253;index:idx_website_host" json:"host"`
	Location        string          `gorm:"column:location;type:text" json:"location"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime;index:idx_website_created_at" json:"createdAt"`
	Title           string          `gorm:"column:title;type:text;index:idx_website_title" json:"title"`
	Webserver       string          `gorm:"column:webserver;type:text" json:"webserver"`
	ResponseBody    string          `gorm:"column:response_body;type:text" json:"responseBody"`
	ContentType     string          `gorm:"column:content_type;type:text" json:"contentType"`
	Tech            pq.StringArray  `gorm:"column:tech;type:varchar(100)[]" json:"tech"`
	StatusCode      *int            `gorm:"column:status_code;index:idx_website_status_code" json:"statusCode"`
	ContentLength   *int            `gorm:"column:content_length" json:"contentLength"`
	Vhost           *bool           `gorm:"column:vhost" json:"vhost"`
	ResponseHeaders string          `gorm:"column:response_headers;type:text" json:"responseHeaders"`
	Target          *AssetTargetRef `gorm:"foreignKey:TargetID" json:"target,omitempty"`
}

func (Website) TableName() string {
	return "website"
}

func (w *Website) BeforeCreate(tx *gorm.DB) error {
	if w.Tech == nil {
		w.Tech = []string{}
	}
	return nil
}

// Endpoint represents an endpoint asset.
type Endpoint struct {
	ID                int             `gorm:"primaryKey;autoIncrement" json:"id"`
	TargetID          int             `gorm:"column:target_id;not null;index:idx_endpoint_target;uniqueIndex:unique_endpoint_url_target,priority:2" json:"targetId"`
	URL               string          `gorm:"column:url;type:text;index:idx_endpoint_url;uniqueIndex:unique_endpoint_url_target,priority:1" json:"url"`
	Host              string          `gorm:"column:host;size:253;index:idx_endpoint_host" json:"host"`
	Location          string          `gorm:"column:location;type:text" json:"location"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime;index:idx_endpoint_created_at" json:"createdAt"`
	Title             string          `gorm:"column:title;type:text;index:idx_endpoint_title" json:"title"`
	Webserver         string          `gorm:"column:webserver;type:text" json:"webserver"`
	ResponseBody      string          `gorm:"column:response_body;type:text" json:"responseBody"`
	ContentType       string          `gorm:"column:content_type;type:text" json:"contentType"`
	Tech              pq.StringArray  `gorm:"column:tech;type:varchar(100)[]" json:"tech"`
	StatusCode        *int            `gorm:"column:status_code;index:idx_endpoint_status_code" json:"statusCode"`
	ContentLength     *int            `gorm:"column:content_length" json:"contentLength"`
	Vhost             *bool           `gorm:"column:vhost" json:"vhost"`
	ResponseHeaders   string          `gorm:"column:response_headers;type:text" json:"responseHeaders"`
	Target            *AssetTargetRef `gorm:"foreignKey:TargetID" json:"target,omitempty"`
}

func (Endpoint) TableName() string {
	return "endpoint"
}

func (e *Endpoint) BeforeCreate(tx *gorm.DB) error {
	if e.Tech == nil {
		e.Tech = []string{}
	}
	return nil
}

// Screenshot represents a screenshot asset.
type Screenshot struct {
	ID         int             `gorm:"primaryKey;autoIncrement" json:"id"`
	TargetID   int             `gorm:"column:target_id;not null;index:idx_screenshot_target;uniqueIndex:unique_screenshot_per_target,priority:1" json:"targetId"`
	URL        string          `gorm:"column:url;type:text;uniqueIndex:unique_screenshot_per_target,priority:2" json:"url"`
	StatusCode *int16          `gorm:"column:status_code" json:"statusCode"`
	Image      []byte          `gorm:"column:image;type:bytea" json:"-"`
	CreatedAt  time.Time       `gorm:"column:created_at;autoCreateTime;index:idx_screenshot_created_at" json:"createdAt"`
	UpdatedAt  time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	Target     *AssetTargetRef `gorm:"foreignKey:TargetID" json:"target,omitempty"`
}

func (Screenshot) TableName() string {
	return "screenshot"
}
