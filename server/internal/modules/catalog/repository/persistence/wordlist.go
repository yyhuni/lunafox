package model

import "time"

// Wordlist represents a dictionary file for scanning.
type Wordlist struct {
	ID          int       `gorm:"primaryKey;index:idx_wordlist_file_size_id,priority:2;index:idx_wordlist_line_count_id,priority:2;index:idx_wordlist_updated_at_id,priority:2" json:"id"`
	FileName    string    `gorm:"column:file_name;size:200;uniqueIndex:unique_wordlist_file_name" json:"fileName"`
	Description string    `gorm:"column:description;size:200" json:"description"`
	Tags        []string  `gorm:"column:tags;type:jsonb;serializer:json" json:"tags"`
	FilePath    string    `gorm:"column:file_path;size:500" json:"filePath"`
	FileSize    int64     `gorm:"column:file_size;default:0;index:idx_wordlist_file_size_id,priority:1" json:"fileSize"`
	LineCount   int       `gorm:"column:line_count;default:0;index:idx_wordlist_line_count_id,priority:1" json:"lineCount"`
	FileHash    string    `gorm:"column:file_hash;size:64" json:"fileHash"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;index:idx_wordlist_updated_at_id,priority:1" json:"updatedAt"`
}

func (Wordlist) TableName() string {
	return "wordlist"
}
