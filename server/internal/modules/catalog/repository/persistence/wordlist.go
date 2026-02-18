package model

import "time"

// Wordlist represents a dictionary file for scanning.
type Wordlist struct {
	ID          int       `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"column:name;size:200;uniqueIndex" json:"name"`
	Description string    `gorm:"column:description;size:200" json:"description"`
	FilePath    string    `gorm:"column:file_path;size:500" json:"filePath"`
	FileSize    int64     `gorm:"column:file_size;default:0" json:"fileSize"`
	LineCount   int       `gorm:"column:line_count;default:0" json:"lineCount"`
	FileHash    string    `gorm:"column:file_hash;size:64" json:"fileHash"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (Wordlist) TableName() string {
	return "wordlist"
}
