package loki

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TimelineBuildOptions struct {
	IDPrefix     string
	MaxLineBytes int
}

type TimelineEntry struct {
	ID         string
	TS         string
	TSNs       string
	Stream     string
	Line       string
	Truncated  bool
	LineHash   string
	Occurrence int
}

type TimelineCursorKey struct {
	LastTsNs       string
	LastID         string
	LastStream     string
	LastLineHash   string
	LastOccurrence int
}

func BuildTimelineEntries(streams []StreamResult, options TimelineBuildOptions, queryDirection string) []TimelineEntry {
	type rawEntry struct {
		tsNs      string
		stream    string
		line      string
		truncated bool
		ts        string
		hash      string
		originSeq int
	}

	idPrefix := strings.TrimSpace(options.IDPrefix)
	if idPrefix == "" {
		idPrefix = "log"
	}

	rawEntries := make([]rawEntry, 0)
	originSeq := 0
	for _, streamResult := range streams {
		source := strings.TrimSpace(streamResult.Stream["source"])
		if source == "" {
			source = "stdout"
		}
		values := NormalizeStreamValuesForTimeline(streamResult.Values, queryDirection)
		for _, value := range values {
			tsNs := strings.TrimSpace(value.TsNs)
			if tsNs == "" {
				continue
			}
			ts := FormatRFC3339FromUnixNano(tsNs)
			if ts == "" {
				continue
			}

			line, truncated := NormalizeTimelineLine(value.Line, options.MaxLineBytes)
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
		if cmp := ComparePositiveNumericStrings(rawEntries[i].tsNs, rawEntries[j].tsNs); cmp != 0 {
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
	entries := make([]TimelineEntry, 0, len(rawEntries))
	for _, item := range rawEntries {
		key := item.tsNs + "\x00" + item.stream + "\x00" + item.hash
		occurrence := sequenceByKey[key]
		sequenceByKey[key] = occurrence + 1

		entries = append(entries, TimelineEntry{
			ID:         fmt.Sprintf("%s:%s:%s:%s:%06d", idPrefix, item.tsNs, item.stream, item.hash, occurrence),
			TS:         item.ts,
			TSNs:       item.tsNs,
			Stream:     item.stream,
			Line:       item.line,
			Truncated:  item.truncated,
			LineHash:   item.hash,
			Occurrence: occurrence,
		})
	}

	SortTimelineEntries(entries)
	return entries
}

func FilterTimelineEntriesAfterCursor(entries []TimelineEntry, cursor TimelineCursorKey) []TimelineEntry {
	if len(entries) == 0 {
		return nil
	}
	filtered := make([]TimelineEntry, 0, len(entries))
	for _, item := range entries {
		if CompareTimelineCursorKey(item, cursor) > 0 {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func FilterTimelineEntriesBeforeCursor(entries []TimelineEntry, cursor TimelineCursorKey) []TimelineEntry {
	if len(entries) == 0 {
		return nil
	}
	filtered := make([]TimelineEntry, 0, len(entries))
	for _, item := range entries {
		if CompareTimelineCursorKey(item, cursor) < 0 {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func NormalizeTimelineLine(raw string, maxBytes int) (string, bool) {
	normalized := strings.TrimRight(raw, "\r\n")
	if maxBytes <= 0 || len(normalized) <= maxBytes {
		return normalized, false
	}
	return normalized[:maxBytes], true
}

func FormatRFC3339FromUnixNano(tsNs string) string {
	ns, err := strconv.ParseInt(strings.TrimSpace(tsNs), 10, 64)
	if err != nil || ns <= 0 {
		return ""
	}
	return time.Unix(0, ns).UTC().Format(time.RFC3339Nano)
}

func NormalizeStreamValuesForTimeline(values []StreamValue, queryDirection string) []StreamValue {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]StreamValue, len(values))
	copy(normalized, values)
	if strings.EqualFold(strings.TrimSpace(queryDirection), "BACKWARD") {
		for left, right := 0, len(normalized)-1; left < right; left, right = left+1, right-1 {
			normalized[left], normalized[right] = normalized[right], normalized[left]
		}
	}
	return normalized
}

func SortTimelineEntries(entries []TimelineEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return CompareTimelineEntryOrder(entries[i], entries[j]) < 0
	})
}

func CompareTimelineEntryOrder(left, right TimelineEntry) int {
	if cmp := ComparePositiveNumericStrings(left.TSNs, right.TSNs); cmp != 0 {
		return cmp
	}
	if c := strings.Compare(left.Stream, right.Stream); c != 0 {
		return c
	}
	if c := strings.Compare(left.LineHash, right.LineHash); c != 0 {
		return c
	}
	if left.Occurrence < right.Occurrence {
		return -1
	}
	if left.Occurrence > right.Occurrence {
		return 1
	}
	return strings.Compare(left.ID, right.ID)
}

func CompareTimelineCursorKey(item TimelineEntry, cursor TimelineCursorKey) int {
	if cmp := ComparePositiveNumericStrings(item.TSNs, cursor.LastTsNs); cmp != 0 {
		return cmp
	}
	if c := strings.Compare(item.Stream, cursor.LastStream); c != 0 {
		return c
	}
	if c := strings.Compare(item.LineHash, cursor.LastLineHash); c != 0 {
		return c
	}
	if item.Occurrence < cursor.LastOccurrence {
		return -1
	}
	if item.Occurrence > cursor.LastOccurrence {
		return 1
	}
	return strings.Compare(item.ID, cursor.LastID)
}

func ComparePositiveNumericStrings(a, b string) int {
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
