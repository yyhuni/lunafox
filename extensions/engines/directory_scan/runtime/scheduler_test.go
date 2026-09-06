package directoryscanruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

type ffufRunnerFunc func(context.Context, ffufInvocation) error

func (run ffufRunnerFunc) Run(ctx context.Context, invocation ffufInvocation) error {
	return run(ctx, invocation)
}

func TestWebsiteSchedulerCapsConcurrencyAndUsesEveryCandidateOnce(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 8)
	config := defaultFFUFConfig()
	config.Concurrency = 3
	started := make(chan uint64, plan.CandidateCount())
	release := make(chan struct{})
	var active atomic.Int32
	var maximum atomic.Int32
	var callsMu sync.Mutex
	calls := make(map[uint64]int)
	runner := ffufRunnerFunc(func(_ context.Context, invocation ffufInvocation) error {
		current := active.Add(1)
		updateMaximum(&maximum, current)
		defer active.Add(-1)
		callsMu.Lock()
		calls[invocation.Candidate.Ordinal]++
		callsMu.Unlock()
		started <- invocation.Candidate.Ordinal
		<-release
		return nil
	})

	result := make(chan schedulerResult, 1)
	workspace := t.TempDir()
	go func() {
		summary, err := runWebsiteScans(context.Background(), plan, config, workspace, websiteSchedulerOptions{
			runner: runner, websiteTimeout: time.Second,
		})
		result <- schedulerResult{summary: summary, err: err}
	}()
	for range int(config.Concurrency) {
		<-started
	}
	select {
	case ordinal := <-started:
		t.Fatalf("candidate %d started before a concurrency slot was released", ordinal)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	completed := <-result
	if completed.err != nil {
		t.Fatalf("runWebsiteScans() error = %v", completed.err)
	}
	if completed.summary.NormalCompletedWebsites != plan.CandidateCount() || completed.summary.TimedOutWebsites != 0 {
		t.Fatalf("scheduler summary = %#v", completed.summary)
	}
	if maximum.Load() != int32(config.Concurrency) || active.Load() != 0 {
		t.Fatalf("maximum active=%d final active=%d", maximum.Load(), active.Load())
	}
	callsMu.Lock()
	defer callsMu.Unlock()
	if uint64(len(calls)) != plan.CandidateCount() {
		t.Fatalf("invoked ordinals = %#v", calls)
	}
	for ordinal, count := range calls {
		if count != 1 {
			t.Fatalf("candidate %d invocation count = %d", ordinal, count)
		}
	}
}

func TestWebsiteSchedulerAcceptsOutOfOrderCompletionWithoutChangingCounts(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 2)
	config := defaultFFUFConfig()
	config.Concurrency = 2
	releaseFirst := make(chan struct{})
	completedOrdinals := make(chan uint64, plan.CandidateCount())
	runner := ffufRunnerFunc(func(_ context.Context, invocation ffufInvocation) error {
		if invocation.Candidate.Ordinal == 0 {
			<-releaseFirst
		}
		completedOrdinals <- invocation.Candidate.Ordinal
		return nil
	})
	result := make(chan schedulerResult, 1)
	workspace := t.TempDir()
	go func() {
		summary, err := runWebsiteScans(context.Background(), plan, config, workspace, websiteSchedulerOptions{runner: runner, websiteTimeout: time.Second})
		result <- schedulerResult{summary: summary, err: err}
	}()
	firstCompleted := <-completedOrdinals
	if firstCompleted == 0 {
		t.Fatal("test did not force out-of-order completion")
	}
	close(releaseFirst)
	completed := <-result
	if completed.err != nil || completed.summary.NormalCompletedWebsites != plan.CandidateCount() {
		t.Fatalf("scheduler result = %#v error=%v", completed.summary, completed.err)
	}
}

func TestWebsiteSchedulerSeparatesPartialAndAllChildDeadlines(t *testing.T) {
	for _, test := range []struct {
		name       string
		normal     map[uint64]bool
		wantNormal uint64
		wantTimed  uint64
		allTimed   bool
	}{
		{name: "partial timeout", normal: map[uint64]bool{0: true}, wantNormal: 1, wantTimed: 2},
		{name: "all timeout", normal: map[uint64]bool{}, wantTimed: 3, allTimed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, _ := schedulerCandidatePlan(t, 1)
			config := defaultFFUFConfig()
			config.Concurrency = 3
			var callsMu sync.Mutex
			calls := make(map[uint64]int)
			runner := ffufRunnerFunc(func(ctx context.Context, invocation ffufInvocation) error {
				callsMu.Lock()
				calls[invocation.Candidate.Ordinal]++
				callsMu.Unlock()
				if test.normal[invocation.Candidate.Ordinal] {
					return nil
				}
				<-ctx.Done()
				return ctx.Err()
			})
			summary, err := runWebsiteScans(context.Background(), plan, config, t.TempDir(), websiteSchedulerOptions{
				runner: runner, websiteTimeout: 15 * time.Millisecond,
			})
			if err != nil {
				t.Fatalf("runWebsiteScans() error = %v", err)
			}
			if summary.NormalCompletedWebsites != test.wantNormal || summary.TimedOutWebsites != test.wantTimed || summary.AllWebsitesTimedOut() != test.allTimed {
				t.Fatalf("scheduler summary = %#v", summary)
			}
			callsMu.Lock()
			defer callsMu.Unlock()
			for ordinal := uint64(0); ordinal < plan.CandidateCount(); ordinal++ {
				if calls[ordinal] != 1 {
					t.Fatalf("candidate %d invocation count = %d", ordinal, calls[ordinal])
				}
			}
		})
	}
}

func TestWebsiteSchedulerReleasesTimedOutSlotForQueuedCandidates(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 2)
	config := defaultFFUFConfig()
	config.Concurrency = 1
	var callsMu sync.Mutex
	var calls []uint64
	runner := ffufRunnerFunc(func(ctx context.Context, invocation ffufInvocation) error {
		callsMu.Lock()
		calls = append(calls, invocation.Candidate.Ordinal)
		callsMu.Unlock()
		if invocation.Candidate.Ordinal == 0 {
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	})
	summary, err := runWebsiteScans(context.Background(), plan, config, t.TempDir(), websiteSchedulerOptions{
		runner: runner, websiteTimeout: 15 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("runWebsiteScans() error = %v", err)
	}
	if summary.TimedOutWebsites != 1 || summary.NormalCompletedWebsites != plan.CandidateCount()-1 {
		t.Fatalf("scheduler summary = %#v", summary)
	}
	callsMu.Lock()
	defer callsMu.Unlock()
	want := []uint64{0, 1, 2, 3}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("invocation order = %#v, want %#v", calls, want)
	}
}

func TestWebsiteSchedulerParentCancellationTakesPrecedenceAndJoinsWorkers(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 8)
	config := defaultFFUFConfig()
	config.Concurrency = 2
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{}, int(config.Concurrency))
	var active atomic.Int32
	runner := ffufRunnerFunc(func(ctx context.Context, _ ffufInvocation) error {
		active.Add(1)
		defer active.Add(-1)
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	})
	result := make(chan schedulerResult, 1)
	workspace := t.TempDir()
	go func() {
		summary, err := runWebsiteScans(ctx, plan, config, workspace, websiteSchedulerOptions{runner: runner, websiteTimeout: time.Second})
		result <- schedulerResult{summary: summary, err: err}
	}()
	<-started
	cancel()
	completed := <-result
	if !errors.Is(completed.err, context.Canceled) {
		t.Fatalf("scheduler error = %v, want context.Canceled", completed.err)
	}
	if active.Load() != 0 {
		t.Fatalf("active workers after return = %d", active.Load())
	}
}

func TestWebsiteSchedulerParentDeadlineTakesPrecedenceOverChildAggregation(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 2)
	config := defaultFFUFConfig()
	config.Concurrency = 2
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	runner := ffufRunnerFunc(func(ctx context.Context, _ ffufInvocation) error {
		<-ctx.Done()
		return ctx.Err()
	})
	summary, err := runWebsiteScans(ctx, plan, config, t.TempDir(), websiteSchedulerOptions{
		runner: runner, websiteTimeout: time.Second,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("scheduler error = %v, want parent context.DeadlineExceeded", err)
	}
	if summary.AllWebsitesTimedOut() {
		t.Fatalf("parent deadline was converted to all-Website timeout: %#v", summary)
	}
}

func TestWebsiteSchedulerFFUFNonZeroFailsFastAndRetainsPrimaryError(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 8)
	config := defaultFFUFConfig()
	config.Concurrency = 2
	wantErr := errors.New("FFUF exit code 7")
	barrier := make(chan struct{})
	var barrierOnce sync.Once
	var startedCount atomic.Int32
	var active atomic.Int32
	var startedMu sync.Mutex
	var started []uint64
	runner := ffufRunnerFunc(func(ctx context.Context, invocation ffufInvocation) error {
		active.Add(1)
		defer active.Add(-1)
		startedMu.Lock()
		started = append(started, invocation.Candidate.Ordinal)
		startedMu.Unlock()
		if startedCount.Add(1) == int32(config.Concurrency) {
			barrierOnce.Do(func() { close(barrier) })
		}
		<-barrier
		if invocation.Candidate.Ordinal == 0 {
			return wantErr
		}
		<-ctx.Done()
		return ctx.Err()
	})
	summary, err := runWebsiteScans(context.Background(), plan, config, t.TempDir(), websiteSchedulerOptions{
		runner: runner, websiteTimeout: time.Second,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("scheduler error = %v, want primary process failure", err)
	}
	if summary.NormalCompletedWebsites != 0 || summary.TimedOutWebsites != 0 || active.Load() != 0 {
		t.Fatalf("scheduler summary=%#v active=%d", summary, active.Load())
	}
	startedMu.Lock()
	defer startedMu.Unlock()
	sort.Slice(started, func(left, right int) bool { return started[left] < started[right] })
	if !reflect.DeepEqual(started, []uint64{0, 1}) {
		t.Fatalf("started candidates after fail-fast = %#v", started)
	}
}

func TestWebsiteSchedulerParentCancellationOverridesConcurrentProcessFailure(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 0)
	config := defaultFFUFConfig()
	config.Concurrency = 2
	ctx, cancel := context.WithCancel(context.Background())
	wantProcessErr := errors.New("process failed")
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	runner := ffufRunnerFunc(func(ctx context.Context, invocation ffufInvocation) error {
		started <- struct{}{}
		if invocation.Candidate.Ordinal == 0 {
			<-release
			return wantProcessErr
		}
		<-ctx.Done()
		return ctx.Err()
	})
	result := make(chan schedulerResult, 1)
	workspace := t.TempDir()
	go func() {
		summary, err := runWebsiteScans(ctx, plan, config, workspace, websiteSchedulerOptions{runner: runner, websiteTimeout: time.Second})
		result <- schedulerResult{summary: summary, err: err}
	}()
	<-started
	<-started
	cancel()
	close(release)
	completed := <-result
	if !errors.Is(completed.err, context.Canceled) {
		t.Fatalf("scheduler error = %v, want parent context.Canceled", completed.err)
	}
}

func TestWebsiteSchedulerPropagatesReplayAndSharedOperationFailures(t *testing.T) {
	t.Run("same-count replay drift", func(t *testing.T) {
		plan, facts := schedulerCandidatePlan(t, 1)
		if err := os.WriteFile(facts, []byte("https://example.com/replaced\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		summary, err := runWebsiteScans(context.Background(), plan, defaultFFUFConfig(), t.TempDir(), websiteSchedulerOptions{
			runner: ffufRunnerFunc(func(context.Context, ffufInvocation) error { return nil }), websiteTimeout: time.Second,
		})
		if err == nil || !strings.Contains(err.Error(), "changed between passes") || summary.WebsiteCandidates != 3 {
			t.Fatalf("replay drift summary=%#v error=%v", summary, err)
		}
	})

	t.Run("deadline plus artifact failure", func(t *testing.T) {
		plan, _ := schedulerCandidatePlan(t, 0)
		wantErr := errors.New("artifact close failed")
		runner := ffufRunnerFunc(func(ctx context.Context, _ ffufInvocation) error {
			<-ctx.Done()
			return newFFUFSharedOperationError("close FFUF raw artifact", wantErr)
		})
		summary, err := runWebsiteScans(context.Background(), plan, defaultFFUFConfig(), t.TempDir(), websiteSchedulerOptions{
			runner: runner, websiteTimeout: 15 * time.Millisecond,
		})
		if !errors.Is(err, wantErr) || summary.TimedOutWebsites != 0 {
			t.Fatalf("shared failure summary=%#v error=%v", summary, err)
		}
	})
}

type schedulerResult struct {
	summary WebsiteScanSummary
	err     error
}

func schedulerCandidatePlan(t *testing.T, additionalWebsites int) (*CandidatePlan, string) {
	t.Helper()
	websites := make([]string, additionalWebsites)
	for index := range websites {
		websites[index] = fmt.Sprintf("https://example.com/%d", index)
	}
	facts := writeWebsiteURLFacts(t, websites...)
	plan, err := PreflightCandidates(context.Background(), testDomainTarget(), facts)
	if err != nil {
		t.Fatalf("PreflightCandidates() error = %v", err)
	}
	return plan, facts
}

func testDomainTarget() enginecontract.Target {
	return enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}
}

func updateMaximum(maximum *atomic.Int32, value int32) {
	for {
		current := maximum.Load()
		if value <= current || maximum.CompareAndSwap(current, value) {
			return
		}
	}
}
