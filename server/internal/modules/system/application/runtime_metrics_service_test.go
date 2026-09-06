package application

import (
	"context"
	"testing"
	"time"
)

type runtimeMetricSamplerStub struct {
	samples []RuntimeMetricSample
	index   int
}

func (stub *runtimeMetricSamplerStub) Sample(context.Context) (RuntimeMetricSample, error) {
	if stub.index >= len(stub.samples) {
		return stub.samples[len(stub.samples)-1], nil
	}
	sample := stub.samples[stub.index]
	stub.index++
	return sample, nil
}

func (stub *runtimeMetricSamplerStub) Calls() int {
	return stub.index
}

func TestRuntimeMetricsServiceRetainsBoundedSeries(t *testing.T) {
	base := time.Now().UTC().Add(-10 * time.Second)
	sampler := &runtimeMetricSamplerStub{samples: []RuntimeMetricSample{
		{SampledAt: base, CPU: 10, Memory: 20, Disk: 30, Scope: RuntimeMetricScopeRuntime},
		{SampledAt: base.Add(2 * time.Second), CPU: 40, Memory: 50, Disk: 60, Scope: RuntimeMetricScopeRuntime},
		{SampledAt: base.Add(10 * time.Second), CPU: 70, Memory: 80, Disk: 90, Scope: RuntimeMetricScopeRuntime},
	}}
	service := NewRuntimeMetricsServiceWithOptions(sampler, RuntimeMetricsServiceOptions{
		SampleInterval: 2 * time.Second,
		Retention:      10 * time.Second,
	})

	for range 3 {
		if err := service.CollectOnce(context.Background()); err != nil {
			t.Fatalf("collect sample: %v", err)
		}
	}

	result, err := service.Current(context.Background())
	if err != nil {
		t.Fatalf("current metrics: %v", err)
	}
	if result.SampleIntervalSeconds != 2 || result.RetentionSeconds != 10 {
		t.Fatalf("unexpected metadata: %+v", result)
	}
	if result.Latest.CPU != 70 || result.Latest.Memory != 80 || result.Latest.Disk != 90 {
		t.Fatalf("unexpected latest sample: %+v", result.Latest)
	}
	if len(result.Series) != 2 {
		t.Fatalf("expected retained series length 2, got %d: %+v", len(result.Series), result.Series)
	}
	if !result.Series[0].SampledAt.Equal(base.Add(2 * time.Second)) {
		t.Fatalf("expected oldest retained sample at +2s, got %s", result.Series[0].SampledAt)
	}
}

func TestRuntimeMetricsServiceCurrentReadsOnlyCollectedSamples(t *testing.T) {
	sampledAt := time.Now().UTC()
	sampler := &runtimeMetricSamplerStub{samples: []RuntimeMetricSample{
		{SampledAt: sampledAt, CPU: 44, Memory: 55, Disk: 66, Scope: RuntimeMetricScopeRuntime},
	}}
	service := NewRuntimeMetricsServiceWithOptions(sampler, RuntimeMetricsServiceOptions{
		SampleInterval: time.Millisecond,
		Retention:      time.Minute,
	})

	if _, err := service.Current(context.Background()); err != ErrRuntimeMetricsUnavailable {
		t.Fatalf("expected unavailable before background collection, got %v", err)
	}
	if sampler.Calls() != 0 {
		t.Fatalf("current must not collect samples directly, got %d sampler calls", sampler.Calls())
	}

	if err := service.CollectOnce(context.Background()); err != nil {
		t.Fatalf("collect sample: %v", err)
	}
	if _, err := service.Current(context.Background()); err != nil {
		t.Fatalf("expected current to return collected sample: %v", err)
	}
	if sampler.Calls() != 1 {
		t.Fatalf("expected one explicit collection, got %d", sampler.Calls())
	}
}

func TestRuntimeMetricsServiceUsesTwoSecondDefaultSampleInterval(t *testing.T) {
	sampler := &runtimeMetricSamplerStub{samples: []RuntimeMetricSample{
		{SampledAt: time.Now().UTC(), CPU: 11, Memory: 22, Disk: 33, Scope: RuntimeMetricScopeRuntime},
	}}
	service := NewRuntimeMetricsServiceWithOptions(sampler, RuntimeMetricsServiceOptions{})

	if err := service.CollectOnce(context.Background()); err != nil {
		t.Fatalf("collect sample: %v", err)
	}
	result, err := service.Current(context.Background())
	if err != nil {
		t.Fatalf("current metrics: %v", err)
	}
	if result.SampleIntervalSeconds != 2 {
		t.Fatalf("expected default sample interval 2s, got %+v", result)
	}
}

func TestRuntimeMetricsServiceStartCollectsOnBackgroundInterval(t *testing.T) {
	base := time.Now().UTC()
	sampler := &runtimeMetricSamplerStub{samples: []RuntimeMetricSample{
		{SampledAt: base, CPU: 10, Memory: 20, Disk: 30, Scope: RuntimeMetricScopeRuntime},
		{SampledAt: base.Add(time.Millisecond), CPU: 40, Memory: 50, Disk: 60, Scope: RuntimeMetricScopeRuntime},
	}}
	service := NewRuntimeMetricsServiceWithOptions(sampler, RuntimeMetricsServiceOptions{
		SampleInterval: time.Millisecond,
		Retention:      time.Minute,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service.Start(ctx)
	waitForRuntimeMetricSamples(t, service, 2)
	cancel()

	result, err := service.Current(context.Background())
	if err != nil {
		t.Fatalf("current metrics: %v", err)
	}
	if len(result.Series) < 2 {
		t.Fatalf("expected background sampler to collect at least two samples, got %d", len(result.Series))
	}
	if result.Latest.CPU != 40 || result.Latest.Memory != 50 || result.Latest.Disk != 60 {
		t.Fatalf("unexpected latest background sample: %+v", result.Latest)
	}
}

func TestRuntimeMetricsServiceRejectsMissingSampler(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected missing sampler to panic")
		}
	}()
	_ = NewRuntimeMetricsServiceWithOptions(nil, RuntimeMetricsServiceOptions{})
}

func waitForRuntimeMetricSamples(t *testing.T, service *RuntimeMetricsService, minSamples int) {
	t.Helper()
	deadline := time.After(200 * time.Millisecond)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			result, err := service.Current(context.Background())
			if err != nil {
				t.Fatalf("wait for samples: %v", err)
			}
			t.Fatalf("expected at least %d samples, got %d", minSamples, len(result.Series))
		case <-ticker.C:
			result, err := service.Current(context.Background())
			if err == nil && len(result.Series) >= minSamples {
				return
			}
		}
	}
}
