package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/loki"
)

const (
	defaultServerLogLimit     = 200
	maxServerLogLimit         = 500
	maxServerLogLineBytes     = 16 * 1024
	serverLogQueryTimeout     = 5 * time.Second
	serverLogInitialLookback  = 15 * 24 * time.Hour
	serverLogCursorFetchLimit = 5000
)

var ErrServerLogQueryTimeout = errors.New("server log query timeout")

type ServerLogQueryClient interface {
	QueryRange(ctx context.Context, input loki.QueryRangeRequest) ([]loki.StreamResult, error)
}

type ServerLogSource struct {
	Component     string
	ContainerName string
}

type ServerLogQueryInput struct {
	Limit     int
	Cursor    string
	Direction string
}

type ServerLogLineItem struct {
	ID         string
	TS         string
	TSNs       string
	Stream     string
	Line       string
	Truncated  bool
	lineHash   string
	occurrence int
}

type ServerLogQueryResult struct {
	Logs           []ServerLogLineItem
	NextCursor     string
	PreviousCursor string
	HasOlder       bool
	HasNewer       bool
	CaughtUp       bool
	Gap            bool
	GapReason      string
}

type ServerLogService struct {
	client      ServerLogQueryClient
	cursorCodec *serverLogCursorCodec
	source      ServerLogSource
}

func NewServerLogService(client ServerLogQueryClient, cursorSecret string) *ServerLogService {
	return NewServerLogServiceWithSource(client, cursorSecret, ServerLogSource{
		Component:     "server",
		ContainerName: "lunafox-server",
	})
}

func NewServerLogServiceWithSource(client ServerLogQueryClient, cursorSecret string, source ServerLogSource) *ServerLogService {
	if client == nil {
		panic("server log query client is required")
	}
	cursorCodec, err := newServerLogCursorCodec(cursorSecret)
	if err != nil {
		panic(err)
	}
	normalized := ServerLogSource{
		Component:     strings.TrimSpace(source.Component),
		ContainerName: strings.TrimSpace(source.ContainerName),
	}
	if normalized.Component == "" || normalized.ContainerName == "" {
		panic("server log source is required")
	}
	return &ServerLogService{
		client:      client,
		cursorCodec: cursorCodec,
		source:      normalized,
	}
}

func (service *ServerLogService) Query(ctx context.Context, input ServerLogQueryInput) (ServerLogQueryResult, error) {
	if service == nil || service.client == nil || service.cursorCodec == nil {
		return ServerLogQueryResult{}, fmt.Errorf("server log query service is not configured")
	}
	if ctx == nil {
		return ServerLogQueryResult{}, context.Canceled
	}
	limit := normalizeServerLogLimit(input.Limit)

	queryCtx, cancel := context.WithTimeout(ctx, serverLogQueryTimeout)
	defer cancel()

	if strings.TrimSpace(input.Cursor) == "" {
		return service.queryInitial(queryCtx, limit)
	}

	cursorPayload, err := service.cursorCodec.Decode(input.Cursor)
	if err != nil {
		return ServerLogQueryResult{}, err
	}
	if cursorPayload.Component != service.source.Component ||
		cursorPayload.ContainerName != service.source.ContainerName {
		return ServerLogQueryResult{}, ErrServerLogCursorQueryMismatch
	}

	switch strings.ToLower(strings.TrimSpace(input.Direction)) {
	case "", "newer":
		if cursorPayload.Kind != "follow" {
			return ServerLogQueryResult{}, ErrServerLogCursorQueryMismatch
		}
		return service.queryAfterCursor(queryCtx, limit, cursorPayload, strings.TrimSpace(input.Cursor))
	case "older":
		if cursorPayload.Kind != "older" {
			return ServerLogQueryResult{}, ErrServerLogCursorQueryMismatch
		}
		return service.queryBeforeCursor(queryCtx, limit, cursorPayload, strings.TrimSpace(input.Cursor))
	default:
		return ServerLogQueryResult{}, fmt.Errorf("%w: invalid query direction", ErrServerLogCursorInvalid)
	}
}

func (service *ServerLogService) queryInitial(ctx context.Context, limit int) (ServerLogQueryResult, error) {
	now := time.Now().UTC()
	internalLimit := limit + 1
	if internalLimit > serverLogCursorFetchLimit {
		internalLimit = serverLogCursorFetchLimit
	}

	var (
		entries             []ServerLogLineItem
		reachedFetchCeiling bool
	)
	for {
		streams, err := service.queryRange(ctx, loki.QueryRangeRequest{
			Query:     service.buildLogQL(),
			StartNs:   strconv.FormatInt(now.Add(-serverLogInitialLookback).UnixNano(), 10),
			EndNs:     strconv.FormatInt(now.UnixNano(), 10),
			Limit:     internalLimit,
			Direction: "BACKWARD",
		})
		if err != nil {
			return ServerLogQueryResult{}, err
		}

		entries = service.buildEntries(streams, "BACKWARD")
		cursorBoundaryComplete := initialServerLogCursorBoundaryComplete(entries, limit, internalLimit)
		reachedFetchCeiling = !cursorBoundaryComplete && internalLimit >= serverLogCursorFetchLimit
		if cursorBoundaryComplete || internalLimit >= serverLogCursorFetchLimit {
			break
		}
		internalLimit *= 2
		if internalLimit > serverLogCursorFetchLimit {
			internalLimit = serverLogCursorFetchLimit
		}
	}
	if len(entries) == 0 {
		return ServerLogQueryResult{
			Logs:     []ServerLogLineItem{},
			CaughtUp: true,
		}, nil
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[len(entries)-limit:]
	}
	if reachedFetchCeiling {
		// The latest rows are usable, but a saturated tie-breaker cannot become a cursor.
		return ServerLogQueryResult{
			Logs:      entries,
			CaughtUp:  false,
			Gap:       true,
			GapReason: "query_limit",
		}, nil
	}

	nextCursor, err := service.encodeCursor("follow", entries[len(entries)-1])
	if err != nil {
		return ServerLogQueryResult{}, err
	}

	var previousCursor string
	if hasMore {
		previousCursor, err = service.encodeCursor("older", entries[0])
		if err != nil {
			return ServerLogQueryResult{}, err
		}
	}

	return ServerLogQueryResult{
		Logs:           entries,
		NextCursor:     nextCursor,
		PreviousCursor: previousCursor,
		HasOlder:       hasMore,
		HasNewer:       false,
		CaughtUp:       true,
		Gap:            false,
	}, nil
}

func (service *ServerLogService) queryAfterCursor(
	ctx context.Context,
	limit int,
	cursor serverLogCursorPayload,
	fallbackCursor string,
) (ServerLogQueryResult, error) {
	nowNs := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	// Loki includes the cursor anchor in an inclusive range. Keep one extra
	// post-filter row so the response can truthfully report hasNewer.
	internalLimit := limit + 2
	if internalLimit > serverLogCursorFetchLimit {
		internalLimit = serverLogCursorFetchLimit
	}

	var (
		filtered            []ServerLogLineItem
		reachedFetchCeiling bool
	)
	for {
		streams, err := service.queryRange(ctx, loki.QueryRangeRequest{
			Query:     service.buildLogQL(),
			StartNs:   cursor.LastTsNs,
			EndNs:     nowNs,
			Limit:     internalLimit,
			Direction: "FORWARD",
		})
		if err != nil {
			return ServerLogQueryResult{}, err
		}

		allEntries := service.buildEntries(streams, "FORWARD")
		filtered = filterServerEntriesAfterCursor(allEntries, cursor)
		reachedFetchCeiling = len(allEntries) == internalLimit && internalLimit >= serverLogCursorFetchLimit
		if len(filtered) > 0 {
			break
		}
		if len(allEntries) < internalLimit || internalLimit >= serverLogCursorFetchLimit {
			break
		}
		internalLimit *= 2
		if internalLimit > serverLogCursorFetchLimit {
			internalLimit = serverLogCursorFetchLimit
		}
	}

	if len(filtered) == 0 {
		if reachedFetchCeiling {
			// A full bounded query cannot prove this cursor is at the live tail.
			return ServerLogQueryResult{
				Logs:       []ServerLogLineItem{},
				NextCursor: fallbackCursor,
				CaughtUp:   false,
				Gap:        true,
				GapReason:  "query_limit",
			}, nil
		}
		return ServerLogQueryResult{
			Logs:       []ServerLogLineItem{},
			NextCursor: fallbackCursor,
			CaughtUp:   true,
		}, nil
	}

	hasNewer := len(filtered) > limit
	if hasNewer {
		filtered = filtered[:limit]
	}

	nextCursor, err := service.encodeCursor("follow", filtered[len(filtered)-1])
	if err != nil {
		return ServerLogQueryResult{}, err
	}

	return ServerLogQueryResult{
		Logs:       filtered,
		NextCursor: nextCursor,
		HasNewer:   hasNewer,
		CaughtUp:   !hasNewer,
	}, nil
}

func (service *ServerLogService) queryBeforeCursor(
	ctx context.Context,
	limit int,
	cursor serverLogCursorPayload,
	fallbackCursor string,
) (ServerLogQueryResult, error) {
	// Loki includes the cursor anchor in an inclusive range. Keep one extra
	// post-filter row so the response can truthfully report hasOlder.
	internalLimit := limit + 2
	if internalLimit > serverLogCursorFetchLimit {
		internalLimit = serverLogCursorFetchLimit
	}

	var (
		filtered            []ServerLogLineItem
		reachedFetchCeiling bool
	)
	for {
		streams, err := service.queryRange(ctx, loki.QueryRangeRequest{
			Query:     service.buildLogQL(),
			EndNs:     cursor.LastTsNs,
			Limit:     internalLimit,
			Direction: "BACKWARD",
		})
		if err != nil {
			return ServerLogQueryResult{}, err
		}

		allEntries := service.buildEntries(streams, "BACKWARD")
		filtered = filterServerEntriesBeforeCursor(allEntries, cursor)
		reachedFetchCeiling = len(allEntries) == internalLimit && internalLimit >= serverLogCursorFetchLimit
		if len(filtered) > 0 {
			break
		}
		if len(allEntries) < internalLimit || internalLimit >= serverLogCursorFetchLimit {
			break
		}
		internalLimit *= 2
		if internalLimit > serverLogCursorFetchLimit {
			internalLimit = serverLogCursorFetchLimit
		}
	}

	if len(filtered) == 0 {
		if reachedFetchCeiling {
			// Do not discard an older cursor when the bounded query cannot locate it.
			return ServerLogQueryResult{
				Logs:           []ServerLogLineItem{},
				PreviousCursor: fallbackCursor,
				HasOlder:       true,
				CaughtUp:       false,
				Gap:            true,
				GapReason:      "query_limit",
			}, nil
		}
		return ServerLogQueryResult{
			Logs:     []ServerLogLineItem{},
			CaughtUp: true,
		}, nil
	}

	hasOlder := len(filtered) > limit
	if hasOlder {
		filtered = filtered[len(filtered)-limit:]
	}

	nextCursor, err := service.encodeCursor("follow", filtered[len(filtered)-1])
	if err != nil {
		return ServerLogQueryResult{}, err
	}

	var previousCursor string
	if hasOlder {
		previousCursor, err = service.encodeCursor("older", filtered[0])
		if err != nil {
			return ServerLogQueryResult{}, err
		}
	}

	return ServerLogQueryResult{
		Logs:           filtered,
		NextCursor:     nextCursor,
		PreviousCursor: previousCursor,
		HasOlder:       hasOlder,
		CaughtUp:       true,
	}, nil
}

func (service *ServerLogService) queryRange(ctx context.Context, req loki.QueryRangeRequest) ([]loki.StreamResult, error) {
	streams, err := service.client.QueryRange(ctx, req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return nil, ErrServerLogQueryTimeout
		}
		return nil, err
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, ErrServerLogQueryTimeout
	}
	return streams, nil
}

func (service *ServerLogService) buildLogQL() string {
	return fmt.Sprintf("{component=%q,container_name=%q}", service.source.Component, service.source.ContainerName)
}

func (service *ServerLogService) encodeCursor(kind string, item ServerLogLineItem) (string, error) {
	return service.cursorCodec.Encode(serverLogCursorPayload{
		V:              serverLogCursorVersion,
		Kind:           kind,
		LastTsNs:       item.TSNs,
		LastID:         item.ID,
		LastStream:     item.Stream,
		LastLineHash:   item.lineHash,
		LastOccurrence: item.occurrence,
		Component:      service.source.Component,
		ContainerName:  service.source.ContainerName,
	})
}

func (service *ServerLogService) buildEntries(streams []loki.StreamResult, queryDirection string) []ServerLogLineItem {
	timelineEntries := loki.BuildTimelineEntries(streams, loki.TimelineBuildOptions{
		IDPrefix:     fmt.Sprintf("srv:%s", service.source.ContainerName),
		MaxLineBytes: maxServerLogLineBytes,
	}, queryDirection)

	entries := make([]ServerLogLineItem, 0, len(timelineEntries))
	for _, item := range timelineEntries {
		entries = append(entries, ServerLogLineItem{
			ID:         item.ID,
			TS:         item.TS,
			TSNs:       item.TSNs,
			Stream:     item.Stream,
			Line:       item.Line,
			Truncated:  item.Truncated,
			lineHash:   item.LineHash,
			occurrence: item.Occurrence,
		})
	}
	return entries
}

func initialServerLogCursorBoundaryComplete(entries []ServerLogLineItem, pageLimit, queryLimit int) bool {
	if len(entries) < queryLimit {
		return true
	}
	if pageLimit <= 0 || len(entries) <= pageLimit {
		return false
	}

	// Older rows cannot change visible cursor occurrences once the overfetch
	// crosses below the first visible row's nanosecond timestamp.
	firstVisible := len(entries) - pageLimit
	return entries[0].TSNs != entries[firstVisible].TSNs
}

func filterServerEntriesAfterCursor(entries []ServerLogLineItem, cursor serverLogCursorPayload) []ServerLogLineItem {
	return filterServerEntriesByTimelineCursor(entries, cursor, loki.FilterTimelineEntriesAfterCursor)
}

func filterServerEntriesBeforeCursor(entries []ServerLogLineItem, cursor serverLogCursorPayload) []ServerLogLineItem {
	return filterServerEntriesByTimelineCursor(entries, cursor, loki.FilterTimelineEntriesBeforeCursor)
}

func normalizeServerLogLimit(limit int) int {
	if limit <= 0 {
		return defaultServerLogLimit
	}
	if limit > maxServerLogLimit {
		return maxServerLogLimit
	}
	return limit
}

func filterServerEntriesByTimelineCursor(
	entries []ServerLogLineItem,
	cursor serverLogCursorPayload,
	filter func([]loki.TimelineEntry, loki.TimelineCursorKey) []loki.TimelineEntry,
) []ServerLogLineItem {
	timelineEntries := make([]loki.TimelineEntry, 0, len(entries))
	entryByID := make(map[string]ServerLogLineItem, len(entries))
	for _, item := range entries {
		timelineEntries = append(timelineEntries, loki.TimelineEntry{
			ID:         item.ID,
			TS:         item.TS,
			TSNs:       item.TSNs,
			Stream:     item.Stream,
			Line:       item.Line,
			Truncated:  item.Truncated,
			LineHash:   item.lineHash,
			Occurrence: item.occurrence,
		})
		entryByID[item.ID] = item
	}

	filteredTimeline := filter(timelineEntries, loki.TimelineCursorKey{
		LastTsNs:       cursor.LastTsNs,
		LastID:         cursor.LastID,
		LastStream:     cursor.LastStream,
		LastLineHash:   cursor.LastLineHash,
		LastOccurrence: cursor.LastOccurrence,
	})

	filtered := make([]ServerLogLineItem, 0, len(filteredTimeline))
	for _, item := range filteredTimeline {
		if original, ok := entryByID[item.ID]; ok {
			filtered = append(filtered, original)
		}
	}
	return filtered
}
