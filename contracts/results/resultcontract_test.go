package results

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func intPointer(value int) *int {
	return &value
}

func TestCanonicalDescriptorsAreStableAndComplete(t *testing.T) {
	descriptors := Descriptors()
	want := []Descriptor{
		{ResultType: ResultKindAssetSubdomain, SchemaRef: "schema.result.asset.subdomain.v1", AuthorResultsFieldName: "Subdomains", ItemTypeName: "Subdomain", EncoderID: ResultEncoderAssetSubdomain, GoEncoderName: "EncodeSubdomain"},
		{ResultType: ResultKindAssetEndpoint, SchemaRef: "schema.result.asset.endpoint.v1", AuthorResultsFieldName: "Endpoints", ItemTypeName: "Endpoint", EncoderID: ResultEncoderAssetEndpoint, GoEncoderName: "EncodeEndpoint"},
		{ResultType: ResultKindAssetHostPort, SchemaRef: "schema.result.asset.host_port.v1", AuthorResultsFieldName: "HostPorts", ItemTypeName: "HostPort", EncoderID: ResultEncoderAssetHostPort, GoEncoderName: "EncodeHostPort"},
		{ResultType: ResultKindAssetWebsite, SchemaRef: "schema.result.asset.website.v1", AuthorResultsFieldName: "Websites", ItemTypeName: "Website", EncoderID: ResultEncoderAssetWebsite, GoEncoderName: "EncodeWebsite"},
		{ResultType: ResultKindAssetWebsiteTechnology, SchemaRef: "schema.result.asset.website_technology.v1", AuthorResultsFieldName: "WebsiteTechnologies", ItemTypeName: "WebsiteTechnology", EncoderID: ResultEncoderAssetWebsiteTechnology, GoEncoderName: "EncodeWebsiteTechnology"},
		{ResultType: ResultKindAssetScreenshot, SchemaRef: "schema.result.asset.screenshot.v1", AuthorResultsFieldName: "Screenshots", ItemTypeName: "Screenshot", EncoderID: ResultEncoderAssetScreenshot, GoEncoderName: "EncodeScreenshot"},
		{ResultType: ResultKindAssetDirectory, SchemaRef: "schema.result.asset.directory.v1", AuthorResultsFieldName: "Directories", ItemTypeName: "Directory", EncoderID: ResultEncoderAssetDirectory, GoEncoderName: "EncodeDirectory"},
		{ResultType: ResultKindAssetVulnerability, SchemaRef: "schema.result.asset.vulnerability.v1", AuthorResultsFieldName: "Vulnerabilities", ItemTypeName: "Vulnerability", EncoderID: ResultEncoderAssetVulnerability, GoEncoderName: "EncodeVulnerability"},
	}
	if !reflect.DeepEqual(descriptors, want) {
		t.Fatalf("canonical descriptors = %#v, want %#v", descriptors, want)
	}

	// Callers receive a detached slice so a generator or catalog projection
	// cannot mutate the registry used by Server validation.
	descriptors[0].ResultType = "asset.changed.v1"
	if got, ok := Lookup(ResultKindAssetSubdomain); !ok || got.ResultType != ResultKindAssetSubdomain {
		t.Fatalf("descriptor registry was mutated through returned slice: %#v, %v", got, ok)
	}
}

func TestCanonicalDescriptorGoAuthorNamesAreExplicitAndUnique(t *testing.T) {
	seenFields := make(map[string]struct{})
	seenItems := make(map[string]struct{})
	seenEncoders := make(map[string]struct{})
	for _, descriptor := range Descriptors() {
		for label, value := range map[string]string{
			"author Results field": descriptor.AuthorResultsFieldName,
			"item type":            descriptor.ItemTypeName,
			"Go encoder":           descriptor.GoEncoderName,
		} {
			if value == "" {
				t.Fatalf("%s descriptor for %q is empty", label, descriptor.ResultType)
			}
			var seen map[string]struct{}
			switch label {
			case "author Results field":
				seen = seenFields
			case "item type":
				seen = seenItems
			case "Go encoder":
				seen = seenEncoders
			}
			if _, exists := seen[value]; exists {
				t.Fatalf("duplicate %s %q", label, value)
			}
			seen[value] = struct{}{}
		}
	}
}

func TestLookupRequiresExplicitCanonicalResultType(t *testing.T) {
	for _, value := range []string{"", " ", " " + ResultKindAssetSubdomain, ResultKindAssetSubdomain + " "} {
		if _, ok := Lookup(value); ok {
			t.Fatalf("Lookup(%q) must reject non-canonical result type", value)
		}
	}
	for _, descriptor := range Descriptors() {
		if got, ok := Lookup(descriptor.ResultType); !ok || got != descriptor {
			t.Fatalf("Lookup(%q) = %#v, %v; want %#v, true", descriptor.ResultType, got, ok, descriptor)
		}
	}
}

func TestCanonicalEncodersAreBackedByRegisteredContracts(t *testing.T) {
	encoded, err := EncodeSubdomain(Subdomain{DNSName: "api.example.com"})
	if err != nil || encoded != `{"dnsName":"api.example.com"}` {
		t.Fatalf("EncodeSubdomain() = %q, %v", encoded, err)
	}
	hostPort, err := EncodeHostPort(HostPort{Host: "api.example.com", IP: "192.0.2.10", Port: 443})
	if err != nil || hostPort != `{"host":"api.example.com","ip":"192.0.2.10","port":443}` {
		t.Fatalf("EncodeHostPort() = %q, %v", hostPort, err)
	}
	website, err := EncodeWebsite(Website{URL: "HTTPS://Api.Example.COM:443/#fragment", Host: "api.example.com"})
	if err != nil || website != `{"url":"HTTPS://Api.Example.COM:443/#fragment","host":"api.example.com"}` {
		t.Fatalf("EncodeWebsite() = %q, %v", website, err)
	}
	endpoint, err := EncodeEndpoint(Endpoint{URL: "HTTPS://Api.Example.COM:443/#fragment", Host: "api.example.com"})
	if err != nil || endpoint != `{"url":"HTTPS://Api.Example.COM:443/#fragment","host":"api.example.com"}` {
		t.Fatalf("EncodeEndpoint() = %q, %v", endpoint, err)
	}
}

func TestSubdomainResultKindContract(t *testing.T) {
	descriptor, ok := Lookup(ResultKindAssetSubdomain)
	if !ok {
		t.Fatalf("expected %s to be registered", ResultKindAssetSubdomain)
	}
	if descriptor.SchemaRef != "schema.result.asset.subdomain.v1" {
		t.Fatalf("unexpected schema ref: %q", descriptor.SchemaRef)
	}
	if err := Validate(ResultKindAssetSubdomain, Subdomain{DNSName: "api.example.com"}); err != nil {
		t.Fatalf("expected valid subdomain result: %v", err)
	}
}

func TestSubdomainResultKindRejectsInvalidItems(t *testing.T) {
	if err := Validate(ResultKindAssetSubdomain, Subdomain{DNSName: " \t "}); err == nil {
		t.Fatalf("expected empty subdomain dnsName to be rejected")
	}
	if err := Validate(ResultKindAssetSubdomain, map[string]string{"name": "api.example.com"}); err == nil {
		t.Fatalf("expected untyped result item to be rejected")
	}
}

func TestHostPortResultKindContract(t *testing.T) {
	descriptor, ok := Lookup(ResultKindAssetHostPort)
	if !ok {
		t.Fatalf("expected %s to be registered", ResultKindAssetHostPort)
	}
	if descriptor.SchemaRef != "schema.result.asset.host_port.v1" {
		t.Fatalf("unexpected schema ref: %q", descriptor.SchemaRef)
	}
	if err := Validate(ResultKindAssetHostPort, HostPort{Host: "api.example.com", IP: "192.0.2.10", Port: 443}); err != nil {
		t.Fatalf("expected valid host-port result: %v", err)
	}
}

func TestHostPortResultKindRejectsInvalidItems(t *testing.T) {
	cases := []HostPort{
		{Host: "", IP: "192.0.2.10", Port: 443},
		{Host: "api.example.com", IP: "not-ip", Port: 443},
		{Host: "api.example.com", IP: "192.0.2.10", Port: 0},
		{Host: "api.example.com", IP: "192.0.2.10", Port: 65536},
		{Host: "bad\nhost", IP: "192.0.2.10", Port: 443},
	}
	for _, item := range cases {
		if err := Validate(ResultKindAssetHostPort, item); err == nil {
			t.Fatalf("expected invalid host-port item to be rejected: %+v", item)
		}
	}
	if err := Validate(ResultKindAssetHostPort, map[string]string{"host": "api.example.com"}); err == nil {
		t.Fatalf("expected untyped host-port result item to be rejected")
	}
}

func TestEncodeHostPortRejectsIncompleteCanonicalFacts(t *testing.T) {
	cases := []HostPort{
		{Host: " ", IP: "192.0.2.10", Port: 443},
		{Host: "api.example.com", IP: "2001:db8::1", Port: 443},
		{Host: "api.example.com", IP: "192.0.2.0/24", Port: 443},
		{Host: "api.example.com", IP: "192.0.2.10", Port: 0},
		{Host: "api.example.com", IP: "192.0.2.10", Port: 65536},
	}
	for _, item := range cases {
		if _, err := EncodeHostPort(item); err == nil {
			t.Fatalf("expected canonical host-port encoding to reject %+v", item)
		}
	}
}

func TestWebsiteResultKindContract(t *testing.T) {
	descriptor, ok := Lookup(ResultKindAssetWebsite)
	if !ok {
		t.Fatalf("expected %s to be registered", ResultKindAssetWebsite)
	}
	if descriptor.SchemaRef != "schema.result.asset.website.v1" {
		t.Fatalf("unexpected schema ref: %q", descriptor.SchemaRef)
	}
	statusCode := 200
	contentLength := 42
	vhost := false
	if err := Validate(ResultKindAssetWebsite, Website{URL: "https://api.example.com", Host: "api.example.com", Title: "API", StatusCode: &statusCode, ContentLength: &contentLength, Tech: []string{"nginx"}, Vhost: &vhost}); err != nil {
		t.Fatalf("expected valid website result: %v", err)
	}
}

func TestWebsiteResultKindRejectsInvalidItems(t *testing.T) {
	cases := []Website{
		{URL: ""},
		{URL: "ftp://example.com"},
		{URL: "https://bad\nhost.example.com"},
		{URL: "https://example.com", Host: "bad\nhost"},
		{URL: "https://example.com", Host: "other.example.com"},
		{URL: "https://example.com", StatusCode: intPointer(600)},
		{URL: "https://example.com", ContentLength: intPointer(-1)},
		{URL: "https://example.com", Title: "bad\x00title"},
		{URL: "https://example.com", ResponseBody: "bad\x00body"},
		{URL: "https://example.com", ResponseHeaders: "bad\x00headers"},
	}
	for _, item := range cases {
		if err := Validate(ResultKindAssetWebsite, item); err == nil {
			t.Fatalf("expected invalid website item to be rejected: %+v", item)
		}
	}
	if err := Validate(ResultKindAssetWebsite, map[string]string{"url": "https://example.com"}); err == nil {
		t.Fatalf("expected untyped website result item to be rejected")
	}
}

func TestWebsiteResultKindAllowsMultilineOptionalMetadata(t *testing.T) {
	item := Website{
		URL:         "https://example.com",
		Host:        "example.com",
		Title:       "first\nsecond",
		Location:    "https://example.com/next\r\ncontinued",
		Webserver:   "gateway\nproxy",
		ContentType: "text/plain\r\ncharset=utf-8",
		Tech:        []string{"first\nsecond"},
	}
	if err := Validate(ResultKindAssetWebsite, item); err != nil {
		t.Fatalf("Validate website failed: %v", err)
	}
	if item.Title != "first\nsecond" || item.Location != "https://example.com/next\r\ncontinued" || item.Webserver != "gateway\nproxy" || item.ContentType != "text/plain\r\ncharset=utf-8" || len(item.Tech) != 1 || item.Tech[0] != "first\nsecond" {
		t.Fatalf("unexpected normalized multiline metadata: %#v", item)
	}
}

func TestEndpointResultKindContract(t *testing.T) {
	if err := Validate(ResultKindAssetEndpoint, Endpoint{URL: "https://api.example.com", Host: "api.example.com"}); err != nil {
		t.Fatalf("expected valid endpoint: %v", err)
	}
}

func TestValidateEndpointRejectsRepairRequiredFields(t *testing.T) {
	statusCode := 999
	contentLength := -1
	if err := Validate(ResultKindAssetEndpoint, Endpoint{
		URL:           "HTTPS://Api.Example.COM:443/a%2Fb?b=2&a=1#fragment",
		Host:          "Api.Example.COM",
		Title:         "title\x00",
		Location:      strings.Repeat("l", 2001),
		Webserver:     strings.Repeat("w", 1025),
		ContentType:   strings.Repeat("c", 1025),
		Tech:          []string{" nginx ", "", strings.Repeat("x", 101), "nginx"},
		StatusCode:    &statusCode,
		ContentLength: &contentLength,
	}); err == nil {
		t.Fatal("Validate accepted an item that would require repair")
	}
	item := Endpoint{URL: "HTTPS://Api.Example.COM:443/a%2Fb?b=2&a=1#fragment", Host: "api.example.com", ResponseBody: "body", Tech: []string{" nginx ", "nginx"}}
	if err := Validate(ResultKindAssetEndpoint, item); err != nil {
		t.Fatalf("Validate rejected an unchanged valid item: %v", err)
	}
}

func TestValidateEndpointRejectsOversizedResponseEvidence(t *testing.T) {
	responseBody := strings.Repeat("界", 1000) // 3,000 UTF-8 bytes.
	responseHeaders := strings.Repeat("界", EndpointResponseHeadersMaxBytes/3+1)
	err := Validate(ResultKindAssetEndpoint, Endpoint{
		URL:             "https://example.com",
		Host:            "example.com",
		ResponseBody:    responseBody,
		ResponseHeaders: responseHeaders,
	})
	if err == nil {
		t.Fatal("Validate accepted oversized response evidence")
	}
}

func TestEndpointStrictBatchRejectsNonCanonicalOrUnrecoverableFieldsBeforeWrites(t *testing.T) {
	tooLongURL := "https://example.com/" + strings.Repeat("x", EndpointURLMaxBytes)
	statusCode := 99
	tests := []struct {
		name string
		json string
	}{
		{"missing host", `{"url":"https://example.com"}`},
		{"host mismatch", `{"url":"https://example.com","host":"api.example.com"}`},
		{"actual URL control", `{"url":"https://example.com/\u0000","host":"example.com"}`},
		{"overlong URL", `{"url":"` + tooLongURL + `","host":"example.com"}`},
		{"unknown field", `{"url":"https://example.com","host":"example.com","extra":true}`},
		{"duplicate field", `{"url":"https://example.com","url":"https://api.example.com","host":"example.com"}`},
		{"NUL metadata", `{"url":"https://example.com","host":"example.com","title":"bad\u0000title"}`},
		{"oversized title", `{"url":"https://example.com","host":"example.com","title":"` + strings.Repeat("x", 2001) + `"}`},
		{"invalid status", `{"url":"https://example.com","host":"example.com","statusCode":` + strconv.Itoa(statusCode) + `}`},
		{"invalid content length", `{"url":"https://example.com","host":"example.com","contentLength":2147483648}`},
		{"oversized response", `{"url":"https://example.com","host":"example.com","responseBody":"` + strings.Repeat("x", EndpointResponseBodyMaxBytes+1) + `"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeEndpointItems([]string{test.json}); err == nil {
				t.Fatalf("DecodeEndpointItems() accepted invalid canonical endpoint %q", test.name)
			}
		})
	}
}

func TestEndpointStrictBatchPreservesOptionalEvidenceAndTruncationFlags(t *testing.T) {
	items, err := DecodeEndpointItems([]string{`{"url":"https://example.com","host":"example.com","tech":["nginx","nginx"],"responseBody":"body","responseBodyTruncated":true,"responseHeaders":"headers","responseHeadersTruncated":true}`})
	if err != nil {
		t.Fatalf("DecodeEndpointItems() error = %v", err)
	}
	if len(items) != 1 || !reflect.DeepEqual(items[0].Tech, []string{"nginx", "nginx"}) || !items[0].ResponseBodyTruncated || !items[0].ResponseHeadersTruncated {
		t.Fatalf("strict decode changed canonical Endpoint evidence: %#v", items)
	}
}
