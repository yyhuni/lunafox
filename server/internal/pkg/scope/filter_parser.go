package scope

import (
	"strconv"
	"strings"
)

const (
	filterPlaceholderPrefix = "__FILTER_"
	filterPlaceholderSuffix = "__"
)

// parseFilter parses filter string into FilterGroup list.
func parseFilter(filterStr string) []FilterGroup {
	if filterStr == "" {
		return nil
	}

	var filtersFound []string
	protected := filterPattern.ReplaceAllStringFunc(filterStr, func(match string) string {
		idx := len(filtersFound)
		filtersFound = append(filtersFound, match)
		return buildFilterPlaceholder(idx)
	})

	normalized := orPattern.ReplaceAllString(protected, " __OR__ ")
	normalized = andPattern.ReplaceAllString(normalized, " __AND__ ")
	tokens := strings.Fields(normalized)

	groups := make([]FilterGroup, 0, len(tokens))
	pendingOp := LogicalAnd

	for _, token := range tokens {
		switch token {
		case "__OR__":
			pendingOp = LogicalOr
		case "__AND__":
			pendingOp = LogicalAnd
		default:
			idx, ok := parseFilterPlaceholder(token)
			if !ok || idx < 0 || idx >= len(filtersFound) {
				continue
			}

			match := filterPattern.FindStringSubmatch(filtersFound[idx])
			if len(match) != 4 {
				continue
			}

			logicalOp := LogicalAnd
			if len(groups) > 0 {
				logicalOp = pendingOp
			}

			groups = append(groups, FilterGroup{
				Filter: ParsedFilter{
					Field:    match[1],
					Operator: match[2],
					Value:    match[3],
				},
				LogicalOp: logicalOp,
			})
			pendingOp = LogicalAnd
		}
	}

	return groups
}

func buildFilterPlaceholder(index int) string {
	return filterPlaceholderPrefix + strconv.Itoa(index) + filterPlaceholderSuffix
}

func parseFilterPlaceholder(token string) (int, bool) {
	if !strings.HasPrefix(token, filterPlaceholderPrefix) || !strings.HasSuffix(token, filterPlaceholderSuffix) {
		return 0, false
	}

	idxStr := strings.TrimPrefix(token, filterPlaceholderPrefix)
	idxStr = strings.TrimSuffix(idxStr, filterPlaceholderSuffix)
	if idxStr == "" {
		return 0, false
	}

	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return 0, false
	}

	return idx, true
}
