package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
)

type boundaryScreenshotReader struct{ image []byte }

func (reader boundaryScreenshotReader) ListByTarget(context.Context, int, ScreenshotQuery) (Page[ScreenshotRecord], error) {
	return Page[ScreenshotRecord]{}, nil
}

func (reader boundaryScreenshotReader) GetImage(context.Context, int, int) ([]byte, error) {
	return append([]byte(nil), reader.image...), nil
}

type boundaryServerLogReader struct{ query LogQuery }

func (reader *boundaryServerLogReader) List(_ context.Context, query LogQuery) (LogPage, error) {
	reader.query = query
	return LogPage{Items: []LogRecord{}}, nil
}

func TestScreenshotImageKeepsBinaryBelowWholeResultBudget(t *testing.T) {
	registry := NewRegistry(Dependencies{Screenshots: boundaryScreenshotReader{image: []byte{1, 2, 3}}})
	result, err := registry.getScreenshotImage(context.Background(), toolRequest(`{"target":"targets/7","screenshot":"targets/7/screenshots/3"}`))
	if err != nil || result == nil || result.IsError || len(result.Content) != 1 {
		t.Fatalf("small screenshot = %#v, %v", result, err)
	}
	image, ok := result.Content[0].(*mcp.ImageContent)
	if !ok || image.MIMEType != "image/webp" || string(image.Data) != string([]byte{1, 2, 3}) {
		t.Fatalf("screenshot content = %#v", result.Content)
	}

	registry = NewRegistry(Dependencies{Screenshots: boundaryScreenshotReader{image: make([]byte, MaxImageBytes+1)}})
	result, err = registry.getScreenshotImage(context.Background(), toolRequest(`{"target":"targets/7","screenshot":"targets/7/screenshots/3"}`))
	if err != nil || result == nil || !result.IsError || toolFailureCategory(t, result) != string(mcpErrors.CategoryResultTooLarge) {
		t.Fatalf("over-budget screenshot = %#v, %v", result, err)
	}
}

func TestLogHandlersNormalizeDirectionBeforeFixedSourceReader(t *testing.T) {
	reader := &boundaryServerLogReader{}
	registry := NewRegistry(Dependencies{ServerLogs: reader})
	result, err := registry.listServerLogEntries(context.Background(), toolRequest(`{"direction":"OLDER","page_size":17}`))
	if err != nil || result == nil || result.IsError {
		t.Fatalf("server log page = %#v, %v", result, err)
	}
	if reader.query.Direction != "older" || reader.query.PageSize != 17 {
		t.Fatalf("server log query = %+v", reader.query)
	}
}
