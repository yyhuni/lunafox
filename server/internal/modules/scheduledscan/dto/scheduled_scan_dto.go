package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type ScheduledScanListQuery struct {
	httpdto.PaginationQuery
	Filter         string `form:"filter" binding:"omitempty"`
	TargetID       int    `form:"targetId" binding:"omitempty"`
	OrganizationID int    `form:"organizationId" binding:"omitempty"`
}

type ScheduledScanOverviewQuery struct {
}

type CreateScheduledScanRequest struct {
	DisplayName    string         `json:"displayName" binding:"required"`
	ScanWorkflow   string         `json:"scanWorkflow" binding:"required"`
	Configuration  map[string]any `json:"configuration"`
	InputSource    string         `json:"inputSource" binding:"required"`
	Organization   string         `json:"organization,omitempty"`
	Target         string         `json:"target,omitempty"`
	Agent          string         `json:"agent,omitempty"`
	CronExpression string         `json:"cronExpression" binding:"required"`
	IsEnabled      *bool          `json:"isEnabled,omitempty"`
}

func (req *CreateScheduledScanRequest) UnmarshalJSON(data []byte) error {
	type rawRequest CreateScheduledScanRequest
	var raw rawRequest
	if err := decodeScheduledScanRequestFields(data, map[string]struct{}{
		"displayName":    {},
		"scanWorkflow":   {},
		"configuration":  {},
		"inputSource":    {},
		"organization":   {},
		"target":         {},
		"agent":          {},
		"cronExpression": {},
		"isEnabled":      {},
	}, &raw); err != nil {
		return err
	}
	*req = CreateScheduledScanRequest(raw)
	return nil
}

type UpdateScheduledScanRequest struct {
	Name           string         `json:"name" binding:"required"`
	DisplayName    *string        `json:"displayName,omitempty"`
	ScanWorkflow   *string        `json:"scanWorkflow,omitempty"`
	Configuration  map[string]any `json:"configuration,omitempty"`
	InputSource    *string        `json:"inputSource,omitempty"`
	Organization   *string        `json:"organization,omitempty"`
	Target         *string        `json:"target,omitempty"`
	Agent          *string        `json:"agent,omitempty"`
	CronExpression *string        `json:"cronExpression,omitempty"`
	IsEnabled      *bool          `json:"isEnabled,omitempty"`
	UpdateMask     string         `json:"updateMask" binding:"required"`
}

func (req *UpdateScheduledScanRequest) UnmarshalJSON(data []byte) error {
	type rawRequest UpdateScheduledScanRequest
	var raw rawRequest
	if err := decodeScheduledScanRequestFields(data, map[string]struct{}{
		"name":           {},
		"displayName":    {},
		"scanWorkflow":   {},
		"configuration":  {},
		"inputSource":    {},
		"organization":   {},
		"target":         {},
		"agent":          {},
		"cronExpression": {},
		"isEnabled":      {},
		"updateMask":     {},
	}, &raw); err != nil {
		return err
	}
	*req = UpdateScheduledScanRequest(raw)
	return nil
}

// BatchUpdateScheduledScanRequest declares one explicit Scheduled Scan status update.
type BatchUpdateScheduledScanRequest struct {
	Name       string `json:"name" binding:"required"`
	IsEnabled  *bool  `json:"isEnabled" binding:"required"`
	UpdateMask string `json:"updateMask" binding:"required"`
}

func (req *BatchUpdateScheduledScanRequest) UnmarshalJSON(data []byte) error {
	type rawRequest BatchUpdateScheduledScanRequest
	var raw rawRequest
	if err := decodeScheduledScanRequestFields(data, map[string]struct{}{
		"name":       {},
		"isEnabled":  {},
		"updateMask": {},
	}, &raw); err != nil {
		return err
	}
	*req = BatchUpdateScheduledScanRequest(raw)
	return nil
}

// BatchUpdateScheduledScansRequest groups up to one hundred atomic status updates.
type BatchUpdateScheduledScansRequest struct {
	Requests []BatchUpdateScheduledScanRequest `json:"requests" binding:"required,min=1,max=100,dive"`
}

func (req *BatchUpdateScheduledScansRequest) UnmarshalJSON(data []byte) error {
	type rawRequest BatchUpdateScheduledScansRequest
	var raw rawRequest
	if err := decodeScheduledScanRequestFields(data, map[string]struct{}{
		"requests": {},
	}, &raw); err != nil {
		return err
	}
	*req = BatchUpdateScheduledScansRequest(raw)
	return nil
}

// BatchUpdateScheduledScansResponse reports the number of committed updates.
type BatchUpdateScheduledScansResponse struct {
	UpdatedCount int `json:"updatedCount"`
}

type ScheduledScanResponse struct {
	ID                     int            `json:"id"`
	Name                   string         `json:"name"`
	DisplayName            string         `json:"displayName"`
	ScanWorkflow           string         `json:"scanWorkflow"`
	Configuration          map[string]any `json:"configuration,omitempty"`
	InputSource            string         `json:"inputSource"`
	Organization           *string        `json:"organization,omitempty"`
	OrganizationID         *int           `json:"organizationId"`
	OrganizationName       *string        `json:"organizationName"`
	Target                 *string        `json:"target,omitempty"`
	TargetID               *int           `json:"targetId"`
	TargetName             *string        `json:"targetName"`
	Agent                  string         `json:"agent,omitempty"`
	AgentID                *int           `json:"agentId,omitempty"`
	ScanMode               string         `json:"scanMode"`
	CronExpression         string         `json:"cronExpression"`
	IsEnabled              bool           `json:"isEnabled"`
	NextRunTime            *time.Time     `json:"nextRunTime"`
	LastRunTime            *time.Time     `json:"lastRunTime"`
	RunCount               int            `json:"runCount"`
	SuccessfulHandoffCount int            `json:"successfulHandoffCount"`
	FailedHandoffCount     int            `json:"failedHandoffCount"`
	CreatedAt              time.Time      `json:"createdAt"`
	UpdatedAt              time.Time      `json:"updatedAt"`
}

type ScheduledScanListResponse struct {
	ScheduledScans []ScheduledScanResponse `json:"scheduledScans"`
	NextPageToken  string                  `json:"nextPageToken,omitempty"`
	TotalSize      int64                   `json:"totalSize,omitempty"`
}

type ScheduledScanOverviewUpcomingResponse struct {
	Name                    string    `json:"name"`
	DisplayName             string    `json:"displayName"`
	Organization            *string   `json:"organization"`
	OrganizationDisplayName *string   `json:"organizationDisplayName"`
	Target                  *string   `json:"target"`
	TargetDisplayName       *string   `json:"targetDisplayName"`
	NextRunTime             time.Time `json:"nextRunTime"`
}

type ScheduledScanOverviewResponse struct {
	AsOfTime                      time.Time                               `json:"asOfTime"`
	EnabledScheduledScanCount     int64                                   `json:"enabledScheduledScanCount"`
	PausedScheduledScanCount      int64                                   `json:"pausedScheduledScanCount"`
	TodayScheduledScanCount       int64                                   `json:"todayScheduledScanCount"`
	Next24HoursScheduledScanCount int64                                   `json:"next24HoursScheduledScanCount"`
	UpcomingScheduledScans        []ScheduledScanOverviewUpcomingResponse `json:"upcomingScheduledScans"`
}

func NewScheduledScanListResponse(data []ScheduledScanResponse, total int64, page, pageSize int) ScheduledScanListResponse {
	paginated := httpdto.NewPaginatedResponse(data, total, page, pageSize)
	return ScheduledScanListResponse{
		ScheduledScans: paginated.Results,
		NextPageToken:  paginated.NextPageToken,
		TotalSize:      paginated.TotalSize,
	}
}

func decodeScheduledScanRequestFields(data []byte, allowed map[string]struct{}, target any) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for key := range fields {
		if _, ok := allowed[key]; ok {
			continue
		}
		return fmt.Errorf("scheduled scan request field %q is not supported; use scanWorkflow: scanWorkflows/{workflow}", key)
	}
	return json.Unmarshal(data, target)
}
