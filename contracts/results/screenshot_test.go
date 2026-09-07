package results

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
)

func screenshotTestChunk(kind string, payload []byte) []byte {
	data := screenshotTestRIFF(screenshotTestRawChunk(kind, payload))
	return data
}

func screenshotTestRawChunk(kind string, payload []byte) []byte {
	data := make([]byte, 8+len(payload))
	copy(data, kind)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(payload)))
	copy(data[8:], payload)
	if len(payload)%2 != 0 {
		data = append(data, 0)
	}
	return data
}

func screenshotTestRIFF(chunks ...[]byte) []byte {
	data := []byte("RIFF....WEBP")
	for _, chunk := range chunks {
		data = append(data, chunk...)
	}
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	return data
}

func screenshotTestVP8(width, height uint16, extra int) []byte {
	payload := make([]byte, 10+extra)
	payload[3], payload[4], payload[5] = 0x9d, 0x01, 0x2a
	binary.LittleEndian.PutUint16(payload[6:8], width)
	binary.LittleEndian.PutUint16(payload[8:10], height)
	return screenshotTestChunk("VP8 ", payload)
}

func screenshotTestVP8X(width, height uint32) []byte {
	payload := make([]byte, 10)
	imageWidth, imageHeight := width, height
	width--
	height--
	payload[4] = byte(width)
	payload[5] = byte(width >> 8)
	payload[6] = byte(width >> 16)
	payload[7] = byte(height)
	payload[8] = byte(height >> 8)
	payload[9] = byte(height >> 16)
	imagePayload := make([]byte, 10)
	imagePayload[3], imagePayload[4], imagePayload[5] = 0x9d, 0x01, 0x2a
	binary.LittleEndian.PutUint16(imagePayload[6:8], uint16(imageWidth))
	binary.LittleEndian.PutUint16(imagePayload[8:10], uint16(imageHeight))
	return screenshotTestRIFF(screenshotTestRawChunk("VP8X", payload), screenshotTestRawChunk("VP8 ", imagePayload))
}

func screenshotTestVP8L(width, height uint32) []byte {
	bits := (width - 1) | ((height - 1) << 14)
	payload := []byte{0x2f, byte(bits), byte(bits >> 8), byte(bits >> 16), byte(bits >> 24)}
	return screenshotTestChunk("VP8L", payload)
}

func TestScreenshotWebPStructureAcceptsAllImageChunkVariants(t *testing.T) {
	for name, image := range map[string][]byte{
		"VP8X": screenshotTestVP8X(800, 720),
		"VP8":  screenshotTestVP8(800, 720, 2),
		"VP8L": screenshotTestVP8L(800, 720),
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateScreenshotWebPStructure(image); err != nil {
				t.Fatalf("ValidateScreenshotWebPStructure() error = %v", err)
			}
		})
	}
}

func TestScreenshotWebPStructureEnforcesWidthAndPixelBounds(t *testing.T) {
	for name, image := range map[string][]byte{
		"width":  screenshotTestVP8X(801, 720),
		"pixels": screenshotTestVP8X(800, 10001),
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateScreenshotWebPStructure(image); err == nil {
				t.Fatal("oversized WebP structure was accepted")
			}
		})
	}
	if err := ValidateScreenshotWebP(screenshotTestVP8X(800, 720)); err != nil {
		t.Fatalf("bounded valid WebP rejected: %v", err)
	}
}

func TestDecodeScreenshotItemsRejectsNonCanonicalWire(t *testing.T) {
	image := screenshotTestVP8(320, 200, 1)
	encoded := base64.StdEncoding.EncodeToString(image)
	url := "https://example.com/"
	cases := map[string]string{
		"missing url":         `{"image":"` + encoded + `"}`,
		"missing image":       `{"url":"` + url + `"}`,
		"null image":          `{"url":"` + url + `","image":null}`,
		"null status":         `{"url":"` + url + `","statusCode":null,"image":"` + encoded + `"}`,
		"unknown field":       `{"url":"` + url + `","image":"` + encoded + `","extra":true}`,
		"duplicate field":     `{"url":"` + url + `","image":"` + encoded + `","image":"` + encoded + `"}`,
		"case variant":        `{"URL":"` + url + `","image":"` + encoded + `"}`,
		"url safe base64":     `{"url":"` + url + `","image":"` + strings.NewReplacer("+", "-", "/", "_").Replace(encoded) + "-" + `"}`,
		"noncanonical base64": `{"url":"` + url + `","image":"` + encoded + "=" + `"}`,
		"png":                 `{"url":"` + url + `","image":"` + base64.StdEncoding.EncodeToString([]byte("\\x89PNG\\r\\n")) + `"}`,
		"jpeg":                `{"url":"` + url + `","image":"` + base64.StdEncoding.EncodeToString([]byte{0xff, 0xd8, 0xff, 0xd9}) + `"}`,
		"avif":                `{"url":"` + url + `","image":"` + base64.StdEncoding.EncodeToString([]byte("....ftypavif")) + `"}`,
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeScreenshotItems([]string{payload}); err == nil {
				t.Fatal("invalid Screenshot wire was accepted")
			}
		})
	}
}

func TestDecodeScreenshotItemsPreservesRawURL(t *testing.T) {
	image := screenshotTestVP8(320, 200, 1)
	rawURL := "HTTPS://EXAMPLE.com:443/%zz?x=%00#fragment"
	payload := `{"url":"` + rawURL + `","image":"` + base64.StdEncoding.EncodeToString(image) + `"}`
	items, err := DecodeScreenshotItems([]string{payload})
	if err != nil || len(items) != 1 || items[0].URL != rawURL {
		t.Fatalf("DecodeScreenshotItems() = %#v, %v", items, err)
	}
}

func TestScreenshotWireRoundTripAndByteBudget(t *testing.T) {
	image := screenshotTestVP8(640, 360, MaxScreenshotWebPBytes-20-10)
	item := Screenshot{URL: "https://example.com/", Image: image}
	encoded, err := EncodeScreenshot(item)
	if err != nil {
		t.Fatalf("EncodeScreenshot() error = %v", err)
	}
	decoded, err := DecodeScreenshotItems([]string{encoded})
	if err != nil || len(decoded) != 1 || string(decoded[0].Image) != string(image) {
		t.Fatalf("Screenshot round trip = %#v/%v", decoded, err)
	}
	if err := ValidateScreenshotWebP(image); err != nil {
		t.Fatalf("exact 256 KiB image rejected: %v", err)
	}
	tooLarge := screenshotTestVP8(640, 360, MaxScreenshotWebPBytes-20-10+2)
	if err := ValidateScreenshotWebP(tooLarge); err == nil {
		t.Fatal("over-budget WebP accepted")
	}
}

func TestValidateCanonicalBatchKeepsSharedFourMiBBudget(t *testing.T) {
	item := []byte(`{"dnsName":"` + strings.Repeat("a", 1010) + `"}`)
	items := make([][]byte, 4096)
	for index := range items {
		items[index] = item
	}
	if _, err := ValidateEncodedBatch("vendor.future.v1", items, DefaultBatchLimits()); err != nil {
		t.Fatalf("batch at shared budget rejected: %v", err)
	}
	items = append(items, item)
	if _, err := ValidateEncodedBatch("vendor.future.v1", items, DefaultBatchLimits()); err == nil {
		t.Fatal("batch over shared 4 MiB budget accepted")
	}
}

func TestScreenshotStatusCodeWireTypeIsStrict(t *testing.T) {
	image := base64.StdEncoding.EncodeToString(screenshotTestVP8(320, 200, 1))
	for _, status := range []string{`"200"`, `200.0`, `true`, `[]`, `{}`} {
		payload := `{"url":"https://example.com/","statusCode":` + status + `,"image":"` + image + `"}`
		var raw map[string]any
		if err := json.Unmarshal([]byte(payload), &raw); err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeScreenshotItems([]string{payload}); err == nil {
			t.Fatalf("statusCode %s was accepted", status)
		}
	}
}
