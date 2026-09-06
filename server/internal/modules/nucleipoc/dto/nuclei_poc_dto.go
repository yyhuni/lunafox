package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type SyncSourceRequest struct {
	RequestID  string `json:"requestId" binding:"required"`
	SourceType string `json:"sourceType" binding:"required"`
	RepoURL    string `json:"repoUrl" binding:"required"`
}

func (request *SyncSourceRequest) UnmarshalJSON(data []byte) error {
	type raw SyncSourceRequest
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	allowed := map[string]struct{}{"requestId": {}, "sourceType": {}, "repoUrl": {}}
	for field := range fields {
		if _, ok := allowed[field]; !ok {
			return fmt.Errorf("unsupported field %q", field)
		}
	}
	var value raw
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*request = SyncSourceRequest(value)
	return nil
}

type SourceResponse struct {
	Name       string     `json:"name"`
	SourceType string     `json:"sourceType"`
	RepoURL    string     `json:"repoUrl"`
	CommitSHA  string     `json:"commitSha"`
	SyncedAt   *time.Time `json:"syncedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type DiagnosticSample struct {
	Category     string `json:"category"`
	RelativePath string `json:"relativePath,omitempty"`
	ReasonCode   string `json:"reasonCode"`
}

type Diagnostics struct {
	Samples   []DiagnosticSample `json:"samples"`
	Total     int                `json:"total"`
	Truncated bool               `json:"truncated"`
}

type SyncCounters struct {
	FilesSeen          *int64 `json:"filesSeen"`
	YAMLFilesSeen      *int64 `json:"yamlFilesSeen"`
	TemplatesValidated *int64 `json:"templatesValidated"`
	BytesRead          *int64 `json:"bytesRead"`
}

type SyncTaskResponse struct {
	Name              string       `json:"name"`
	RequestID         string       `json:"requestId"`
	SourceType        string       `json:"sourceType"`
	State             string       `json:"state"`
	Phase             string       `json:"phase"`
	Counters          SyncCounters `json:"counters"`
	CommitSHA         string       `json:"commitSha,omitempty"`
	CommittedPOCCount int64        `json:"committedPocCount,omitempty"`
	FailureCode       string       `json:"failureCode,omitempty"`
	FailureSummary    string       `json:"failureSummary,omitempty"`
	Diagnostics       Diagnostics  `json:"diagnostics"`
	CleanupStatus     string       `json:"cleanupStatus"`
	CreatedAt         time.Time    `json:"createdAt"`
	StartedAt         *time.Time   `json:"startedAt"`
	CompletedAt       *time.Time   `json:"completedAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
}

type NucleiPocListQuery struct {
	PageSize  int    `form:"pageSize" binding:"omitempty,min=1,max=1000"`
	PageToken string `form:"pageToken" binding:"omitempty"`
	Filter    string `form:"filter" binding:"omitempty"`
	OrderBy   string `form:"orderBy" binding:"omitempty"`
}

type NucleiPocResponse struct {
	Name          string    `json:"name"`
	TemplateID    string    `json:"templateId"`
	DisplayName   string    `json:"displayName"`
	Severity      string    `json:"severity"`
	Tags          []string  `json:"tags"`
	Author        string    `json:"author"`
	Description   string    `json:"description"`
	CVE           []string  `json:"cve"`
	CWE           []string  `json:"cwe"`
	References    []string  `json:"references"`
	Remediation   string    `json:"remediation"`
	RelativePath  string    `json:"relativePath"`
	ContentSHA256 string    `json:"contentSha256"`
	IsEnabled     bool      `json:"isEnabled"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Content       string    `json:"content,omitempty"`
}

type NucleiPocListResponse struct {
	Results       []NucleiPocResponse `json:"results"`
	NextPageToken string              `json:"nextPageToken,omitempty"`
	TotalSize     int64               `json:"totalSize"`
}

type UpdateNucleiPocRequest struct {
	Name       string   `json:"name" binding:"required"`
	IsEnabled  *bool    `json:"isEnabled"`
	UpdateMask []string `json:"updateMask" binding:"required"`
}

// SetNucleiPocActivationRequest deliberately uses pointers so omitted values
// remain distinguishable from explicit false and selected-scope commands.
type SetNucleiPocActivationRequest struct {
	Enabled *bool     `json:"enabled"`
	Names   *[]string `json:"names"`
}

func (request *SetNucleiPocActivationRequest) UnmarshalJSON(data []byte) error {
	type raw SetNucleiPocActivationRequest
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	allowed := map[string]struct{}{"enabled": {}, "names": {}}
	for field := range fields {
		if _, ok := allowed[field]; !ok {
			return fmt.Errorf("unsupported field %q", field)
		}
	}
	var value raw
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	// Preserve field presence so a JSON null names value cannot be mistaken for
	// an omitted scope, which would otherwise widen the mutation to the catalog.
	if rawNames, ok := fields["names"]; ok && string(rawNames) == "null" {
		empty := []string(nil)
		value.Names = &empty
	}
	*request = SetNucleiPocActivationRequest(value)
	return nil
}

type SetNucleiPocActivationResponse struct {
	Enabled       bool  `json:"enabled"`
	AffectedCount int64 `json:"affectedCount"`
}

// UnmarshalJSON accepts only the canonical array updateMask. Keeping parsing
// strict prevents a string mask from silently widening the mutable surface.
func (request *UpdateNucleiPocRequest) UnmarshalJSON(data []byte) error {
	type raw UpdateNucleiPocRequest
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	allowed := map[string]struct{}{"name": {}, "isEnabled": {}, "updateMask": {}}
	for field := range fields {
		if _, ok := allowed[field]; !ok {
			return fmt.Errorf("unsupported field %q", field)
		}
	}
	var value raw
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*request = UpdateNucleiPocRequest(value)
	return nil
}

func (request UpdateNucleiPocRequest) MaskSet() map[string]struct{} {
	set := make(map[string]struct{}, len(request.UpdateMask))
	for _, field := range request.UpdateMask {
		field = strings.TrimSpace(field)
		if field != "" {
			set[field] = struct{}{}
		}
	}
	return set
}
