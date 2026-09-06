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
	defaultLokiLogLimit     = 200
	maxLokiLogLimit         = 500
	maxLokiLogLineBytes     = 16 * 1024
	lokiQueryTimeout        = 5 * time.Second
	lokiInitialLookback     = 15 * 24 * time.Hour
	lokiCursorMaxFetchLimit = 5000
)

var (
	ErrLokiContainerNotFound = errors.New("loki container not found")
	ErrLokiQueryTimeout      = errors.New("loki query timeout")
)

type LokiLogQueryClient interface {
	QueryRange(ctx context.Context, input loki.QueryRangeRequest) ([]loki.StreamResult, error)
}

type LokiLogQueryInput struct {
	AgentID   int
	Container string
	Limit     int
	Cursor    string
	Direction string
}

type LokiLogLineItem struct {
	ID         string
	TS         string
	TSNs       string
	Stream     string
	Line       string
	Truncated  bool
	lineHash   string
	occurrence int
}

type LokiLogQueryResult struct {
	Logs           []LokiLogLineItem
	NextCursor     string
	PreviousCursor string
	HasOlder       bool
	HasNewer       bool
	CaughtUp       bool
	Gap            bool
	GapReason      string
}

type LokiLogQueryService struct {
	client      LokiLogQueryClient
	cursorCodec *logCursorCodec
}

func NewLokiLogQueryService(client LokiLogQueryClient, cursorSecret string) *LokiLogQueryService {
	if client == nil {
		panic("loki query client is required")
	}
	cursorCodec, err := newLogCursorCodec(cursorSecret)
	if err != nil {
		panic(err)
	}
	return &LokiLogQueryService{
		client:      client,
		cursorCodec: cursorCodec,
	}
}

func (service *LokiLogQueryService) Query(ctx context.Context, input LokiLogQueryInput) (LokiLogQueryResult, error) {
	if service == nil || service.client == nil || service.cursorCodec == nil {
		return LokiLogQueryResult{}, fmt.Errorf("Loki log query service is not configured")
	}
	if ctx == nil {
		return LokiLogQueryResult{}, context.Canceled
	}
	agentID := input.AgentID
	container := strings.TrimSpace(input.Container)
	if agentID <= 0 || container == "" {
		return LokiLogQueryResult{}, fmt.Errorf("%w: invalid query input", ErrLogCursorInvalid)
	}

	limit := normalizeLokiLogLimit(input.Limit)

	queryCtx, cancel := context.WithTimeout(ctx, lokiQueryTimeout)
	defer cancel()

	if strings.TrimSpace(input.Cursor) == "" {
		return service.queryInitial(queryCtx, agentID, container, limit)
	}

	cursorPayload, err := service.cursorCodec.Decode(input.Cursor)
	if err != nil {
		return LokiLogQueryResult{}, err
	}
	if cursorPayload.AgentID != agentID ||
		cursorPayload.Container != container {
		return LokiLogQueryResult{}, ErrLogCursorQueryMismatch
	}

	switch strings.ToLower(strings.TrimSpace(input.Direction)) {
	case "", "newer":
		if cursorPayload.Kind != "follow" {
			return LokiLogQueryResult{}, ErrLogCursorQueryMismatch
		}
		return service.queryAfterCursor(queryCtx, agentID, container, limit, cursorPayload, strings.TrimSpace(input.Cursor))
	case "older":
		if cursorPayload.Kind != "older" {
			return LokiLogQueryResult{}, ErrLogCursorQueryMismatch
		}
		return service.queryBeforeCursor(queryCtx, agentID, container, limit, cursorPayload, strings.TrimSpace(input.Cursor))
	default:
		return LokiLogQueryResult{}, fmt.Errorf("%w: invalid query direction", ErrLogCursorInvalid)
	}
}

func (service *LokiLogQueryService) queryInitial(
	ctx context.Context,
	agentID int,
	container string,
	limit int,
) (LokiLogQueryResult, error) {
	query := buildLokiLogQL(agentID, container)
	now := time.Now().UTC()

	internalLimit := limit*4 + 1
	if internalLimit < limit+1 {
		internalLimit = limit + 1
	}
	if internalLimit > lokiCursorMaxFetchLimit {
		internalLimit = lokiCursorMaxFetchLimit
	}

	var (
		entries             []LokiLogLineItem
		reachedFetchCeiling bool
	)
	for {
		streams, err := service.queryRange(ctx, loki.QueryRangeRequest{
			Query:     query,
			StartNs:   strconv.FormatInt(now.Add(-lokiInitialLookback).UnixNano(), 10),
			EndNs:     strconv.FormatInt(now.UnixNano(), 10),
			Limit:     internalLimit,
			Direction: "BACKWARD",
		})
		if err != nil {
			return LokiLogQueryResult{}, err
		}

		entries = buildLokiEntries(agentID, container, streams, "BACKWARD")
		reachedFetchCeiling = len(entries) == internalLimit && internalLimit >= lokiCursorMaxFetchLimit
		if len(entries) < internalLimit || internalLimit >= lokiCursorMaxFetchLimit {
			break
		}
		internalLimit *= 2
		if internalLimit > lokiCursorMaxFetchLimit {
			internalLimit = lokiCursorMaxFetchLimit
		}
	}
	if len(entries) == 0 {
		return LokiLogQueryResult{}, ErrLokiContainerNotFound
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[len(entries)-limit:]
	}
	if reachedFetchCeiling {
		// The latest rows are usable, but a saturated tie-breaker cannot become a cursor.
		return LokiLogQueryResult{
			Logs:      entries,
			CaughtUp:  false,
			Gap:       true,
			GapReason: "query_limit",
		}, nil
	}

	last := entries[len(entries)-1]
	nextCursor, err := service.cursorCodec.Encode(logCursorPayload{
		V:              logCursorVersion,
		Kind:           "follow",
		LastTsNs:       last.TSNs,
		LastID:         last.ID,
		LastStream:     last.Stream,
		LastLineHash:   last.lineHash,
		LastOccurrence: last.occurrence,
		AgentID:        agentID,
		Container:      container,
	})
	if err != nil {
		return LokiLogQueryResult{}, err
	}

	var previousCursor string
	if hasMore {
		first := entries[0]
		previousCursor, err = service.cursorCodec.Encode(logCursorPayload{
			V:              logCursorVersion,
			Kind:           "older",
			LastTsNs:       first.TSNs,
			LastID:         first.ID,
			LastStream:     first.Stream,
			LastLineHash:   first.lineHash,
			LastOccurrence: first.occurrence,
			AgentID:        agentID,
			Container:      container,
		})
		if err != nil {
			return LokiLogQueryResult{}, err
		}
	}

	return LokiLogQueryResult{
		Logs:           entries,
		NextCursor:     nextCursor,
		PreviousCursor: previousCursor,
		HasOlder:       hasMore,
		HasNewer:       false,
		CaughtUp:       true,
		Gap:            false,
	}, nil
}

func (service *LokiLogQueryService) queryAfterCursor(
	ctx context.Context,
	agentID int,
	container string,
	limit int,
	cursor logCursorPayload,
	fallbackCursor string,
) (LokiLogQueryResult, error) {
	query := buildLokiLogQL(agentID, container)
	nowNs := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

	internalLimit := limit*4 + 1
	if internalLimit < limit+1 {
		internalLimit = limit + 1
	}
	if internalLimit > lokiCursorMaxFetchLimit {
		internalLimit = lokiCursorMaxFetchLimit
	}

	var (
		filtered            []LokiLogLineItem
		reachedFetchCeiling bool
	)
	for {
		streams, err := service.queryRange(ctx, loki.QueryRangeRequest{
			Query:     query,
			StartNs:   cursor.LastTsNs,
			EndNs:     nowNs,
			Limit:     internalLimit,
			Direction: "FORWARD",
		})
		if err != nil {
			return LokiLogQueryResult{}, err
		}

		allEntries := buildLokiEntries(agentID, container, streams, "FORWARD")
		filtered = filterEntriesAfterCursor(allEntries, cursor)
		reachedFetchCeiling = len(allEntries) == internalLimit && internalLimit >= lokiCursorMaxFetchLimit
		if len(filtered) > 0 {
			break
		}
		if len(allEntries) < internalLimit || internalLimit >= lokiCursorMaxFetchLimit {
			break
		}
		internalLimit *= 2
		if internalLimit > lokiCursorMaxFetchLimit {
			internalLimit = lokiCursorMaxFetchLimit
		}
	}

	if len(filtered) == 0 {
		if reachedFetchCeiling {
			// A full bounded query cannot prove this cursor is at the live tail.
			return LokiLogQueryResult{
				Logs:           []LokiLogLineItem{},
				NextCursor:     fallbackCursor,
				PreviousCursor: "",
				HasOlder:       false,
				HasNewer:       false,
				CaughtUp:       false,
				Gap:            true,
				GapReason:      "query_limit",
			}, nil
		}
		return LokiLogQueryResult{
			Logs:           []LokiLogLineItem{},
			NextCursor:     fallbackCursor,
			PreviousCursor: "",
			HasOlder:       false,
			HasNewer:       false,
			CaughtUp:       true,
			Gap:            false,
		}, nil
	}

	hasNewer := len(filtered) > limit
	if hasNewer {
		filtered = filtered[:limit]
	}

	last := filtered[len(filtered)-1]
	nextCursor, err := service.cursorCodec.Encode(logCursorPayload{
		V:              logCursorVersion,
		Kind:           "follow",
		LastTsNs:       last.TSNs,
		LastID:         last.ID,
		LastStream:     last.Stream,
		LastLineHash:   last.lineHash,
		LastOccurrence: last.occurrence,
		AgentID:        agentID,
		Container:      container,
	})
	if err != nil {
		return LokiLogQueryResult{}, err
	}

	return LokiLogQueryResult{
		Logs:           filtered,
		NextCursor:     nextCursor,
		PreviousCursor: "",
		HasOlder:       false,
		HasNewer:       hasNewer,
		CaughtUp:       !hasNewer,
		Gap:            false,
	}, nil
}

func (service *LokiLogQueryService) queryBeforeCursor(
	ctx context.Context,
	agentID int,
	container string,
	limit int,
	cursor logCursorPayload,
	fallbackCursor string,
) (LokiLogQueryResult, error) {
	query := buildLokiLogQL(agentID, container)

	internalLimit := limit*4 + 1
	if internalLimit < limit+1 {
		internalLimit = limit + 1
	}
	if internalLimit > lokiCursorMaxFetchLimit {
		internalLimit = lokiCursorMaxFetchLimit
	}

	var (
		filtered            []LokiLogLineItem
		reachedFetchCeiling bool
	)
	for {
		streams, err := service.queryRange(ctx, loki.QueryRangeRequest{
			Query:     query,
			EndNs:     cursor.LastTsNs,
			Limit:     internalLimit,
			Direction: "BACKWARD",
		})
		if err != nil {
			return LokiLogQueryResult{}, err
		}

		allEntries := buildLokiEntries(agentID, container, streams, "BACKWARD")
		filtered = filterEntriesBeforeCursor(allEntries, cursor)
		reachedFetchCeiling = len(allEntries) == internalLimit && internalLimit >= lokiCursorMaxFetchLimit
		if len(filtered) > 0 {
			break
		}
		if len(allEntries) < internalLimit || internalLimit >= lokiCursorMaxFetchLimit {
			break
		}
		internalLimit *= 2
		if internalLimit > lokiCursorMaxFetchLimit {
			internalLimit = lokiCursorMaxFetchLimit
		}
	}

	if len(filtered) == 0 {
		if reachedFetchCeiling {
			// Do not discard an older cursor when the bounded query cannot locate it.
			return LokiLogQueryResult{
				Logs:           []LokiLogLineItem{},
				NextCursor:     "",
				PreviousCursor: fallbackCursor,
				HasOlder:       true,
				HasNewer:       false,
				CaughtUp:       false,
				Gap:            true,
				GapReason:      "query_limit",
			}, nil
		}
		return LokiLogQueryResult{
			Logs:           []LokiLogLineItem{},
			PreviousCursor: "",
			HasOlder:       false,
			HasNewer:       false,
			CaughtUp:       true,
			Gap:            false,
		}, nil
	}

	hasOlder := len(filtered) > limit
	if hasOlder {
		filtered = filtered[len(filtered)-limit:]
	}

	last := filtered[len(filtered)-1]
	nextCursor, err := service.cursorCodec.Encode(logCursorPayload{
		V:              logCursorVersion,
		Kind:           "follow",
		LastTsNs:       last.TSNs,
		LastID:         last.ID,
		LastStream:     last.Stream,
		LastLineHash:   last.lineHash,
		LastOccurrence: last.occurrence,
		AgentID:        agentID,
		Container:      container,
	})
	if err != nil {
		return LokiLogQueryResult{}, err
	}

	var previousCursor string
	if hasOlder {
		first := filtered[0]
		previousCursor, err = service.cursorCodec.Encode(logCursorPayload{
			V:              logCursorVersion,
			Kind:           "older",
			LastTsNs:       first.TSNs,
			LastID:         first.ID,
			LastStream:     first.Stream,
			LastLineHash:   first.lineHash,
			LastOccurrence: first.occurrence,
			AgentID:        agentID,
			Container:      container,
		})
		if err != nil {
			return LokiLogQueryResult{}, err
		}
	}

	return LokiLogQueryResult{
		Logs:           filtered,
		NextCursor:     nextCursor,
		PreviousCursor: previousCursor,
		HasOlder:       hasOlder,
		HasNewer:       false,
		CaughtUp:       true,
		Gap:            false,
	}, nil
}

func (service *LokiLogQueryService) queryRange(ctx context.Context, req loki.QueryRangeRequest) ([]loki.StreamResult, error) {
	streams, err := service.client.QueryRange(ctx, req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return nil, ErrLokiQueryTimeout
		}
		return nil, err
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, ErrLokiQueryTimeout
	}
	return streams, nil
}

func buildLokiEntries(agentID int, container string, streams []loki.StreamResult, queryDirection string) []LokiLogLineItem {
	timelineEntries := loki.BuildTimelineEntries(streams, loki.TimelineBuildOptions{
		IDPrefix:     fmt.Sprintf("agt_%d:%s", agentID, container),
		MaxLineBytes: maxLokiLogLineBytes,
	}, queryDirection)

	entries := make([]LokiLogLineItem, 0, len(timelineEntries))
	for _, item := range timelineEntries {
		entries = append(entries, LokiLogLineItem{
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

func filterEntriesAfterCursor(entries []LokiLogLineItem, cursor logCursorPayload) []LokiLogLineItem {
	return filterLokiEntriesByTimelineCursor(entries, cursor, loki.FilterTimelineEntriesAfterCursor)
}

func filterEntriesBeforeCursor(entries []LokiLogLineItem, cursor logCursorPayload) []LokiLogLineItem {
	return filterLokiEntriesByTimelineCursor(entries, cursor, loki.FilterTimelineEntriesBeforeCursor)
}

func normalizeLokiLogLimit(limit int) int {
	if limit <= 0 {
		return defaultLokiLogLimit
	}
	if limit > maxLokiLogLimit {
		return maxLokiLogLimit
	}
	return limit
}

func buildLokiLogQL(agentID int, container string) string {
	return fmt.Sprintf("{agent_id=%q,container_name=%q}", strconv.Itoa(agentID), container)
}

func filterLokiEntriesByTimelineCursor(
	entries []LokiLogLineItem,
	cursor logCursorPayload,
	filter func([]loki.TimelineEntry, loki.TimelineCursorKey) []loki.TimelineEntry,
) []LokiLogLineItem {
	timelineEntries := make([]loki.TimelineEntry, 0, len(entries))
	entryByID := make(map[string]LokiLogLineItem, len(entries))
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

	filtered := make([]LokiLogLineItem, 0, len(filteredTimeline))
	for _, item := range filteredTimeline {
		if original, ok := entryByID[item.ID]; ok {
			filtered = append(filtered, original)
		}
	}
	return filtered
}
