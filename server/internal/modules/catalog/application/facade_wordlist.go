package application

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

type WordlistFacade struct {
	queryService *WordlistQueryService
	cmdService   *WordlistCommandService
}

// NewWordlistFacade creates a new wordlist service.
func NewWordlistFacade(queryService *WordlistQueryService, cmdService *WordlistCommandService) *WordlistFacade {
	return &WordlistFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// List returns paginated wordlists.
func (service *WordlistFacade) List(query *dto.WordlistListQuery) (*WordlistListResult, error) {
	return service.ListContext(context.Background(), query)
}

// ListContext preserves request cancellation for MCP callers.
func (service *WordlistFacade) ListContext(ctx context.Context, query *dto.WordlistListQuery) (*WordlistListResult, error) {
	if service == nil || service.queryService == nil {
		return nil, fmt.Errorf("wordlist query service is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	if query == nil {
		return nil, ErrUnsupportedWordlistFilter
	}
	return service.queryService.ListWordlists(ctx, WordlistListQueryInput{
		PageSize:  query.GetPageSize(),
		PageToken: query.PageToken,
		Filter:    query.Filter,
		OrderBy:   query.OrderBy,
	})
}

func (service *WordlistFacade) ListTags(query *dto.WordlistTagListQuery) ([]WordlistTagSummary, int64, error) {
	return service.queryService.ListWordlistTagSummaries(context.Background(), query.GetPage(), query.GetPageSize(), query.Filter)
}

// ListAll returns all wordlists without pagination.
func (service *WordlistFacade) ListAll() ([]Wordlist, error) {
	return service.ListAllContext(context.Background())
}

func (service *WordlistFacade) ListAllContext(ctx context.Context) ([]Wordlist, error) {
	if service == nil || service.queryService == nil {
		return nil, fmt.Errorf("wordlist query service is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	return service.queryService.ListAllWordlists(ctx)
}

// GetByID returns a wordlist by ID.
func (service *WordlistFacade) GetByID(id int) (*Wordlist, error) {
	return service.GetByIDContext(context.Background(), id)
}

func (service *WordlistFacade) GetByIDContext(ctx context.Context, id int) (*Wordlist, error) {
	if service == nil || service.queryService == nil {
		return nil, fmt.Errorf("wordlist query service is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	wordlist, err := service.queryService.GetWordlistByID(ctx, id)
	if err != nil {
		return nil, mapWordlistBoundaryError(err)
	}
	return wordlist, nil
}

func (service *WordlistFacade) UpdateMetadata(id int, req *dto.UpdateWordlistRequest) (*Wordlist, error) {
	wordlist, err := service.cmdService.UpdateWordlistMetadata(context.Background(), id, req.Description, req.Tags, req.UpdateMask)
	if err != nil {
		if isRecordNotFound(err) {
			return nil, ErrWordlistNotFound
		}
		if errors.Is(err, ErrWordlistExists) {
			return nil, ErrWordlistExists
		}
		if errors.Is(err, ErrEmptyFileName) {
			return nil, ErrEmptyFileName
		}
		if errors.Is(err, ErrFileNameTooLong) {
			return nil, ErrFileNameTooLong
		}
		if errors.Is(err, ErrInvalidFileName) {
			return nil, ErrInvalidFileName
		}
		return nil, err
	}
	return wordlist, nil
}

// Delete deletes a wordlist and its file.
func (service *WordlistFacade) Delete(id int) error {
	err := service.cmdService.DeleteWordlist(context.Background(), id)
	if err != nil {
		return mapWordlistBoundaryError(err)
	}
	return nil
}

func (service *WordlistFacade) Create(fileName, description string, tags []string, storedFileName string, fileContent io.Reader) (*Wordlist, error) {
	wordlist, err := service.cmdService.CreateWordlist(context.Background(), fileName, description, tags, storedFileName, fileContent)
	if err != nil {
		if errors.Is(err, ErrWordlistExists) {
			return nil, ErrWordlistExists
		}
		if errors.Is(err, ErrEmptyFileName) {
			return nil, ErrEmptyFileName
		}
		if errors.Is(err, ErrFileNameTooLong) {
			return nil, ErrFileNameTooLong
		}
		if errors.Is(err, ErrInvalidFileName) {
			return nil, ErrInvalidFileName
		}
		if errors.Is(err, ErrInvalidFileType) {
			return nil, ErrInvalidFileType
		}
		if errors.Is(err, ErrLineTooLong) {
			return nil, ErrLineTooLong
		}
		return nil, err
	}
	return wordlist, nil
}

// GetFilePathByID returns the file path of a wordlist for download.
func (service *WordlistFacade) GetFilePathByID(id int) (string, error) {
	filePath, err := service.queryService.GetWordlistFilePathByID(context.Background(), id)
	if err != nil {
		return "", mapWordlistFileBoundaryError(err)
	}
	return filePath, nil
}
