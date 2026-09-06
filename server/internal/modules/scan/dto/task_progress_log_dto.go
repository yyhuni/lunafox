package dto

import "time"

type TaskProgressLogListQuery struct {
	PageSize  int    `form:"pageSize" binding:"omitempty,min=1,max=1000"`
	PageToken string `form:"pageToken" binding:"omitempty"`
}

type TaskProgressLogListResponse struct {
	Results       []TaskProgressLogResponse `json:"results"`
	NextPageToken string                    `json:"nextPageToken,omitempty"`
}

type TaskProgressLogResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	ScanID    int       `json:"scanId"`
	TaskID    int       `json:"taskId"`
	Level     string    `json:"level"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}
