package application

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
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
	Logs       []LokiLogLineItem
	NextCursor string
	HasMore    bool
}

type LokiLogQueryService struct {
	client      LokiLogQueryClient
	cursorCodec *logCursorCodec
}

func NewLokiLogQueryService(client LokiLogQueryClient, cursorSecret string) *LokiLogQueryService {
	if client == nil {
		panic("loki query client is required")
	}
	return &LokiLogQueryService{
		client:      client,
		cursorCodec: newLogCursorCodec(cursorSecret),
	}
}

func (service *LokiLogQueryService) Query(ctx context.Context, input LokiLogQueryInput) (LokiLogQueryResult, error) {
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

	return service.queryAfterCursor(queryCtx, agentID, container, limit, cursorPayload, strings.TrimSpace(input.Cursor))
}

func (service *LokiLogQueryService) queryInitial(
	ctx context.Context,
	agentID int,
	container string,
	limit int,
) (LokiLogQueryResult, error) {
	query := buildLokiLogQL(agentID, container)
	now := time.Now().UTC()

	streams, err := service.queryRange(ctx, loki.QueryRangeRequest{
		Query:     query,
		StartNs:   strconv.FormatInt(now.Add(-lokiInitialLookback).UnixNano(), 10),
		EndNs:     strconv.FormatInt(now.UnixNano(), 10),
		Limit:     limit + 1,
		Direction: "BACKWARD",
	})
	if err != nil {
		return LokiLogQueryResult{}, err
	}

	entries := buildLokiEntries(agentID, container, streams, "BACKWARD")
	if len(entries) == 0 {
		return LokiLogQueryResult{}, ErrLokiContainerNotFound
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[len(entries)-limit:]
	}

	last := entries[len(entries)-1]
	nextCursor, err := service.cursorCodec.Encode(logCursorPayload{
		V:              logCursorVersion,
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
		Logs:       entries,
		NextCursor: nextCursor,
		HasMore:    hasMore,
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

	var filtered []LokiLogLineItem
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
		return LokiLogQueryResult{
			Logs:       []LokiLogLineItem{},
			NextCursor: fallbackCursor,
			HasMore:    false,
		}, nil
	}

	hasMore := len(filtered) > limit
	if hasMore {
		filtered = filtered[:limit]
	}

	last := filtered[len(filtered)-1]
	nextCursor, err := service.cursorCodec.Encode(logCursorPayload{
		V:              logCursorVersion,
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
		Logs:       filtered,
		NextCursor: nextCursor,
		HasMore:    hasMore,
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
	type rawEntry struct {
		tsNs      string
		stream    string
		line      string
		truncated bool
		ts        string
		hash      string
		originSeq int
	}

	rawEntries := make([]rawEntry, 0)
	originSeq := 0
	for _, streamResult := range streams {
		source := strings.TrimSpace(streamResult.Stream["source"])
		if source == "" {
			source = "stdout"
		}
		values := normalizeStreamValuesForStableOrder(streamResult.Values, queryDirection)
		for _, value := range values {
			tsNs := strings.TrimSpace(value.TsNs)
			if tsNs == "" {
				continue
			}
			ts := formatRFC3339FromUnixNano(tsNs)
			if ts == "" {
				continue
			}

			line, truncated := normalizeLokiLine(value.Line)
			sum := sha1.Sum([]byte(source + "\x00" + line))
			hash := hex.EncodeToString(sum[:8])

			rawEntries = append(rawEntries, rawEntry{
				tsNs:      tsNs,
				stream:    source,
				line:      line,
				truncated: truncated,
				ts:        ts,
				hash:      hash,
				originSeq: originSeq,
			})
			originSeq++
		}
	}

	sort.Slice(rawEntries, func(i, j int) bool {
		if cmp := comparePositiveNumericStrings(rawEntries[i].tsNs, rawEntries[j].tsNs); cmp != 0 {
			return cmp < 0
		}
		if c := strings.Compare(rawEntries[i].stream, rawEntries[j].stream); c != 0 {
			return c < 0
		}
		if c := strings.Compare(rawEntries[i].hash, rawEntries[j].hash); c != 0 {
			return c < 0
		}
		if c := strings.Compare(rawEntries[i].line, rawEntries[j].line); c != 0 {
			return c < 0
		}
		return rawEntries[i].originSeq < rawEntries[j].originSeq
	})

	sequenceByKey := make(map[string]int)
	entries := make([]LokiLogLineItem, 0, len(rawEntries))
	for _, item := range rawEntries {
		key := item.tsNs + "\x00" + item.stream + "\x00" + item.hash
		occurrence := sequenceByKey[key]
		sequenceByKey[key] = occurrence + 1

		id := fmt.Sprintf("agt_%d:%s:%s:%s:%s:%06d", agentID, container, item.tsNs, item.stream, item.hash, occurrence)
		entries = append(entries, LokiLogLineItem{
			ID:         id,
			TS:         item.ts,
			TSNs:       item.tsNs,
			Stream:     item.stream,
			Line:       item.line,
			Truncated:  item.truncated,
			lineHash:   item.hash,
			occurrence: occurrence,
		})
	}

	sortLokiLogItems(entries)
	return entries
}

func filterEntriesAfterCursor(entries []LokiLogLineItem, cursor logCursorPayload) []LokiLogLineItem {
	if len(entries) == 0 {
		return nil
	}
	filtered := make([]LokiLogLineItem, 0, len(entries))
	for _, item := range entries {
		if compareLokiCursorKey(item, cursor) > 0 {
			filtered = append(filtered, item)
		}
	}
	return filtered
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

func normalizeLokiLine(raw string) (string, bool) {
	normalized := strings.TrimRight(raw, "\r\n")
	if len(normalized) <= maxLokiLogLineBytes {
		return normalized, false
	}
	return normalized[:maxLokiLogLineBytes], true
}

func formatRFC3339FromUnixNano(tsNs string) string {
	ns, err := strconv.ParseInt(strings.TrimSpace(tsNs), 10, 64)
	if err != nil || ns <= 0 {
		return ""
	}
	return time.Unix(0, ns).UTC().Format(time.RFC3339Nano)
}

func buildLokiLogQL(agentID int, container string) string {
	return fmt.Sprintf("{agent_id=%q,container_name=%q}", strconv.Itoa(agentID), container)
}

func comparePositiveNumericStrings(a, b string) int {
	trimmedA := strings.TrimLeft(strings.TrimSpace(a), "0")
	trimmedB := strings.TrimLeft(strings.TrimSpace(b), "0")
	if trimmedA == "" {
		trimmedA = "0"
	}
	if trimmedB == "" {
		trimmedB = "0"
	}
	if len(trimmedA) < len(trimmedB) {
		return -1
	}
	if len(trimmedA) > len(trimmedB) {
		return 1
	}
	return strings.Compare(trimmedA, trimmedB)
}

func sortLokiLogItems(items []LokiLogLineItem) {
	sort.Slice(items, func(i, j int) bool {
		return compareLokiLogItemOrder(items[i], items[j]) < 0
	})
}

func normalizeStreamValuesForStableOrder(values []loki.StreamValue, queryDirection string) []loki.StreamValue {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]loki.StreamValue, len(values))
	copy(normalized, values)

	if strings.EqualFold(strings.TrimSpace(queryDirection), "BACKWARD") {
		for left, right := 0, len(normalized)-1; left < right; left, right = left+1, right-1 {
			normalized[left], normalized[right] = normalized[right], normalized[left]
		}
	}
	return normalized
}

func compareLokiLogItemOrder(left, right LokiLogLineItem) int {
	if cmp := comparePositiveNumericStrings(left.TSNs, right.TSNs); cmp != 0 {
		return cmp
	}
	if c := strings.Compare(left.Stream, right.Stream); c != 0 {
		return c
	}
	if c := strings.Compare(left.lineHash, right.lineHash); c != 0 {
		return c
	}
	if left.occurrence < right.occurrence {
		return -1
	}
	if left.occurrence > right.occurrence {
		return 1
	}
	return strings.Compare(left.ID, right.ID)
}

func compareLokiCursorKey(item LokiLogLineItem, cursor logCursorPayload) int {
	if cmp := comparePositiveNumericStrings(item.TSNs, cursor.LastTsNs); cmp != 0 {
		return cmp
	}
	if c := strings.Compare(item.Stream, cursor.LastStream); c != 0 {
		return c
	}
	if c := strings.Compare(item.lineHash, cursor.LastLineHash); c != 0 {
		return c
	}
	if item.occurrence < cursor.LastOccurrence {
		return -1
	}
	if item.occurrence > cursor.LastOccurrence {
		return 1
	}
	return strings.Compare(item.ID, cursor.LastID)
}
