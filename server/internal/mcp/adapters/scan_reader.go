package adapters

import (
	"context"
	"fmt"
	"strings"

	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

type scanReader struct{ facade *scanapp.ScanFacade }

// NewScanReader creates the MCP projection for scan query records.
func NewScanReader(facade *scanapp.ScanFacade) tools.ScanReader { return &scanReader{facade: facade} }

func (reader *scanReader) List(ctx context.Context, query tools.ScanQuery) (tools.Page[tools.ScanRecord], error) {
	if reader.facade == nil {
		return tools.Page[tools.ScanRecord]{}, mcpErrors.ErrInternal
	}
	if !validScanStatus(query.Status) {
		return tools.Page[tools.ScanRecord]{}, invalidReaderInput()
	}
	shape := struct {
		TargetID int    `json:"targetId"`
		Status   string `json:"status"`
		OrderBy  string `json:"orderBy"`
	}{query.TargetID, query.Status, query.OrderBy}
	cursor, err := decodeListCursor(query.PageToken, tools.ToolListScans, query.PageSize, shape)
	if err != nil {
		return tools.Page[tools.ScanRecord]{}, err
	}
	items, total, err := reader.facade.ListContext(ctx, scanapp.ScanListFilter{
		Page: cursor.Page, PageSize: query.PageSize, TargetID: query.TargetID, Status: query.Status, OrderBy: query.OrderBy,
	})
	if err != nil {
		if errorsIs(err, scanapp.ErrUnsupportedScanFilter, scanapp.ErrUnsupportedScanOrderBy) {
			return tools.Page[tools.ScanRecord]{}, invalidReaderInput()
		}
		return tools.Page[tools.ScanRecord]{}, mapReaderError(err, scanapp.ErrScanNotFound, scanapp.ErrTargetNotFound)
	}
	records := make([]tools.ScanRecord, 0, len(items))
	for _, item := range items {
		records = append(records, scanRecord(item))
	}
	next := ""
	if int64(cursor.Page*query.PageSize) < total {
		next, err = encodeNextCursor(tools.ToolListScans, query.PageSize, shape, cursor.Page+1, "")
		if err != nil {
			return tools.Page[tools.ScanRecord]{}, err
		}
	}
	return tools.Page[tools.ScanRecord]{Items: records, NextPageToken: next, TotalSize: total}, nil
}

func (reader *scanReader) Get(ctx context.Context, id int) (tools.ScanRecord, error) {
	if reader.facade == nil {
		return tools.ScanRecord{}, mcpErrors.ErrInternal
	}
	item, err := reader.facade.GetByIDContext(ctx, id)
	if err != nil {
		return tools.ScanRecord{}, mapReaderError(err, scanapp.ErrScanNotFound)
	}
	if item == nil {
		return tools.ScanRecord{}, mcpErrors.ErrNotFound
	}
	return scanRecord(*item), nil
}

func validScanStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case "", string(scandomain.ScanStatusPending), string(scandomain.ScanStatusRunning), string(scandomain.ScanStatusSucceeded), string(scandomain.ScanStatusFailed), string(scandomain.ScanStatusCancelled):
		return true
	default:
		return false
	}
}

func scanFilterError(value string) error {
	return fmt.Errorf("%w: invalid scan status %q", mcpErrors.ErrInvalidInput, value)
}
