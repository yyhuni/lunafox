package results

import (
	"bytes"
	"strings"
	"testing"

	engineexecution "github.com/yyhuni/lunafox/engine-go/protocol"
)

func TestDefaultBatchLimitsMatchEngineProtocolCeilings(t *testing.T) {
	limits := DefaultBatchLimits()
	if limits.MaxItems != int(engineexecution.ResultBatchMaxItemsCeiling) || limits.MaxBytes != int(engineexecution.ResultBatchMaxBytesCeiling) {
		t.Fatalf("default batch limits = %#v; want Engine Protocol ceilings", limits)
	}
}

func TestValidateEncodedBatchPreservesExactBytesAndAllowsUnknownSyntax(t *testing.T) {
	original := [][]byte{
		[]byte(`{"dnsName":"api.example.com"}`),
		[]byte(`{"dnsName":"www.example.com"}`),
	}
	batch, err := ValidateEncodedBatch("vendor.future.v1", original, BatchLimits{MaxItems: 2, MaxBytes: 1024})
	if err != nil {
		t.Fatalf("ValidateEncodedBatch failed: %v", err)
	}
	if batch.ResultType != "vendor.future.v1" || batch.TotalBytes != len(original[0])+len(original[1]) {
		t.Fatalf("unexpected batch: %+v", batch)
	}
	original[0][0] = 'X'
	if !bytes.Equal(batch.Items[0], []byte(`{"dnsName":"api.example.com"}`)) {
		t.Fatalf("batch did not retain exact cloned bytes: %q", batch.Items[0])
	}
}

func TestValidateEncodedBatchRejectsMalformedTypeEmptyItemAndLimits(t *testing.T) {
	cases := []struct {
		name       string
		resultType string
		items      [][]byte
		limit      BatchLimits
	}{
		{name: "missing type", resultType: "", items: [][]byte{[]byte(`{}`)}, limit: BatchLimits{MaxItems: 1, MaxBytes: 10}},
		{name: "uppercase type", resultType: "Asset.Subdomain.v1", items: [][]byte{[]byte(`{}`)}, limit: BatchLimits{MaxItems: 1, MaxBytes: 10}},
		{name: "empty type segment", resultType: "asset..v1", items: [][]byte{[]byte(`{}`)}, limit: BatchLimits{MaxItems: 1, MaxBytes: 10}},
		{name: "type segment edge", resultType: "asset.-subdomain.v1", items: [][]byte{[]byte(`{}`)}, limit: BatchLimits{MaxItems: 1, MaxBytes: 10}},
		{name: "empty batch", resultType: "asset.subdomain.v1", items: nil, limit: BatchLimits{MaxItems: 1, MaxBytes: 10}},
		{name: "empty item", resultType: "asset.subdomain.v1", items: [][]byte{nil}, limit: BatchLimits{MaxItems: 1, MaxBytes: 10}},
		{name: "array item", resultType: "asset.subdomain.v1", items: [][]byte{[]byte(`[]`)}, limit: BatchLimits{MaxItems: 1, MaxBytes: 10}},
		{name: "item limit", resultType: "asset.subdomain.v1", items: [][]byte{[]byte(`{}`), []byte(`{}`)}, limit: BatchLimits{MaxItems: 1, MaxBytes: 10}},
		{name: "byte limit", resultType: "asset.subdomain.v1", items: [][]byte{[]byte(`{"x":1}`)}, limit: BatchLimits{MaxItems: 1, MaxBytes: 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ValidateEncodedBatch(tc.resultType, tc.items, tc.limit); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateEncodedBatchEnvelopePreservesLocatableItemFailuresForServerClassification(t *testing.T) {
	items := [][]byte{
		[]byte(`{"dnsName":"api.example.com"}`),
		[]byte(`{"dnsName":`),
		{0xff},
		nil,
	}
	batch, err := ValidateEncodedBatchEnvelope(ResultKindAssetSubdomain, items, BatchLimits{MaxItems: len(items), MaxBytes: 1024})
	if err != nil {
		t.Fatalf("ValidateEncodedBatchEnvelope returned error: %v", err)
	}
	if len(batch.Items) != len(items) || !bytes.Equal(batch.Items[2], []byte{0xff}) || batch.Items[3] != nil {
		t.Fatalf("envelope changed locatable item bytes: %#v", batch.Items)
	}
	if _, err := ValidateEncodedBatch(ResultKindAssetSubdomain, items, BatchLimits{MaxItems: len(items), MaxBytes: 1024}); err == nil {
		t.Fatal("Agent transport validation accepted malformed item content")
	}
}

func TestValidateCanonicalBatchRejectsUnknownAndIPv6BeforeWrites(t *testing.T) {
	if _, err := ValidateCanonicalBatch("asset.unknown.v1", [][]byte{[]byte(`{"x":1}`)}, DefaultBatchLimits()); err == nil {
		t.Fatal("unknown result type must be rejected by the Server validator")
	}
	for _, item := range []string{
		`{"host":"[2001:db8::1]","ip":"2001:db8::1","port":443}`,
		`{"url":"https://[2001:db8::1]/"}`,
	} {
		kind := ResultKindAssetHostPort
		if strings.Contains(item, `"url"`) {
			kind = ResultKindAssetWebsite
		}
		if _, err := ValidateCanonicalBatch(kind, [][]byte{[]byte(item)}, DefaultBatchLimits()); err == nil {
			t.Fatalf("IPv6 item %s must be hard-invalid", item)
		}
	}
}

func TestValidateCanonicalBatchDoesNotRejectParserHostileWebsitePortText(t *testing.T) {
	for _, port := range []string{"0", "65536", "http"} {
		item := []byte(`{"url":"https://example.com:` + port + `","host":"example.com"}`)
		if _, err := ValidateCanonicalBatch(ResultKindAssetWebsite, [][]byte{item}, DefaultBatchLimits()); err != nil {
			t.Fatalf("website port text %q must remain an accepted raw URL: %v", port, err)
		}
	}
}
