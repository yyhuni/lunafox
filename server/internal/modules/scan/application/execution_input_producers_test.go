package application

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
)

type dnsCursorForExecutionInputTest struct {
	names []string
}

func (cursor *dnsCursorForExecutionInputTest) ForEachDNSNameByScanID(_ context.Context, _ int, visit func(string) error) error {
	for _, name := range cursor.names {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}

type hostPortCursorForExecutionInputTest struct {
	evidence []HostPortEvidence
}

type websiteURLCursorForExecutionInputTest struct {
	urls []string
}

type endpointURLCursorForExecutionInputTest struct {
	urls []string
	err  error
}

func (cursor *websiteURLCursorForExecutionInputTest) ForEachWebsiteURLByScanID(_ context.Context, _ int, visit func(string) error) error {
	for _, url := range cursor.urls {
		if err := visit(url); err != nil {
			return err
		}
	}
	return nil
}

func (cursor *endpointURLCursorForExecutionInputTest) ForEachEndpointURLByScanID(_ context.Context, _ int, visit func(string) error) error {
	if cursor.err != nil {
		return cursor.err
	}
	for _, url := range cursor.urls {
		if err := visit(url); err != nil {
			return err
		}
	}
	return nil
}

func (cursor *hostPortCursorForExecutionInputTest) ForEachHostPortByScanID(_ context.Context, _ int, visit func(HostPortEvidence) error) error {
	for _, evidence := range cursor.evidence {
		if err := visit(evidence); err != nil {
			return err
		}
	}
	return nil
}

func collectExecutionInputRecords(t *testing.T, produce func(func(string) error) error) []string {
	t.Helper()
	var records []string
	if err := produce(func(value string) error {
		records = append(records, value)
		return nil
	}); err != nil {
		t.Fatalf("producer returned error: %v", err)
	}
	return records
}

func collectHostPortRecords(t *testing.T, produce HostPortsProducer) []HostPortEvidence {
	t.Helper()
	var records []HostPortEvidence
	if err := produce(func(value HostPortEvidence) error {
		records = append(records, value)
		return nil
	}); err != nil {
		t.Fatalf("producer returned error: %v", err)
	}
	return records
}

func executionInputBlacklistFilterForTest(t *testing.T, patterns ...string) *ExecutionInputBlacklistFilter {
	t.Helper()
	matcher, err := blacklistdomain.CompileMatcher(patterns)
	if err != nil {
		t.Fatalf("compile blacklist matcher: %v", err)
	}
	filter, err := NewExecutionInputBlacklistFilter(matcher)
	if err != nil {
		t.Fatalf("new blacklist filter: %v", err)
	}
	return filter
}

func TestNewSubdomainsProducerProjectsFinalizedFactsVerbatim(t *testing.T) {
	producer, err := NewSubdomainsProducer(context.Background(), 7, &dnsCursorForExecutionInputTest{
		names: []string{"api.example.com", "example.com", "www.example.com"},
	}, executionInputBlacklistFilterForTest(t))
	if err != nil {
		t.Fatalf("NewSubdomainsProducer returned error: %v", err)
	}
	got := collectExecutionInputRecords(t, producer)
	want := []string{"api.example.com", "example.com", "www.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("records = %#v, want %#v", got, want)
	}
}

func TestNewSubdomainsProducerAllowsZeroFacts(t *testing.T) {
	producer, err := NewSubdomainsProducer(context.Background(), 7, &dnsCursorForExecutionInputTest{}, executionInputBlacklistFilterForTest(t))
	if err != nil {
		t.Fatalf("NewSubdomainsProducer returned error: %v", err)
	}
	if got := collectExecutionInputRecords(t, producer); len(got) != 0 {
		t.Fatalf("records = %#v, want empty", got)
	}
}

func TestNewHostPortsProducerProjectsCompleteFactsWithoutDerivation(t *testing.T) {
	want := []HostPortEvidence{{Host: "a.example.com", IP: "192.0.2.10", Port: 80}, {Host: "a.example.com", IP: "192.0.2.11", Port: 80}, {Host: "192.0.2.12", IP: "192.0.2.12", Port: 8443}}
	producer, err := NewHostPortsProducer(context.Background(), 8, &hostPortCursorForExecutionInputTest{evidence: want}, executionInputBlacklistFilterForTest(t))
	if err != nil {
		t.Fatalf("NewHostPortsProducer returned error: %v", err)
	}
	if got := collectHostPortRecords(t, producer); !reflect.DeepEqual(got, want) {
		t.Fatalf("records = %#v, want %#v", got, want)
	}
}

func TestNewHostPortsProducerAllowsZeroFacts(t *testing.T) {
	producer, err := NewHostPortsProducer(context.Background(), 1, &hostPortCursorForExecutionInputTest{}, executionInputBlacklistFilterForTest(t))
	if err != nil {
		t.Fatalf("NewHostPortsProducer returned error: %v", err)
	}
	if got := collectHostPortRecords(t, producer); len(got) != 0 {
		t.Fatalf("records = %#v, want empty", got)
	}
}

func TestNewWebsiteURLsProducerProjectsConfirmedURLsVerbatim(t *testing.T) {
	want := []string{
		"HTTPS://EXAMPLE.com:443/a%2Fb?b=2&a=1#fragment",
		"https://api.example.com:8443/?payload=%0d%0aInjected",
		"https://www.example.com/%zz",
	}
	producer, err := NewWebsiteURLsProducer(context.Background(), 7, &websiteURLCursorForExecutionInputTest{urls: want}, executionInputBlacklistFilterForTest(t))
	if err != nil {
		t.Fatalf("NewWebsiteURLsProducer() error = %v", err)
	}
	if got := collectExecutionInputRecords(t, producer); !reflect.DeepEqual(got, want) {
		t.Fatalf("WebsiteURLs = %#v, want %#v", got, want)
	}
}

func TestNewWebsiteURLsProducerRejectsDamagedStoredRecordWithoutRepair(t *testing.T) {
	for name, value := range map[string]string{
		"actual NUL":   "https://example.com/\x00payload",
		"actual CR":    "https://example.com/\rpayload",
		"actual LF":    "https://example.com/\npayload",
		"over limit":   "https://example.com/" + strings.Repeat("x", 2000),
		"wrong prefix": "ftp://example.com",
	} {
		t.Run(name, func(t *testing.T) {
			producer, err := NewWebsiteURLsProducer(context.Background(), 7, &websiteURLCursorForExecutionInputTest{urls: []string{value}}, executionInputBlacklistFilterForTest(t))
			if err != nil {
				t.Fatalf("NewWebsiteURLsProducer() error = %v", err)
			}
			if err := producer(func(string) error { return nil }); err == nil {
				t.Fatal("producer accepted a damaged stored URL")
			}
		})
	}
}

func TestWriteObservedURLLineStreamPreservesRawBytes(t *testing.T) {
	values := []string{
		"HTTPS://EXAMPLE.com:443/a%2Fb?b=2&a=1#fragment",
		"https://example.com/?payload=%00",
		"https://example.com/?payload=%0d%0aInjected",
		"https://example.com/%zz",
	}
	producer := func(emit func(string) error) error {
		for _, value := range values {
			if err := emit(value); err != nil {
				return err
			}
		}
		return nil
	}
	var output bytes.Buffer
	count, err := WriteObservedURLLineStream(context.Background(), &output, producer)
	if err != nil {
		t.Fatalf("WriteObservedURLLineStream() error = %v", err)
	}
	if count != uint64(len(values)) || output.String() != strings.Join(values, "\n")+"\n" {
		t.Fatalf("observed URL line product = count %d content %q", count, output.String())
	}
}

func TestNewWebsiteURLsProducerAllowsZeroFacts(t *testing.T) {
	producer, err := NewWebsiteURLsProducer(context.Background(), 7, &websiteURLCursorForExecutionInputTest{}, executionInputBlacklistFilterForTest(t))
	if err != nil {
		t.Fatalf("NewWebsiteURLsProducer() error = %v", err)
	}
	if got := collectExecutionInputRecords(t, producer); len(got) != 0 {
		t.Fatalf("WebsiteURLs = %#v, want empty", got)
	}
}

func TestNewEndpointURLsProducerProjectsAndFiltersURLs(t *testing.T) {
	filter := executionInputBlacklistFilterForTest(t, "blocked.example")
	values := []string{"https://blocked.example/api", "HTTPS://api.example.com:443/v1", "https://api.example.com/%zz"}
	producer, err := NewEndpointURLsProducer(context.Background(), 7, &endpointURLCursorForExecutionInputTest{urls: values}, filter)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := collectExecutionInputRecords(t, producer), []string{"HTTPS://api.example.com:443/v1", "https://api.example.com/%zz"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("endpoint URLs = %#v, want %#v", got, want)
	}
}

func TestNewEndpointURLsProducerFailsClosedOnCursorError(t *testing.T) {
	readErr := errors.New("endpoint repository unavailable")
	producer, err := NewEndpointURLsProducer(context.Background(), 7, &endpointURLCursorForExecutionInputTest{err: readErr}, executionInputBlacklistFilterForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := producer(func(string) error { return nil }); !errors.Is(err, readErr) {
		t.Fatalf("producer error = %v, want %v", err, readErr)
	}
}

func TestProducersPropagateCancellation(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	producer, err := NewWebsiteURLsProducer(cancelled, 7, &websiteURLCursorForExecutionInputTest{urls: []string{"https://example.com"}}, executionInputBlacklistFilterForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := producer(func(string) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled producer error = %v", err)
	}
}

func TestExecutionInputProducersExcludeOnlyFrozenBlacklistMatches(t *testing.T) {
	t.Run("subdomains exact and arbitrary-depth wildcard", func(t *testing.T) {
		filter := executionInputBlacklistFilterForTest(t, "*.blocked.example", "blocked.example")
		cursor := &dnsCursorForExecutionInputTest{names: []string{"blocked.example", "one.blocked.example", "two.one.blocked.example", "keep.example"}}
		producer, err := NewSubdomainsProducer(context.Background(), 7, cursor, filter)
		if err != nil {
			t.Fatalf("NewSubdomainsProducer() error = %v", err)
		}
		if got, want := collectExecutionInputRecords(t, producer), []string{"keep.example"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("subdomains = %#v, want %#v", got, want)
		}
		if got, want := filter.Stats(), (ExecutionInputBlacklistStats{Examined: 4, Excluded: 3, Emitted: 1}); got != want {
			t.Fatalf("subdomain stats = %#v, want %#v", got, want)
		}
		if got, want := cursor.names, []string{"blocked.example", "one.blocked.example", "two.one.blocked.example", "keep.example"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("producer changed finalized facts: %#v", got)
		}
	})

	t.Run("hostPorts match host or IPv4 CIDR", func(t *testing.T) {
		filter := executionInputBlacklistFilterForTest(t, "192.0.2.0/24", "blocked.example")
		cursor := &hostPortCursorForExecutionInputTest{evidence: []HostPortEvidence{
			{Host: "keep.example", IP: "192.0.2.10", Port: 443},
			{Host: "blocked.example", IP: "198.51.100.10", Port: 443},
			{Host: "keep.example", IP: "198.51.100.11", Port: 443},
		}}
		producer, err := NewHostPortsProducer(context.Background(), 8, cursor, filter)
		if err != nil {
			t.Fatalf("NewHostPortsProducer() error = %v", err)
		}
		want := []HostPortEvidence{{Host: "keep.example", IP: "198.51.100.11", Port: 443}}
		if got := collectHostPortRecords(t, producer); !reflect.DeepEqual(got, want) {
			t.Fatalf("hostPorts = %#v, want %#v", got, want)
		}
		if got, want := filter.Stats(), (ExecutionInputBlacklistStats{Examined: 3, Excluded: 2, Emitted: 1}); got != want {
			t.Fatalf("HostPort stats = %#v, want %#v", got, want)
		}
	})

	t.Run("website URLs match hostname only", func(t *testing.T) {
		filter := executionInputBlacklistFilterForTest(t, "192.0.2.0/24", "blocked.example")
		cursor := &websiteURLCursorForExecutionInputTest{urls: []string{
			"https://blocked.example:8443/path?q=1#fragment",
			"http://192.0.2.17/admin",
			"HTTPS://keep.example:443/path%zz?q=%0d%0a",
		}}
		producer, err := NewWebsiteURLsProducer(context.Background(), 9, cursor, filter)
		if err != nil {
			t.Fatalf("NewWebsiteURLsProducer() error = %v", err)
		}
		if got, want := collectExecutionInputRecords(t, producer), []string{"HTTPS://keep.example:443/path%zz?q=%0d%0a"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("website URLs = %#v, want %#v", got, want)
		}
		if got, want := filter.Stats(), (ExecutionInputBlacklistStats{Examined: 3, Excluded: 2, Emitted: 1}); got != want {
			t.Fatalf("website URL stats = %#v, want %#v", got, want)
		}
	})
}

func TestExecutionInputProducersKeepValidAllExcludedProductsEmpty(t *testing.T) {
	filter := executionInputBlacklistFilterForTest(t, "*.blocked.example")
	producer, err := NewSubdomainsProducer(context.Background(), 7, &dnsCursorForExecutionInputTest{names: []string{"one.blocked.example", "two.blocked.example"}}, filter)
	if err != nil {
		t.Fatalf("NewSubdomainsProducer() error = %v", err)
	}
	var output bytes.Buffer
	count, err := WriteCanonicalLineStream(context.Background(), &output, producer)
	if err != nil || count != 0 || output.Len() != 0 {
		t.Fatalf("all-excluded product = count %d bytes %q error %v", count, output.String(), err)
	}
	if got, want := filter.Stats(), (ExecutionInputBlacklistStats{Examined: 2, Excluded: 2}); got != want {
		t.Fatalf("all-excluded stats = %#v, want %#v", got, want)
	}
}

func TestExecutionInputProducersRequireBlacklistFilter(t *testing.T) {
	if _, err := NewSubdomainsProducer(context.Background(), 7, &dnsCursorForExecutionInputTest{}, nil); err == nil {
		t.Fatal("expected missing blacklist filter to fail")
	}
}

func TestWriteCanonicalLineStreamAllowsZeroAndCleansCancellation(t *testing.T) {
	var output bytes.Buffer
	count, err := WriteCanonicalLineStream(context.Background(), &output, func(_ func(string) error) error { return nil })
	if err != nil || count != 0 || output.Len() != 0 {
		t.Fatalf("zero stream = count %d err %v bytes %q", count, err, output.String())
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = WriteCanonicalLineStream(cancelled, &output, func(emit func(string) error) error { return emit("example.com") })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestWriteCanonicalLineStreamRejectsNewlineRecords(t *testing.T) {
	var output bytes.Buffer
	_, err := WriteCanonicalLineStream(context.Background(), &output, func(emit func(string) error) error { return emit("bad\nvalue") })
	if err == nil {
		t.Fatal("expected newline record rejection")
	}
}
