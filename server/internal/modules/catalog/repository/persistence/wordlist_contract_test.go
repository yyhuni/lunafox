package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestWordlistModelDeclaresSortableFieldIndexes(t *testing.T) {
	wordlistType := reflect.TypeOf(Wordlist{})
	required := map[string][]string{
		"FileSize":  {"index:idx_wordlist_file_size_id,priority:1"},
		"LineCount": {"index:idx_wordlist_line_count_id,priority:1"},
		"UpdatedAt": {"index:idx_wordlist_updated_at_id,priority:1"},
		"ID": {
			"index:idx_wordlist_file_size_id,priority:2",
			"index:idx_wordlist_line_count_id,priority:2",
			"index:idx_wordlist_updated_at_id,priority:2",
		},
	}

	for fieldName, fragments := range required {
		field, ok := wordlistType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("Wordlist.%s field is missing", fieldName)
		}
		gormTag := field.Tag.Get("gorm")
		for _, fragment := range fragments {
			if !strings.Contains(gormTag, fragment) {
				t.Fatalf("Wordlist.%s gorm tag must contain %q, got %q", fieldName, fragment, gormTag)
			}
		}
	}
}

func TestWordlistFileNameUsesCanonicalUniqueIndex(t *testing.T) {
	field, ok := reflect.TypeOf(Wordlist{}).FieldByName("FileName")
	if !ok {
		t.Fatal("Wordlist.FileName field is missing")
	}
	if tag := string(field.Tag); !strings.Contains(tag, "uniqueIndex:unique_wordlist_file_name") {
		t.Fatalf("Wordlist.FileName must use canonical unique_wordlist_file_name index, got tag %q", tag)
	}
}
