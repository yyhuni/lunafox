package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
	"github.com/yyhuni/lunafox/server/internal/modules/system/dto"
)

type RuntimeMetricsHandler struct {
	metricsService runtimeMetricsService
}

type runtimeMetricsService interface {
	Current(ctx context.Context) (systemapp.RuntimeMetricsReport, error)
}

func NewRuntimeMetricsHandler(metricsService runtimeMetricsService) *RuntimeMetricsHandler {
	return &RuntimeMetricsHandler{metricsService: metricsService}
}

// Current returns the Server/control-plane runtime metrics short-window report.
// GET /v1/admin/system/runtimeMetrics/current
func (h *RuntimeMetricsHandler) Current(c *gin.Context) {
	if h.metricsService == nil {
		httpdto.Error(c, http.StatusInternalServerError, "internal_error", "Runtime metrics service is not configured")
		return
	}

	report, err := h.metricsService.Current(c.Request.Context())
	if err != nil {
		if errors.Is(err, systemapp.ErrRuntimeMetricsUnavailable) {
			httpdto.Error(c, http.StatusServiceUnavailable, "runtime_metrics_unavailable", "Runtime metrics are unavailable")
			return
		}
		httpdto.Error(c, http.StatusServiceUnavailable, "runtime_metrics_unavailable", "Runtime metrics are unavailable")
		return
	}

	httpdto.Success(c, toRuntimeMetricsOutput(report))
}

func toRuntimeMetricsOutput(report systemapp.RuntimeMetricsReport) dto.RuntimeMetricsReport {
	series := make([]dto.RuntimeMetricSeriesPoint, 0, len(report.Series))
	for _, sample := range report.Series {
		series = append(series, dto.RuntimeMetricSeriesPoint{
			Time:      sample.SampledAt.UTC().Format("15:04"),
			CPU:       sample.CPU,
			Memory:    sample.Memory,
			Disk:      sample.Disk,
			SampledAt: sample.SampledAt.UTC().Format(time.RFC3339),
		})
	}
	return dto.RuntimeMetricsReport{
		Scope: string(report.Scope),
		Latest: dto.RuntimeMetricLatest{
			CPU:       report.Latest.CPU,
			Memory:    report.Latest.Memory,
			Disk:      report.Latest.Disk,
			UpdatedAt: report.Latest.SampledAt.UTC().Format(time.RFC3339),
		},
		Series: series,
		Capacity: dto.RuntimeMetricCapacity{
			CPUCores:      report.Capacity.CPUCores,
			MemoryTotalGB: report.Capacity.MemoryTotalGB,
			DiskTotalGB:   report.Capacity.DiskTotalGB,
		},
		Source: dto.RuntimeMetricSource{
			Hostname: report.Source.Hostname,
			DiskPath: report.Source.DiskPath,
		},
		SampleIntervalSeconds: report.SampleIntervalSeconds,
		RetentionSeconds:      report.RetentionSeconds,
	}
}
