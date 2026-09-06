package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
)

type runtimeMetricsServiceStub struct {
	result systemapp.RuntimeMetricsReport
	err    error
}

func (stub *runtimeMetricsServiceStub) Current(context.Context) (systemapp.RuntimeMetricsReport, error) {
	return stub.result, stub.err
}

func TestRuntimeMetricsHandlerCurrentReturnsReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sampledAt := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	handler := NewRuntimeMetricsHandler(&runtimeMetricsServiceStub{result: systemapp.RuntimeMetricsReport{
		Scope:                 systemapp.RuntimeMetricScopeHost,
		SampleIntervalSeconds: 2,
		RetentionSeconds:      600,
		Latest: systemapp.RuntimeMetricSample{
			SampledAt: sampledAt,
			CPU:       23.5,
			Memory:    61.2,
			Disk:      48,
			Scope:     systemapp.RuntimeMetricScopeHost,
		},
		Series:   []systemapp.RuntimeMetricSample{{SampledAt: sampledAt, CPU: 23.5, Memory: 61.2, Disk: 48}},
		Capacity: systemapp.RuntimeMetricCapacity{CPUCores: 8, MemoryTotalGB: 32, DiskTotalGB: 512},
		Source:   systemapp.RuntimeMetricSource{Hostname: "srv-1", DiskPath: "/data"},
	}})
	router := gin.New()
	router.GET("/v1/admin/system/runtimeMetrics/current", handler.Current)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/system/runtimeMetrics/current", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Scope  string `json:"scope"`
		Latest struct {
			CPU       float64 `json:"cpu"`
			Memory    float64 `json:"memory"`
			Disk      float64 `json:"disk"`
			UpdatedAt string  `json:"updatedAt"`
		} `json:"latest"`
		Series []struct {
			Time      string  `json:"time"`
			CPU       float64 `json:"cpu"`
			Memory    float64 `json:"memory"`
			Disk      float64 `json:"disk"`
			SampledAt string  `json:"sampledAt"`
		} `json:"series"`
		Capacity struct {
			CPUCores      int     `json:"cpuCores"`
			MemoryTotalGB float64 `json:"memoryTotalGb"`
			DiskTotalGB   float64 `json:"diskTotalGb"`
		} `json:"capacity"`
		Source struct {
			Hostname string `json:"hostname"`
			DiskPath string `json:"diskPath"`
		} `json:"source"`
		SampleIntervalSeconds int `json:"sampleIntervalSeconds"`
		RetentionSeconds      int `json:"retentionSeconds"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Scope != "host" || payload.Latest.CPU != 23.5 || payload.Latest.UpdatedAt == "" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.Series) != 1 || payload.Series[0].Time != "10:00" || payload.Series[0].SampledAt == "" {
		t.Fatalf("unexpected series: %+v", payload.Series)
	}
	if payload.Capacity.CPUCores != 8 || payload.Source.DiskPath != "/data" {
		t.Fatalf("unexpected metadata: %+v", payload)
	}
}

func TestRuntimeMetricsHandlerCurrentMapsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewRuntimeMetricsHandler(&runtimeMetricsServiceStub{err: errors.New("boom")})
	router := gin.New()
	router.GET("/v1/admin/system/runtimeMetrics/current", handler.Current)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/system/runtimeMetrics/current", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
}
