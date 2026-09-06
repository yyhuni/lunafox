package adapters

import (
	"context"
	"fmt"
	"strings"

	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type targetReader struct{ facade *catalogapp.TargetFacade }

// NewTargetReader creates the MCP projection for targets.
func NewTargetReader(facade *catalogapp.TargetFacade) tools.TargetReader {
	return &targetReader{facade: facade}
}

func (reader *targetReader) List(ctx context.Context, query tools.TargetQuery) (tools.Page[tools.TargetRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.TargetRecord]{}, mcpErrors.ErrInternal
	}
	filter := targetFilter(query)
	cursor, err := decodeListCursor(query.PageToken, tools.ToolListTargets, query.PageSize, struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		OrderBy string `json:"orderBy"`
	}{query.Name, query.Type, query.OrderBy})
	if err != nil {
		return tools.Page[tools.TargetRecord]{}, err
	}
	result, err := reader.facade.ListContext(ctx, catalogapp.TargetListQueryInput{
		PageSize: query.PageSize, PageToken: cursor.InnerToken, Filter: filter, OrderBy: query.OrderBy,
	})
	if err != nil {
		if errorsIsTargetInput(err) {
			return tools.Page[tools.TargetRecord]{}, invalidReaderInput()
		}
		return tools.Page[tools.TargetRecord]{}, mapReaderError(err, catalogapp.ErrTargetNotFound)
	}
	if result == nil {
		return tools.Page[tools.TargetRecord]{}, mcpErrors.ErrInternal
	}
	items := make([]tools.TargetRecord, 0, len(result.Targets))
	for _, target := range result.Targets {
		items = append(items, targetRecord(target, nil))
	}
	next := ""
	if result.NextPageToken != "" {
		next, err = encodeNextCursor(tools.ToolListTargets, query.PageSize, struct {
			Name    string `json:"name"`
			Type    string `json:"type"`
			OrderBy string `json:"orderBy"`
		}{query.Name, query.Type, query.OrderBy}, cursor.Page+1, result.NextPageToken)
		if err != nil {
			return tools.Page[tools.TargetRecord]{}, err
		}
	}
	return tools.Page[tools.TargetRecord]{Items: items, NextPageToken: next, TotalSize: result.TotalSize}, nil
}

func (reader *targetReader) Get(ctx context.Context, id int) (tools.TargetRecord, error) {
	if reader.facade == nil {
		return tools.TargetRecord{}, mcpErrors.ErrInternal
	}
	target, summary, err := reader.facade.GetDetailContext(ctx, id)
	if err != nil {
		return tools.TargetRecord{}, mapReaderError(err, catalogapp.ErrTargetNotFound)
	}
	if target == nil {
		return tools.TargetRecord{}, mcpErrors.ErrNotFound
	}
	return targetRecord(*target, summary), nil
}

func targetFilter(query tools.TargetQuery) string {
	if value := strings.TrimSpace(query.Filter); value != "" {
		return value
	}
	parts := make([]string, 0, 2)
	if value := strings.TrimSpace(query.Name); value != "" {
		parts = append(parts, `displayName==`+fmt.Sprintf("%q", value))
	}
	if value := strings.TrimSpace(query.Type); value != "" {
		parts = append(parts, `type==`+fmt.Sprintf("%q", value))
	}
	return strings.Join(parts, " && ")
}

func errorsIsTargetInput(err error) bool {
	return errorsIs(err, catalogapp.ErrUnsupportedTargetFilter, catalogapp.ErrUnsupportedTargetOrderBy, catalogapp.ErrInvalidTargetPageToken)
}

func targetRecord(target domain.Target, summary *catalogapp.TargetSummary) tools.TargetRecord {
	record := tools.TargetRecord{ID: target.ID, Name: target.Name, Type: target.Type, CreatedAt: target.CreatedAt.UTC(), LastScannedAt: utcPtr(target.LastScannedAt)}
	if summary == nil || summary.Vulnerabilities == nil {
		return record
	}
	record.Summary = &tools.TargetSummary{
		Subdomains: summary.Subdomains, Websites: summary.Websites, Endpoints: summary.Endpoints, IPs: summary.IPs,
		Directories: summary.Directories, Screenshots: summary.Screenshots,
		Vulnerabilities: tools.VulnerabilityCount{
			Total: summary.Vulnerabilities.Total, Critical: summary.Vulnerabilities.Critical, High: summary.Vulnerabilities.High,
			Medium: summary.Vulnerabilities.Medium, Low: summary.Vulnerabilities.Low,
		},
	}
	return record
}
