package application

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/loki"
)

type lokiQueryClientStub struct {
	fn func(ctx context.Context, input loki.QueryRangeRequest) ([]loki.StreamResult, error)
}

func (stub *lokiQueryClientStub) QueryRange(ctx context.Context, input loki.QueryRangeRequest) ([]loki.StreamResult, error) {
	if stub.fn != nil {
		return stub.fn(ctx, input)
	}
	return nil, nil
}

type timelineEntry struct {
	tsNs   string
	line   string
	source string
}

type timelineLokiClientStub struct {
	entries []timelineEntry
}

func (stub *timelineLokiClientStub) SetEntries(entries []timelineEntry) {
	stub.entries = make([]timelineEntry, len(entries))
	copy(stub.entries, entries)
}

func (stub *timelineLokiClientStub) QueryRange(_ context.Context, input loki.QueryRangeRequest) ([]loki.StreamResult, error) {
	filtered := make([]loki.StreamValue, 0, len(stub.entries))
	for _, entry := range stub.entries {
		if input.StartNs != "" && loki.ComparePositiveNumericStrings(entry.tsNs, input.StartNs) < 0 {
			continue
		}
		if input.EndNs != "" && loki.ComparePositiveNumericStrings(entry.tsNs, input.EndNs) > 0 {
			continue
		}
		filtered = append(filtered, loki.StreamValue{
			TsNs: entry.tsNs,
			Line: entry.line,
		})
	}

	if strings.EqualFold(input.Direction, "BACKWARD") {
		for left, right := 0, len(filtered)-1; left < right; left, right = left+1, right-1 {
			filtered[left], filtered[right] = filtered[right], filtered[left]
		}
	}

	if input.Limit > 0 && len(filtered) > input.Limit {
		filtered = filtered[:input.Limit]
	}

	return []loki.StreamResult{
		{
			Stream: map[string]string{"source": "stdout"},
			Values: filtered,
		},
	}, nil
}

func TestLokiLogQueryServiceCursorRoundTripNoNewLogs(t *testing.T) {
	client := &lokiQueryClientStub{
		fn: func(context.Context, loki.QueryRangeRequest) ([]loki.StreamResult, error) {
			return []loki.StreamResult{
				{
					Stream: map[string]string{"source": "stdout"},
					Values: []loki.StreamValue{
						{TsNs: "1740381601000000000", Line: "line-1"},
					},
				},
			}, nil
		},
	}
	service := NewLokiLogQueryService(client, "test-secret")

	first, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     50,
	})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}
	if len(first.Logs) != 1 {
		t.Fatalf("expected first query to return 1 log, got %d", len(first.Logs))
	}
	if first.NextCursor == "" {
		t.Fatalf("expected non-empty nextCursor")
	}

	second, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     50,
		Cursor:    first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second query error: %v", err)
	}
	if len(second.Logs) != 0 {
		t.Fatalf("expected no new logs, got %d", len(second.Logs))
	}
	if second.NextCursor != first.NextCursor {
		t.Fatalf("expected fallback cursor to stay unchanged")
	}
}

func TestLokiLogQueryServiceRejectsCrossQueryCursorReuse(t *testing.T) {
	client := &lokiQueryClientStub{
		fn: func(context.Context, loki.QueryRangeRequest) ([]loki.StreamResult, error) {
			return []loki.StreamResult{
				{
					Stream: map[string]string{"source": "stdout"},
					Values: []loki.StreamValue{
						{TsNs: "1740381601000000000", Line: "line-1"},
					},
				},
			}, nil
		},
	}
	service := NewLokiLogQueryService(client, "test-secret")

	first, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
	})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}

	_, err = service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "another-container",
		Cursor:    first.NextCursor,
	})
	if !errors.Is(err, ErrLogCursorQueryMismatch) {
		t.Fatalf("expected ErrLogCursorQueryMismatch, got %v", err)
	}
}

func TestLokiLogQueryServiceMapsTimeout(t *testing.T) {
	client := &lokiQueryClientStub{
		fn: func(ctx context.Context, input loki.QueryRangeRequest) ([]loki.StreamResult, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	service := NewLokiLogQueryService(client, "test-secret")

	_, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
	})
	if !errors.Is(err, ErrLokiQueryTimeout) {
		t.Fatalf("expected ErrLokiQueryTimeout, got %v", err)
	}
}

func TestLokiLogQueryServiceStableCursorForDuplicateTimestamp(t *testing.T) {
	baseNs := time.Now().UTC().Add(-1 * time.Minute).UnixNano()
	ts := strconv.FormatInt(baseNs, 10)

	client := &timelineLokiClientStub{}
	client.SetEntries([]timelineEntry{
		{tsNs: ts, line: "line-a", source: "stdout"},
	})

	service := NewLokiLogQueryService(client, "test-secret")

	first, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}
	if len(first.Logs) != 1 {
		t.Fatalf("expected first query to return 1 log, got %d", len(first.Logs))
	}
	if first.NextCursor == "" {
		t.Fatalf("expected non-empty nextCursor")
	}

	client.SetEntries([]timelineEntry{
		{tsNs: ts, line: "line-a", source: "stdout"},
		{tsNs: ts, line: "line-a", source: "stdout"},
		{tsNs: ts, line: "line-a", source: "stdout"},
	})

	second, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     10,
		Cursor:    first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second query error: %v", err)
	}
	if len(second.Logs) != 2 {
		t.Fatalf("expected 2 incremental logs, got %d", len(second.Logs))
	}

	seen := make(map[string]struct{}, len(second.Logs))
	for _, item := range second.Logs {
		if _, exists := seen[item.ID]; exists {
			t.Fatalf("expected unique ids, duplicated: %s", item.ID)
		}
		seen[item.ID] = struct{}{}
	}
}

func TestLokiLogQueryServiceCursorStableWhenLimitChanges(t *testing.T) {
	baseNs := time.Now().UTC().Add(-1 * time.Minute).UnixNano()
	ts := func(offset int64) string {
		return strconv.FormatInt(baseNs+offset, 10)
	}

	client := &timelineLokiClientStub{}
	client.SetEntries([]timelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
		{tsNs: ts(3), line: "line-3", source: "stdout"},
		{tsNs: ts(4), line: "line-4", source: "stdout"},
	})

	service := NewLokiLogQueryService(client, "test-secret")

	first, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}
	if len(first.Logs) != 2 {
		t.Fatalf("expected 2 logs in first page, got %d", len(first.Logs))
	}

	client.SetEntries([]timelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
		{tsNs: ts(3), line: "line-3", source: "stdout"},
		{tsNs: ts(4), line: "line-4", source: "stdout"},
		{tsNs: ts(5), line: "line-5", source: "stdout"},
		{tsNs: ts(6), line: "line-6", source: "stdout"},
	})

	second, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     4,
		Cursor:    first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second query error: %v", err)
	}
	if len(second.Logs) != 2 {
		t.Fatalf("expected 2 new logs after cursor, got %d", len(second.Logs))
	}
	if second.Logs[0].Line != "line-5" || second.Logs[1].Line != "line-6" {
		t.Fatalf("unexpected incremental logs: %+v", second.Logs)
	}
}

func TestLokiLogQueryServiceInitialLatestWindowReturnsOlderAndFollowAnchors(t *testing.T) {
	baseNs := time.Now().UTC().Add(-1 * time.Minute).UnixNano()
	ts := func(offset int64) string {
		return strconv.FormatInt(baseNs+offset, 10)
	}

	client := &timelineLokiClientStub{}
	client.SetEntries([]timelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
		{tsNs: ts(3), line: "line-3", source: "stdout"},
	})

	service := NewLokiLogQueryService(client, "test-secret")

	result, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if len(result.Logs) != 2 {
		t.Fatalf("expected latest 2 logs, got %d", len(result.Logs))
	}
	if result.Logs[0].Line != "line-2" || result.Logs[1].Line != "line-3" {
		t.Fatalf("expected oldest-to-newest order, got %+v", result.Logs)
	}
	if result.NextCursor == "" {
		t.Fatal("expected follow cursor")
	}
	if result.PreviousCursor == "" {
		t.Fatal("expected older cursor")
	}
	if !result.HasOlder {
		t.Fatal("expected hasOlder=true")
	}
	if result.HasNewer {
		t.Fatal("expected hasNewer=false")
	}
	if !result.CaughtUp {
		t.Fatal("expected caughtUp=true")
	}
}

func TestLokiLogQueryServiceOlderPageReturnsPrependFriendlyOrder(t *testing.T) {
	baseNs := time.Now().UTC().Add(-1 * time.Minute).UnixNano()
	ts := func(offset int64) string {
		return strconv.FormatInt(baseNs+offset, 10)
	}

	client := &timelineLokiClientStub{}
	client.SetEntries([]timelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
		{tsNs: ts(3), line: "line-3", source: "stdout"},
		{tsNs: ts(4), line: "line-4", source: "stdout"},
	})

	service := NewLokiLogQueryService(client, "test-secret")

	first, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}
	if first.PreviousCursor == "" {
		t.Fatal("expected previous cursor on first page")
	}

	older, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     2,
		Cursor:    first.PreviousCursor,
		Direction: "older",
	})
	if err != nil {
		t.Fatalf("older query error: %v", err)
	}
	if len(older.Logs) != 2 {
		t.Fatalf("expected 2 older logs, got %d", len(older.Logs))
	}
	if older.Logs[0].Line != "line-1" || older.Logs[1].Line != "line-2" {
		t.Fatalf("expected prepend-friendly oldest-to-newest order, got %+v", older.Logs)
	}
	if older.HasOlder {
		t.Fatal("expected hasOlder=false after reaching beginning")
	}
	if older.NextCursor == "" {
		t.Fatal("expected follow cursor for rejoining latest flow")
	}
}

func TestLokiLogQueryServiceOlderPagesPreserveDuplicateTimestampTimeline(t *testing.T) {
	ts := strconv.FormatInt(time.Now().UTC().Add(-time.Minute).UnixNano(), 10)
	entries := make([]timelineEntry, 6)
	for index := range entries {
		entries[index] = timelineEntry{tsNs: ts, line: "duplicate", source: "stdout"}
	}
	client := &timelineLokiClientStub{}
	client.SetEntries(entries)
	service := NewLokiLogQueryService(client, "test-secret")

	page, err := service.Query(context.Background(), LokiLogQueryInput{AgentID: 1, Container: "lunafox-agent", Limit: 2})
	if err != nil {
		t.Fatalf("initial query: %v", err)
	}
	seen := append([]LokiLogLineItem{}, page.Logs...)
	for page.HasOlder {
		page, err = service.Query(context.Background(), LokiLogQueryInput{
			AgentID: 1, Container: "lunafox-agent", Limit: 2, Cursor: page.PreviousCursor, Direction: "older",
		})
		if err != nil {
			t.Fatalf("older query: %v", err)
		}
		seen = append(page.Logs, seen...)
	}
	if len(seen) != 6 {
		t.Fatalf("expected 6 entries across pages, got %d", len(seen))
	}
	ids := map[string]struct{}{}
	for _, item := range seen {
		if _, exists := ids[item.ID]; exists {
			t.Fatalf("duplicate entry across pages: %s", item.ID)
		}
		ids[item.ID] = struct{}{}
	}
}

func TestLokiLogQueryServiceFollowPageSurfacesBacklogMetadata(t *testing.T) {
	baseNs := time.Now().UTC().Add(-1 * time.Minute).UnixNano()
	ts := func(offset int64) string {
		return strconv.FormatInt(baseNs+offset, 10)
	}

	client := &timelineLokiClientStub{}
	client.SetEntries([]timelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
	})

	service := NewLokiLogQueryService(client, "test-secret")

	first, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}

	client.SetEntries([]timelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
		{tsNs: ts(3), line: "line-3", source: "stdout"},
		{tsNs: ts(4), line: "line-4", source: "stdout"},
	})

	follow, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID:   1,
		Container: "lunafox-agent",
		Limit:     1,
		Cursor:    first.NextCursor,
		Direction: "newer",
	})
	if err != nil {
		t.Fatalf("follow query error: %v", err)
	}
	if len(follow.Logs) != 1 || follow.Logs[0].Line != "line-3" {
		t.Fatalf("expected first backlog line, got %+v", follow.Logs)
	}
	if !follow.HasNewer {
		t.Fatal("expected hasNewer=true when backlog exceeds one page")
	}
	if follow.CaughtUp {
		t.Fatal("expected caughtUp=false while backlog remains")
	}
}

func TestLokiLogQueryServiceFetchCeilingKeepsUncertainFollowCursor(t *testing.T) {
	baseNs := strconv.FormatInt(time.Now().UTC().Add(-time.Minute).UnixNano(), 10)
	entries := make([]timelineEntry, lokiCursorMaxFetchLimit)
	for index := range entries {
		entries[index] = timelineEntry{tsNs: baseNs, line: "duplicate", source: "stdout"}
	}
	client := &timelineLokiClientStub{}
	client.SetEntries(entries)
	service := NewLokiLogQueryService(client, "test-secret")

	anchor := buildLokiEntries(1, "lunafox-agent", []loki.StreamResult{{
		Stream: map[string]string{"source": "stdout"},
		Values: []loki.StreamValue{{TsNs: baseNs, Line: "duplicate"}},
	}}, "FORWARD")[0]
	cursor, err := service.cursorCodec.Encode(logCursorPayload{
		V:              logCursorVersion,
		Kind:           "follow",
		LastTsNs:       anchor.TSNs,
		LastID:         anchor.ID,
		LastStream:     anchor.Stream,
		LastLineHash:   anchor.lineHash,
		LastOccurrence: lokiCursorMaxFetchLimit,
		AgentID:        1,
		Container:      "lunafox-agent",
	})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	result, err := service.Query(context.Background(), LokiLogQueryInput{
		AgentID: 1, Container: "lunafox-agent", Limit: 500, Cursor: cursor, Direction: "newer",
	})
	if err != nil {
		t.Fatalf("query after saturated cursor: %v", err)
	}
	if !result.Gap || result.GapReason != "query_limit" || result.CaughtUp {
		t.Fatalf("expected uncertain boundary gap, got %+v", result)
	}
	if result.NextCursor != cursor {
		t.Fatalf("expected follow cursor to remain unchanged")
	}
}

func TestBuildLokiLogQL_UsesProviderCompatibleLabels(t *testing.T) {
	query := buildLokiLogQL(7, "lunafox-agent")
	if query != `{agent_id="7",container_name="lunafox-agent"}` {
		t.Fatalf("unexpected LogQL selector: %q", query)
	}
	if strings.Contains(query, "agent.id") || strings.Contains(query, "container.name") {
		t.Fatalf("expected provider-compatible labels, got %q", query)
	}
}
