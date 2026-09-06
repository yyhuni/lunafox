package application

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

const (
	globalAssetSearchDefaultPageSize = 10
	globalAssetSearchMaxPageSize     = 100
	globalAssetSearchTokenVersion    = 1
)

var (
	// ErrInvalidGlobalAssetSearchQuery is returned before a repository call when
	// a global-search request cannot satisfy the bounded query contract.
	ErrInvalidGlobalAssetSearchQuery = errors.New("invalid global asset search query")
	// ErrInvalidGlobalAssetSearchPageToken is returned when an opaque cursor is
	// malformed or was issued for another query shape.
	ErrInvalidGlobalAssetSearchPageToken = errors.New("invalid global asset search pageToken")
	// ErrGlobalAssetSearchTimeout prevents PostgreSQL cancellation details from
	// crossing the application boundary as an implementation-specific error.
	ErrGlobalAssetSearchTimeout = errors.New("global asset search timed out")
)

// GlobalAssetSearchAssetType selects one current-state asset table.
type GlobalAssetSearchAssetType string

const (
	GlobalAssetSearchAssetTypeWebsite  GlobalAssetSearchAssetType = "website"
	GlobalAssetSearchAssetTypeEndpoint GlobalAssetSearchAssetType = "endpoint"
)

// GlobalAssetSearchMode distinguishes an exact ordinary URL lookup from the
// strict DSL.
type GlobalAssetSearchMode string

const (
	GlobalAssetSearchModePlainURL   GlobalAssetSearchMode = "plain_url"
	GlobalAssetSearchModeStructured GlobalAssetSearchMode = "structured"
)

// GlobalAssetSearchField is an approved structured-search field.
type GlobalAssetSearchField string

const (
	GlobalAssetSearchFieldURL        GlobalAssetSearchField = "url"
	GlobalAssetSearchFieldHost       GlobalAssetSearchField = "host"
	GlobalAssetSearchFieldTitle      GlobalAssetSearchField = "title"
	GlobalAssetSearchFieldStatusCode GlobalAssetSearchField = "statusCode"
	GlobalAssetSearchFieldTech       GlobalAssetSearchField = "tech"
)

// GlobalAssetSearchOperator controls the approved field-specific match mode.
// URL conditions are always converted to Exact during parsing so a future
// repository caller cannot accidentally widen an observed-URL lookup.
type GlobalAssetSearchOperator string

const (
	GlobalAssetSearchOperatorContains GlobalAssetSearchOperator = "="
	GlobalAssetSearchOperatorExact    GlobalAssetSearchOperator = "=="
)

// GlobalAssetSearchCondition is a fully typed structured-search predicate.
// StatusCode is populated only for statusCode predicates; Text is populated
// for the four string-backed fields.
type GlobalAssetSearchCondition struct {
	Field      GlobalAssetSearchField
	Operator   GlobalAssetSearchOperator
	Text       string
	StatusCode *int
}

// GlobalAssetSearchAST is the query representation accepted by the repository.
// It intentionally has no generic expression nodes because P0 only permits an
// exact plain URL predicate or a flat conjunction of typed conditions.
type GlobalAssetSearchAST struct {
	Mode       GlobalAssetSearchMode
	PlainURL   string
	Conditions []GlobalAssetSearchCondition
}

// GlobalAssetSearchInput is the application input for the global search API.
type GlobalAssetSearchInput struct {
	Query     string
	AssetType GlobalAssetSearchAssetType
	// PageSize is nil only when the caller omitted pageSize. Keeping that
	// distinction prevents an explicit zero from silently becoming page 1.
	PageSize  *int
	PageToken string
}

// GlobalAssetSearchCursor is the stable descending-sort cursor persisted in a
// search page token.
type GlobalAssetSearchCursor struct {
	CreatedAt time.Time
	ID        int
}

// GlobalAssetSearchStoreQuery is passed only after all public request
// validation and page-token binding are complete.
type GlobalAssetSearchStoreQuery struct {
	AST      GlobalAssetSearchAST
	PageSize int
	Cursor   *GlobalAssetSearchCursor
}

// GlobalWebsiteSearchStore executes a query against the Website current-state
// table only.
type GlobalWebsiteSearchStore interface {
	SearchGlobalWebsites(ctx context.Context, query GlobalAssetSearchStoreQuery) ([]assetdomain.Website, error)
}

// GlobalEndpointSearchStore executes a query against the Endpoint current-state
// table only.
type GlobalEndpointSearchStore interface {
	SearchGlobalEndpoints(ctx context.Context, query GlobalAssetSearchStoreQuery) ([]assetdomain.Endpoint, error)
}

// GlobalAssetSearchResult contains exactly one selected asset type and no
// count-derived metadata.
type GlobalAssetSearchResult struct {
	AssetType     GlobalAssetSearchAssetType
	Websites      []assetdomain.Website
	Endpoints     []assetdomain.Endpoint
	NextPageToken string
}

// GlobalAssetSearchService validates and dispatches current-state search
// requests without widening existing Target-scoped list services.
type GlobalAssetSearchService struct {
	websiteStore  GlobalWebsiteSearchStore
	endpointStore GlobalEndpointSearchStore
}

// NewGlobalAssetSearchService creates the single-type global search service.
func NewGlobalAssetSearchService(websiteStore GlobalWebsiteSearchStore, endpointStore GlobalEndpointSearchStore) *GlobalAssetSearchService {
	return &GlobalAssetSearchService{websiteStore: websiteStore, endpointStore: endpointStore}
}

// Search validates the request before selecting exactly one backing table.
func (service *GlobalAssetSearchService) Search(ctx context.Context, input GlobalAssetSearchInput) (*GlobalAssetSearchResult, error) {
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ast, err := ParseGlobalAssetSearchQuery(input.Query)
	if err != nil {
		return nil, err
	}
	if !isGlobalAssetSearchAssetType(input.AssetType) {
		return nil, fmt.Errorf("%w: assetType must be website or endpoint", ErrInvalidGlobalAssetSearchQuery)
	}

	pageSize, err := normalizeGlobalAssetSearchPageSize(input.PageSize)
	if err != nil {
		return nil, err
	}
	cursor, err := decodeAndValidateGlobalAssetSearchPageToken(input.PageToken, input.AssetType, ast, pageSize)
	if err != nil {
		return nil, err
	}
	query := GlobalAssetSearchStoreQuery{AST: ast, PageSize: pageSize, Cursor: cursor}

	result := &GlobalAssetSearchResult{AssetType: input.AssetType}
	switch input.AssetType {
	case GlobalAssetSearchAssetTypeWebsite:
		if service.websiteStore == nil {
			return nil, errors.New("global Website search store unavailable")
		}
		items, err := service.websiteStore.SearchGlobalWebsites(ctx, query)
		if err != nil {
			return nil, mapGlobalAssetSearchStoreError(err)
		}
		result.Websites, result.NextPageToken, err = finalizeGlobalWebsiteSearchPage(items, input.AssetType, ast, pageSize)
		if err != nil {
			return nil, err
		}
	case GlobalAssetSearchAssetTypeEndpoint:
		if service.endpointStore == nil {
			return nil, errors.New("global Endpoint search store unavailable")
		}
		items, err := service.endpointStore.SearchGlobalEndpoints(ctx, query)
		if err != nil {
			return nil, mapGlobalAssetSearchStoreError(err)
		}
		result.Endpoints, result.NextPageToken, err = finalizeGlobalEndpointSearchPage(items, input.AssetType, ast, pageSize)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func isGlobalAssetSearchAssetType(assetType GlobalAssetSearchAssetType) bool {
	return assetType == GlobalAssetSearchAssetTypeWebsite || assetType == GlobalAssetSearchAssetTypeEndpoint
}

func normalizeGlobalAssetSearchPageSize(pageSize *int) (int, error) {
	if pageSize == nil {
		return globalAssetSearchDefaultPageSize, nil
	}
	if *pageSize < 1 || *pageSize > globalAssetSearchMaxPageSize {
		return 0, fmt.Errorf("%w: pageSize must be between 1 and %d", ErrInvalidGlobalAssetSearchQuery, globalAssetSearchMaxPageSize)
	}
	return *pageSize, nil
}

func mapGlobalAssetSearchStoreError(err error) error {
	if errors.Is(err, ErrGlobalAssetSearchTimeout) {
		return ErrGlobalAssetSearchTimeout
	}
	return err
}

func finalizeGlobalWebsiteSearchPage(items []assetdomain.Website, assetType GlobalAssetSearchAssetType, ast GlobalAssetSearchAST, pageSize int) ([]assetdomain.Website, string, error) {
	if len(items) <= pageSize {
		return items, "", nil
	}
	page := append([]assetdomain.Website(nil), items[:pageSize]...)
	last := page[len(page)-1]
	token, err := encodeGlobalAssetSearchPageToken(assetType, ast, pageSize, GlobalAssetSearchCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	if err != nil {
		return nil, "", err
	}
	return page, token, nil
}

func finalizeGlobalEndpointSearchPage(items []assetdomain.Endpoint, assetType GlobalAssetSearchAssetType, ast GlobalAssetSearchAST, pageSize int) ([]assetdomain.Endpoint, string, error) {
	if len(items) <= pageSize {
		return items, "", nil
	}
	page := append([]assetdomain.Endpoint(nil), items[:pageSize]...)
	last := page[len(page)-1]
	token, err := encodeGlobalAssetSearchPageToken(assetType, ast, pageSize, GlobalAssetSearchCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	if err != nil {
		return nil, "", err
	}
	return page, token, nil
}

type globalAssetSearchPageToken struct {
	Version     int    `json:"v"`
	AssetType   string `json:"a"`
	QueryDigest string `json:"q"`
	PageSize    int    `json:"s"`
	CreatedAt   string `json:"c"`
	ID          int    `json:"i"`
}

func encodeGlobalAssetSearchPageToken(assetType GlobalAssetSearchAssetType, ast GlobalAssetSearchAST, pageSize int, cursor GlobalAssetSearchCursor) (string, error) {
	if cursor.ID <= 0 || cursor.CreatedAt.IsZero() {
		return "", fmt.Errorf("%w: incomplete cursor", ErrInvalidGlobalAssetSearchPageToken)
	}
	payload := globalAssetSearchPageToken{
		Version:     globalAssetSearchTokenVersion,
		AssetType:   string(assetType),
		QueryDigest: globalAssetSearchQueryDigest(ast),
		PageSize:    pageSize,
		CreatedAt:   cursor.CreatedAt.UTC().Format(time.RFC3339Nano),
		ID:          cursor.ID,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeAndValidateGlobalAssetSearchPageToken(raw string, assetType GlobalAssetSearchAssetType, ast GlobalAssetSearchAST, pageSize int) (*GlobalAssetSearchCursor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: malformed", ErrInvalidGlobalAssetSearchPageToken)
	}
	var payload globalAssetSearchPageToken
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("%w: malformed", ErrInvalidGlobalAssetSearchPageToken)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, payload.CreatedAt)
	if err != nil || createdAt.IsZero() || payload.Version != globalAssetSearchTokenVersion || payload.ID <= 0 || payload.PageSize < 1 || payload.QueryDigest == "" || !isGlobalAssetSearchAssetType(GlobalAssetSearchAssetType(payload.AssetType)) {
		return nil, fmt.Errorf("%w: malformed", ErrInvalidGlobalAssetSearchPageToken)
	}
	if payload.AssetType != string(assetType) || payload.QueryDigest != globalAssetSearchQueryDigest(ast) || payload.PageSize != pageSize {
		return nil, fmt.Errorf("%w: query shape mismatch", ErrInvalidGlobalAssetSearchPageToken)
	}
	return &GlobalAssetSearchCursor{CreatedAt: createdAt.UTC(), ID: payload.ID}, nil
}

func globalAssetSearchQueryDigest(ast GlobalAssetSearchAST) string {
	canonical := globalAssetSearchCanonicalQuery(ast)
	digest := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(digest[:])
}

func globalAssetSearchCanonicalQuery(ast GlobalAssetSearchAST) string {
	if ast.Mode == GlobalAssetSearchModePlainURL {
		return "plain:url=" + strconvQuote(ast.PlainURL)
	}
	conditions := make([]string, 0, len(ast.Conditions))
	for _, condition := range ast.Conditions {
		value := condition.Text
		if condition.StatusCode != nil {
			value = fmt.Sprintf("%d", *condition.StatusCode)
		}
		conditions = append(conditions, string(condition.Field)+string(condition.Operator)+strconvQuote(value))
	}
	sort.Strings(conditions)
	return "structured:" + strings.Join(conditions, "&&")
}

func strconvQuote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
