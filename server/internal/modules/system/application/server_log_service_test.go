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

type serverLogLokiClientStub struct {
	requests []loki.QueryRangeRequest
	results  []loki.StreamResult
	err      error
}

func (stub *serverLogLokiClientStub) QueryRange(_ context.Context, input loki.QueryRangeRequest) ([]loki.StreamResult, error) {
	stub.requests = append(stub.requests, input)
	return stub.results, stub.err
}

type serverTimelineEntry struct {
	tsNs   string
	line   string
	source string
}

type serverTimelineLokiClientStub struct {
	entries          []serverTimelineEntry
	maxAcceptedLimit int
	requests         []loki.QueryRangeRequest
}

func (stub *serverTimelineLokiClientStub) SetEntries(entries []serverTimelineEntry) {
	stub.entries = make([]serverTimelineEntry, len(entries))
	copy(stub.entries, entries)
}

func (stub *serverTimelineLokiClientStub) QueryRange(_ context.Context, input loki.QueryRangeRequest) ([]loki.StreamResult, error) {
	stub.requests = append(stub.requests, input)
	if stub.maxAcceptedLimit > 0 && input.Limit > stub.maxAcceptedLimit {
		return nil, errors.New("loki query response exceeds transfer budget")
	}

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

func TestServerLogServiceInitialPageStopsAtCompleteCursorBoundary(t *testing.T) {
	baseNs := time.Now().UTC().Add(-time.Hour).UnixNano()
	entries := make([]serverTimelineEntry, 3000)
	for index := range entries {
		entries[index] = serverTimelineEntry{
			tsNs:   strconv.FormatInt(baseNs+int64(index), 10),
			line:   "large-log-line",
			source: "stdout",
		}
	}
	client := &serverTimelineLokiClientStub{
		maxAcceptedLimit: 501,
	}
	client.SetEntries(entries)
	service := NewServerLogService(client, "test-secret")

	result, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 500})
	if err != nil {
		t.Fatalf("query initial server logs: %v", err)
	}
	if len(result.Logs) != 500 {
		t.Fatalf("initial log count = %d, want 500", len(result.Logs))
	}
	if result.Gap || !result.CaughtUp || result.NextCursor == "" || result.PreviousCursor == "" {
		t.Fatalf("unexpected initial pagination metadata: %+v", result)
	}
	if len(client.requests) != 1 || client.requests[0].Limit != 501 {
		t.Fatalf("Loki requests = %+v, want one bounded request with limit 501", client.requests)
	}
}

func TestServerLogServiceInitialPageKeepsUncertainSameTimestampBoundary(t *testing.T) {
	ts := strconv.FormatInt(time.Now().UTC().Add(-time.Minute).UnixNano(), 10)
	entries := make([]serverTimelineEntry, serverLogCursorFetchLimit)
	for index := range entries {
		entries[index] = serverTimelineEntry{tsNs: ts, line: "duplicate", source: "stdout"}
	}
	client := &serverTimelineLokiClientStub{}
	client.SetEntries(entries)
	service := NewServerLogService(client, "test-secret")

	result, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 500})
	if err != nil {
		t.Fatalf("query initial server logs: %v", err)
	}
	if len(result.Logs) != 500 {
		t.Fatalf("initial log count = %d, want 500", len(result.Logs))
	}
	if !result.Gap || result.GapReason != "query_limit" || result.CaughtUp {
		t.Fatalf("expected uncertain initial boundary gap, got %+v", result)
	}
	if len(client.requests) < 2 || client.requests[len(client.requests)-1].Limit != serverLogCursorFetchLimit {
		t.Fatalf("Loki requests = %+v, want expansion through fetch ceiling", client.requests)
	}
}

func TestServerLogServiceFollowPageUsesBoundedOverfetch(t *testing.T) {
	baseNs := time.Now().UTC().Add(-time.Hour).UnixNano()
	client := &serverTimelineLokiClientStub{
		maxAcceptedLimit: 502,
	}
	client.SetEntries([]serverTimelineEntry{
		{tsNs: strconv.FormatInt(baseNs, 10), line: "anchor", source: "stdout"},
	})
	service := NewServerLogService(client, "test-secret")
	initial, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 500})
	if err != nil {
		t.Fatalf("query initial server logs: %v", err)
	}

	entries := make([]serverTimelineEntry, 1001)
	for index := range entries {
		entries[index] = serverTimelineEntry{
			tsNs:   strconv.FormatInt(baseNs+int64(index), 10),
			line:   "large-log-line",
			source: "stdout",
		}
	}
	entries[0].line = "anchor"
	client.SetEntries(entries)
	client.requests = nil

	result, err := service.Query(context.Background(), ServerLogQueryInput{
		Limit: 500, Cursor: initial.NextCursor, Direction: "newer",
	})
	if err != nil {
		t.Fatalf("query newer server logs: %v", err)
	}
	if len(result.Logs) != 500 || !result.HasNewer || result.CaughtUp {
		t.Fatalf("unexpected follow page: %+v", result)
	}
	if len(client.requests) != 1 || client.requests[0].Limit != 502 {
		t.Fatalf("Loki requests = %+v, want one bounded request with limit 502", client.requests)
	}
}

func TestServerLogServiceOlderPageUsesBoundedOverfetch(t *testing.T) {
	baseNs := time.Now().UTC().Add(-time.Hour).UnixNano()
	entries := make([]serverTimelineEntry, 3000)
	for index := range entries {
		entries[index] = serverTimelineEntry{
			tsNs:   strconv.FormatInt(baseNs+int64(index), 10),
			line:   "large-log-line",
			source: "stdout",
		}
	}
	client := &serverTimelineLokiClientStub{
		maxAcceptedLimit: 502,
	}
	client.SetEntries(entries)
	service := NewServerLogService(client, "test-secret")
	initial, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 500})
	if err != nil {
		t.Fatalf("query initial server logs: %v", err)
	}
	client.requests = nil

	result, err := service.Query(context.Background(), ServerLogQueryInput{
		Limit: 500, Cursor: initial.PreviousCursor, Direction: "older",
	})
	if err != nil {
		t.Fatalf("query older server logs: %v", err)
	}
	if len(result.Logs) != 500 || !result.HasOlder || result.PreviousCursor == "" {
		t.Fatalf("unexpected older page: %+v", result)
	}
	if len(client.requests) != 1 || client.requests[0].Limit != 502 {
		t.Fatalf("Loki requests = %+v, want one bounded request with limit 502", client.requests)
	}
}

func TestServerLogServiceLatestWindowUsesFixedServerLabels(t *testing.T) {
	client := &serverLogLokiClientStub{
		results: []loki.StreamResult{
			{
				Stream: map[string]string{"source": "stdout"},
				Values: []loki.StreamValue{
					{TsNs: "1740381601000000000", Line: `{"level":"info","msg":"server started"}`},
				},
			},
		},
	}
	service := NewServerLogService(client, "test-secret")

	result, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 50})
	if err != nil {
		t.Fatalf("query server logs: %v", err)
	}
	if len(result.Logs) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(result.Logs))
	}
	if !strings.HasPrefix(result.Logs[0].ID, "srv:lunafox-server:") {
		t.Fatalf("expected server log id prefix, got %q", result.Logs[0].ID)
	}
	if result.NextCursor == "" {
		t.Fatalf("expected follow cursor")
	}
	if len(client.requests) != 1 {
		t.Fatalf("expected one Loki request, got %d", len(client.requests))
	}
	if got, want := client.requests[0].Query, `{component="server",container_name="lunafox-server"}`; got != want {
		t.Fatalf("unexpected Loki query: got %q want %q", got, want)
	}
}

func TestServerLogServiceRejectsMismatchedCursor(t *testing.T) {
	client := &serverLogLokiClientStub{
		results: []loki.StreamResult{
			{
				Stream: map[string]string{"source": "stdout"},
				Values: []loki.StreamValue{
					{TsNs: "1740381601000000000", Line: "line-1"},
				},
			},
		},
	}
	service := NewServerLogService(client, "test-secret")
	initial, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 50})
	if err != nil {
		t.Fatalf("query initial server logs: %v", err)
	}

	otherSource := NewServerLogServiceWithSource(client, "test-secret", ServerLogSource{
		Component:     "server",
		ContainerName: "other-server",
	})
	_, err = otherSource.Query(context.Background(), ServerLogQueryInput{
		Limit:     50,
		Cursor:    initial.NextCursor,
		Direction: "newer",
	})
	if err == nil {
		t.Fatalf("expected mismatched cursor error")
	}
	if err != ErrServerLogCursorQueryMismatch {
		t.Fatalf("expected ErrServerLogCursorQueryMismatch, got %v", err)
	}
}

func TestServerLogServiceMapsTimeout(t *testing.T) {
	client := &serverLogLokiClientStub{
		err: context.DeadlineExceeded,
	}
	service := NewServerLogService(client, "test-secret")

	_, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 50})
	if !errors.Is(err, ErrServerLogQueryTimeout) {
		t.Fatalf("expected ErrServerLogQueryTimeout, got %v", err)
	}
}

func TestServerLogServiceCursorRoundTripNoNewLogs(t *testing.T) {
	client := &serverLogLokiClientStub{
		results: []loki.StreamResult{
			{
				Stream: map[string]string{"source": "stdout"},
				Values: []loki.StreamValue{
					{TsNs: "1740381601000000000", Line: "line-1"},
				},
			},
		},
	}
	service := NewServerLogService(client, "test-secret")

	first, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 50})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}
	if first.NextCursor == "" {
		t.Fatalf("expected non-empty nextCursor")
	}

	second, err := service.Query(context.Background(), ServerLogQueryInput{
		Limit:  50,
		Cursor: first.NextCursor,
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

func TestServerLogServiceStableCursorForDuplicateTimestamp(t *testing.T) {
	baseNs := time.Now().UTC().Add(-1 * time.Minute).UnixNano()
	ts := strconv.FormatInt(baseNs, 10)

	client := &serverTimelineLokiClientStub{}
	client.SetEntries([]serverTimelineEntry{
		{tsNs: ts, line: "line-a", source: "stdout"},
	})

	service := NewServerLogService(client, "test-secret")

	first, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 10})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}
	if len(first.Logs) != 1 {
		t.Fatalf("expected first query to return 1 log, got %d", len(first.Logs))
	}

	client.SetEntries([]serverTimelineEntry{
		{tsNs: ts, line: "line-a", source: "stdout"},
		{tsNs: ts, line: "line-a", source: "stdout"},
		{tsNs: ts, line: "line-a", source: "stdout"},
	})

	second, err := service.Query(context.Background(), ServerLogQueryInput{
		Limit:  10,
		Cursor: first.NextCursor,
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

func TestServerLogServiceCursorStableWhenLimitChanges(t *testing.T) {
	baseNs := time.Now().UTC().Add(-1 * time.Minute).UnixNano()
	ts := func(offset int64) string {
		return strconv.FormatInt(baseNs+offset, 10)
	}

	client := &serverTimelineLokiClientStub{}
	client.SetEntries([]serverTimelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
		{tsNs: ts(3), line: "line-3", source: "stdout"},
		{tsNs: ts(4), line: "line-4", source: "stdout"},
	})

	service := NewServerLogService(client, "test-secret")

	first, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 2})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}
	if len(first.Logs) != 2 {
		t.Fatalf("expected 2 logs in first page, got %d", len(first.Logs))
	}

	client.SetEntries([]serverTimelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
		{tsNs: ts(3), line: "line-3", source: "stdout"},
		{tsNs: ts(4), line: "line-4", source: "stdout"},
		{tsNs: ts(5), line: "line-5", source: "stdout"},
		{tsNs: ts(6), line: "line-6", source: "stdout"},
	})

	second, err := service.Query(context.Background(), ServerLogQueryInput{
		Limit:  4,
		Cursor: first.NextCursor,
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

func TestServerLogServiceInitialLatestWindowReturnsOlderAndFollowAnchors(t *testing.T) {
	baseNs := time.Now().UTC().Add(-1 * time.Minute).UnixNano()
	ts := func(offset int64) string {
		return strconv.FormatInt(baseNs+offset, 10)
	}

	client := &serverTimelineLokiClientStub{}
	client.SetEntries([]serverTimelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
		{tsNs: ts(3), line: "line-3", source: "stdout"},
	})

	service := NewServerLogService(client, "test-secret")

	result, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 2})
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

func TestServerLogServiceOlderPageReturnsPrependFriendlyOrder(t *testing.T) {
	baseNs := time.Now().UTC().Add(-1 * time.Minute).UnixNano()
	ts := func(offset int64) string {
		return strconv.FormatInt(baseNs+offset, 10)
	}

	client := &serverTimelineLokiClientStub{}
	client.SetEntries([]serverTimelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
		{tsNs: ts(3), line: "line-3", source: "stdout"},
		{tsNs: ts(4), line: "line-4", source: "stdout"},
	})

	service := NewServerLogService(client, "test-secret")

	first, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 2})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}
	if first.PreviousCursor == "" {
		t.Fatal("expected previous cursor on first page")
	}

	older, err := service.Query(context.Background(), ServerLogQueryInput{
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

func TestServerLogServiceOlderPagesPreserveDuplicateTimestampTimeline(t *testing.T) {
	ts := strconv.FormatInt(time.Now().UTC().Add(-time.Minute).UnixNano(), 10)
	entries := make([]serverTimelineEntry, 6)
	for index := range entries {
		entries[index] = serverTimelineEntry{tsNs: ts, line: "duplicate", source: "stdout"}
	}
	client := &serverTimelineLokiClientStub{}
	client.SetEntries(entries)
	service := NewServerLogService(client, "test-secret")

	page, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 2})
	if err != nil {
		t.Fatalf("initial query: %v", err)
	}
	seen := append([]ServerLogLineItem{}, page.Logs...)
	for page.HasOlder {
		page, err = service.Query(context.Background(), ServerLogQueryInput{
			Limit: 2, Cursor: page.PreviousCursor, Direction: "older",
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

func TestServerLogServiceFollowPageSurfacesBacklogMetadata(t *testing.T) {
	baseNs := time.Now().UTC().Add(-1 * time.Minute).UnixNano()
	ts := func(offset int64) string {
		return strconv.FormatInt(baseNs+offset, 10)
	}

	client := &serverTimelineLokiClientStub{}
	client.SetEntries([]serverTimelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
	})

	service := NewServerLogService(client, "test-secret")

	first, err := service.Query(context.Background(), ServerLogQueryInput{Limit: 1})
	if err != nil {
		t.Fatalf("first query error: %v", err)
	}

	client.SetEntries([]serverTimelineEntry{
		{tsNs: ts(1), line: "line-1", source: "stdout"},
		{tsNs: ts(2), line: "line-2", source: "stdout"},
		{tsNs: ts(3), line: "line-3", source: "stdout"},
		{tsNs: ts(4), line: "line-4", source: "stdout"},
	})

	follow, err := service.Query(context.Background(), ServerLogQueryInput{
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

func TestServerLogServiceFetchCeilingKeepsUncertainFollowCursor(t *testing.T) {
	baseNs := strconv.FormatInt(time.Now().UTC().Add(-time.Minute).UnixNano(), 10)
	entries := make([]serverTimelineEntry, serverLogCursorFetchLimit)
	for index := range entries {
		entries[index] = serverTimelineEntry{tsNs: baseNs, line: "duplicate", source: "stdout"}
	}
	client := &serverTimelineLokiClientStub{}
	client.SetEntries(entries)
	service := NewServerLogService(client, "test-secret")

	anchor := service.buildEntries([]loki.StreamResult{{
		Stream: map[string]string{"source": "stdout"},
		Values: []loki.StreamValue{{TsNs: baseNs, Line: "duplicate"}},
	}}, "FORWARD")[0]
	cursor, err := service.encodeCursor("follow", ServerLogLineItem{
		ID:         anchor.ID,
		TSNs:       anchor.TSNs,
		Stream:     anchor.Stream,
		lineHash:   anchor.lineHash,
		occurrence: serverLogCursorFetchLimit,
	})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	result, err := service.Query(context.Background(), ServerLogQueryInput{
		Limit: 500, Cursor: cursor, Direction: "newer",
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
