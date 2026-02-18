package application

import (
	"context"
	"io"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type WordlistCommandService struct {
	store     WordlistCommandStore
	fileStore WordlistFileStore
	basePath  string
}

func NewWordlistCommandService(store WordlistCommandStore, basePath string, fileStore WordlistFileStore) *WordlistCommandService {
	return &WordlistCommandService{store: store, fileStore: fileStore, basePath: basePath}
}

func (service *WordlistCommandService) CreateWordlist(
	ctx context.Context,
	name, description, filename string,
	fileContent io.Reader,
) (*catalogdomain.Wordlist, error) {
	_ = ctx

	wordlist, err := catalogdomain.NewWordlist(name, description)
	if err != nil {
		return nil, err
	}

	exists, err := service.store.ExistsByName(wordlist.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrWordlistExists
	}

	metadata, err := service.fileStore.Save(service.basePath, filename, fileContent)
	if err != nil {
		return nil, err
	}

	wordlist.AttachFile(metadata.FilePath, metadata.FileSize, metadata.LineCount, metadata.FileHash)

	if err := service.store.Create(wordlist); err != nil {
		_ = service.fileStore.Remove(metadata.FilePath)
		return nil, err
	}

	return wordlist, nil
}

func (service *WordlistCommandService) DeleteWordlist(ctx context.Context, id int) error {
	_ = ctx

	wordlist, err := service.store.GetByID(id)
	if err != nil {
		return err
	}

	_ = service.fileStore.Remove(wordlist.FilePath)
	return service.store.Delete(id)
}

func (service *WordlistCommandService) UpdateWordlistContent(ctx context.Context, id int, content string) (*catalogdomain.Wordlist, error) {
	_ = ctx

	wordlist, err := service.store.GetByID(id)
	if err != nil {
		return nil, err
	}

	if wordlist.FilePath == "" {
		return nil, ErrFileNotFound
	}

	metadata, err := service.fileStore.Write(wordlist.FilePath, content)
	if err != nil {
		return nil, err
	}

	wordlist.UpdateFileStats(metadata.FileSize, metadata.LineCount, metadata.FileHash)

	if err := service.store.Update(wordlist); err != nil {
		return nil, err
	}

	return wordlist, nil
}

func (service *WordlistCommandService) GetWordlistContent(ctx context.Context, id int) (string, error) {
	_ = ctx

	wordlist, err := service.store.GetByID(id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return "", ErrWordlistNotFound
		}
		return "", err
	}

	if wordlist.FilePath == "" {
		return "", ErrFileNotFound
	}

	content, err := service.fileStore.Read(wordlist.FilePath)
	if err != nil {
		return "", ErrFileNotFound
	}
	return content, nil
}
