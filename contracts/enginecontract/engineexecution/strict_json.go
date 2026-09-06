package engineexecution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// ValidateStrictJSONFields rejects duplicate object members and explicit null
// values recursively. encoding/json otherwise accepts duplicate members with
// last-value-wins semantics and collapses null into Go zero values, which can
// silently remove optional Engine declarations.
func ValidateStrictJSONFields(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := validateStrictJSONValue(decoder, "$"); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON content")
		}
		return err
	}
	return nil
}

func validateStrictJSONValue(decoder *json.Decoder, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return fmt.Errorf("null JSON value at %s is not allowed", path)
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key at %s must be string", path)
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON field %q at %s", key, path)
			}
			seen[key] = struct{}{}
			if err := validateStrictJSONValue(decoder, path+"."+key); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		index := 0
		for decoder.More() {
			if err := validateStrictJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
			index++
		}
		_, err = decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q at %s", delimiter, path)
	}
}
