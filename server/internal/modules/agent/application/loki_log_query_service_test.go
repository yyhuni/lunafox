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
		if input.StartNs != "" && comparePositiveNumericStrings(entry.tsNs, input.StartNs) < 0 {
			continue
		}
		if input.EndNs != "" && comparePositiveNumericStrings(entry.tsNs, input.EndNs) > 0 {
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
