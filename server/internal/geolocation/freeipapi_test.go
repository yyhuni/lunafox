package geolocation

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (failingReader) Close() error             { return nil }

func TestFreeIPAPIClientUsesFixedCredentialFreeTargetContract(t *testing.T) {
	resolvedAt := time.Date(2026, 8, 4, 12, 0, 0, 123456789, time.UTC)
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.Method != http.MethodGet || request.URL.Path != "/api/json/8.8.8.8" {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.String())
		}
		if request.URL.RawQuery != "" || request.Header.Get("Authorization") != "" || request.Header.Get("X-API-Key") != "" {
			t.Errorf("FreeIPAPI request must not carry credentials or query settings: query=%q headers=%v", request.URL.RawQuery, request.Header)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{
			"ipVersion": 4,
			"ipAddress": "8.8.8.8",
			"latitude": 37.4219999,
			"longitude": -122.0840575,
			"accuracyRadiusKm": null,
			"countryName": "must not enter the model",
			"asn": "15169"
		}`)
	}))
	defer server.Close()

	client := newFreeIPAPITestClient(server.Client(), server.URL+"/api/json", resolvedAt)
	location, err := client.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("request count = %d, want exactly one", requests.Load())
	}
	if location.ObservedIP != "8.8.8.8" || location.Latitude != 37.4219999 || location.Longitude != -122.0840575 || location.AccuracyRadiusKM != nil || location.ProviderKey != FreeIPAPIProviderKey || !location.ResolvedAt.Equal(resolvedAt) {
		t.Fatalf("normalized location = %#v", location)
	}
}

func TestFreeIPAPIClientSupportsSelfLookupAndNullableRadius(t *testing.T) {
	resolvedAt := time.Date(2026, 8, 4, 12, 5, 0, 0, time.UTC)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/json" {
			t.Errorf("self lookup path = %q", request.URL.Path)
		}
		_, _ = io.WriteString(writer, `{"ipAddress":"1.1.1.1","latitude":-33.86,"longitude":151.21,"accuracyRadiusKm":12.5}`)
	}))
	defer server.Close()

	client := newFreeIPAPITestClient(server.Client(), server.URL+"/api/json", resolvedAt)
	location, err := client.Lookup(context.Background(), "")
	if err != nil {
		t.Fatalf("Lookup self: %v", err)
	}
	if location.AccuracyRadiusKM == nil || *location.AccuracyRadiusKM != 12.5 || location.ObservedIP != "1.1.1.1" {
		t.Fatalf("self location = %#v", location)
	}
}

func TestFreeIPAPIClientRejectsInvalidOrPartialPayloads(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		target  string
	}{
		{name: "missing IP", payload: `{"latitude":1,"longitude":2}`, target: "8.8.8.8"},
		{name: "private IP", payload: `{"ipAddress":"10.0.0.1","latitude":1,"longitude":2}`},
		{name: "mismatched target IP", payload: `{"ipAddress":"1.1.1.1","latitude":1,"longitude":2}`, target: "8.8.8.8"},
		{name: "missing latitude", payload: `{"ipAddress":"8.8.8.8","longitude":2}`, target: "8.8.8.8"},
		{name: "null latitude", payload: `{"ipAddress":"8.8.8.8","latitude":null,"longitude":2}`, target: "8.8.8.8"},
		{name: "non-numeric latitude", payload: `{"ipAddress":"8.8.8.8","latitude":"1","longitude":2}`, target: "8.8.8.8"},
		{name: "non-finite latitude", payload: `{"ipAddress":"8.8.8.8","latitude":1e999,"longitude":2}`, target: "8.8.8.8"},
		{name: "latitude below range", payload: `{"ipAddress":"8.8.8.8","latitude":-90.1,"longitude":2}`, target: "8.8.8.8"},
		{name: "latitude above range", payload: `{"ipAddress":"8.8.8.8","latitude":90.1,"longitude":2}`, target: "8.8.8.8"},
		{name: "missing longitude", payload: `{"ipAddress":"8.8.8.8","latitude":1}`, target: "8.8.8.8"},
		{name: "longitude below range", payload: `{"ipAddress":"8.8.8.8","latitude":1,"longitude":-180.1}`, target: "8.8.8.8"},
		{name: "longitude above range", payload: `{"ipAddress":"8.8.8.8","latitude":1,"longitude":180.1}`, target: "8.8.8.8"},
		{name: "negative radius", payload: `{"ipAddress":"8.8.8.8","latitude":1,"longitude":2,"accuracyRadiusKm":-1}`, target: "8.8.8.8"},
		{name: "non-numeric radius", payload: `{"ipAddress":"8.8.8.8","latitude":1,"longitude":2,"accuracyRadiusKm":"3"}`, target: "8.8.8.8"},
		{name: "malformed JSON", payload: `{"ipAddress":`, target: "8.8.8.8"},
		{name: "trailing JSON", payload: `{"ipAddress":"8.8.8.8","latitude":1,"longitude":2}{}`, target: "8.8.8.8"},
		{name: "top-level array", payload: `[]`, target: "8.8.8.8"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, attempts := staticFreeIPAPIClient(http.StatusOK, nil, test.payload)
			_, err := client.Lookup(context.Background(), test.target)
			requireLookupFailureClass(t, err, FailureInvalidPayload)
			if attempts.Load() != 1 {
				t.Fatalf("attempts = %d, want one", attempts.Load())
			}
		})
	}
}

func TestFreeIPAPIClientRejectsInvalidTargetsWithoutProviderAttempt(t *testing.T) {
	client, attempts := staticFreeIPAPIClient(http.StatusOK, nil, `{}`)
	for _, target := range []string{"not-an-ip", "10.0.0.1", "192.0.2.1"} {
		_, err := client.Lookup(context.Background(), target)
		requireLookupFailureClass(t, err, FailureInvalidTarget)
	}
	if attempts.Load() != 0 {
		t.Fatalf("invalid target attempts = %d, want zero", attempts.Load())
	}
}

func TestFreeIPAPIClientBoundsResponseBodyAndReadFailures(t *testing.T) {
	t.Run("body too large", func(t *testing.T) {
		client, attempts := staticFreeIPAPIClient(http.StatusOK, nil, strings.Repeat(" ", freeIPAPIMaxBodyBytes+1))
		_, err := client.Lookup(context.Background(), "8.8.8.8")
		requireLookupFailureClass(t, err, FailureBodyTooLarge)
		if attempts.Load() != 1 {
			t.Fatalf("attempts = %d, want one", attempts.Load())
		}
	})

	t.Run("body read failure", func(t *testing.T) {
		var attempts atomic.Int32
		adapter := NewFreeIPAPIClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts.Add(1)
			return &http.Response{StatusCode: http.StatusOK, Body: failingReader{}, Header: make(http.Header)}, nil
		})})
		_, err := adapter.Lookup(context.Background(), "8.8.8.8")
		requireLookupFailureClass(t, err, FailureNetwork)
		if attempts.Load() != 1 {
			t.Fatalf("attempts = %d, want one", attempts.Load())
		}
	})
}

func TestFreeIPAPIClientClassifiesTerminalHTTPAndTransportFailuresWithoutRetry(t *testing.T) {
	for _, test := range []struct {
		name       string
		statusCode int
		want       FailureClass
	}{
		{name: "bad request", statusCode: http.StatusBadRequest, want: FailureHTTPStatus},
		{name: "server failure", statusCode: http.StatusServiceUnavailable, want: FailureHTTPStatus},
		{name: "rate limited", statusCode: http.StatusTooManyRequests, want: FailureRateLimited},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, attempts := staticFreeIPAPIClient(test.statusCode, nil, `provider text is ignored`)
			_, err := client.Lookup(context.Background(), "8.8.8.8")
			requireLookupFailureClass(t, err, test.want)
			if attempts.Load() != 1 {
				t.Fatalf("attempts = %d, want one", attempts.Load())
			}
		})
	}

	t.Run("network", func(t *testing.T) {
		var attempts atomic.Int32
		adapter := NewFreeIPAPIClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts.Add(1)
			return nil, errors.New("network unavailable")
		})})
		_, err := adapter.Lookup(context.Background(), "8.8.8.8")
		requireLookupFailureClass(t, err, FailureNetwork)
		if attempts.Load() != 1 {
			t.Fatalf("attempts = %d, want one", attempts.Load())
		}
	})

	t.Run("canceled", func(t *testing.T) {
		var attempts atomic.Int32
		adapter := NewFreeIPAPIClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			attempts.Add(1)
			<-request.Context().Done()
			return nil, request.Context().Err()
		})})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := adapter.Lookup(ctx, "8.8.8.8")
		requireLookupFailureClass(t, err, FailureCanceled)
		if attempts.Load() > 1 {
			t.Fatalf("attempts = %d, want at most one", attempts.Load())
		}
	})

	t.Run("deadline", func(t *testing.T) {
		var attempts atomic.Int32
		adapter := NewFreeIPAPIClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			attempts.Add(1)
			<-request.Context().Done()
			return nil, request.Context().Err()
		})})
		adapter.timeout = 5 * time.Millisecond
		_, err := adapter.Lookup(context.Background(), "8.8.8.8")
		requireLookupFailureClass(t, err, FailureTimeout)
		if attempts.Load() != 1 {
			t.Fatalf("attempts = %d, want one", attempts.Load())
		}
	})
}

func TestFreeIPAPIClientAllowsOnlyHTTPSRedirects(t *testing.T) {
	t.Run("reject HTTP redirect", func(t *testing.T) {
		var attempts atomic.Int32
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			attempts.Add(1)
			http.Redirect(writer, &http.Request{}, "http://example.invalid/result", http.StatusFound)
		}))
		defer server.Close()
		client := newFreeIPAPITestClient(server.Client(), server.URL, time.Now().UTC())
		_, err := client.Lookup(context.Background(), "")
		requireLookupFailureClass(t, err, FailureRedirect)
		if attempts.Load() != 1 {
			t.Fatalf("attempts = %d, want one", attempts.Load())
		}
	})

	t.Run("bound HTTPS redirect chain", func(t *testing.T) {
		var requests atomic.Int32
		server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requests.Add(1)
			http.Redirect(writer, request, request.URL.Path, http.StatusFound)
		}))
		defer server.Close()
		client := newFreeIPAPITestClient(server.Client(), server.URL, time.Now().UTC())
		_, err := client.Lookup(context.Background(), "")
		requireLookupFailureClass(t, err, FailureRedirect)
		if requests.Load() != 10 {
			t.Fatalf("redirect requests = %d, want bounded chain of 10", requests.Load())
		}
	})
}

func TestFreeIPAPIRetryAfterParsing(t *testing.T) {
	now := time.Date(2026, 8, 4, 13, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name string
		raw  string
		want time.Time
	}{
		{name: "delta seconds", raw: "30", want: now.Add(30 * time.Second)},
		{name: "HTTP date", raw: now.Add(2 * time.Minute).Format(http.TimeFormat), want: now.Add(2 * time.Minute)},
		{name: "zero is not future", raw: "0", want: now.Add(time.Minute)},
		{name: "invalid", raw: "later", want: now.Add(time.Minute)},
		{name: "past date", raw: now.Add(-time.Minute).Format(http.TimeFormat), want: now.Add(time.Minute)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := parseRetryAfter(test.raw, now); !got.Equal(test.want) {
				t.Fatalf("parseRetryAfter(%q) = %s, want %s", test.raw, got, test.want)
			}
		})
	}
}

func newFreeIPAPITestClient(httpClient *http.Client, endpoint string, resolvedAt time.Time) *FreeIPAPIClient {
	client := NewFreeIPAPIClient(httpClient)
	client.endpoint = endpoint
	client.now = func() time.Time { return resolvedAt }
	return client
}

func staticFreeIPAPIClient(statusCode int, headers http.Header, payload string) (*FreeIPAPIClient, *atomic.Int32) {
	var attempts atomic.Int32
	client := NewFreeIPAPIClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts.Add(1)
		if headers == nil {
			headers = make(http.Header)
		}
		return &http.Response{
			StatusCode: statusCode,
			Header:     headers.Clone(),
			Body:       io.NopCloser(strings.NewReader(payload)),
		}, nil
	})})
	client.now = func() time.Time { return time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC) }
	return client, &attempts
}

func requireLookupFailureClass(t *testing.T, err error, want FailureClass) *LookupError {
	t.Helper()
	var failure *LookupError
	if !errors.As(err, &failure) {
		t.Fatalf("error = %v, want LookupError(%s)", err, want)
	}
	if failure.Class != want {
		t.Fatalf("failure class = %q, want %q", failure.Class, want)
	}
	return failure
}
