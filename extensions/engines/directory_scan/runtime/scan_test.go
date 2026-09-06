package directoryscanruntime

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestScanAndParseDirectoriesProcessesPreDeadlineRecordsBeforeAllTimeoutFailure(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 0)
	workspace := t.TempDir()
	record := marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/found", Status: 200, ContentLength: 7, ContentType: "text/plain", Duration: 10})
	runner := ffufRunnerFunc(func(ctx context.Context, invocation ffufInvocation) error {
		path, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, append(append([]byte{}, record...), '\n'), 0o600); err != nil {
			return err
		}
		<-ctx.Done()
		return ctx.Err()
	})
	var observations []DirectoryObservation
	summary, err := scanAndParseDirectories(context.Background(), plan, defaultFFUFConfig(), workspace, func(observation DirectoryObservation) error {
		observations = append(observations, observation)
		return nil
	}, websiteSchedulerOptions{runner: runner, websiteTimeout: 15 * time.Millisecond})
	if !errors.Is(err, ErrAllWebsitesTimedOut) {
		t.Fatalf("scan error = %v, want ErrAllWebsitesTimedOut", err)
	}
	if !summary.Websites.AllWebsitesTimedOut() || summary.Records != (FFUFParseSummary{SourceRecords: 2, ParsedItems: 2}) {
		t.Fatalf("scan summary = %#v", summary)
	}
	if len(observations) != 2 || observations[0].CandidateOrdinal != 0 || observations[1].CandidateOrdinal != 1 {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestScanAndParseDirectoriesAllowsPartialTimeoutAndTrueZeroResults(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 1)
	workspace := t.TempDir()
	runner := ffufRunnerFunc(func(ctx context.Context, invocation ffufInvocation) error {
		path, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			return err
		}
		if invocation.Candidate.Ordinal == 0 {
			return nil
		}
		<-ctx.Done()
		return ctx.Err()
	})
	summary, err := scanAndParseDirectories(context.Background(), plan, defaultFFUFConfig(), workspace, func(DirectoryObservation) error {
		t.Fatal("zero-result artifacts produced an observation")
		return nil
	}, websiteSchedulerOptions{runner: runner, websiteTimeout: 15 * time.Millisecond})
	if err != nil {
		t.Fatalf("scan error = %v", err)
	}
	if summary.Websites.NormalCompletedWebsites != 1 || summary.Websites.TimedOutWebsites != 2 || summary.Records != (FFUFParseSummary{}) {
		t.Fatalf("scan summary = %#v", summary)
	}
}

func TestScanAndParseDirectoriesRoutesAllRejectedRecordsToFailure(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 0)
	workspace := t.TempDir()
	invalid := []byte(`{"url":null,"status":200,"length":1,"content-type":"","duration":1}` + "\n")
	runner := ffufRunnerFunc(func(_ context.Context, invocation ffufInvocation) error {
		path, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
		if err != nil {
			return err
		}
		return os.WriteFile(path, invalid, 0o600)
	})
	summary, err := scanAndParseDirectories(context.Background(), plan, defaultFFUFConfig(), workspace, func(DirectoryObservation) error {
		t.Fatal("invalid records produced an observation")
		return nil
	}, websiteSchedulerOptions{runner: runner, websiteTimeout: time.Second})
	if !errors.Is(err, ErrAllFFUFRecordsRejected) {
		t.Fatalf("scan error = %v, want ErrAllFFUFRecordsRejected", err)
	}
	if summary.Records != (FFUFParseSummary{SourceRecords: 2, InvalidRecords: 2}) {
		t.Fatalf("scan summary = %#v", summary)
	}
}

func TestScanAndParseDirectoriesKeepsAllWebsiteTimeoutPrecedenceOverRejectedRecords(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 0)
	workspace := t.TempDir()
	invalid := []byte(`{"url":null,"status":200,"length":1,"content-type":"","duration":1}` + "\n")
	runner := ffufRunnerFunc(func(ctx context.Context, invocation ffufInvocation) error {
		path, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, invalid, 0o600); err != nil {
			return err
		}
		<-ctx.Done()
		return ctx.Err()
	})
	summary, err := scanAndParseDirectories(context.Background(), plan, defaultFFUFConfig(), workspace, func(DirectoryObservation) error { return nil }, websiteSchedulerOptions{
		runner: runner, websiteTimeout: 15 * time.Millisecond,
	})
	if !errors.Is(err, ErrAllWebsitesTimedOut) || errors.Is(err, ErrAllFFUFRecordsRejected) {
		t.Fatalf("scan error = %v, want only ErrAllWebsitesTimedOut", err)
	}
	if !summary.Websites.AllWebsitesTimedOut() || summary.Records.InvalidRecords != 2 {
		t.Fatalf("scan summary = %#v", summary)
	}
}

func TestScanAndParseDirectoriesDoesNotParseAfterTaskFatalProcessFailure(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 0)
	workspace := t.TempDir()
	wantErr := errors.New("FFUF exited non-zero")
	var visits atomic.Int32
	record := marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/not-submitted", Status: 200, ContentLength: 1, ContentType: "", Duration: 1})
	runner := ffufRunnerFunc(func(ctx context.Context, invocation ffufInvocation) error {
		path, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
		if err != nil {
			return err
		}
		payload := append(append([]byte(nil), record...), '\n')
		if err := os.WriteFile(path, payload, 0o600); err != nil {
			return err
		}
		if invocation.Candidate.Ordinal == 0 {
			return wantErr
		}
		<-ctx.Done()
		return ctx.Err()
	})
	_, err := scanAndParseDirectories(context.Background(), plan, defaultFFUFConfig(), workspace, func(DirectoryObservation) error {
		visits.Add(1)
		return nil
	}, websiteSchedulerOptions{runner: runner, websiteTimeout: time.Second})
	if !errors.Is(err, wantErr) || visits.Load() != 0 {
		t.Fatalf("scan error=%v visits=%d", err, visits.Load())
	}
}

func TestScanAndParseDirectoriesPropagatesArtifactReadAndCallbackFailures(t *testing.T) {
	t.Run("missing artifact", func(t *testing.T) {
		plan, _ := schedulerCandidatePlan(t, 0)
		_, err := scanAndParseDirectories(context.Background(), plan, defaultFFUFConfig(), t.TempDir(), func(DirectoryObservation) error { return nil }, websiteSchedulerOptions{
			runner: ffufRunnerFunc(func(context.Context, ffufInvocation) error { return nil }), websiteTimeout: time.Second,
		})
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing artifact error = %v", err)
		}
	})

	t.Run("callback", func(t *testing.T) {
		plan, _ := schedulerCandidatePlan(t, 0)
		workspace := t.TempDir()
		wantErr := errors.New("stage failed")
		record := marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com", Status: 200, ContentLength: 1, ContentType: "", Duration: 1})
		runner := ffufRunnerFunc(func(_ context.Context, invocation ffufInvocation) error {
			path, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
			if err != nil {
				return err
			}
			payload := append(append([]byte(nil), record...), '\n')
			return os.WriteFile(path, payload, 0o600)
		})
		summary, err := scanAndParseDirectories(context.Background(), plan, defaultFFUFConfig(), workspace, func(DirectoryObservation) error {
			return wantErr
		}, websiteSchedulerOptions{runner: runner, websiteTimeout: time.Second})
		if !errors.Is(err, wantErr) || summary.Records.SourceRecords != 1 || summary.Records.ParsedItems != 0 {
			t.Fatalf("callback summary=%#v error=%v", summary, err)
		}
	})
}
