package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type WordlistListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter" binding:"omitempty"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *WordlistListQuery) ValidatePageToken() error {
	return nil
}

type WordlistTagListQuery struct {
	httpdto.PaginationQuery
	Filter string `form:"filter" binding:"omitempty"`
}

type UpdateWordlistRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description" binding:"omitempty,max=200"`
	Tags        []string `json:"tags" binding:"omitempty"`
	UpdateMask  string   `json:"updateMask" binding:"required"`
}

type UpdateWordlistContentRequest struct {
	Name    string `json:"name" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type WordlistResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	FileName    string    `json:"fileName"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	FilePath    string    `json:"filePath"`
	FileSize    int64     `json:"fileSize"`
	LineCount   int       `json:"lineCount"`
	FileHash    string    `json:"fileHash"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type WordlistTagSummaryResponse struct {
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	WordlistCount int64  `json:"wordlistCount"`
}

type WordlistContentResponse struct {
	Name       string    `json:"name"`
	Content    string    `json:"content"`
	UpdateTime time.Time `json:"updateTime"`
}
