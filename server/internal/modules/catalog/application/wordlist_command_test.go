package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"gorm.io/gorm"
)

type wordlistCommandStoreStub struct {
	wordlistByID map[int]*catalogdomain.Wordlist
	nameExists   map[string]bool
	created      *catalogdomain.Wordlist
	updated      *catalogdomain.Wordlist
	deletedID    int
	findByIDErr  error
	createErr    error
	updateErr    error
	deleteErr    error
}

func (stub *wordlistCommandStoreStub) GetByID(id int) (*catalogdomain.Wordlist, error) {
	if stub.findByIDErr != nil {
		return nil, stub.findByIDErr
	}
	item, ok := stub.wordlistByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

func (stub *wordlistCommandStoreStub) ExistsByName(name string) (bool, error) {
	return stub.nameExists[name], nil
}

func (stub *wordlistCommandStoreStub) Create(wordlist *catalogdomain.Wordlist) error {
	if stub.createErr != nil {
		return stub.createErr
	}
	copyItem := *wordlist
	stub.created = &copyItem
	return nil
}

func (stub *wordlistCommandStoreStub) Update(wordlist *catalogdomain.Wordlist) error {
	if stub.updateErr != nil {
		return stub.updateErr
	}
	copyItem := *wordlist
	stub.updated = &copyItem
	return nil
}

func (stub *wordlistCommandStoreStub) Delete(id int) error {
	if stub.deleteErr != nil {
		return stub.deleteErr
	}
	stub.deletedID = id
	return nil
}

func TestWordlistCommandServiceCreateWordlist(t *testing.T) {
	t.Run("空名称", func(t *testing.T) {
		service := NewWordlistCommandService(&wordlistCommandStoreStub{}, t.TempDir(), newWordlistFileStoreTestStub())
		_, err := service.CreateWordlist(context.Background(), "", "", "a.txt", strings.NewReader("a"))
		if !errors.Is(err, ErrEmptyName) {
			t.Fatalf("expected ErrEmptyName, got %v", err)
		}
	})

	t.Run("成功创建", func(t *testing.T) {
		store := &wordlistCommandStoreStub{nameExists: map[string]bool{}}
		baseDir := t.TempDir()
		service := NewWordlistCommandService(store, baseDir, newWordlistFileStoreTestStub())

		wordlist, err := service.CreateWordlist(
			context.Background(),
			"dict",
			"desc",
			"dict.txt",
			strings.NewReader("a\nb\n"),
		)
		if err != nil {
			t.Fatalf("create wordlist failed: %v", err)
		}
		if wordlist.FilePath == "" || store.created == nil {
			t.Fatalf("expected file path and created record")
		}
		if !strings.HasPrefix(wordlist.FilePath, baseDir) {
			t.Fatalf("expected file under base dir, got %s", wordlist.FilePath)
		}
	})
}

func TestWordlistCommandServiceUpdateAndDelete(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "dict.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write fixture file failed: %v", err)
	}

	store := &wordlistCommandStoreStub{
		wordlistByID: map[int]*catalogdomain.Wordlist{
			1: {ID: 1, Name: "dict", FilePath: filePath},
		},
	}
	service := NewWordlistCommandService(store, baseDir, newWordlistFileStoreTestStub())

	updated, err := service.UpdateWordlistContent(context.Background(), 1, "x\ny")
	if err != nil {
		t.Fatalf("update content failed: %v", err)
	}
	if updated.LineCount != 2 {
		t.Fatalf("expected line count 2, got %d", updated.LineCount)
	}

	if err := service.DeleteWordlist(context.Background(), 1); err != nil {
		t.Fatalf("delete wordlist failed: %v", err)
	}
	if store.deletedID != 1 {
		t.Fatalf("expected deleted id 1, got %d", store.deletedID)
	}
}

func TestWordlistCommandServiceGetContent(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "dict.txt")
	if err := os.WriteFile(filePath, []byte("abc"), 0o644); err != nil {
		t.Fatalf("write fixture file failed: %v", err)
	}

	store := &wordlistCommandStoreStub{
		wordlistByID: map[int]*catalogdomain.Wordlist{5: {ID: 5, FilePath: filePath}},
	}
	service := NewWordlistCommandService(store, baseDir, newWordlistFileStoreTestStub())

	content, err := service.GetWordlistContent(context.Background(), 5)
	if err != nil {
		t.Fatalf("get content failed: %v", err)
	}
	if content != "abc" {
		t.Fatalf("unexpected content: %s", content)
	}
}
