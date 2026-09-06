package adapters

import (
	"strings"
	"testing"

	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
)

func TestListCursorIsOpaqueAndBoundToQueryShape(t *testing.T) {
	shape := struct {
		TargetID int    `json:"targetId"`
		Filter   string `json:"filter"`
	}{TargetID: 7, Filter: `url=="https://example.test"`}
	token, err := encodeNextCursor(tools.ToolListWebsites, 20, shape, 2, "inner-cursor")
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	if strings.Contains(token, "example.test") || strings.Contains(token, "inner-cursor") {
		t.Fatalf("cursor exposes query or underlying pagination data: %q", token)
	}

	cursor, err := decodeListCursor(token, tools.ToolListWebsites, 20, shape)
	if err != nil || cursor.Page != 2 || cursor.InnerToken != "inner-cursor" {
		t.Fatalf("decode matching cursor = %#v, %v", cursor, err)
	}

	for _, mismatch := range []struct {
		name     string
		resource string
		pageSize int
		shape    any
	}{
		{name: "resource", resource: tools.ToolListEndpoints, pageSize: 20, shape: shape},
		{name: "page size", resource: tools.ToolListWebsites, pageSize: 21, shape: shape},
		{name: "target", resource: tools.ToolListWebsites, pageSize: 20, shape: struct {
			TargetID int    `json:"targetId"`
			Filter   string `json:"filter"`
		}{TargetID: 8, Filter: shape.Filter}},
		{name: "filter", resource: tools.ToolListWebsites, pageSize: 20, shape: struct {
			TargetID int    `json:"targetId"`
			Filter   string `json:"filter"`
		}{TargetID: 7, Filter: `url=="https://other.test"`}},
	} {
		t.Run(mismatch.name, func(t *testing.T) {
			_, err := decodeListCursor(token, mismatch.resource, mismatch.pageSize, mismatch.shape)
			if err == nil || !strings.Contains(err.Error(), mcpErrors.ErrInvalidInput.Error()) {
				t.Fatalf("mismatched cursor error = %v", err)
			}
		})
	}
}

func TestWrapInnerCursorDoesNotInventContinuationForAnEmptyOrLastPage(t *testing.T) {
	shape := struct{ TargetID int }{TargetID: 7}
	for _, page := range []int{1, 2} {
		token, err := wrapInnerCursor("targetScreenshots", 20, shape, page, "")
		if err != nil || token != "" {
			t.Fatalf("empty inner cursor at page %d = %q, %v", page, token, err)
		}
	}
	token, err := wrapInnerCursor("targetScreenshots", 20, shape, 2, "underlying-next")
	if err != nil || token == "" {
		t.Fatalf("present inner cursor = %q, %v", token, err)
	}
}
