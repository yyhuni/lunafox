package repository

import (
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWordlistRepositoryCreatePersistsEmptyTagsAsAnEmptyJSONArray(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE wordlist (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			tags JSON NOT NULL DEFAULT '[]',
			file_path TEXT NOT NULL DEFAULT '',
			file_size INTEGER NOT NULL DEFAULT 0,
			line_count INTEGER NOT NULL DEFAULT 0,
			file_hash TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`).Error; err != nil {
		t.Fatalf("create wordlist table: %v", err)
	}

	wordlist := &catalogdomain.Wordlist{
		FileName: "empty-tags.txt",
		Tags:     []string{},
	}
	if err := NewWordlistRepository(db).Create(wordlist); err != nil {
		t.Fatalf("create wordlist with empty tags: %v", err)
	}

	var stored model.Wordlist
	if err := db.First(&stored, wordlist.ID).Error; err != nil {
		t.Fatalf("read created wordlist: %v", err)
	}
	if stored.Tags == nil {
		t.Fatal("stored tags must be a non-nil empty slice")
	}
	if len(stored.Tags) != 0 {
		t.Fatalf("stored tags = %v, want empty", stored.Tags)
	}
}
