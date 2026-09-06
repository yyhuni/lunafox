package application

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const (
	defaultPOCPageSize = 50
	maxPOCPageSize     = 1000
	pocTokenVersion    = 1
)

type pocPageToken struct {
	Version     int    `json:"v"`
	PageSize    int    `json:"s"`
	Filter      string `json:"f"`
	OrderBy     string `json:"r"`
	CursorValue string `json:"c"`
	CursorID    string `json:"i"`
}

func normalizePOCListQuery(input POCListQuery) (POCListQuery, error) {
	pageSize := input.PageSize
	if pageSize == 0 {
		pageSize = defaultPOCPageSize
	}
	if pageSize < 0 {
		return POCListQuery{}, fmt.Errorf("%w: pageSize must be positive", ErrInvalidQuery)
	}
	if pageSize > maxPOCPageSize {
		return POCListQuery{}, fmt.Errorf("%w: pageSize exceeds maximum", ErrInvalidQuery)
	}
	filter, err := normalizePOCFilter(input.Filter)
	if err != nil {
		return POCListQuery{}, err
	}
	orderBy, err := normalizePOCOrderBy(input.OrderBy)
	if err != nil {
		return POCListQuery{}, err
	}
	output := POCListQuery{PageSize: pageSize, Filter: filter, OrderBy: orderBy}
	if strings.TrimSpace(input.PageToken) != "" {
		payload, err := decodePOCPageToken(input.PageToken)
		if err != nil {
			return POCListQuery{}, err
		}
		if payload.PageSize != pageSize || payload.Filter != filter || payload.OrderBy != orderBy || payload.CursorValue == "" || payload.CursorID == "" {
			return POCListQuery{}, fmt.Errorf("%w: pageToken query shape mismatch", ErrInvalidQuery)
		}
		output.Cursor = &POCCursor{Value: payload.CursorValue, TemplateID: payload.CursorID}
	}
	return output, nil
}

func CompletePOCListResult(query POCListQuery, result *POCListResult) error {
	if result == nil {
		return fmt.Errorf("%w: empty list result", ErrInvalidQuery)
	}
	if result.HasMore && result.LastCursor.Value != "" && result.LastCursor.TemplateID != "" {
		encoded, err := json.Marshal(pocPageToken{Version: pocTokenVersion, PageSize: query.PageSize, Filter: query.Filter, OrderBy: query.OrderBy, CursorValue: result.LastCursor.Value, CursorID: result.LastCursor.TemplateID})
		if err != nil {
			return err
		}
		result.NextPageToken = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return nil
}

func decodePOCPageToken(value string) (pocPageToken, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return pocPageToken{}, fmt.Errorf("%w: malformed pageToken", ErrInvalidQuery)
	}
	var token pocPageToken
	decoder := json.NewDecoder(strings.NewReader(string(decoded)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&token); err != nil || token.Version != pocTokenVersion || token.PageSize <= 0 || token.CursorValue == "" || token.CursorID == "" {
		return pocPageToken{}, fmt.Errorf("%w: malformed pageToken", ErrInvalidQuery)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return pocPageToken{}, fmt.Errorf("%w: malformed pageToken", ErrInvalidQuery)
	}
	return token, nil
}

var allowedPOCKeywordFields = map[string]struct{}{"templateId": {}, "name": {}, "author": {}, "description": {}, "tags": {}, "cve": {}, "cwe": {}}

// POCFilterTerm is the parsed, transport-neutral representation of one
// approved catalog filter expression. Repository code consumes this parser so
// normalization and actual matching cannot drift apart.
type POCFilterTerm struct {
	Field string
	Value string
	Facet bool
}

type POCFilterGroup []POCFilterTerm

// ParsePOCFilter parses AND/OR expressions while respecting quoted values and
// escaped quotes/backslashes. It intentionally accepts only the tiny grammar
// used by the catalog boundary; callers still apply field/operator policy.
func ParsePOCFilter(value string) ([]POCFilterGroup, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	andParts, err := splitFilterOperators(value, "AND")
	if err != nil {
		return nil, fmt.Errorf("%w: unsupported filter expression", ErrInvalidQuery)
	}
	groups := make([]POCFilterGroup, 0, len(andParts))
	for _, andPart := range andParts {
		orParts, err := splitFilterOperators(andPart, "OR")
		if err != nil {
			return nil, fmt.Errorf("%w: unsupported filter expression", ErrInvalidQuery)
		}
		group := make(POCFilterGroup, 0, len(orParts))
		for _, raw := range orParts {
			raw = strings.TrimSpace(raw)
			operator := "="
			operatorIndex := findFilterOperator(raw)
			if operatorIndex < 0 {
				return nil, fmt.Errorf("%w: unsupported filter expression", ErrInvalidQuery)
			}
			if strings.HasPrefix(raw[operatorIndex:], "==") {
				operator = "=="
			}
			field := strings.TrimSpace(raw[:operatorIndex])
			quoted := strings.TrimSpace(raw[operatorIndex+len(operator):])
			if field == "" || quoted == "" || quoted[0] != '"' {
				return nil, fmt.Errorf("%w: filter values must be quoted", ErrInvalidQuery)
			}
			parsedValue, err := strconv.Unquote(quoted)
			if err != nil || strings.TrimSpace(parsedValue) == "" || strings.ContainsAny(parsedValue, "\r\n") {
				return nil, fmt.Errorf("%w: filter values must be quoted", ErrInvalidQuery)
			}
			group = append(group, POCFilterTerm{Field: field, Value: parsedValue, Facet: operator == "=="})
		}
		if len(group) == 0 {
			return nil, fmt.Errorf("%w: unsupported filter expression", ErrInvalidQuery)
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func findFilterOperator(value string) int {
	inQuote, escaped := false, false
	for index, character := range value {
		if inQuote {
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				inQuote = false
			}
			continue
		}
		if character == '"' {
			inQuote = true
			continue
		}
		if character == '=' {
			return index
		}
	}
	return -1
}

func splitFilterOperators(value, operator string) ([]string, error) {
	parts := make([]string, 0, 2)
	start := 0
	inQuote, escaped := false, false
	for index := 0; index < len(value); index++ {
		character := rune(value[index])
		if inQuote {
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				inQuote = false
			}
			continue
		}
		if character == '"' {
			inQuote = true
			continue
		}
		if index+1 < len(value) && ((operator == "AND" && value[index:index+2] == "&&") || (operator == "OR" && value[index:index+2] == "||")) {
			parts = append(parts, value[start:index])
			start = index + 2
			index++
			continue
		}
		if index+len(operator) <= len(value) && strings.EqualFold(value[index:index+len(operator)], operator) {
			beforeOK := index == 0 || unicode.IsSpace(rune(value[index-1]))
			afterIndex := index + len(operator)
			afterOK := afterIndex == len(value) || unicode.IsSpace(rune(value[afterIndex]))
			if beforeOK && afterOK {
				parts = append(parts, value[start:index])
				start = afterIndex
				index = afterIndex - 1
			}
		}
	}
	if inQuote || escaped {
		return nil, fmt.Errorf("unterminated quoted filter value")
	}
	parts = append(parts, value[start:])
	return parts, nil
}

func normalizePOCFilter(value string) (string, error) {
	groups, err := ParsePOCFilter(value)
	if err != nil {
		return "", err
	}
	if len(groups) == 0 {
		return "", nil
	}
	normalized := make([]string, 0, len(groups))
	for _, group := range groups {
		normalizedTerms := make([]string, 0, len(group))
		for _, term := range group {
			if _, ok := allowedPOCKeywordFields[term.Field]; !ok {
				if term.Field != "severity" || !term.Facet {
					return "", fmt.Errorf("%w: unsupported filter field", ErrInvalidQuery)
				}
			}
			if term.Facet && term.Field != "severity" && term.Field != "tags" {
				return "", fmt.Errorf("%w: facet is not supported for this field", ErrInvalidQuery)
			}
			if term.Field == "severity" && !isAllowedSeverity(term.Value) {
				return "", fmt.Errorf("%w: unsupported severity", ErrInvalidQuery)
			}
			if strings.ContainsAny(term.Value, "*%") {
				return "", fmt.Errorf("%w: wildcard filter is not supported", ErrInvalidQuery)
			}
			op := "="
			if term.Facet {
				op = "=="
			}
			normalizedTerms = append(normalizedTerms, term.Field+op+strconv.Quote(strings.ToLower(term.Value)))
		}
		normalized = append(normalized, strings.Join(normalizedTerms, " OR "))
	}
	return strings.Join(normalized, " AND "), nil
}

func isAllowedSeverity(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "info", "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func normalizePOCOrderBy(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "templateId asc", nil
	}
	parts := strings.Fields(value)
	if len(parts) > 2 {
		return "", fmt.Errorf("%w: unsupported orderBy", ErrInvalidQuery)
	}
	allowed := map[string]struct{}{"templateId": {}, "name": {}, "severity": {}, "updatedAt": {}}
	if _, ok := allowed[parts[0]]; !ok {
		return "", fmt.Errorf("%w: unsupported orderBy field", ErrInvalidQuery)
	}
	direction := "asc"
	if len(parts) == 2 {
		direction = strings.ToLower(parts[1])
	}
	if direction != "asc" && direction != "desc" {
		return "", fmt.Errorf("%w: unsupported orderBy direction", ErrInvalidQuery)
	}
	return parts[0] + " " + direction, nil
}
