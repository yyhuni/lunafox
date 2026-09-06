package application

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	globalAssetSearchMaxQueryBytes    = 2048
	globalAssetSearchMaxConditions    = 10
	globalAssetSearchMinContainsRunes = 3
)

// ParseGlobalAssetSearchQuery parses and validates the only two search modes
// accepted by P0. A malformed structural expression never falls back to a
// plain URL search because that could widen the query unexpectedly.
func ParseGlobalAssetSearchQuery(raw string) (GlobalAssetSearchAST, error) {
	if !utf8.ValidString(raw) {
		return GlobalAssetSearchAST{}, fmt.Errorf("%w: q must be valid UTF-8", ErrInvalidGlobalAssetSearchQuery)
	}
	if len(raw) > globalAssetSearchMaxQueryBytes {
		return GlobalAssetSearchAST{}, fmt.Errorf("%w: q exceeds %d UTF-8 bytes", ErrInvalidGlobalAssetSearchQuery, globalAssetSearchMaxQueryBytes)
	}
	if strings.TrimSpace(raw) == "" {
		return GlobalAssetSearchAST{}, fmt.Errorf("%w: q is required", ErrInvalidGlobalAssetSearchQuery)
	}

	if isGlobalAssetSearchPlainURL(raw) || !looksLikeGlobalAssetSearchStructure(raw) {
		// Plain q is an exact observed-URL value. Preserve it verbatim rather
		// than treating surrounding bytes as presentation whitespace.
		if utf8.RuneCountInString(raw) < globalAssetSearchMinContainsRunes {
			return GlobalAssetSearchAST{}, fmt.Errorf("%w: url contains value must contain at least %d Unicode characters", ErrInvalidGlobalAssetSearchQuery, globalAssetSearchMinContainsRunes)
		}
		return GlobalAssetSearchAST{Mode: GlobalAssetSearchModePlainURL, PlainURL: raw}, nil
	}

	parser := globalAssetSearchParser{input: strings.TrimSpace(raw)}
	conditions, err := parser.parseStructured()
	if err != nil {
		return GlobalAssetSearchAST{}, fmt.Errorf("%w: %v", ErrInvalidGlobalAssetSearchQuery, err)
	}
	if len(conditions) == 0 || len(conditions) > globalAssetSearchMaxConditions {
		return GlobalAssetSearchAST{}, fmt.Errorf("%w: structured query must contain at most %d conditions", ErrInvalidGlobalAssetSearchQuery, globalAssetSearchMaxConditions)
	}
	return GlobalAssetSearchAST{Mode: GlobalAssetSearchModeStructured, Conditions: conditions}, nil
}

func isGlobalAssetSearchPlainURL(raw string) bool {
	return len(raw) >= len("http://") && strings.EqualFold(raw[:len("http://")], "http://") ||
		len(raw) >= len("https://") && strings.EqualFold(raw[:len("https://")], "https://")
}

func looksLikeGlobalAssetSearchStructure(raw string) bool {
	if strings.Contains(raw, "&&") || strings.Contains(raw, "||") || strings.Contains(raw, "!=") || strings.ContainsAny(raw, "()") {
		return true
	}
	for index := 0; index < len(raw); {
		if index > 0 && !isGlobalAssetSearchSpace(raw[index-1]) {
			index++
			continue
		}
		if !isGlobalAssetSearchIdentifierStart(raw[index]) {
			index++
			continue
		}
		index++
		for index < len(raw) && isGlobalAssetSearchIdentifierPart(raw[index]) {
			index++
		}
		for index < len(raw) && isGlobalAssetSearchSpace(raw[index]) {
			index++
		}
		if index < len(raw) && raw[index] == '=' {
			return true
		}
	}
	return false
}

func isGlobalAssetSearchIdentifierStart(ch byte) bool {
	return ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z'
}

func isGlobalAssetSearchIdentifierPart(ch byte) bool {
	return isGlobalAssetSearchIdentifierStart(ch) || ch >= '0' && ch <= '9' || ch == '_'
}

func isGlobalAssetSearchSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

type globalAssetSearchParser struct {
	input string
	pos   int
}

func (parser *globalAssetSearchParser) parseStructured() ([]GlobalAssetSearchCondition, error) {
	conditions := make([]GlobalAssetSearchCondition, 0, 1)
	parser.skipSpace()
	for {
		condition, err := parser.parseCondition()
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, condition)
		if len(conditions) > globalAssetSearchMaxConditions {
			return nil, fmt.Errorf("structured query must contain at most %d conditions", globalAssetSearchMaxConditions)
		}
		parser.skipSpace()
		if parser.atEnd() {
			return conditions, nil
		}
		if !parser.consume("&&") {
			return nil, fmt.Errorf("expected && between conditions at byte %d", parser.pos)
		}
		parser.skipSpace()
		if parser.atEnd() {
			return nil, fmt.Errorf("missing condition after &&")
		}
	}
}

func (parser *globalAssetSearchParser) parseCondition() (GlobalAssetSearchCondition, error) {
	fieldRaw, err := parser.parseIdentifier()
	if err != nil {
		return GlobalAssetSearchCondition{}, err
	}
	field, err := globalAssetSearchField(fieldRaw)
	if err != nil {
		return GlobalAssetSearchCondition{}, err
	}
	parser.skipSpace()
	operator, err := parser.parseOperator()
	if err != nil {
		return GlobalAssetSearchCondition{}, err
	}
	parser.skipSpace()
	value, err := parser.parseQuotedValue()
	if err != nil {
		return GlobalAssetSearchCondition{}, err
	}
	return newGlobalAssetSearchCondition(field, operator, value)
}

func (parser *globalAssetSearchParser) parseIdentifier() (string, error) {
	if parser.atEnd() || !isGlobalAssetSearchIdentifierStart(parser.input[parser.pos]) {
		return "", fmt.Errorf("expected field name at byte %d", parser.pos)
	}
	start := parser.pos
	parser.pos++
	for !parser.atEnd() && isGlobalAssetSearchIdentifierPart(parser.input[parser.pos]) {
		parser.pos++
	}
	return parser.input[start:parser.pos], nil
}

func (parser *globalAssetSearchParser) parseOperator() (GlobalAssetSearchOperator, error) {
	if parser.consume("==") {
		return GlobalAssetSearchOperatorExact, nil
	}
	if parser.consume("=") {
		return GlobalAssetSearchOperatorContains, nil
	}
	return "", fmt.Errorf("expected = or == at byte %d", parser.pos)
}

func (parser *globalAssetSearchParser) parseQuotedValue() (string, error) {
	if parser.atEnd() || parser.input[parser.pos] != '"' {
		return "", fmt.Errorf("expected quoted value at byte %d", parser.pos)
	}
	start := parser.pos
	parser.pos++
	escaped := false
	for !parser.atEnd() {
		ch := parser.input[parser.pos]
		parser.pos++
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			literal := parser.input[start:parser.pos]
			value, err := strconv.Unquote(literal)
			if err != nil {
				return "", fmt.Errorf("invalid quoted value: %w", err)
			}
			return value, nil
		}
	}
	return "", fmt.Errorf("unterminated quoted value")
}

func (parser *globalAssetSearchParser) skipSpace() {
	for !parser.atEnd() && isGlobalAssetSearchSpace(parser.input[parser.pos]) {
		parser.pos++
	}
}

func (parser *globalAssetSearchParser) consume(value string) bool {
	if strings.HasPrefix(parser.input[parser.pos:], value) {
		parser.pos += len(value)
		return true
	}
	return false
}

func (parser *globalAssetSearchParser) atEnd() bool {
	return parser.pos >= len(parser.input)
}

func globalAssetSearchField(raw string) (GlobalAssetSearchField, error) {
	switch raw {
	case string(GlobalAssetSearchFieldURL):
		return GlobalAssetSearchFieldURL, nil
	case string(GlobalAssetSearchFieldHost):
		return GlobalAssetSearchFieldHost, nil
	case string(GlobalAssetSearchFieldTitle):
		return GlobalAssetSearchFieldTitle, nil
	case string(GlobalAssetSearchFieldStatusCode):
		return GlobalAssetSearchFieldStatusCode, nil
	case string(GlobalAssetSearchFieldTech):
		return GlobalAssetSearchFieldTech, nil
	default:
		return "", fmt.Errorf("unsupported field %q", raw)
	}
}

func newGlobalAssetSearchCondition(field GlobalAssetSearchField, operator GlobalAssetSearchOperator, rawValue string) (GlobalAssetSearchCondition, error) {
	if field == GlobalAssetSearchFieldStatusCode {
		if rawValue == "" {
			return GlobalAssetSearchCondition{}, fmt.Errorf("statusCode must be an integer")
		}
		for _, character := range rawValue {
			if character < '0' || character > '9' {
				return GlobalAssetSearchCondition{}, fmt.Errorf("statusCode must be an integer")
			}
		}
		value, err := strconv.Atoi(rawValue)
		if err != nil {
			return GlobalAssetSearchCondition{}, fmt.Errorf("statusCode must be an integer")
		}
		return GlobalAssetSearchCondition{Field: field, Operator: operator, StatusCode: &value}, nil
	}
	if field == GlobalAssetSearchFieldURL {
		// Observed URL identity is exact even when callers retain the existing
		// url="..." filter spelling. The value is not URL-decoded or rebuilt.
		operator = GlobalAssetSearchOperatorExact
	}
	if field == GlobalAssetSearchFieldHost || field == GlobalAssetSearchFieldTitle {
		if operator == GlobalAssetSearchOperatorContains && utf8.RuneCountInString(strings.TrimSpace(rawValue)) < globalAssetSearchMinContainsRunes {
			return GlobalAssetSearchCondition{}, fmt.Errorf("%s contains value must contain at least %d Unicode characters", field, globalAssetSearchMinContainsRunes)
		}
	}
	return GlobalAssetSearchCondition{Field: field, Operator: operator, Text: rawValue}, nil
}
