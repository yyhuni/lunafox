package installapp

import "strings"

const (
	DefaultMaxLogLines = 2000
	DefaultMaxLineSize = 8 * 1024
)

type LogRing struct {
	maxLines int
	maxBytes int
	lines    []string
}

func NewLogRing(maxLines, maxBytes int) *LogRing {
	if maxLines <= 0 {
		maxLines = DefaultMaxLogLines
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxLineSize
	}
	return &LogRing{
		maxLines: maxLines,
		maxBytes: maxBytes,
		lines:    make([]string, 0, maxLines),
	}
}

func (ring *LogRing) Append(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.SplitAfter(raw, "\n")
	if len(parts) == 0 {
		parts = []string{raw}
	}
	appended := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		clipped := clipString(part, ring.maxBytes)
		ring.lines = append(ring.lines, clipped)
		appended = append(appended, clipped)
	}
	if len(ring.lines) > ring.maxLines {
		start := len(ring.lines) - ring.maxLines
		ring.lines = append([]string(nil), ring.lines[start:]...)
	}
	return appended
}

func (ring *LogRing) String() string {
	if len(ring.lines) == 0 {
		return ""
	}
	return strings.Join(ring.lines, "")
}

func clipString(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	const suffix = "...(truncated)\n"
	if limit <= len(suffix) {
		return value[:limit]
	}
	return value[:limit-len(suffix)] + suffix
}
