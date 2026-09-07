package scope

import (
	"fmt"
	"strconv"
	"strings"
)

// ExtractField removes one structured field from a filter expression and
// returns its parsed occurrences. Callers can validate the extracted field's
// operator/value independently before applying the remaining user filters.
func ExtractField(filterStr, field string) (string, []FilterGroup, error) {
	if EmptyIfBlank(filterStr) == "" {
		return "", nil, nil
	}
	if hasHTTPURLPrefix(filterStr) || !hasFilterSyntax(filterStr) {
		return filterStr, nil, nil
	}
	trimmed := strings.TrimSpace(filterStr)
	if !isWellFormedFilterExpression(trimmed) {
		return "", nil, fmt.Errorf("invalid filter")
	}

	groups := parseFilter(trimmed)
	if len(groups) == 0 {
		return "", nil, fmt.Errorf("invalid filter")
	}
	remaining := make([]FilterGroup, 0, len(groups))
	matched := make([]FilterGroup, 0, 1)
	for _, group := range groups {
		if strings.EqualFold(group.Filter.Field, field) {
			matched = append(matched, group)
			continue
		}
		remaining = append(remaining, group)
	}
	if len(matched) == 0 {
		return trimmed, nil, nil
	}
	return RenderFilterGroups(remaining), matched, nil
}

// isWellFormedFilterExpression accepts the existing filter grammar while
// ensuring that ExtractField never drops a malformed remainder silently.
func isWellFormedFilterExpression(filterStr string) bool {
	filtersFound := 0
	protected := filterPattern.ReplaceAllStringFunc(filterStr, func(string) string {
		placeholder := buildFilterPlaceholder(filtersFound)
		filtersFound++
		return placeholder
	})
	if filtersFound == 0 {
		return false
	}

	normalized := orPattern.ReplaceAllString(protected, " __OR__ ")
	normalized = andPattern.ReplaceAllString(normalized, " __AND__ ")
	normalized = strings.ReplaceAll(normalized, "(", " ( ")
	normalized = strings.ReplaceAll(normalized, ")", " ) ")

	expectOperand := true
	parentheses := 0
	for _, token := range strings.Fields(normalized) {
		switch token {
		case "__AND__", "__OR__":
			if expectOperand {
				return false
			}
			expectOperand = true
		case "(":
			if !expectOperand {
				// The legacy grammar treats adjacent conditions as AND.
				expectOperand = true
			}
			parentheses++
		case ")":
			if expectOperand || parentheses == 0 {
				return false
			}
			parentheses--
		default:
			if _, ok := parseFilterPlaceholder(token); !ok {
				return false
			}
			// Adjacent conditions are an implicit AND in the established parser.
			expectOperand = false
		}
	}

	return !expectOperand && parentheses == 0
}

// RenderFilterGroups serializes parsed groups into the canonical filter syntax
// understood by this package. It is used only after a field has been removed.
func RenderFilterGroups(groups []FilterGroup) string {
	if len(groups) == 0 {
		return ""
	}
	parts := make([]string, 0, len(groups)*2-1)
	for index, group := range groups {
		if index > 0 {
			if group.LogicalOp == LogicalOr {
				parts = append(parts, "||")
			} else {
				parts = append(parts, "&&")
			}
		}
		parts = append(parts, group.Filter.Field+group.Filter.Operator+strconv.Quote(group.Filter.Value))
	}
	return strings.Join(parts, " ")
}
