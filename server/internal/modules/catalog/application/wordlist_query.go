package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

const (
	wordlistListDefaultPage     = 1
	wordlistListDefaultPageSize = 20
	wordlistListMaxPageSize     = 1000
	wordlistListTokenVersion    = 1
)

var wordlistQueryFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"fileName":    {Column: "file_name"},
	"description": {Column: "description"},
	"tags":        {Column: "tags", IsJSONBStringArray: true},
	"lineCount":   {Column: "line_count", IsNumeric: true},
	"fileSize":    {Column: "file_size", IsNumeric: true},
	"fileHash":    {Column: "file_hash"},
})

var wordlistOrderByFields = map[string]struct{}{
	"fileName":  {},
	"lineCount": {},
	"fileSize":  {},
	"updatedAt": {},
}

type WordlistQueryService struct {
	store     WordlistQueryStore
	fileStore WordlistFileStore
}

// WordlistListQueryInput carries the normalized HTTP list query shape into the application layer.
type WordlistListQueryInput struct {
	PageSize  int
	PageToken string
	Filter    string
	OrderBy   string
}

// WordlistListResult is the application result for the paginated wordlist collection.
type WordlistListResult struct {
	Wordlists     []catalogdomain.Wordlist
	TotalSize     int64
	Page          int
	PageSize      int
	NextPageToken string
}

type wordlistListPageTokenPayload struct {
	Version  int    `json:"v"`
	Page     int    `json:"p"`
	PageSize int    `json:"s"`
	Filter   string `json:"f"`
	OrderBy  string `json:"o"`
	Cursor   string `json:"c,omitempty"`
	ID       int    `json:"i,omitempty"`
}

func NewWordlistQueryService(store WordlistQueryStore, fileStore WordlistFileStore) *WordlistQueryService {
	return &WordlistQueryService{store: store, fileStore: fileStore}
}

func (service *WordlistQueryService) ListWordlists(ctx context.Context, input WordlistListQueryInput) (*WordlistListResult, error) {
	if service == nil || service.store == nil {
		return nil, fmt.Errorf("wordlist query store is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	pageSize := normalizeWordlistListPageSize(input.PageSize)
	filter := strings.TrimSpace(input.Filter)
	if err := scope.ValidateFilterDefault(filter, wordlistQueryFilterMapping, "fileName"); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedWordlistFilter, err)
	}
	normalizedOrderBy, err := normalizeWordlistOrderBy(input.OrderBy)
	if err != nil {
		return nil, err
	}

	page := wordlistListDefaultPage
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodeWordlistListPageToken(input.PageToken)
		if err != nil {
			return nil, err
		}
		if payload.Filter != filter || payload.OrderBy != normalizedOrderBy || payload.PageSize != pageSize {
			return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidWordlistPageToken)
		}
		page = payload.Page
	}

	var wordlists []catalogdomain.Wordlist
	var total int64
	if store, ok := service.store.(WordlistQueryStoreContext); ok {
		wordlists, total, err = store.ListContext(ctx, page, pageSize, filter, normalizedOrderBy)
	} else {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		wordlists, total, err = service.store.List(page, pageSize, filter, normalizedOrderBy)
	}
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	if total > 0 && int64(page*pageSize) < total {
		nextPageToken, err = encodeWordlistListPageToken(wordlistListPageTokenPayload{
			Version:  wordlistListTokenVersion,
			Page:     page + 1,
			PageSize: pageSize,
			Filter:   filter,
			OrderBy:  normalizedOrderBy,
		})
		if err != nil {
			return nil, err
		}
	}

	return &WordlistListResult{
		Wordlists:     wordlists,
		TotalSize:     total,
		Page:          page,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func normalizeWordlistListPageSize(pageSize int) int {
	if pageSize <= 0 {
		return wordlistListDefaultPageSize
	}
	if pageSize > wordlistListMaxPageSize {
		return wordlistListMaxPageSize
	}
	return pageSize
}

func normalizeWordlistOrderBy(orderBy string) (string, error) {
	trimmed := strings.TrimSpace(orderBy)
	if trimmed == "" {
		return "updatedAt desc", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedWordlistOrderBy, orderBy)
	}
	field := parts[0]
	if _, ok := wordlistOrderByFields[field]; !ok {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedWordlistOrderBy, orderBy)
	}
	direction := "asc"
	if field == "updatedAt" {
		direction = "desc"
	}
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	switch direction {
	case "asc":
		return field + " asc", nil
	case "desc":
		return field + " desc", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedWordlistOrderBy, orderBy)
	}
}

func encodeWordlistListPageToken(payload wordlistListPageTokenPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = wordlistListTokenVersion
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeWordlistListPageToken(token string) (wordlistListPageTokenPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return wordlistListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidWordlistPageToken)
	}
	var payload wordlistListPageTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return wordlistListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidWordlistPageToken)
	}
	if payload.Version != wordlistListTokenVersion || payload.Page <= 0 || payload.PageSize <= 0 {
		return wordlistListPageTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidWordlistPageToken)
	}
	return payload, nil
}

func (service *WordlistQueryService) ListWordlistTagSummaries(ctx context.Context, page, pageSize int, filter string) ([]catalogdomain.WordlistTagSummary, int64, error) {
	_ = ctx
	return service.store.ListTagSummaries(page, pageSize, filter)
}

func (service *WordlistQueryService) ListAllWordlists(ctx context.Context) ([]catalogdomain.Wordlist, error) {
	if service == nil || service.store == nil {
		return nil, fmt.Errorf("wordlist query store is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	var wordlists []catalogdomain.Wordlist
	var err error
	if store, ok := service.store.(WordlistQueryStoreContext); ok {
		wordlists, err = store.ListAllContext(ctx)
	} else {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		wordlists, err = service.store.ListAll()
	}
	if err != nil {
		return nil, err
	}

	for index := range wordlists {
		service.syncWordlistFileStats(&wordlists[index])
	}

	return wordlists, nil
}

func (service *WordlistQueryService) GetWordlistByID(ctx context.Context, id int) (*catalogdomain.Wordlist, error) {
	if service == nil || service.store == nil {
		return nil, fmt.Errorf("wordlist query store is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	var wordlist *catalogdomain.Wordlist
	var err error
	if _, ok := service.store.(WordlistQueryStoreContext); ok {
		// GetByIDContext is part of the base interface and is already used by
		// execution readers; call it here to preserve request cancellation.
		wordlist, err = service.store.GetByIDContext(ctx, id)
	} else {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		wordlist, err = service.store.GetByID(id)
	}
	if err != nil {
		return nil, err
	}
	if wordlist == nil {
		return nil, ErrWordlistNotFound
	}

	service.syncWordlistFileStats(wordlist)
	return wordlist, nil
}

// GetExecutionWordlistByResourceName returns persisted immutable metadata without the
// legacy UI path's best-effort file-stat refresh and database write-back.
func (service *WordlistQueryService) GetExecutionWordlistByResourceName(ctx context.Context, resourceName string) (*catalogdomain.Wordlist, error) {
	if service == nil || service.store == nil {
		return nil, fmt.Errorf("wordlist query store is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, err := resourcenames.ParseWordlist(resourceName)
	if err != nil || resourcenames.Wordlist(id) != strings.TrimSpace(resourceName) {
		return nil, ErrWordlistNotFound
	}
	return service.store.GetByIDContext(ctx, id)
}

// GetExecutionWordlistFilePath resolves the persisted file path under the
// caller context without refreshing or mutating catalog metadata.
func (service *WordlistQueryService) GetExecutionWordlistFilePathByResourceName(ctx context.Context, resourceName string) (string, error) {
	wordlist, err := service.GetExecutionWordlistByResourceName(ctx, resourceName)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !service.fileStore.Exists(wordlist.FilePath) {
		return "", ErrFileNotFound
	}
	return wordlist.FilePath, nil
}

func (service *WordlistQueryService) GetWordlistFilePathByID(ctx context.Context, id int) (string, error) {
	wordlist, err := service.GetWordlistByID(ctx, id)
	if err != nil {
		return "", err
	}

	if !service.fileStore.Exists(wordlist.FilePath) {
		return "", ErrFileNotFound
	}

	return wordlist.FilePath, nil
}

func (service *WordlistQueryService) syncWordlistFileStats(wordlist *catalogdomain.Wordlist) {
	if service == nil || wordlist == nil || service.fileStore == nil || service.store == nil {
		return
	}
	metadata, changed, err := service.fileStore.RefreshMetadata(wordlist.FilePath, wordlist.FileSize, wordlist.UpdatedAt)
	if err != nil || !changed || metadata == nil {
		return
	}

	wordlist.UpdateFileStats(metadata.FileSize, metadata.LineCount, metadata.FileHash)
	_ = service.store.Update(wordlist)
}
