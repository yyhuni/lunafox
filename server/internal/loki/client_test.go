package loki

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if err := client.CheckReady(context.Background()); err != nil {
		t.Fatalf("expected ready check success, got %v", err)
	}
}

func TestQueryRangeParsesStreams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/query_range" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"success",
			"data":{
				"resultType":"streams",
				"result":[
					{
						"stream":{"source":"stdout"},
						"values":[["1740381601000000000","hello"]]
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	results, err := client.QueryRange(context.Background(), QueryRangeRequest{
		Query:     `{agent_id="1",container_name="lunafox-agent"}`,
		Limit:     10,
		Direction: "FORWARD",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(results[0].Values))
	}
	if results[0].Values[0].Line != "hello" {
		t.Fatalf("expected line=hello, got %s", results[0].Values[0].Line)
	}
}

