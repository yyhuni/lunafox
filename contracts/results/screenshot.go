package results

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

const MaxScreenshotWebPBytes = 256 * 1024

func ValidateScreenshot(item Screenshot) error {
	if _, err := ValidateObservedAssetURL(item.URL); err != nil {
		return fmt.Errorf("screenshot url is invalid: %w", err)
	}
	if item.StatusCode != nil && (*item.StatusCode < 100 || *item.StatusCode > 599) {
		return fmt.Errorf("screenshot statusCode must be between 100 and 599")
	}
	return ValidateScreenshotWebP(item.Image)
}

func ValidateScreenshotWebP(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("screenshot image is required")
	}
	if len(data) > MaxScreenshotWebPBytes {
		return fmt.Errorf("screenshot image exceeds 256 KiB")
	}
	return ValidateScreenshotWebPStructure(data)
}

// ValidateScreenshotWebPStructure checks the bounded WebP structure and
// decoded canvas constraints without applying the persistent byte budget.
// Engine uses this for the first encoding attempt so an oversized but valid
// WebP can take the one allowed lower-size fallback.
func ValidateScreenshotWebPStructure(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("screenshot image is required")
	}
	width, height, err := webpDimensions(data)
	if err != nil {
		return err
	}
	if width == 0 || height == 0 {
		return fmt.Errorf("screenshot WebP dimensions are invalid")
	}
	if width > 800 {
		return fmt.Errorf("screenshot WebP width exceeds 800 pixels")
	}
	if uint64(width)*uint64(height) > 8_000_000 {
		return fmt.Errorf("screenshot WebP pixel count exceeds 8000000")
	}
	return nil
}

func validateScreenshotWireImage(payload string, item Screenshot) error {
	var wire struct {
		Image      json.RawMessage `json:"image"`
		StatusCode json.RawMessage `json:"statusCode"`
	}
	if err := json.Unmarshal([]byte(payload), &wire); err != nil {
		return fmt.Errorf("image is invalid")
	}
	if len(wire.StatusCode) > 0 && string(wire.StatusCode) == "null" {
		return fmt.Errorf("statusCode must be omitted or an integer")
	}
	var encoded string
	if err := json.Unmarshal(wire.Image, &encoded); err != nil || encoded == "" {
		return fmt.Errorf("image must be a non-empty standard base64 string")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) == 0 || base64.StdEncoding.EncodeToString(decoded) != encoded {
		return fmt.Errorf("image must use canonical standard base64")
	}
	if string(decoded) != string(item.Image) {
		return fmt.Errorf("image payload is inconsistent")
	}
	return nil
}

func webpDimensions(data []byte) (uint32, uint32, error) {
	if len(data) < 20 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0, fmt.Errorf("screenshot image must be a WebP")
	}
	if uint64(binary.LittleEndian.Uint32(data[4:8]))+8 != uint64(len(data)) {
		return 0, 0, fmt.Errorf("screenshot WebP RIFF length is invalid")
	}
	var width, height uint32
	var hasExtendedHeader bool
	var hasImagePayload bool
	offset := 12
	for offset+8 <= len(data) {
		kind := string(data[offset : offset+4])
		size64 := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		if size64 == 0 || size64 > uint64(len(data)-offset-8) {
			return 0, 0, fmt.Errorf("screenshot WebP chunk is invalid")
		}
		size := int(size64)
		payload := data[offset+8 : offset+8+size]
		switch kind {
		case "VP8X":
			if hasExtendedHeader || hasImagePayload || len(payload) != 10 {
				return 0, 0, fmt.Errorf("screenshot VP8X chunk is invalid")
			}
			width = 1 + (uint32(payload[4]) | uint32(payload[5])<<8 | uint32(payload[6])<<16)
			height = 1 + (uint32(payload[7]) | uint32(payload[8])<<8 | uint32(payload[9])<<16)
			hasExtendedHeader = true
		case "VP8 ":
			if hasImagePayload || len(payload) < 10 || payload[3] != 0x9d || payload[4] != 0x01 || payload[5] != 0x2a {
				return 0, 0, fmt.Errorf("screenshot VP8 chunk is invalid")
			}
			imageWidth := uint32(binary.LittleEndian.Uint16(payload[6:8]) & 0x3fff)
			imageHeight := uint32(binary.LittleEndian.Uint16(payload[8:10]) & 0x3fff)
			if hasExtendedHeader && (imageWidth != width || imageHeight != height) {
				return 0, 0, fmt.Errorf("screenshot VP8 dimensions do not match VP8X")
			}
			width, height = imageWidth, imageHeight
			hasImagePayload = true
		case "VP8L":
			if hasImagePayload || len(payload) < 5 || payload[0] != 0x2f {
				return 0, 0, fmt.Errorf("screenshot VP8L chunk is invalid")
			}
			bits := uint32(payload[1]) | uint32(payload[2])<<8 | uint32(payload[3])<<16 | uint32(payload[4])<<24
			imageWidth := 1 + (bits & 0x3fff)
			imageHeight := 1 + ((bits >> 14) & 0x3fff)
			if hasExtendedHeader && (imageWidth != width || imageHeight != height) {
				return 0, 0, fmt.Errorf("screenshot VP8L dimensions do not match VP8X")
			}
			width, height = imageWidth, imageHeight
			hasImagePayload = true
		}
		offset += 8 + size
		if size%2 != 0 {
			if offset >= len(data) {
				return 0, 0, fmt.Errorf("screenshot WebP chunk padding is missing")
			}
			offset++
		}
	}
	if offset != len(data) || !hasImagePayload {
		return 0, 0, fmt.Errorf("screenshot WebP payload is missing")
	}
	if width == 0 || height == 0 {
		return 0, 0, fmt.Errorf("screenshot WebP dimensions are invalid")
	}
	return width, height, nil
}
