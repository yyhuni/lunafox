package dto

import "time"

type UpdateWordlistContentRequest struct {
	Content string `json:"content" binding:"required"`
}

type WordlistResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FilePath    string    `json:"filePath"`
	FileSize    int64     `json:"fileSize"`
	LineCount   int       `json:"lineCount"`
	FileHash    string    `json:"fileHash"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type WordlistContentResponse struct {
	Content string `json:"content"`
}
