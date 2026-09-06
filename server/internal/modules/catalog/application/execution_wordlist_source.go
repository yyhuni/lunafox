package application

import (
	"context"
	"errors"
)

var errExecutionWordlistSourceNotInitialized = errors.New("execution wordlist source is not initialized")

// ExecutionWordlistSource exposes only context-aware catalog reads used while
// authorizing and streaming one saved execution plan's immutable wordlist.
type ExecutionWordlistSource struct {
	queryService *WordlistQueryService
}

// NewExecutionWordlistSource binds execution-only reads to the catalog query service.
func NewExecutionWordlistSource(queryService *WordlistQueryService) *ExecutionWordlistSource {
	return &ExecutionWordlistSource{queryService: queryService}
}

// GetByResourceName returns persisted wordlist metadata under the caller context.
func (source *ExecutionWordlistSource) GetByResourceName(ctx context.Context, resourceName string) (*Wordlist, error) {
	if source == nil || source.queryService == nil {
		return nil, errExecutionWordlistSourceNotInitialized
	}
	wordlist, err := source.queryService.GetExecutionWordlistByResourceName(ctx, resourceName)
	if err != nil {
		return nil, mapWordlistBoundaryError(err)
	}
	return wordlist, nil
}

// GetFilePathByResourceName returns the persisted wordlist path under the caller context.
func (source *ExecutionWordlistSource) GetFilePathByResourceName(ctx context.Context, resourceName string) (string, error) {
	if source == nil || source.queryService == nil {
		return "", errExecutionWordlistSourceNotInitialized
	}
	path, err := source.queryService.GetExecutionWordlistFilePathByResourceName(ctx, resourceName)
	if err != nil {
		return "", mapWordlistFileBoundaryError(err)
	}
	return path, nil
}
