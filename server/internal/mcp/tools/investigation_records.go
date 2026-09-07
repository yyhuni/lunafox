package tools

import (
	"encoding/json"
	"fmt"
	"time"
)

type OrganizationRecord struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	TargetCount int64     `json:"targetCount"`
}

type OrganizationTargetRecord struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastScannedAt *time.Time `json:"lastScannedAt,omitempty"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
}

type ScreenshotRecord struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	TargetID   int       `json:"targetId"`
	URL        string    `json:"url"`
	StatusCode *int16    `json:"statusCode,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type ScanWorkflowRecord struct {
	Name         string    `json:"name"`
	DisplayName  string    `json:"displayName"`
	Description  string    `json:"description"`
	Stages       any       `json:"stages"`
	IsBuiltin    bool      `json:"isBuiltin"`
	IsExecutable bool      `json:"isExecutable"`
	ETag         string    `json:"etag"`
	CreateTime   time.Time `json:"createTime"`
	UpdateTime   time.Time `json:"updateTime"`
}

type ScanWorkflowProfileRecord struct {
	Name          string         `json:"name"`
	ScanWorkflow  string         `json:"scanWorkflow"`
	Configuration map[string]any `json:"configuration"`
}

type EngineConfigParamResourceRecord struct {
	Kind string `json:"kind"`
}

type EngineConfigParamRecord struct {
	Key       string                           `json:"key"`
	Type      string                           `json:"type"`
	Default   any                              `json:"default,omitempty"`
	Minimum   *int                             `json:"minimum,omitempty"`
	Maximum   *int                             `json:"maximum,omitempty"`
	MinLength *int                             `json:"minLength,omitempty"`
	MaxLength *int                             `json:"maxLength,omitempty"`
	MinItems  *int                             `json:"minItems,omitempty"`
	MaxItems  *int                             `json:"maxItems,omitempty"`
	Pattern   string                           `json:"pattern,omitempty"`
	Enum      []string                         `json:"enum,omitempty"`
	Resource  *EngineConfigParamResourceRecord `json:"resource,omitempty"`
}

type EngineConfigSectionRecord struct {
	ID              string                    `json:"id"`
	DefaultEnabled  bool                      `json:"defaultEnabled"`
	RequiredEnabled bool                      `json:"requiredEnabled"`
	Params          []EngineConfigParamRecord `json:"params,omitempty"`
}

type EngineRecord struct {
	Name                 string                      `json:"name"`
	EngineID             string                      `json:"engineId"`
	ManifestVersion      string                      `json:"manifestVersion,omitempty"`
	Publisher            string                      `json:"publisher,omitempty"`
	PackageVersion       string                      `json:"packageVersion,omitempty"`
	ArtifactRef          string                      `json:"artifactRef,omitempty"`
	PackageDigest        string                      `json:"packageDigest,omitempty"`
	EngineAPIMajor       uint32                      `json:"engineApiMajor"`
	SupportedTargetTypes []string                    `json:"supportedTargetTypes,omitempty"`
	ExecutionResources   []string                    `json:"executionResources,omitempty"`
	ConfigSections       []EngineConfigSectionRecord `json:"configSections,omitempty"`
}

type WordlistRecord struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	FileName    string    `json:"fileName"`
	Description string    `json:"description,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	FileSize    int64     `json:"fileSize"`
	LineCount   int       `json:"lineCount"`
	FileHash    string    `json:"fileHash,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type LogRecord struct {
	ID        string `json:"id"`
	TS        string `json:"ts"`
	TSNs      string `json:"tsNs"`
	Stream    string `json:"stream"`
	Line      string `json:"line"`
	Truncated bool   `json:"truncated"`
}

type LogPage struct {
	Items             []LogRecord `json:"items"`
	NextPageToken     string      `json:"next_page_token,omitempty"`
	PreviousPageToken string      `json:"previous_page_token,omitempty"`
	HasOlder          bool        `json:"hasOlder"`
	HasNewer          bool        `json:"hasNewer"`
	CaughtUp          bool        `json:"caughtUp"`
	Gap               bool        `json:"gap"`
	GapReason         string      `json:"gapReason,omitempty"`
}

type VulnerabilityActionRecord struct {
	Name     string `json:"name"`
	Reviewed bool   `json:"reviewed"`
}

type VulnerabilityBatchActionRecord struct {
	RequestedCount int  `json:"requestedCount"`
	ProcessedCount int  `json:"processedCount"`
	Reviewed       bool `json:"reviewed"`
}

type ActionViolation struct {
	Index  int    `json:"index"`
	Name   string `json:"name,omitempty"`
	Reason string `json:"reason"`
}

// VulnerabilityBatchValidationError carries bounded, pre-write violations for
// the explicit MCP batch disposition surface.
type VulnerabilityBatchValidationError struct{ Violations []ActionViolation }

func (err *VulnerabilityBatchValidationError) Error() string {
	if err == nil {
		return "invalid vulnerability batch"
	}
	return fmt.Sprintf("invalid vulnerability batch with %d violations", len(err.Violations))
}

type StartScanInput struct {
	Target        string
	ScanWorkflow  string
	Configuration map[string]any
	Agent         string
	RequestID     string
}

type StartScanOutput struct {
	Operation string `json:"operation"`
	Scan      string `json:"scan"`
}

type OperationRecord struct {
	Name        string         `json:"name"`
	Scan        string         `json:"scan"`
	Target      string         `json:"target"`
	Status      string         `json:"status"`
	Phase       string         `json:"phase,omitempty"`
	Progress    int            `json:"progress"`
	CurrentTask string         `json:"currentTask,omitempty"`
	CreateTime  time.Time      `json:"createTime"`
	UpdateTime  time.Time      `json:"updateTime"`
	Response    map[string]any `json:"response,omitempty"`
	Error       map[string]any `json:"error,omitempty"`
}

// Ensure the records remain JSON serializable even when a catalog projection
// carries a typed map from a generated contract.
var _ = json.RawMessage{}
