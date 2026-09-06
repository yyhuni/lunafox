package results

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDirectoryResultDescriptorIsCanonical(t *testing.T) {
	descriptor, ok := Lookup(ResultKindAssetDirectory)
	if !ok {
		t.Fatalf("expected %s to be registered", ResultKindAssetDirectory)
	}
	if descriptor != (Descriptor{
		ResultType:             ResultKindAssetDirectory,
		SchemaRef:              "schema.result.asset.directory.v1",
		AuthorResultsFieldName: "Directories",
		ItemTypeName:           "Directory",
		EncoderID:              ResultEncoderAssetDirectory,
		GoEncoderName:          "EncodeDirectory",
	}) {
		t.Fatalf("unexpected Directory descriptor: %#v", descriptor)
	}
}

func TestDirectoryResultTypeRejectsAliasesCaseVariantsAndShapeInference(t *testing.T) {
	item := Directory{URL: "https://example.com/admin", Status: 200, ContentType: "text/html"}
	for _, resultType := range []string{
		"asset.directories.v1",
		"asset.Directory.v1",
		"asset.directory.V1",
		"asset.directory.v01",
		"directory",
		" " + ResultKindAssetDirectory,
		ResultKindAssetDirectory + " ",
	} {
		if _, ok := Lookup(resultType); ok {
			t.Fatalf("Lookup(%q) accepted a Directory result alias", resultType)
		}
		if err := Validate(resultType, item); err == nil {
			t.Fatalf("Validate(%q) accepted a Directory result alias", resultType)
		}
	}
	if err := Validate(ResultKindAssetDirectory, Screenshot{}); err == nil {
		t.Fatal("Directory result type was inferred from a non-Directory Go item")
	}
	if err := Validate(ResultKindAssetWebsite, item); err == nil {
		t.Fatal("Directory Go item shape was inferred as another result type")
	}
	payload := []byte(`{"url":"https://example.com/admin","status":200,"contentLength":0,"contentType":"text/html","duration":0}`)
	if _, err := ValidateCanonicalBatch("asset.directories.v1", [][]byte{payload}, DefaultBatchLimits()); err == nil {
		t.Fatal("canonical batch accepted a Directory result alias")
	}
}

func TestDecodeDirectoryItemsPreservesCompleteObservation(t *testing.T) {
	payload := `{"url":"HTTPS://EXAMPLE.com:443/a%2Fb?x=1#fragment%00","status":0,"contentLength":9223372036854775807,"contentType":" Text/HTML ","duration":9223372036854775807}`
	items, err := DecodeDirectoryItems([]string{payload})
	if err != nil {
		t.Fatalf("DecodeDirectoryItems() error = %v", err)
	}
	want := Directory{
		URL:           "HTTPS://EXAMPLE.com:443/a%2Fb?x=1#fragment%00",
		Status:        0,
		ContentLength: math.MaxInt64,
		ContentType:   " Text/HTML ",
		Duration:      math.MaxInt64,
	}
	if len(items) != 1 || items[0] != want {
		t.Fatalf("DecodeDirectoryItems() = %#v, want %#v", items, want)
	}
}

func TestDecodeDirectoryItemsAcceptsExactStringAndIntegerBoundaries(t *testing.T) {
	url := "https://example.com/" + strings.Repeat("界", 660)
	contentType := strings.Repeat("界", 341) + "a"
	if !utf8.ValidString(url) || len([]byte(url)) != 2000 {
		t.Fatalf("URL fixture is %d bytes", len([]byte(url)))
	}
	if !utf8.ValidString(contentType) || len([]byte(contentType)) != 1024 {
		t.Fatalf("contentType fixture is %d bytes", len([]byte(contentType)))
	}
	payload := `{"url":"` + url + `","status":999,"contentLength":0,"contentType":"` + contentType + `","duration":0}`
	items, err := DecodeDirectoryItems([]string{payload})
	if err != nil {
		t.Fatalf("DecodeDirectoryItems() error = %v", err)
	}
	if len(items) != 1 || items[0].URL != url || items[0].Status != 999 || items[0].ContentLength != 0 || items[0].ContentType != contentType || items[0].Duration != 0 {
		t.Fatalf("boundary observation changed: %#v", items)
	}
}

func TestDecodeDirectoryItemsRejectsNonCanonicalWire(t *testing.T) {
	valid := `{"url":"https://example.com/admin","status":200,"contentLength":12,"contentType":"text/html","duration":34}`
	invalidUTF8URL := string(append([]byte(`{"url":"`), append([]byte{0xff}, []byte(`","status":200,"contentLength":12,"contentType":"text/html","duration":34}`)...)...))
	invalidUTF8ContentType := string(append([]byte(`{"url":"https://example.com/admin","status":200,"contentLength":12,"contentType":"`), append([]byte{0xff}, []byte(`","duration":34}`)...)...))
	tests := []struct {
		name    string
		payload string
	}{
		{name: "not object", payload: `[]`},
		{name: "trailing content", payload: valid + ` {}`},
		{name: "unknown field", payload: strings.Replace(valid, `}`, `,"extra":true}`, 1)},
		{name: "case variant", payload: strings.Replace(valid, `"url"`, `"URL"`, 1)},
		{name: "duplicate field", payload: strings.Replace(valid, `}`, `,"url":"https://other.example/"}`, 1)},
		{name: "missing url", payload: `{"status":200,"contentLength":12,"contentType":"text/html","duration":34}`},
		{name: "missing status", payload: `{"url":"https://example.com/admin","contentLength":12,"contentType":"text/html","duration":34}`},
		{name: "missing content length", payload: `{"url":"https://example.com/admin","status":200,"contentType":"text/html","duration":34}`},
		{name: "missing content type", payload: `{"url":"https://example.com/admin","status":200,"contentLength":12,"duration":34}`},
		{name: "missing duration", payload: `{"url":"https://example.com/admin","status":200,"contentLength":12,"contentType":"text/html"}`},
		{name: "null url", payload: `{"url":null,"status":200,"contentLength":12,"contentType":"text/html","duration":34}`},
		{name: "null status", payload: `{"url":"https://example.com/admin","status":null,"contentLength":12,"contentType":"text/html","duration":34}`},
		{name: "null content length", payload: `{"url":"https://example.com/admin","status":200,"contentLength":null,"contentType":"text/html","duration":34}`},
		{name: "null content type", payload: `{"url":"https://example.com/admin","status":200,"contentLength":12,"contentType":null,"duration":34}`},
		{name: "null duration", payload: `{"url":"https://example.com/admin","status":200,"contentLength":12,"contentType":"text/html","duration":null}`},
		{name: "url wrong type", payload: `{"url":1,"status":200,"contentLength":12,"contentType":"text/html","duration":34}`},
		{name: "status string", payload: `{"url":"https://example.com/admin","status":"200","contentLength":12,"contentType":"text/html","duration":34}`},
		{name: "status decimal", payload: `{"url":"https://example.com/admin","status":200.0,"contentLength":12,"contentType":"text/html","duration":34}`},
		{name: "content length string", payload: `{"url":"https://example.com/admin","status":200,"contentLength":"12","contentType":"text/html","duration":34}`},
		{name: "content type wrong type", payload: `{"url":"https://example.com/admin","status":200,"contentLength":12,"contentType":[],"duration":34}`},
		{name: "duration string", payload: `{"url":"https://example.com/admin","status":200,"contentLength":12,"contentType":"text/html","duration":"34"}`},
		{name: "empty url", payload: strings.Replace(valid, "https://example.com/admin", "", 1)},
		{name: "url NUL", payload: strings.Replace(valid, "https://example.com/admin", `https://example.com/\u0000`, 1)},
		{name: "content type NUL", payload: strings.Replace(valid, "text/html", `text/\u0000html`, 1)},
		{name: "invalid UTF-8 url", payload: invalidUTF8URL},
		{name: "invalid UTF-8 content type", payload: invalidUTF8ContentType},
		{name: "URL over byte limit", payload: strings.Replace(valid, "https://example.com/admin", observedURLAtByteLimit(ObservedAssetURLMaxBytes+1), 1)},
		{name: "content type over byte limit", payload: strings.Replace(valid, "text/html", strings.Repeat("c", 1025), 1)},
		{name: "negative status", payload: strings.Replace(valid, `"status":200`, `"status":-1`, 1)},
		{name: "status over limit", payload: strings.Replace(valid, `"status":200`, `"status":1000`, 1)},
		{name: "negative content length", payload: strings.Replace(valid, `"contentLength":12`, `"contentLength":-1`, 1)},
		{name: "content length overflow", payload: strings.Replace(valid, `"contentLength":12`, `"contentLength":9223372036854775808`, 1)},
		{name: "negative duration", payload: strings.Replace(valid, `"duration":34`, `"duration":-1`, 1)},
		{name: "duration overflow", payload: strings.Replace(valid, `"duration":34`, `"duration":9223372036854775808`, 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items, err := DecodeDirectoryItems([]string{test.payload})
			if err == nil || items != nil {
				t.Fatalf("DecodeDirectoryItems() = %#v, %v; want nil, error", items, err)
			}
		})
	}
}

func TestDecodeDirectoryItemsRejectsWholeBatch(t *testing.T) {
	valid := `{"url":"https://example.com/admin","status":200,"contentLength":12,"contentType":"text/html","duration":34}`
	invalid := `{"url":"https://other.example/admin","status":1000,"contentLength":12,"contentType":"text/html","duration":34}`
	items, err := DecodeDirectoryItems([]string{valid, invalid})
	if err == nil || items != nil {
		t.Fatalf("DecodeDirectoryItems() = %#v, %v; want nil, error", items, err)
	}
}

func TestEncodeDirectoryPreservesExactValuesAndFieldOrder(t *testing.T) {
	item := Directory{
		URL:           "HTTPS://EXAMPLE.com:443/a%2Fb?x=1#fragment%00",
		Status:        0,
		ContentLength: math.MaxInt64,
		ContentType:   " Text/HTML ",
		Duration:      math.MaxInt64,
	}
	encoded, err := EncodeDirectory(item)
	if err != nil {
		t.Fatalf("EncodeDirectory() error = %v", err)
	}
	want := `{"url":"HTTPS://EXAMPLE.com:443/a%2Fb?x=1#fragment%00","status":0,"contentLength":9223372036854775807,"contentType":" Text/HTML ","duration":9223372036854775807}`
	if encoded != want {
		t.Fatalf("EncodeDirectory() = %q, want %q", encoded, want)
	}
	decoded, err := DecodeDirectoryItems([]string{encoded})
	if err != nil || len(decoded) != 1 || decoded[0] != item {
		t.Fatalf("encoded Directory did not round-trip exactly: %#v, %v", decoded, err)
	}
}

func TestValidateDirectoryRejectsUntypedAndInvalidItems(t *testing.T) {
	if err := Validate(ResultKindAssetDirectory, map[string]any{"url": "https://example.com"}); err == nil {
		t.Fatal("untyped Directory item must be rejected")
	}
	for _, item := range []Directory{
		{URL: "", Status: 200},
		{URL: "https://example.com/\x00", Status: 200},
		{URL: "https://example.com", Status: -1},
		{URL: "https://example.com", Status: 1000},
		{URL: "https://example.com", Status: 200, ContentLength: -1},
		{URL: "https://example.com", Status: 200, Duration: -1},
		{URL: "https://example.com", Status: 200, ContentType: "text/\x00html"},
	} {
		if err := Validate(ResultKindAssetDirectory, item); err == nil {
			t.Fatalf("Validate() accepted invalid Directory item: %#v", item)
		}
	}
}
