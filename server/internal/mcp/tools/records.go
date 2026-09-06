package tools

import (
	"encoding/json"
	"time"
)

// Page is the MCP collection envelope shared by every list adapter.
type Page[T any] struct {
	Items         []T    `json:"items"`
	NextPageToken string `json:"next_page_token"`
	TotalSize     int64  `json:"total_size"`
}

type TargetRecord struct {
	ID            int            `json:"id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	CreatedAt     time.Time      `json:"createdAt"`
	LastScannedAt *time.Time     `json:"lastScannedAt,omitempty"`
	Summary       *TargetSummary `json:"summary,omitempty"`
}

type TargetSummary struct {
	Subdomains      int64              `json:"subdomains"`
	Websites        int64              `json:"websites"`
	Endpoints       int64              `json:"endpoints"`
	IPs             int64              `json:"ips"`
	Directories     int64              `json:"directories"`
	Screenshots     int64              `json:"screenshots"`
	Vulnerabilities VulnerabilityCount `json:"vulnerabilities"`
}

type VulnerabilityCount struct {
	Total    int64 `json:"total"`
	Critical int64 `json:"critical"`
	High     int64 `json:"high"`
	Medium   int64 `json:"medium"`
	Low      int64 `json:"low"`
}

type ScanRecord struct {
	ID             int                 `json:"id"`
	TargetID       int                 `json:"targetId"`
	TargetName     string              `json:"targetName,omitempty"`
	Workflow       string              `json:"workflow,omitempty"`
	Status         string              `json:"status"`
	InputSource    string              `json:"inputSource,omitempty"`
	TriggerType    string              `json:"triggerType,omitempty"`
	AssignmentMode string              `json:"assignmentMode,omitempty"`
	Progress       int                 `json:"progress"`
	CurrentStage   string              `json:"currentStage,omitempty"`
	CreatedAt      time.Time           `json:"createdAt"`
	StoppedAt      *time.Time          `json:"stoppedAt,omitempty"`
	AgentID        *int                `json:"agentId,omitempty"`
	AgentName      string              `json:"agentName,omitempty"`
	AgentStatus    string              `json:"agentStatus,omitempty"`
	AgentDeleted   bool                `json:"agentDeleted,omitempty"`
	CachedAssets   AssetCount          `json:"cachedAssets,omitempty"`
	Failure        *ScanFailure        `json:"failure,omitempty"`
	RuntimeTasks   []RuntimeTaskRecord `json:"runtimeTasks,omitempty"`
}

type AssetCount struct {
	Subdomains      int `json:"subdomains"`
	Websites        int `json:"websites"`
	Endpoints       int `json:"endpoints"`
	IPs             int `json:"ips"`
	Directories     int `json:"directories"`
	Screenshots     int `json:"screenshots"`
	Vulnerabilities int `json:"vulnerabilities"`
}

type ScanFailure struct {
	Kind    string `json:"kind,omitempty"`
	Message string `json:"message,omitempty"`
}

type RuntimeTaskRecord struct {
	ID          int        `json:"id"`
	StepID      string     `json:"stepId,omitempty"`
	StageID     string     `json:"stageId,omitempty"`
	EngineID    string     `json:"engineId,omitempty"`
	Status      string     `json:"status"`
	SkipReason  string     `json:"skipReason,omitempty"`
	Order       int        `json:"order"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Duration    *float64   `json:"duration,omitempty"`
	FailureKind string     `json:"failureKind,omitempty"`
}

type WebsiteRecord struct {
	ID            int       `json:"id"`
	TargetID      int       `json:"targetId"`
	URL           string    `json:"url"`
	Host          string    `json:"host,omitempty"`
	Location      string    `json:"location,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	Title         string    `json:"title,omitempty"`
	Webserver     string    `json:"webserver,omitempty"`
	ContentType   string    `json:"contentType,omitempty"`
	Tech          []string  `json:"tech,omitempty"`
	StatusCode    *int      `json:"statusCode,omitempty"`
	ContentLength *int      `json:"contentLength,omitempty"`
	Vhost         *bool     `json:"vhost,omitempty"`
}

type SubdomainRecord struct {
	ID        int       `json:"id"`
	TargetID  int       `json:"targetId"`
	DNSName   string    `json:"dnsName"`
	CreatedAt time.Time `json:"createdAt"`
}

type EndpointRecord struct {
	ID            int       `json:"id"`
	TargetID      int       `json:"targetId"`
	URL           string    `json:"url"`
	Host          string    `json:"host,omitempty"`
	Location      string    `json:"location,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	Title         string    `json:"title,omitempty"`
	Webserver     string    `json:"webserver,omitempty"`
	ContentType   string    `json:"contentType,omitempty"`
	Tech          []string  `json:"tech,omitempty"`
	StatusCode    *int      `json:"statusCode,omitempty"`
	ContentLength *int      `json:"contentLength,omitempty"`
	Vhost         *bool     `json:"vhost,omitempty"`
}

type DirectoryRecord struct {
	ID            int       `json:"id"`
	TargetID      int       `json:"targetId"`
	URL           string    `json:"url"`
	Status        *int      `json:"status,omitempty"`
	ContentLength *int64    `json:"contentLength,omitempty"`
	ContentType   string    `json:"contentType,omitempty"`
	Duration      *int64    `json:"duration,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type HostPortRecord struct {
	IP        string    `json:"ip"`
	Hosts     []string  `json:"hosts,omitempty"`
	Ports     []int     `json:"ports,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type VulnerabilityRecord struct {
	ID          int             `json:"id"`
	TargetID    int             `json:"targetId"`
	URL         string          `json:"url"`
	VulnType    string          `json:"vulnType"`
	Severity    string          `json:"severity"`
	Source      string          `json:"source,omitempty"`
	CVSSScore   *float64        `json:"cvssScore,omitempty"`
	Description string          `json:"description,omitempty"`
	RawOutput   json.RawMessage `json:"rawOutput,omitempty"`
	Reviewed    bool            `json:"reviewed"`
	CreatedAt   time.Time       `json:"createdAt"`
}
