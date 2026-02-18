package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type WordlistQueryService struct {
	store     WordlistQueryStore
	fileStore WordlistFileStore
}

func NewWordlistQueryService(store WordlistQueryStore, fileStore WordlistFileStore) *WordlistQueryService {
	return &WordlistQueryService{store: store, fileStore: fileStore}
}

func (service *WordlistQueryService) ListWordlists(ctx context.Context, page, pageSize int) ([]catalogdomain.Wordlist, int64, error) {
	_ = ctx
	return service.store.FindAll(page, pageSize)
}

func (service *WordlistQueryService) ListAllWordlists(ctx context.Context) ([]catalogdomain.Wordlist, error) {
	_ = ctx

	wordlists, err := service.store.List()
	if err != nil {
		return nil, err
	}

	for index := range wordlists {
		service.syncWordlistFileStats(&wordlists[index])
	}

	return wordlists, nil
}

func (service *WordlistQueryService) GetWordlistByID(ctx context.Context, id int) (*catalogdomain.Wordlist, error) {
	_ = ctx

	wordlist, err := service.store.GetByID(id)
	if err != nil {
		return nil, err
	}

	service.syncWordlistFileStats(wordlist)
	return wordlist, nil
}

func (service *WordlistQueryService) GetWordlistByName(ctx context.Context, name string) (*catalogdomain.Wordlist, error) {
	_ = ctx

	name = catalogdomain.NormalizeWordlistName(name)
	if name == "" {
		return nil, ErrWordlistNotFound
	}

	wordlist, err := service.store.FindByName(name)
	if err != nil {
		return nil, err
	}

	service.syncWordlistFileStats(wordlist)
	return wordlist, nil
}

func (service *WordlistQueryService) GetWordlistFilePath(ctx context.Context, name string) (string, error) {
	wordlist, err := service.GetWordlistByName(ctx, name)
	if err != nil {
		return "", err
	}

	if !service.fileStore.Exists(wordlist.FilePath) {
		return "", ErrFileNotFound
	}

	return wordlist.FilePath, nil
}

func (service *WordlistQueryService) syncWordlistFileStats(wordlist *catalogdomain.Wordlist) {
	metadata, changed, err := service.fileStore.RefreshMetadata(wordlist.FilePath, wordlist.FileSize, wordlist.UpdatedAt)
	if err != nil || !changed || metadata == nil {
		return
	}

	wordlist.UpdateFileStats(metadata.FileSize, metadata.LineCount, metadata.FileHash)
	_ = service.store.Update(wordlist)
}
