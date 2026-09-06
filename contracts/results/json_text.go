package results

import (
	"errors"
	"unicode/utf8"
)

var (
	ErrInvalidJSONTextUTF8   = errors.New("json text must be valid UTF-8")
	ErrUnpairedJSONSurrogate = errors.New("json text contains an unpaired Unicode surrogate escape")
	errInvalidUnicodeEscape  = errors.New("json text contains an invalid Unicode escape")
)

// JSONTextValidator incrementally checks the Unicode transport properties
// that encoding/json otherwise recovers from. JSON syntax remains the
// decoder's responsibility; this validator only prevents lossy text replace.
type JSONTextValidator struct {
	pendingUTF8          []byte
	inString             bool
	afterEscape          bool
	unicodeDigits        int
	unicodeValue         uint16
	requiresLowSurrogate bool
}

// ValidateJSONText applies the same lossless JSON-text check to one complete
// value.
func ValidateJSONText(payload []byte) error {
	var validator JSONTextValidator
	if err := validator.Write(payload); err != nil {
		return err
	}
	return validator.Finalize()
}

// Write checks one chunk while retaining only an incomplete UTF-8 rune between
// calls. It is safe for callers that stream a request body or JSONL artifact.
func (validator *JSONTextValidator) Write(payload []byte) error {
	input := payload
	if len(validator.pendingUTF8) > 0 {
		input = make([]byte, 0, len(validator.pendingUTF8)+len(payload))
		input = append(input, validator.pendingUTF8...)
		input = append(input, payload...)
		validator.pendingUTF8 = nil
	}

	for len(input) > 0 {
		if input[0] < utf8.RuneSelf {
			if err := validator.consumeASCII(input[0]); err != nil {
				return err
			}
			input = input[1:]
			continue
		}
		if !utf8.FullRune(input) {
			validator.pendingUTF8 = append(validator.pendingUTF8[:0], input...)
			return nil
		}
		runeValue, size := utf8.DecodeRune(input)
		if runeValue == utf8.RuneError && size == 1 {
			return ErrInvalidJSONTextUTF8
		}
		if validator.requiresLowSurrogate {
			return ErrUnpairedJSONSurrogate
		}
		if validator.afterEscape || validator.unicodeDigits > 0 {
			return errInvalidUnicodeEscape
		}
		input = input[size:]
	}
	return nil
}

// Finalize rejects a trailing partial UTF-8 rune or an unpaired high
// surrogate. Other malformed JSON is deliberately left to encoding/json.
func (validator *JSONTextValidator) Finalize() error {
	if len(validator.pendingUTF8) > 0 {
		return ErrInvalidJSONTextUTF8
	}
	if validator.requiresLowSurrogate {
		return ErrUnpairedJSONSurrogate
	}
	return nil
}

func (validator *JSONTextValidator) consumeASCII(value byte) error {
	if !validator.inString {
		if value == '"' {
			validator.inString = true
		}
		return nil
	}
	if validator.unicodeDigits > 0 {
		digit, ok := hexadecimalDigit(value)
		if !ok {
			return errInvalidUnicodeEscape
		}
		validator.unicodeValue = validator.unicodeValue<<4 | uint16(digit)
		validator.unicodeDigits--
		if validator.unicodeDigits == 0 {
			return validator.completeUnicodeEscape()
		}
		return nil
	}
	if validator.afterEscape {
		if validator.requiresLowSurrogate && value != 'u' {
			return ErrUnpairedJSONSurrogate
		}
		validator.afterEscape = false
		if value == 'u' {
			validator.unicodeDigits = 4
			validator.unicodeValue = 0
		}
		return nil
	}
	if validator.requiresLowSurrogate {
		if value != '\\' {
			return ErrUnpairedJSONSurrogate
		}
		validator.afterEscape = true
		return nil
	}
	switch value {
	case '\\':
		validator.afterEscape = true
	case '"':
		validator.inString = false
	}
	return nil
}

func (validator *JSONTextValidator) completeUnicodeEscape() error {
	value := validator.unicodeValue
	switch {
	case value >= 0xd800 && value <= 0xdbff:
		if validator.requiresLowSurrogate {
			return ErrUnpairedJSONSurrogate
		}
		validator.requiresLowSurrogate = true
	case value >= 0xdc00 && value <= 0xdfff:
		if !validator.requiresLowSurrogate {
			return ErrUnpairedJSONSurrogate
		}
		validator.requiresLowSurrogate = false
	case validator.requiresLowSurrogate:
		return ErrUnpairedJSONSurrogate
	}
	return nil
}

func hexadecimalDigit(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}
