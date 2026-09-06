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
	wordlistByID   map[int]*catalogdomain.Wordlist
	fileNameExists map[string]bool
	created        *catalogdomain.Wordlist
	updated        *catalogdomain.Wordlist
	deletedID      int
	findByIDErr    error
	createErr      error
	updateErr      error
	deleteErr      error
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

func (stub *wordlistCommandStoreStub) ExistsByFileName(fileName string, excludeID ...int) (bool, error) {
	_ = excludeID
	return stub.fileNameExists[fileName], nil
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
	delete(stub.wordlistByID, id)
	return nil
}

func TestWordlistCommandServiceCreateWordlist(t *testing.T) {
	t.Run("empty file name", func(t *testing.T) {
		service := NewWordlistCommandService(&wordlistCommandStoreStub{}, t.TempDir(), newWordlistFileStoreTestStub())
		_, err := service.CreateWordlist(context.Background(), "", "", nil, "a.txt", strings.NewReader("a"))
		if !errors.Is(err, ErrEmptyFileName) {
			t.Fatalf("expected ErrEmptyFileName, got %v", err)
		}
	})

	t.Run("create succeeds", func(t *testing.T) {
		store := &wordlistCommandStoreStub{fileNameExists: map[string]bool{}}
		baseDir := t.TempDir()
		service := NewWordlistCommandService(store, baseDir, newWordlistFileStoreTestStub())

		wordlist, err := service.CreateWordlist(
			context.Background(),
			"dict",
			"desc",
			[]string{" 子域名扫描 ", "fuzz", "fuzz"},
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
		if got, want := strings.Join(wordlist.Tags, ","), "fuzz,子域名扫描"; got != want {
			t.Fatalf("expected normalized tags %q, got %q", want, got)
		}
	})
}

func TestWordlistCommandServiceCreateRemovesFileWhenCatalogCreateFails(t *testing.T) {
	baseDir := t.TempDir()
	store := &wordlistCommandStoreStub{
		fileNameExists: map[string]bool{},
		createErr:      errors.New("unique constraint violation"),
	}
	service := NewWordlistCommandService(store, baseDir, NewLocalWordlistFileStore())

	if _, err := service.CreateWordlist(context.Background(), "dict.txt", "", nil, "dict.txt", strings.NewReader("one\ntwo\n")); err == nil {
		t.Fatal("expected Catalog create failure")
	}
	if _, err := os.Stat(filepath.Join(baseDir, "dict.txt")); !os.IsNotExist(err) {
		t.Fatalf("new file survived failed Catalog create: %v", err)
	}
}

func TestWordlistCommandServiceUpdateAndDelete(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "dict.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write fixture file failed: %v", err)
	}

	store := &wordlistCommandStoreStub{
		wordlistByID: map[int]*catalogdomain.Wordlist{
			1: {ID: 1, FileName: "dict.txt", FilePath: filePath},
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

func TestWordlistCommandServiceDeletesOrdinaryAndInstallationSeededWordlistsWithoutReplacement(t *testing.T) {
	for _, test := range []struct {
		name         string
		wordlistName string
	}{
		{name: "ordinary", wordlistName: "custom-subdomains.txt"},
		{name: "installation seeded", wordlistName: "subdomains-top1million-110000.txt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			filePath := filepath.Join(t.TempDir(), test.wordlistName)
			if err := os.WriteFile(filePath, []byte("example\n"), 0o644); err != nil {
				t.Fatalf("write fixture file: %v", err)
			}
			store := &wordlistCommandStoreStub{wordlistByID: map[int]*catalogdomain.Wordlist{
				7: {ID: 7, FileName: test.wordlistName, FilePath: filePath},
			}}
			service := NewWordlistCommandService(store, t.TempDir(), newWordlistFileStoreTestStub())

			if err := service.DeleteWordlist(context.Background(), 7); err != nil {
				t.Fatalf("DeleteWordlist() error = %v", err)
			}
			if store.deletedID != 7 {
				t.Fatalf("deleted ID = %d, want 7", store.deletedID)
			}
			if _, exists := store.wordlistByID[7]; exists {
				t.Fatal("deleted Catalog row was restored")
			}
			if store.created != nil || store.updated != nil {
				t.Fatalf("delete rewrote or replaced Catalog consumer state: created=%#v updated=%#v", store.created, store.updated)
			}
			if _, err := os.Stat(filePath); !os.IsNotExist(err) {
				t.Fatalf("deleted wordlist file still exists: %v", err)
			}
		})
	}
}

func TestWordlistCommandServiceUpdateWordlistMetadata(t *testing.T) {
	store := &wordlistCommandStoreStub{
		wordlistByID: map[int]*catalogdomain.Wordlist{
			1: {ID: 1, FileName: "old.txt", Description: "old desc", Tags: []string{"old"}},
		},
		fileNameExists: map[string]bool{},
	}
	service := NewWordlistCommandService(store, t.TempDir(), newWordlistFileStoreTestStub())

	updated, err := service.UpdateWordlistMetadata(context.Background(), 1, " desc ", []string{"fuzz", " custom ", "fuzz"}, "description,tags")
	if err != nil {
		t.Fatalf("update metadata failed: %v", err)
	}
	if updated.FileName != "old.txt" || updated.Description != "desc" {
		t.Fatalf("unexpected metadata: %+v", updated)
	}
	if got, want := strings.Join(updated.Tags, ","), "custom,fuzz"; got != want {
		t.Fatalf("expected tags %q, got %q", want, got)
	}
	if store.updated == nil || store.updated.FileName != "old.txt" {
		t.Fatalf("expected persisted update, got %+v", store.updated)
	}
}

func TestWordlistCommandServiceUpdateWordlistMetadataRejectsImmutableIdentityMasks(t *testing.T) {
	for _, updateMask := range []string{"name", "displayName", "fileName"} {
		t.Run(updateMask, func(t *testing.T) {
			store := &wordlistCommandStoreStub{
				wordlistByID: map[int]*catalogdomain.Wordlist{1: {ID: 1, FileName: "old.txt"}},
			}
			service := NewWordlistCommandService(store, t.TempDir(), newWordlistFileStoreTestStub())

			_, err := service.UpdateWordlistMetadata(context.Background(), 1, "", nil, updateMask)
			if err == nil || !strings.Contains(err.Error(), "unsupported updateMask field") {
				t.Fatalf("expected unsupported updateMask error, got %v", err)
			}
			if store.updated != nil {
				t.Fatalf("immutable identity mask reached persistence: %+v", store.updated)
			}
		})
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
