package results

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateJSONTextRejectsOnlyLossyUnicodeInput(t *testing.T) {
	invalidUTF8 := append([]byte("{\"url\":\"https://example.com/"), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte("\"}")...)

	for name, test := range map[string]struct {
		payload []byte
		want    error
	}{
		"valid UTF-8": {
			payload: []byte("{\"url\":\"https://example.com/界\"}"),
		},
		"paired surrogate": {
			payload: []byte("{\"url\":\"https://example.com/\\uD83D\\uDE00\"}"),
		},
		"escaped NUL remains decoder concern": {
			payload: []byte("{\"url\":\"https://example.com/\\u0000\"}"),
		},
		"literal replacement rune": {
			payload: []byte("{\"url\":\"https://example.com/\\uFFFD\"}"),
		},
		"invalid UTF-8": {
			payload: invalidUTF8,
			want:    ErrInvalidJSONTextUTF8,
		},
		"unpaired high surrogate": {
			payload: []byte("{\"url\":\"https://example.com/\\uD800\"}"),
			want:    ErrUnpairedJSONSurrogate,
		},
		"unpaired low surrogate": {
			payload: []byte("{\"url\":\"https://example.com/\\uDC00\"}"),
			want:    ErrUnpairedJSONSurrogate,
		},
		"high surrogate followed by non-low escape": {
			payload: []byte("{\"url\":\"https://example.com/\\uD800\\u0000\"}"),
			want:    ErrUnpairedJSONSurrogate,
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateJSONText(test.payload)
			if !errors.Is(err, test.want) {
				t.Fatalf("ValidateJSONText(%q) error = %v, want %v", test.payload, err, test.want)
			}
		})
	}
}

func TestValidateJSONTextAcceptsOrdinaryJSONEscapes(t *testing.T) {
	for _, payload := range []string{
		"{\"value\":\"quote: \\\"; slash: \\/; backslash: \\\\; tab: \\t; newline: \\n; carriage return: \\r; form feed: \\f; backspace: \\b\"}",
		"{\"value\":\"\\u0000 \\uD83D\\uDE00 \\uFFFD\"}",
	} {
		if !json.Valid([]byte(payload)) {
			t.Fatalf("test fixture is not valid JSON: %q", payload)
		}
		if err := ValidateJSONText([]byte(payload)); err != nil {
			t.Fatalf("ValidateJSONText(%s): %v", payload, err)
		}
	}
}

func TestJSONTextValidatorRetainsEscapeAndUTF8StateAcrossWrites(t *testing.T) {
	var validator JSONTextValidator
	for _, chunk := range [][]byte{
		[]byte("{\"url\":\"https://example.com/\\uD8"),
		[]byte("3D\\uDE"),
		[]byte("00/"),
		[]byte("界"),
		[]byte("\"}"),
	} {
		if err := validator.Write(chunk); err != nil {
			t.Fatalf("Write(%q): %v", chunk, err)
		}
	}
	if err := validator.Finalize(); err != nil {
		t.Fatalf("Finalize(): %v", err)
	}
}
