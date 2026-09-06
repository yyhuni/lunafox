package application

import (
	"context"
	"errors"
	"reflect"

	"github.com/shopspring/decimal"
	contractresults "github.com/yyhuni/lunafox/contracts/results"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type resultIngestScope struct {
	taskID   int
	scanID   int
	targetID int
}

type materializationSummary struct {
	receivedItems      int
	duplicateItems     int
	scopeFilteredItems int
	unsupportedItems   int
	invalidItems       int
	snapshotCount      int64
	assetCount         int64
	currentOnlyCount   int
}

type preparedResultBatch struct {
	receivedItems int
	currentOnly   bool
	materialize   func(context.Context, resultIngestScope) (materializationSummary, error)
}

type resultIngestHandler interface {
	prepare(context.Context, [][]byte) (preparedResultBatch, error)
}

type resultIngestRegistry struct {
	handlers map[string]resultIngestHandler
}

func newResultIngestRegistry(deps ResultIngestFacadeDependencies) resultIngestRegistry {
	return resultIngestRegistry{handlers: map[string]resultIngestHandler{
		contractresults.ResultKindAssetSubdomain: newTypedResultIngestHandler(
			contractresults.DecodeSubdomainItems,
			func(item contractresults.Subdomain) snapshotapp.SubdomainSnapshotItem {
				return snapshotapp.SubdomainSnapshotItem{DNSName: item.DNSName}
			},
			func(ctx context.Context, scope resultIngestScope, items []snapshotapp.SubdomainSnapshotItem) (snapshotapp.MaterializationSummary, error) {
				if isNilResultDependency(deps.Subdomains) {
					return snapshotapp.MaterializationSummary{}, ErrResultMaterializerUnavailable
				}
				return deps.Subdomains.SaveAndSyncContext(ctx, scope.scanID, scope.targetID, items)
			},
		),
		contractresults.ResultKindAssetHostPort: newTypedResultIngestHandler(
			contractresults.DecodeHostPortItems,
			func(item contractresults.HostPort) snapshotapp.HostPortSnapshotItem {
				return snapshotapp.HostPortSnapshotItem{Host: item.Host, IP: item.IP, Port: item.Port}
			},
			func(ctx context.Context, scope resultIngestScope, items []snapshotapp.HostPortSnapshotItem) (snapshotapp.MaterializationSummary, error) {
				if isNilResultDependency(deps.HostPorts) {
					return snapshotapp.MaterializationSummary{}, ErrResultMaterializerUnavailable
				}
				return deps.HostPorts.SaveAndSyncContext(ctx, scope.scanID, scope.targetID, items)
			},
		),
		contractresults.ResultKindAssetWebsite: newTypedResultIngestHandler(
			contractresults.DecodeWebsiteItems,
			func(item contractresults.Website) snapshotapp.WebsiteSnapshotItem {
				return snapshotapp.WebsiteSnapshotItem{
					URL: item.URL, Host: item.Host, Title: item.Title, StatusCode: item.StatusCode,
					ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver,
					ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody,
					Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders,
				}
			},
			func(ctx context.Context, scope resultIngestScope, items []snapshotapp.WebsiteSnapshotItem) (snapshotapp.MaterializationSummary, error) {
				if isNilResultDependency(deps.Websites) {
					return snapshotapp.MaterializationSummary{}, ErrResultMaterializerUnavailable
				}
				return deps.Websites.SaveAndSyncContext(ctx, scope.scanID, scope.targetID, items)
			},
		),
		contractresults.ResultKindAssetWebsiteTechnology: newWebsiteTechnologyResultIngestHandler(deps.WebsiteTechnologies),
		contractresults.ResultKindAssetEndpoint: newTypedResultIngestHandler(
			contractresults.DecodeEndpointItems,
			func(item contractresults.Endpoint) snapshotapp.EndpointSnapshotItem {
				return snapshotapp.EndpointSnapshotItem{URL: item.URL, Host: item.Host, Title: item.Title, StatusCode: item.StatusCode, ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver, ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody, ResponseBodyTruncated: item.ResponseBodyTruncated, Vhost: item.Vhost, ResponseHeaders: item.ResponseHeaders, ResponseHeadersTruncated: item.ResponseHeadersTruncated}
			},
			func(ctx context.Context, scope resultIngestScope, items []snapshotapp.EndpointSnapshotItem) (snapshotapp.MaterializationSummary, error) {
				if isNilResultDependency(deps.Endpoints) {
					return snapshotapp.MaterializationSummary{}, ErrResultMaterializerUnavailable
				}
				return deps.Endpoints.SaveAndSyncContext(ctx, scope.scanID, scope.targetID, items)
			},
		),
		contractresults.ResultKindAssetDirectory: newTypedResultIngestHandler(
			contractresults.DecodeDirectoryItems,
			func(item contractresults.Directory) snapshotapp.DirectorySnapshotItem {
				return snapshotapp.DirectorySnapshotItem{
					URL: item.URL, Status: intPtr(item.Status), ContentLength: int64Ptr(item.ContentLength),
					ContentType: item.ContentType, Duration: int64Ptr(item.Duration),
				}
			},
			func(ctx context.Context, scope resultIngestScope, items []snapshotapp.DirectorySnapshotItem) (snapshotapp.MaterializationSummary, error) {
				if isNilResultDependency(deps.Directories) {
					return snapshotapp.MaterializationSummary{}, ErrResultMaterializerUnavailable
				}
				return deps.Directories.SaveResultBatchContext(ctx, scope.scanID, scope.targetID, items)
			},
		),
		contractresults.ResultKindAssetScreenshot: newTypedResultIngestHandler(
			contractresults.DecodeScreenshotItems,
			func(item contractresults.Screenshot) snapshotapp.ScreenshotSnapshotItem {
				return snapshotapp.ScreenshotSnapshotItem{URL: item.URL, StatusCode: int16Ptr(item.StatusCode), Image: append([]byte(nil), item.Image...)}
			},
			func(ctx context.Context, scope resultIngestScope, items []snapshotapp.ScreenshotSnapshotItem) (snapshotapp.MaterializationSummary, error) {
				if isNilResultDependency(deps.Screenshots) {
					return snapshotapp.MaterializationSummary{}, ErrResultMaterializerUnavailable
				}
				return deps.Screenshots.SaveResultBatchContext(ctx, scope.scanID, scope.targetID, items)
			},
		),
		contractresults.ResultKindAssetVulnerability: newTypedResultIngestHandler(
			contractresults.DecodeVulnerabilityItems,
			func(item contractresults.Vulnerability) snapshotapp.VulnerabilitySnapshotItem {
				var score *decimal.Decimal
				if item.CVSSScore != nil {
					value := decimal.NewFromFloat(*item.CVSSScore)
					score = &value
				}
				return snapshotapp.VulnerabilitySnapshotItem{
					URL: item.URL, VulnType: item.VulnType, Severity: item.Severity,
					Source: item.Source, CVSSScore: score, Description: item.Description,
					RawOutput: cloneJSONMap(item.RawOutput),
				}
			},
			func(ctx context.Context, scope resultIngestScope, items []snapshotapp.VulnerabilitySnapshotItem) (snapshotapp.MaterializationSummary, error) {
				if isNilResultDependency(deps.Vulnerabilities) {
					return snapshotapp.MaterializationSummary{}, ErrResultMaterializerUnavailable
				}
				return deps.Vulnerabilities.SaveResultBatchContext(ctx, scope.scanID, scope.targetID, items)
			},
		),
	}}
}

func cloneJSONMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func newWebsiteTechnologyResultIngestHandler(materializer websiteTechnologyResultMaterializer) resultIngestHandler {
	return &websiteTechnologyResultIngestHandler{materializer: materializer}
}

type websiteTechnologyResultIngestHandler struct {
	materializer websiteTechnologyResultMaterializer
}

func (handler *websiteTechnologyResultIngestHandler) prepare(ctx context.Context, encoded [][]byte) (preparedResultBatch, error) {
	if ctx == nil {
		return preparedResultBatch{}, ErrResultIngestContextRequired
	}
	if handler == nil || isNilResultDependency(handler.materializer) {
		return preparedResultBatch{}, ErrResultMaterializerUnavailable
	}
	materialized := make([]assetapp.WebsiteTechnologyUpsertItem, 0, len(encoded))
	for index := range encoded {
		if err := ctx.Err(); err != nil {
			return preparedResultBatch{}, err
		}
		items, err := contractresults.DecodeWebsiteTechnologyItems([]string{string(encoded[index])})
		if err != nil {
			// The complete batch is validated before the transaction callback is
			// created. A malformed final item must have no materialization effect.
			return preparedResultBatch{}, err
		}
		if len(items) != 1 {
			return preparedResultBatch{}, errors.New("canonical decoder returned an unexpected item count")
		}
		item := items[0]
		materialized = append(materialized, assetdomain.WebsiteTechnology{URL: item.URL, Tech: append([]string(nil), item.Tech...)})
	}
	return preparedResultBatch{
		receivedItems: len(encoded),
		currentOnly:   true,
		materialize: func(materializeCtx context.Context, scope resultIngestScope) (materializationSummary, error) {
			if err := materializeCtx.Err(); err != nil {
				return materializationSummary{}, err
			}
			result, err := handler.materializer.BatchUpsertTechnologyContext(materializeCtx, scope.targetID, materialized)
			return materializationSummary{
				receivedItems:      result.ReceivedItems,
				duplicateItems:     result.DuplicateItems,
				scopeFilteredItems: result.ScopeFilteredItems,
				assetCount:         result.AssetCount,
				currentOnlyCount:   result.ReceivedItems - result.DuplicateItems - result.ScopeFilteredItems,
			}, err
		},
	}, nil
}

func int16Ptr(value *int) *int16 {
	if value == nil {
		return nil
	}
	converted := int16(*value)
	return &converted
}

func intPtr(value int) *int {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func (registry resultIngestRegistry) lookup(resultType string) (resultIngestHandler, bool) {
	if _, ok := contractresults.Lookup(resultType); !ok {
		return nil, false
	}
	handler, ok := registry.handlers[resultType]
	return handler, ok && handler != nil
}

type typedResultIngestHandler[ContractItem, MaterializedItem any] struct {
	decode      func([]string) ([]ContractItem, error)
	mapItem     func(ContractItem) MaterializedItem
	materialize func(context.Context, resultIngestScope, []MaterializedItem) (snapshotapp.MaterializationSummary, error)
}

func newTypedResultIngestHandler[ContractItem, MaterializedItem any](
	decode func([]string) ([]ContractItem, error),
	mapItem func(ContractItem) MaterializedItem,
	materialize func(context.Context, resultIngestScope, []MaterializedItem) (snapshotapp.MaterializationSummary, error),
) resultIngestHandler {
	handler := &typedResultIngestHandler[ContractItem, MaterializedItem]{decode: decode, mapItem: mapItem, materialize: materialize}
	return handler
}

func (handler *typedResultIngestHandler[ContractItem, MaterializedItem]) prepare(ctx context.Context, encoded [][]byte) (preparedResultBatch, error) {
	if ctx == nil {
		return preparedResultBatch{}, ErrResultIngestContextRequired
	}
	if handler == nil || handler.decode == nil || handler.mapItem == nil || handler.materialize == nil {
		return preparedResultBatch{}, ErrResultMaterializerUnavailable
	}
	items := make([]MaterializedItem, 0, len(encoded))
	for index := range encoded {
		if err := ctx.Err(); err != nil {
			return preparedResultBatch{}, err
		}
		decoded, err := handler.decode([]string{string(encoded[index])})
		if err != nil {
			// Registry validation only accepts or rejects. It cannot filter a
			// malformed item and then materialize the rest of the batch.
			return preparedResultBatch{}, err
		}
		if len(decoded) != 1 {
			return preparedResultBatch{}, errors.New("canonical decoder returned an unexpected item count")
		}
		items = append(items, handler.mapItem(decoded[0]))
	}
	return preparedResultBatch{
		receivedItems: len(encoded),
		materialize: func(materializeCtx context.Context, scope resultIngestScope) (materializationSummary, error) {
			if materializeCtx == nil {
				return materializationSummary{}, ErrResultIngestContextRequired
			}
			if err := materializeCtx.Err(); err != nil {
				return materializationSummary{}, err
			}
			summary, err := handler.materialize(materializeCtx, scope, items)
			return materializationSummary{
				receivedItems: summary.ReceivedItems, duplicateItems: summary.DuplicateItems,
				scopeFilteredItems: summary.ScopeFilteredItems, unsupportedItems: summary.UnsupportedItems,
				invalidItems: summary.InvalidItems, snapshotCount: summary.SnapshotCount, assetCount: summary.AssetCount,
			}, err
		},
	}, nil
}

func isNilResultDependency(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
