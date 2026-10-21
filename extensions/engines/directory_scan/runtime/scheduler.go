package directoryscanruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

// WebsiteScanSummary contains only task-wide outcome counts. Raw artifact
// paths are derived from stable ordinals, so the scheduler retains no
// candidate-sized result collection.
type WebsiteScanSummary struct {
	WebsiteCandidates       uint64
	NormalCompletedWebsites uint64
	TimedOutWebsites        uint64
	// WAFSkippedWebsites counts candidates skipped by the optional WAF pre-scan.
	// They are also counted in NormalCompletedWebsites so the existing completion
	// contract stays unchanged.
	WAFSkippedWebsites uint64
}

func (summary WebsiteScanSummary) AllWebsitesTimedOut() bool {
	return summary.WebsiteCandidates > 0 &&
		summary.NormalCompletedWebsites == 0 &&
		summary.TimedOutWebsites == summary.WebsiteCandidates
}

type websiteSchedulerOptions struct {
	runner         ffufProcessRunner
	websiteTimeout time.Duration
	// prepareWordlist optionally stages a candidate-specific wordlist. When set,
	// the returned cleanup runs after that candidate's invocation and the path is
	// passed to FFUF as -w instead of the shared config wordlist.
	prepareWordlist func(context.Context, WebsiteCandidate) (string, func() error, error)
	// skipCandidate lets the caller skip an invocation (for example when a WAF
	// pre-scan already proved the origin is blocked). Skipped candidates count
	// as normally completed.
	skipCandidate func(context.Context, WebsiteCandidate) (bool, string, error)
}

type websiteSchedulerState struct {
	mu              sync.Mutex
	stopped         bool
	primaryErr      error
	normalCompleted uint64
	timedOut        uint64
	wafSkipped      uint64
}

func RunWebsiteScans(
	ctx context.Context,
	plan *CandidatePlan,
	config enginecontract.FfufConfig,
	workspace string,
) (WebsiteScanSummary, error) {
	return runWebsiteScans(ctx, plan, config, workspace, websiteSchedulerOptions{
		runner:         newContainerFFUFProcessRunner(),
		websiteTimeout: time.Duration(config.Timeout) * time.Second,
	})
}

func runWebsiteScans(
	ctx context.Context,
	plan *CandidatePlan,
	config enginecontract.FfufConfig,
	workspace string,
	options websiteSchedulerOptions,
) (WebsiteScanSummary, error) {
	if ctx == nil {
		return WebsiteScanSummary{}, errors.New("Website scheduler context is required")
	}
	if plan == nil {
		return WebsiteScanSummary{}, errors.New("candidate plan is required")
	}
	if err := ValidateConfig(config); err != nil {
		return WebsiteScanSummary{}, err
	}
	if _, err := requireDirectoryWorkspace(workspace); err != nil {
		return WebsiteScanSummary{}, err
	}
	if options.runner == nil || options.websiteTimeout <= 0 {
		return WebsiteScanSummary{}, errors.New("Website scheduler options are incomplete")
	}

	summary := WebsiteScanSummary{WebsiteCandidates: plan.CandidateCount()}
	schedulerContext, cancelScheduler := context.WithCancel(ctx)
	defer cancelScheduler()
	replay, err := plan.StartReplay(schedulerContext, int(config.Concurrency))
	if err != nil {
		return summary, err
	}
	state := &websiteSchedulerState{}
	replayDone := make(chan error, 1)
	go func() {
		replayErr := replay.Wait()
		if replayErr != nil && ctx.Err() == nil && !errors.Is(replayErr, context.Canceled) {
			if state.stop(fmt.Errorf("replay Website candidates: %w", replayErr)) {
				cancelScheduler()
			}
		}
		replayDone <- replayErr
	}()

	var workers sync.WaitGroup
	workers.Add(int(config.Concurrency))
	for range int(config.Concurrency) {
		go func() {
			defer workers.Done()
			for {
				select {
				case <-schedulerContext.Done():
					return
				case candidate, ok := <-replay.Candidates:
					if !ok {
						return
					}
					if schedulerContext.Err() != nil {
						return
					}
					if !state.beginInvocation() {
						return
					}
					if options.skipCandidate != nil {
						blocked, _, skipErr := options.skipCandidate(schedulerContext, candidate)
						if skipErr == nil && blocked {
							artifactPath, artifactErr := rawFFUFArtifactPath(workspace, candidate.Ordinal)
							if artifactErr != nil {
								if state.stop(fmt.Errorf("prepare skipped artifact %d: %w", candidate.Ordinal, artifactErr)) {
									replay.Stop()
									cancelScheduler()
								}
								return
							}
							if artifactErr := os.WriteFile(artifactPath, nil, 0o600); artifactErr != nil {
								if state.stop(fmt.Errorf("write skipped artifact %d: %w", candidate.Ordinal, artifactErr)) {
									replay.Stop()
									cancelScheduler()
								}
								return
							}
							if !state.recordWAFSkip() {
								return
							}
							continue
						}
					}
					candidateConfig := config
					cleanupWordlist := func() error { return nil }
					if options.prepareWordlist != nil {
						path, cleanup, prepareErr := options.prepareWordlist(schedulerContext, candidate)
						if prepareErr != nil {
							if schedulerContext.Err() == nil && state.stop(fmt.Errorf("prepare candidate wordlist %d: %w", candidate.Ordinal, prepareErr)) {
								replay.Stop()
								cancelScheduler()
							}
							return
						}
						candidateConfig.Wordlist = path
						if cleanup != nil {
							cleanupWordlist = cleanup
						}
					}
					childContext, cancelChild := context.WithTimeout(schedulerContext, options.websiteTimeout)
					runErr := options.runner.Run(childContext, ffufInvocation{
						Candidate: candidate,
						Config:    candidateConfig,
						Workspace: workspace,
					})
					childContextErr := childContext.Err()
					cancelChild()
					if cleanupErr := cleanupWordlist(); cleanupErr != nil {
						if state.stop(fmt.Errorf("cleanup candidate wordlist %d: %w", candidate.Ordinal, cleanupErr)) {
							replay.Stop()
							cancelScheduler()
						}
						return
					}

					if ctx.Err() != nil || state.isStopped() {
						return
					}
					if runErr == nil {
						if !state.recordNormalCompletion() {
							return
						}
						continue
					}
					if errors.Is(childContextErr, context.DeadlineExceeded) && !isFFUFSharedOperationError(runErr) {
						if !state.recordTimeout() {
							return
						}
						continue
					}
					if schedulerContext.Err() != nil {
						return
					}
					if state.stop(fmt.Errorf("FFUF invocation %d failed: %w", candidate.Ordinal, runErr)) {
						replay.Stop()
						cancelScheduler()
					}
					return
				}
			}
		}()
	}
	workers.Wait()
	replayErr := <-replayDone

	summary.NormalCompletedWebsites, summary.TimedOutWebsites, summary.WAFSkippedWebsites, err = state.snapshot()
	if contextErr := ctx.Err(); contextErr != nil {
		return summary, contextErr
	}
	if err != nil {
		return summary, err
	}
	if replayErr != nil && !errors.Is(replayErr, context.Canceled) {
		return summary, fmt.Errorf("replay Website candidates: %w", replayErr)
	}
	if summary.NormalCompletedWebsites+summary.TimedOutWebsites != summary.WebsiteCandidates {
		return summary, errors.New("Website scheduler outcome count does not match candidate count")
	}
	return summary, nil
}

func (state *websiteSchedulerState) beginInvocation() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	return !state.stopped
}

func (state *websiteSchedulerState) isStopped() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.stopped
}

func (state *websiteSchedulerState) recordNormalCompletion() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.stopped {
		return false
	}
	state.normalCompleted++
	return true
}

func (state *websiteSchedulerState) recordWAFSkip() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.stopped {
		return false
	}
	state.normalCompleted++
	state.wafSkipped++
	return true
}

func (state *websiteSchedulerState) recordTimeout() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.stopped {
		return false
	}
	state.timedOut++
	return true
}

func (state *websiteSchedulerState) stop(err error) bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.stopped {
		return false
	}
	state.stopped = true
	state.primaryErr = err
	return true
}

func (state *websiteSchedulerState) snapshot() (uint64, uint64, uint64, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.normalCompleted, state.timedOut, state.wafSkipped, state.primaryErr
}
