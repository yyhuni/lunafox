package results

import (
	"strings"
	"testing"
)

func TestDecodeSubdomainItemsRejectsNonCanonicalWithoutRepair(t *testing.T) {
	if _, err := DecodeSubdomainItems([]string{`{"dnsName":" Api.Example.COM. "}`}); err == nil {
		t.Fatal("DecodeSubdomainItems silently repaired a non-canonical DNS name")
	}
	items, err := DecodeSubdomainItems([]string{`{"dnsName":"api.example.com"}`})
	if err != nil || len(items) != 1 || items[0].DNSName != "api.example.com" {
		t.Fatalf("canonical subdomain decode = %#v, %v", items, err)
	}
}

func TestDecodeSubdomainItemsRejectsInvalidJSON(t *testing.T) {
	_, err := DecodeSubdomainItems([]string{`{"dnsName":`})

	if err == nil || !strings.Contains(err.Error(), "itemsJson[0]") {
		t.Fatalf("expected item JSON rejection, got %v", err)
	}
}

func TestDecodeWebsiteItemsRejectsUnpairedSurrogateBeforeJSONDecode(t *testing.T) {
	_, err := DecodeWebsiteItems([]string{`{"url":"https://example.com/\uD800","host":"example.com"}`})
	if err == nil || !strings.Contains(err.Error(), "unpaired Unicode surrogate") {
		t.Fatalf("expected unpaired-surrogate rejection, got %v", err)
	}
}

func TestDecodeSubdomainItemsRejectsMissingDNSName(t *testing.T) {
	_, err := DecodeSubdomainItems([]string{`{"dnsName":" "}`})

	if err == nil || !strings.Contains(err.Error(), "dnsName") {
		t.Fatalf("expected dnsName rejection, got %v", err)
	}
}

func TestDecodeSubdomainItemsRejectsUnknownDuplicateAndCaseVariantFields(t *testing.T) {
	for _, payload := range []string{
		`{"dnsName":"api.example.com","extra":true}`,
		`{"dnsName":"api.example.com","dnsName":"other.example.com"}`,
		`{"DNSNAME":"api.example.com"}`,
	} {
		_, err := DecodeSubdomainItems([]string{payload})
		if err == nil || !strings.Contains(err.Error(), "itemsJson[0]") {
			t.Fatalf("expected strict field rejection for %s, got %v", payload, err)
		}
	}
}

func TestDecodeSubdomainItemsRejectsIPDNSName(t *testing.T) {
	_, err := DecodeSubdomainItems([]string{`{"dnsName":"127.0.0.1"}`})

	if err == nil || !strings.Contains(err.Error(), "dnsName") {
		t.Fatalf("expected dnsName rejection, got %v", err)
	}
}

func TestDecodeSubdomainItemsRejectsIDNRepair(t *testing.T) {
	if _, err := DecodeSubdomainItems([]string{`{"dnsName":"bücher.example"}`}); err == nil {
		t.Fatal("DecodeSubdomainItems silently converted IDN to punycode")
	}
}

func TestDecodeHostPortItemsRejectsNonCanonicalWithoutRepair(t *testing.T) {
	if _, err := DecodeHostPortItems([]string{`{"host":" Api.Example.COM. ","ip":"192.0.2.10","port":443}`}); err == nil {
		t.Fatal("DecodeHostPortItems silently repaired a Host")
	}
	items, err := DecodeHostPortItems([]string{`{"host":"api.example.com","ip":"192.0.2.10","port":443}`})
	if err != nil || len(items) != 1 || items[0].Host != "api.example.com" || items[0].IP != "192.0.2.10" {
		t.Fatalf("canonical host-port decode = %#v, %v", items, err)
	}
}

func TestDecodeHostPortItemsAcceptsIPHost(t *testing.T) {
	items, err := DecodeHostPortItems([]string{`{"host":"192.0.2.10","ip":"192.0.2.10","port":443}`})

	if err != nil {
		t.Fatalf("DecodeHostPortItems failed: %v", err)
	}
	if len(items) != 1 || items[0].Host != "192.0.2.10" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestDecodeHostPortItemsRejectsInvalidFields(t *testing.T) {
	cases := []string{
		`{"host":" ","ip":"192.0.2.10","port":443}`,
		`{"host":"api.example.com","ip":"not-ip","port":443}`,
		`{"host":"api.example.com","ip":"192.0.2.10","port":70000}`,
		`{"host":"api.example.com","ip":"192.0.2.10","port":"443"}`,
	}
	for _, payload := range cases {
		_, err := DecodeHostPortItems([]string{payload})
		if err == nil || !strings.Contains(err.Error(), "itemsJson[0]") {
			t.Fatalf("expected item rejection for %s, got %v", payload, err)
		}
	}
}

func TestDecodeHostPortItemsRejectsUnknownDuplicateAndIPv6Fields(t *testing.T) {
	for _, payload := range []string{
		`{"host":"api.example.com","ip":"192.0.2.10","port":443,"extra":true}`,
		`{"host":"api.example.com","ip":"192.0.2.10","port":443,"port":80}`,
		`{"HOST":"api.example.com","ip":"192.0.2.10","port":443}`,
	} {
		_, err := DecodeHostPortItems([]string{payload})
		if err == nil || !strings.Contains(err.Error(), "itemsJson[0]") {
			t.Fatalf("expected strict host-port rejection for %s, got %v", payload, err)
		}
	}
}

func TestDecodeWebsiteItemsPreservesURLAndRejectsHostRepair(t *testing.T) {
	if _, err := DecodeWebsiteItems([]string{`{"url":"HTTPS://Api.Example.COM:443/login?b=2&a=1#fragment","host":" Api.Example.COM. "}`}); err == nil {
		t.Fatal("DecodeWebsiteItems silently repaired a Host assertion")
	}
	items, err := DecodeWebsiteItems([]string{`{"url":"HTTPS://Api.Example.COM:443/login?b=2&a=1#fragment","host":"api.example.com","title":"API","statusCode":200,"contentLength":42,"location":"https://api.example.com/home","webserver":"nginx","contentType":"text/html","tech":["nginx","go"],"responseBody":"ok","vhost":false,"responseHeaders":"HTTP/1.1 200 OK"}`})

	if err != nil {
		t.Fatalf("DecodeWebsiteItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 website item, got %d", len(items))
	}
	item := items[0]
	if item.URL != "HTTPS://Api.Example.COM:443/login?b=2&a=1#fragment" || item.Host != "api.example.com" || item.Title != "API" {
		t.Fatalf("unexpected website item: %#v", item)
	}
	if item.StatusCode == nil || *item.StatusCode != 200 || item.ContentLength == nil || *item.ContentLength != 42 {
		t.Fatalf("unexpected numeric fields: %#v", item)
	}
	if len(item.Tech) != 2 || item.Tech[0] != "nginx" || item.Tech[1] != "go" {
		t.Fatalf("unexpected tech values: %#v", item.Tech)
	}
	if item.Vhost == nil || *item.Vhost {
		t.Fatalf("unexpected vhost value: %#v", item.Vhost)
	}
}

func TestDecodeWebsiteItemsRequiresExplicitHostAssertion(t *testing.T) {
	if _, err := DecodeWebsiteItems([]string{`{"url":"http://www.example.com:80"}`}); err == nil {
		t.Fatal("DecodeWebsiteItems accepted a missing host assertion")
	}
	items, err := DecodeWebsiteItems([]string{`{"url":"http://www.example.com:80","host":"www.example.com"}`})
	if err != nil || len(items) != 1 || items[0].URL != "http://www.example.com:80" || items[0].Host != "www.example.com" {
		t.Fatalf("DecodeWebsiteItems raw URL = %#v, %v", items, err)
	}
}

func TestDecodeWebsiteItemsRejectsInvalidFields(t *testing.T) {
	cases := []string{
		`{"url":" "}`,
		`{"url":"ftp://example.com"}`,
		`{"url":"https://example.com","host":"bad host"}`,
		`{"url":"https://example.com","statusCode":"200"}`,
		`{"url":"https://example.com","tech":["ok",42]}`,
	}
	for _, payload := range cases {
		_, err := DecodeWebsiteItems([]string{payload})
		if err == nil || !strings.Contains(err.Error(), "itemsJson[0]") {
			t.Fatalf("expected item rejection for %s, got %v", payload, err)
		}
	}
}

func TestDecodeWebsiteItemsRejectsUnknownDuplicateAndCaseVariantFields(t *testing.T) {
	for _, payload := range []string{
		`{"url":"https://example.com","extra":true}`,
		`{"url":"https://example.com","url":"https://other.example.com"}`,
		`{"URL":"https://example.com"}`,
	} {
		_, err := DecodeWebsiteItems([]string{payload})
		if err == nil || !strings.Contains(err.Error(), "itemsJson[0]") {
			t.Fatalf("expected strict website rejection for %s, got %v", payload, err)
		}
	}
}
