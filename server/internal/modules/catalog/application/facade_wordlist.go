package application

import (
	"context"
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

type WordlistFacade struct {
	queryService *WordlistQueryService
	cmdService   *WordlistCommandService
}

// NewWordlistFacade creates a new wordlist service.
func NewWordlistFacade(queryStore WordlistQueryStore, commandStore WordlistCommandStore, basePath string) *WordlistFacade {
	fileStore := NewLocalWordlistFileStore()
	return &WordlistFacade{
		queryService: NewWordlistQueryService(queryStore, fileStore),
		cmdService:   NewWordlistCommandService(commandStore, basePath, fileStore),
	}
}

// List returns paginated wordlists.
func (service *WordlistFacade) List(query *dto.PaginationQuery) ([]Wordlist, int64, error) {
	return service.queryService.ListWordlists(context.Background(), query.GetPage(), query.GetPageSize())
}

// ListAll returns all wordlists without pagination.
func (service *WordlistFacade) ListAll() ([]Wordlist, error) {
	return service.queryService.ListAllWordlists(context.Background())
}

// GetByID returns a wordlist by ID.
func (service *WordlistFacade) GetByID(id int) (*Wordlist, error) {
	wordlist, err := service.queryService.GetWordlistByID(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrWordlistNotFound
		}
		return nil, err
	}
	return wordlist, nil
}

// GetByName returns a wordlist by name.
func (service *WordlistFacade) GetByName(name string) (*Wordlist, error) {
	wordlist, err := service.queryService.GetWordlistByName(context.Background(), name)
	if err != nil {
		if errors.Is(err, ErrWordlistNotFound) {
			return nil, ErrWordlistNotFound
		}
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrWordlistNotFound
		}
		return nil, err
	}
	return wordlist, nil
}

// Delete deletes a wordlist and its file.
func (service *WordlistFacade) Delete(id int) error {
	err := service.cmdService.DeleteWordlist(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return ErrWordlistNotFound
		}
		return err
	}
	return nil
}

// GetFilePath returns the file path of a wordlist by name (for download).
func (service *WordlistFacade) GetFilePath(name string) (string, error) {
	filePath, err := service.queryService.GetWordlistFilePath(context.Background(), name)
	if err != nil {
		if errors.Is(err, ErrWordlistNotFound) {
			return "", ErrWordlistNotFound
		}
		if errors.Is(err, ErrFileNotFound) {
			return "", ErrFileNotFound
		}
		return "", err
	}
	return filePath, nil
}
