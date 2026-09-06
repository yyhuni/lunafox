package adapters

import (
	"context"
	"fmt"
	"strings"

	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

type websiteReader struct{ facade *assetapp.WebsiteFacade }
type subdomainReader struct{ facade *assetapp.SubdomainFacade }
type endpointReader struct{ facade *assetapp.EndpointFacade }
type directoryReader struct{ facade *assetapp.DirectoryFacade }
type hostPortReader struct{ facade *assetapp.HostPortFacade }

// NewWebsiteReader creates the MCP projection for website assets.
func NewWebsiteReader(facade *assetapp.WebsiteFacade) tools.WebsiteReader {
	return &websiteReader{facade: facade}
}

// NewSubdomainReader creates the MCP projection for subdomain assets.
func NewSubdomainReader(facade *assetapp.SubdomainFacade) tools.SubdomainReader {
	return &subdomainReader{facade: facade}
}

// NewEndpointReader creates the MCP projection for endpoint assets.
func NewEndpointReader(facade *assetapp.EndpointFacade) tools.EndpointReader {
	return &endpointReader{facade: facade}
}

// NewDirectoryReader creates the MCP projection for directory assets.
func NewDirectoryReader(facade *assetapp.DirectoryFacade) tools.DirectoryReader {
	return &directoryReader{facade: facade}
}

// NewHostPortReader creates the MCP projection for host-port assets.
func NewHostPortReader(facade *assetapp.HostPortFacade) tools.HostPortReader {
	return &hostPortReader{facade: facade}
}

func (reader *websiteReader) List(ctx context.Context, targetID int, query tools.AssetQuery) (tools.Page[tools.WebsiteRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.WebsiteRecord]{}, mcpErrors.ErrInternal
	}
	filter, err := websiteOrEndpointFilter(query)
	if err != nil {
		return tools.Page[tools.WebsiteRecord]{}, err
	}
	orderBy, err := assetOrder(query.OrderBy, "website")
	if err != nil {
		return tools.Page[tools.WebsiteRecord]{}, err
	}
	shape := assetQueryShape{TargetID: targetID, Filter: filter, OrderBy: orderBy}
	cursor, err := decodeListCursor(query.PageToken, tools.ToolListWebsites, query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.WebsiteRecord]{}, err
	}
	result, err := reader.facade.ListByTargetContext(ctx, targetID, assetapp.WebsiteListQueryInput{
		PageSize: query.PageSize, PageToken: cursor.InnerToken, Filter: filter, OrderBy: orderBy,
	})
	if err != nil {
		return tools.Page[tools.WebsiteRecord]{}, mapAssetListError(err)
	}
	if result == nil {
		// A nil result with no error is not a valid facade contract. Keep the
		// MCP boundary fail-closed instead of dereferencing it below.
		return tools.Page[tools.WebsiteRecord]{}, mcpErrors.ErrInternal
	}
	items := make([]tools.WebsiteRecord, 0, len(result.Websites))
	for _, website := range result.Websites {
		items = append(items, websiteRecord(website.Website))
	}
	next, err := assetNextCursor(tools.ToolListWebsites, query.PageSize, shape, cursor.Page+1, result.NextPageToken)
	if err != nil {
		return tools.Page[tools.WebsiteRecord]{}, err
	}
	return tools.Page[tools.WebsiteRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

func (reader *websiteReader) Get(context.Context, int) (tools.WebsiteRecord, error) {
	return tools.WebsiteRecord{}, fmt.Errorf("%w: website detail tools are not registered", mcpErrors.ErrInvalidInput)
}

func (reader *subdomainReader) List(ctx context.Context, targetID int, query tools.AssetQuery) (tools.Page[tools.SubdomainRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.SubdomainRecord]{}, mcpErrors.ErrInternal
	}
	filter := stringFilter("dnsName", query.DNSName)
	orderBy, err := assetOrder(query.OrderBy, "subdomain")
	if err != nil {
		return tools.Page[tools.SubdomainRecord]{}, err
	}
	shape := assetQueryShape{TargetID: targetID, Filter: filter, OrderBy: orderBy}
	cursor, err := decodeListCursor(query.PageToken, tools.ToolListSubdomains, query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.SubdomainRecord]{}, err
	}
	result, err := reader.facade.ListByTargetContext(ctx, targetID, assetapp.SubdomainListQueryInput{
		PageSize: query.PageSize, PageToken: cursor.InnerToken, Filter: filter, OrderBy: orderBy,
	})
	if err != nil {
		return tools.Page[tools.SubdomainRecord]{}, mapAssetListError(err)
	}
	if result == nil {
		return tools.Page[tools.SubdomainRecord]{}, mcpErrors.ErrInternal
	}
	items := make([]tools.SubdomainRecord, 0, len(result.Subdomains))
	for _, item := range result.Subdomains {
		items = append(items, subdomainRecord(item))
	}
	next, err := assetNextCursor(tools.ToolListSubdomains, query.PageSize, shape, cursor.Page+1, result.NextPageToken)
	if err != nil {
		return tools.Page[tools.SubdomainRecord]{}, err
	}
	return tools.Page[tools.SubdomainRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

func (reader *endpointReader) List(ctx context.Context, targetID int, query tools.AssetQuery) (tools.Page[tools.EndpointRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.EndpointRecord]{}, mcpErrors.ErrInternal
	}
	filter, err := websiteOrEndpointFilter(query)
	if err != nil {
		return tools.Page[tools.EndpointRecord]{}, err
	}
	orderBy, err := assetOrder(query.OrderBy, "endpoint")
	if err != nil {
		return tools.Page[tools.EndpointRecord]{}, err
	}
	shape := assetQueryShape{TargetID: targetID, Filter: filter, OrderBy: orderBy}
	cursor, err := decodeListCursor(query.PageToken, tools.ToolListEndpoints, query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.EndpointRecord]{}, err
	}
	result, err := reader.facade.ListByTargetContext(ctx, targetID, assetapp.EndpointListQueryInput{
		PageSize: query.PageSize, PageToken: cursor.InnerToken, Filter: filter, OrderBy: orderBy,
	})
	if err != nil {
		return tools.Page[tools.EndpointRecord]{}, mapAssetListError(err)
	}
	if result == nil {
		return tools.Page[tools.EndpointRecord]{}, mcpErrors.ErrInternal
	}
	items := make([]tools.EndpointRecord, 0, len(result.Endpoints))
	for _, item := range result.Endpoints {
		items = append(items, endpointRecord(item))
	}
	next, err := assetNextCursor(tools.ToolListEndpoints, query.PageSize, shape, cursor.Page+1, result.NextPageToken)
	if err != nil {
		return tools.Page[tools.EndpointRecord]{}, err
	}
	return tools.Page[tools.EndpointRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

func (reader *endpointReader) Get(context.Context, int) (tools.EndpointRecord, error) {
	return tools.EndpointRecord{}, fmt.Errorf("%w: endpoint detail tools are not registered", mcpErrors.ErrInvalidInput)
}

func (reader *directoryReader) List(ctx context.Context, targetID int, query tools.AssetQuery) (tools.Page[tools.DirectoryRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.DirectoryRecord]{}, mcpErrors.ErrInternal
	}
	filter := joinFilters(
		stringFilter("url", query.URL),
		intFilter("status", query.StatusCode),
		stringFilter("contentType", query.ContentType),
	)
	orderBy, err := assetOrder(query.OrderBy, "directory")
	if err != nil {
		return tools.Page[tools.DirectoryRecord]{}, err
	}
	shape := assetQueryShape{TargetID: targetID, Filter: filter, OrderBy: orderBy}
	cursor, err := decodeListCursor(query.PageToken, tools.ToolListDirectories, query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.DirectoryRecord]{}, err
	}
	result, err := reader.facade.ListByTargetContext(ctx, targetID, assetapp.DirectoryListQueryInput{
		PageSize: query.PageSize, PageToken: cursor.InnerToken, Filter: filter, OrderBy: orderBy,
	})
	if err != nil {
		return tools.Page[tools.DirectoryRecord]{}, mapAssetListError(err)
	}
	if result == nil {
		return tools.Page[tools.DirectoryRecord]{}, mcpErrors.ErrInternal
	}
	items := make([]tools.DirectoryRecord, 0, len(result.Directories))
	for _, item := range result.Directories {
		items = append(items, directoryRecord(item))
	}
	next, err := assetNextCursor(tools.ToolListDirectories, query.PageSize, shape, cursor.Page+1, result.NextPageToken)
	if err != nil {
		return tools.Page[tools.DirectoryRecord]{}, err
	}
	return tools.Page[tools.DirectoryRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

func (reader *hostPortReader) List(ctx context.Context, targetID int, query tools.AssetQuery) (tools.Page[tools.HostPortRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.HostPortRecord]{}, mcpErrors.ErrInternal
	}
	filter := joinFilters(stringFilter("ip", query.IP), stringFilter("host", query.Host), intFilter("port", query.Port))
	orderBy, err := assetOrder(query.OrderBy, "hostPort")
	if err != nil {
		return tools.Page[tools.HostPortRecord]{}, err
	}
	shape := assetQueryShape{TargetID: targetID, Filter: filter, OrderBy: orderBy}
	cursor, err := decodeListCursor(query.PageToken, tools.ToolListHostPorts, query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.HostPortRecord]{}, err
	}
	result, err := reader.facade.ListByTargetContext(ctx, targetID, assetapp.HostPortListQueryInput{
		PageSize: query.PageSize, PageToken: cursor.InnerToken, Filter: filter, OrderBy: orderBy,
	})
	if err != nil {
		return tools.Page[tools.HostPortRecord]{}, mapAssetListError(err)
	}
	if result == nil {
		return tools.Page[tools.HostPortRecord]{}, mcpErrors.ErrInternal
	}
	items := make([]tools.HostPortRecord, 0, len(result.HostPorts))
	for _, item := range result.HostPorts {
		items = append(items, tools.HostPortRecord{IP: item.IP, Hosts: append([]string(nil), item.Hosts...), Ports: append([]int(nil), item.Ports...), CreatedAt: item.CreatedAt.UTC()})
	}
	next, err := assetNextCursor(tools.ToolListHostPorts, query.PageSize, shape, cursor.Page+1, result.NextPageToken)
	if err != nil {
		return tools.Page[tools.HostPortRecord]{}, err
	}
	return tools.Page[tools.HostPortRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

type assetQueryShape struct {
	TargetID int    `json:"targetId"`
	Filter   string `json:"filter"`
	OrderBy  string `json:"orderBy"`
}

func assetNextCursor(resource string, pageSize int, shape assetQueryShape, page int, inner string) (string, error) {
	if inner == "" {
		return "", nil
	}
	return encodeNextCursor(resource, pageSize, shape, page, inner)
}

func websiteOrEndpointFilter(query tools.AssetQuery) (string, error) {
	if query.Host != "" {
		return "", invalidReaderInput()
	}
	return joinFilters(
		stringFilter("url", query.URL), intFilter("statusCode", query.StatusCode), stringFilter("webserver", query.Webserver),
		stringFilter("contentType", query.ContentType), stringFilter("tech", query.Tech), boolFilter("vhost", query.Vhost),
	), nil
}

func assetOrder(value, kind string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	allowed := map[string]map[string]struct{}{
		"website":   {"createdAt asc": {}, "createdAt desc": {}, "statusCode asc": {}, "statusCode desc": {}, "contentLength asc": {}, "contentLength desc": {}},
		"endpoint":  {"createdAt asc": {}, "createdAt desc": {}, "statusCode asc": {}, "statusCode desc": {}, "contentLength asc": {}, "contentLength desc": {}},
		"directory": {"createdAt asc": {}, "createdAt desc": {}, "statusCode asc": {}, "statusCode desc": {}, "contentLength asc": {}, "contentLength desc": {}},
		"subdomain": {"createdAt asc": {}, "createdAt desc": {}, "dnsName asc": {}, "dnsName desc": {}},
		"hostPort":  {"createdAt asc": {}, "createdAt desc": {}, "ip asc": {}, "ip desc": {}},
	}
	if _, ok := allowed[kind][trimmed]; !ok {
		return "", invalidReaderInput()
	}
	if kind == "directory" && strings.HasPrefix(trimmed, "statusCode") {
		return strings.Replace(trimmed, "statusCode", "status", 1), nil
	}
	return trimmed, nil
}

func stringFilter(field, value string) string {
	if value = strings.TrimSpace(value); value == "" {
		return ""
	}
	return field + "==" + fmt.Sprintf("%q", value)
}

func intFilter(field string, value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%s==%d", field, *value)
}

func boolFilter(field string, value *bool) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%s==%t", field, *value)
}

func joinFilters(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " && ")
}

func mapAssetListError(err error) error {
	if errorsIs(err,
		assetapp.ErrTargetNotFound,
		assetapp.ErrUnsupportedWebsiteFilter, assetapp.ErrUnsupportedWebsiteOrderBy, assetapp.ErrInvalidWebsitePageToken,
		assetapp.ErrUnsupportedSubdomainFilter, assetapp.ErrUnsupportedSubdomainOrderBy, assetapp.ErrInvalidSubdomainPageToken,
		assetapp.ErrUnsupportedEndpointFilter, assetapp.ErrUnsupportedEndpointOrderBy, assetapp.ErrInvalidEndpointPageToken,
		assetapp.ErrUnsupportedDirectoryFilter, assetapp.ErrUnsupportedDirectoryOrderBy, assetapp.ErrInvalidDirectoryPageToken,
		assetapp.ErrUnsupportedHostPortFilter, assetapp.ErrUnsupportedHostPortOrderBy, assetapp.ErrInvalidHostPortPageToken,
	) {
		if errorsIs(err, assetapp.ErrTargetNotFound) {
			return mapReaderError(err, assetapp.ErrTargetNotFound)
		}
		return invalidReaderInput()
	}
	return mapReaderError(err, assetapp.ErrTargetNotFound)
}
