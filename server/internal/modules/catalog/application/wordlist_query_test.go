package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type wordlistQueryStoreStub struct {
	listItems  []catalogdomain.Wordlist
	findByID   map[int]*catalogdomain.Wordlist
	findByName map[string]*catalogdomain.Wordlist
	listErr    error
	findErr    error
	updated    *catalogdomain.Wordlist
}

func (stub *wordlistQueryStoreStub) FindAll(page, pageSize int) ([]catalogdomain.Wordlist, int64, error) {
	_ = page
	_ = pageSize
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	result := append([]catalogdomain.Wordlist(nil), stub.listItems...)
	return result, int64(len(result)), nil
}

func (stub *wordlistQueryStoreStub) List() ([]catalogdomain.Wordlist, error) {
	if stub.listErr != nil {
		return nil, stub.listErr
	}
	return append([]catalogdomain.Wordlist(nil), stub.listItems...), nil
}

func (stub *wordlistQueryStoreStub) GetByID(id int) (*catalogdomain.Wordlist, error) {
	if stub.findErr != nil {
		return nil, stub.findErr
	}
	item, ok := stub.findByID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	copyItem := *item
	return &copyItem, nil
}

func (stub *wordlistQueryStoreStub) FindByName(name string) (*catalogdomain.Wordlist, error) {
	if stub.findErr != nil {
		return nil, stub.findErr
	}
	item, ok := stub.findByName[name]
	if !ok {
		return nil, errors.New("not found")
	}
	copyItem := *item
	return &copyItem, nil
}

func (stub *wordlistQueryStoreStub) Update(wordlist *catalogdomain.Wordlist) error {
	copyItem := *wordlist
	stub.updated = &copyItem
	return nil
}

func TestWordlistQueryServiceGetAndPath(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "dict.txt")
	if err := os.WriteFile(filePath, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatalf("write fixture file failed: %v", err)
	}

	store := &wordlistQueryStoreStub{
		findByID: map[int]*catalogdomain.Wordlist{
			1: {ID: 1, Name: "dict", FilePath: filePath},
		},
		findByName: map[string]*catalogdomain.Wordlist{
			"dict": {ID: 1, Name: "dict", FilePath: filePath},
		},
	}
	service := NewWordlistQueryService(store, newWordlistFileStoreTestStub())

	wordlist, err := service.GetWordlistByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if wordlist.ID != 1 {
		t.Fatalf("unexpected wordlist: %+v", wordlist)
	}

	path, err := service.GetWordlistFilePath(context.Background(), "dict")
	if err != nil {
		t.Fatalf("get file path failed: %v", err)
	}
	if path != filePath {
		t.Fatalf("unexpected file path: %s", path)
	}
}

func TestWordlistQueryServiceListAll(t *testing.T) {
	store := &wordlistQueryStoreStub{
		listItems: []catalogdomain.Wordlist{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}},
	}
	service := NewWordlistQueryService(store, newWordlistFileStoreTestStub())

	items, err := service.ListAllWordlists(context.Background())
	if err != nil {
		t.Fatalf("list all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}
