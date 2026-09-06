package fingerprintdetectionruntime

import (
	"errors"
	"unicode/utf8"
)

// validateJSONText prevents encoding/json from replacing malformed scanner
// text before this Engine can decide whether to use the record.
func validateJSONText(payload []byte) error {
	if !utf8.Valid(payload) {
		return errors.New("JSON text must be valid UTF-8")
	}
	inString, escaped, pendingHighSurrogate := false, false, false
	for index := 0; index < len(payload); index++ {
		value := payload[index]
		if !inString {
			if value == '"' {
				inString = true
			}
			continue
		}
		if escaped {
			escaped = false
			if value != 'u' {
				if pendingHighSurrogate {
					return errors.New("JSON text contains an unpaired Unicode surrogate escape")
				}
				continue
			}
			if index+4 >= len(payload) {
				return errors.New("JSON text contains an invalid Unicode escape")
			}
			var code uint16
			for offset := 1; offset <= 4; offset++ {
				digit, ok := hexadecimalDigit(payload[index+offset])
				if !ok {
					return errors.New("JSON text contains an invalid Unicode escape")
				}
				code = code<<4 | uint16(digit)
			}
			index += 4
			switch {
			case code >= 0xd800 && code <= 0xdbff:
				if pendingHighSurrogate {
					return errors.New("JSON text contains an unpaired Unicode surrogate escape")
				}
				pendingHighSurrogate = true
			case code >= 0xdc00 && code <= 0xdfff:
				if !pendingHighSurrogate {
					return errors.New("JSON text contains an unpaired Unicode surrogate escape")
				}
				pendingHighSurrogate = false
			case pendingHighSurrogate:
				return errors.New("JSON text contains an unpaired Unicode surrogate escape")
			}
			continue
		}
		if pendingHighSurrogate {
			if value != '\\' {
				return errors.New("JSON text contains an unpaired Unicode surrogate escape")
			}
			escaped = true
			continue
		}
		switch value {
		case '\\':
			escaped = true
		case '"':
			inString = false
		}
	}
	if pendingHighSurrogate {
		return errors.New("JSON text contains an unpaired Unicode surrogate escape")
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
