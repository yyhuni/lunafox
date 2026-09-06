package application

import (
	"context"
	"errors"
	"math"
	"runtime"
	"sync"
	"time"
)

const (
	DefaultRuntimeMetricSampleInterval = 2 * time.Second
	DefaultRuntimeMetricRetention      = 10 * time.Minute
)

var ErrRuntimeMetricsUnavailable = errors.New("runtime metrics unavailable")

type RuntimeMetricScope string

const (
	RuntimeMetricScopeHost      RuntimeMetricScope = "host"
	RuntimeMetricScopeContainer RuntimeMetricScope = "container"
	RuntimeMetricScopeProcess   RuntimeMetricScope = "process"
	RuntimeMetricScopeRuntime   RuntimeMetricScope = "runtime"
)

type RuntimeMetricSample struct {
	SampledAt time.Time
	CPU       float64
	Memory    float64
	Disk      float64
	Scope     RuntimeMetricScope
}

type RuntimeMetricCapacity struct {
	CPUCores      int
	MemoryTotalGB float64
	DiskTotalGB   float64
}

type RuntimeMetricSource struct {
	Hostname string
	DiskPath string
}

type RuntimeMetricsReport struct {
	Scope                 RuntimeMetricScope
	Latest                RuntimeMetricSample
	Series                []RuntimeMetricSample
	Capacity              RuntimeMetricCapacity
	Source                RuntimeMetricSource
	SampleIntervalSeconds int
	RetentionSeconds      int
}

type RuntimeMetricSampler interface {
	Sample(ctx context.Context) (RuntimeMetricSample, error)
}

type RuntimeMetricMetadataProvider interface {
	Capacity(ctx context.Context) (RuntimeMetricCapacity, error)
	Source(ctx context.Context) (RuntimeMetricSource, error)
}

type RuntimeMetricsServiceOptions struct {
	SampleInterval time.Duration
	Retention      time.Duration
	Metadata       RuntimeMetricMetadataProvider
}

type RuntimeMetricsService struct {
	sampler        RuntimeMetricSampler
	metadata       RuntimeMetricMetadataProvider
	sampleInterval time.Duration
	retention      time.Duration

	mu      sync.RWMutex
	series  []RuntimeMetricSample
	started bool
	stop    chan struct{}
}

func NewRuntimeMetricsService(sampler RuntimeMetricSampler, metadata RuntimeMetricMetadataProvider) *RuntimeMetricsService {
	return NewRuntimeMetricsServiceWithOptions(sampler, RuntimeMetricsServiceOptions{Metadata: metadata})
}

func NewRuntimeMetricsServiceWithOptions(sampler RuntimeMetricSampler, options RuntimeMetricsServiceOptions) *RuntimeMetricsService {
	if sampler == nil {
		panic("runtime metric sampler is required")
	}
	sampleInterval := options.SampleInterval
	if sampleInterval <= 0 {
		sampleInterval = DefaultRuntimeMetricSampleInterval
	}
	retention := options.Retention
	if retention <= 0 {
		retention = DefaultRuntimeMetricRetention
	}
	return &RuntimeMetricsService{
		sampler:        sampler,
		metadata:       options.Metadata,
		sampleInterval: sampleInterval,
		retention:      retention,
		stop:           make(chan struct{}),
	}
}

func (service *RuntimeMetricsService) Start(ctx context.Context) {
	service.mu.Lock()
	if service.started {
		service.mu.Unlock()
		return
	}
	service.started = true
	service.mu.Unlock()

	go func() {
		_ = service.CollectOnce(ctx)
		ticker := time.NewTicker(service.sampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-service.stop:
				return
			case <-ticker.C:
				_ = service.CollectOnce(ctx)
			}
		}
	}()
}

func (service *RuntimeMetricsService) Stop() {
	service.mu.Lock()
	defer service.mu.Unlock()
	if !service.started {
		return
	}
	close(service.stop)
	service.stop = make(chan struct{})
	service.started = false
}

func (service *RuntimeMetricsService) CollectOnce(ctx context.Context) error {
	sample, err := service.sampler.Sample(ctx)
	if err != nil {
		return err
	}
	sample.CPU = clampPercent(sample.CPU)
	sample.Memory = clampPercent(sample.Memory)
	sample.Disk = clampPercent(sample.Disk)
	if sample.Scope == "" {
		sample.Scope = RuntimeMetricScopeRuntime
	}
	if sample.SampledAt.IsZero() {
		sample.SampledAt = time.Now().UTC()
	}
	sample.SampledAt = sample.SampledAt.UTC()

	service.mu.Lock()
	defer service.mu.Unlock()
	service.series = append(service.series, sample)
	service.pruneLocked(sample.SampledAt)
	return nil
}

func (service *RuntimeMetricsService) Current(ctx context.Context) (RuntimeMetricsReport, error) {
	service.mu.Lock()
	service.pruneLocked(time.Now().UTC())
	series := append([]RuntimeMetricSample(nil), service.series...)
	sampleInterval := service.sampleInterval
	retention := service.retention
	service.mu.Unlock()

	if len(series) == 0 {
		return RuntimeMetricsReport{}, ErrRuntimeMetricsUnavailable
	}

	latest := series[len(series)-1]
	report := RuntimeMetricsReport{
		Scope:                 latest.Scope,
		Latest:                latest,
		Series:                series,
		Capacity:              RuntimeMetricCapacity{CPUCores: runtime.NumCPU()},
		SampleIntervalSeconds: int(sampleInterval.Seconds()),
		RetentionSeconds:      int(retention.Seconds()),
	}
	if service.metadata != nil {
		if capacity, err := service.metadata.Capacity(ctx); err == nil {
			report.Capacity = capacity
		}
		if source, err := service.metadata.Source(ctx); err == nil {
			report.Source = source
		}
	}
	return report, nil
}

func (service *RuntimeMetricsService) pruneLocked(now time.Time) {
	cutoff := now.Add(-service.retention)
	keep := 0
	for keep < len(service.series) && !service.series[keep].SampledAt.After(cutoff) {
		keep++
	}
	if keep > 0 {
		service.series = append([]RuntimeMetricSample(nil), service.series[keep:]...)
	}
}

func clampPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return math.Round(value*10) / 10
}
