package dto

import "time"

type ScanLogListQuery struct {
	AfterID int64 `form:"afterId" binding:"omitempty,min=0"`
	Limit   int   `form:"limit" binding:"omitempty,min=1,max=1000"`
}

type ScanLogListResponse struct {
	Results []ScanLogResponse `json:"results"`
	HasMore bool              `json:"hasMore"`
}

type ScanLogResponse struct {
	ID        int64     `json:"id"`
	ScanID    int       `json:"scanId"`
	Level     string    `json:"level"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type ScanLogItem struct {
	Level   string `json:"level" binding:"required,oneof=info warning error"`
	Content string `json:"content" binding:"required"`
}

type BulkCreateScanLogsRequest struct {
	Logs []ScanLogItem `json:"logs" binding:"required,min=1,max=1000,dive"`
}

type BulkCreateScanLogsResponse struct {
	CreatedCount int `json:"createdCount"`
}
