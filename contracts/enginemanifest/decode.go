package enginemanifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func decodeStrict(payload []byte, source string, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %q: %w", source, err)
	}
	if err := consumeJSONEOF(decoder); err != nil {
		return fmt.Errorf("decode %q: %w", source, err)
	}
	return nil
}

func consumeJSONEOF(decoder *json.Decoder) error {
	if decoder == nil {
		return nil
	}
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("unexpected trailing JSON content")
}
