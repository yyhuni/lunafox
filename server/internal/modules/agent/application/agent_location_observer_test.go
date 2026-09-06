package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func TestAgentLocationObserverSuppressesUnknownNonPublicAndCurrentSameIP(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	lookup := &agentLocationLookupStub{}
	observer := newAgentLocationObserverForTest(t, &agentLocationRepositoryStub{}, lookup, base)

	observer.ObserveReadyConnection(nil)
	observer.ObserveReadyConnection(&agentdomain.Agent{ID: 1, ObservedIPGeneration: 1})
	observer.ObserveReadyConnection(&agentdomain.Agent{ID: 2, ObservedSourceIP: "10.0.0.1", ObservedIPGeneration: 1})
	observer.ObserveReadyConnection(&agentdomain.Agent{
		ID:                   3,
		ObservedSourceIP:     "8.8.8.8",
		ObservedIPGeneration: 2,
		Location: &agentdomain.AgentLocationSnapshot{
			AgentID:          3,
			SourceObservedIP: "8.8.8.8",
			ResolvedAt:       base.Add(-6 * 24 * time.Hour),
		},
	})

	if submissions := lookup.submissionCount(); submissions != 0 {
		t.Fatalf("suppressed observations submitted %d lookups", submissions)
	}
}

func TestAgentLocationObserverSubmitsFirstChangedAndExpiredObservations(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	lookup := &agentLocationLookupStub{}
	observer := newAgentLocationObserverForTest(t, &agentLocationRepositoryStub{}, lookup, base)
	agents := []*agentdomain.Agent{
		{ID: 1, ObservedSourceIP: "8.8.8.8", ObservedIPGeneration: 1},
		{
			ID:                   2,
			ObservedSourceIP:     "1.1.1.1",
			ObservedIPGeneration: 2,
			Location: &agentdomain.AgentLocationSnapshot{
				AgentID:          2,
				SourceObservedIP: "8.8.8.8",
				ResolvedAt:       base,
			},
		},
		{
			ID:                   3,
			ObservedSourceIP:     "9.9.9.9",
			ObservedIPGeneration: 4,
			Location: &agentdomain.AgentLocationSnapshot{
				AgentID:          3,
				SourceObservedIP: "9.9.9.9",
				ResolvedAt:       base.Add(-7 * 24 * time.Hour),
			},
		},
		{
			ID:                   4,
			ObservedSourceIP:     "208.67.222.222",
			ObservedIPGeneration: 5,
			Location: &agentdomain.AgentLocationSnapshot{
				AgentID:          4,
				SourceObservedIP: "208.67.222.222",
				ResolvedAt:       base,
				ForcedExpired:    true,
			},
		},
	}
	for _, agent := range agents {
		observer.ObserveReadyConnection(agent)
	}

	submissions := lookup.snapshotSubmissions()
	if len(submissions) != 4 {
		t.Fatalf("eligible submissions = %d, want 4", len(submissions))
	}
	for index, submission := range submissions {
		if submission.sourceIP != agents[index].ObservedSourceIP {
			t.Fatalf("submission %d source = %q, want %q", index, submission.sourceIP, agents[index].ObservedSourceIP)
		}
	}
}

func TestAgentLocationObserverPersistsCompleteSuccessWithCapturedGeneration(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	repository := &agentLocationRepositoryStub{replaceCalls: make(chan agentLocationReplaceCall, 1)}
	lookup := &agentLocationLookupStub{}
	observer := newAgentLocationObserverForTest(t, repository, lookup, base)
	observer.ObserveReadyConnection(&agentdomain.Agent{ID: 7, ObservedSourceIP: "8.8.8.8", ObservedIPGeneration: 9})
	submission := lookup.waitSubmission(t)
	radius := 12.5
	submission.completion(AgentLocationLookupOutcome{
		ProviderAttempted: true,
		CompletedAt:       base.Add(time.Second),
		Location: &AgentLocationResolution{
			ObservedIP:       "8.8.8.8",
			Latitude:         37.4219999,
			Longitude:        -122.0840575,
			AccuracyRadiusKM: &radius,
			ProviderKey:      "freeipapi",
			ResolvedAt:       base.Add(time.Second),
		},
	})

	select {
	case call := <-repository.replaceCalls:
		if call.generation != 9 || call.snapshot.AgentID != 7 || call.snapshot.SourceObservedIP != "8.8.8.8" || call.snapshot.ProviderKey != "freeipapi" || call.snapshot.ForcedExpired || !call.snapshot.ResolvedAt.Equal(base.Add(time.Second)) || !call.snapshot.UpdatedAt.Equal(base) || call.snapshot.AccuracyRadiusKM == nil || *call.snapshot.AccuracyRadiusKM != radius {
			t.Fatalf("persisted success call = %#v", call)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for success persistence")
	}
}

func TestAgentLocationObserverExpiresOnlyProviderFailureOutcomes(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	repository := &agentLocationRepositoryStub{expireCalls: make(chan agentLocationExpireCall, 2)}
	lookup := &agentLocationLookupStub{}
	observer := newAgentLocationObserverForTest(t, repository, lookup, base)

	observer.ObserveReadyConnection(&agentdomain.Agent{ID: 7, ObservedSourceIP: "8.8.8.8", ObservedIPGeneration: 3})
	lookup.waitSubmission(t).completion(AgentLocationLookupOutcome{FailureClass: "network", ProviderAttempted: true, CompletedAt: base})
	select {
	case call := <-repository.expireCalls:
		if call.agentID != 7 || call.sourceIP != "8.8.8.8" || call.generation != 3 {
			t.Fatalf("provider failure expiration = %#v", call)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for provider failure expiration")
	}

	observer.ObserveReadyConnection(&agentdomain.Agent{ID: 8, ObservedSourceIP: "1.1.1.1", ObservedIPGeneration: 4})
	lookup.waitSubmission(t).completion(AgentLocationLookupOutcome{Canceled: true, ProviderAttempted: true, CompletedAt: base})
	select {
	case call := <-repository.expireCalls:
		t.Fatalf("cancellation unexpectedly expired a location: %#v", call)
	case <-time.After(10 * time.Millisecond):
	}
}

func TestAgentLocationObserverSubmissionFailureIsNonBlockingAndDoesNotPersist(t *testing.T) {
	repository := &agentLocationRepositoryStub{
		replaceCalls: make(chan agentLocationReplaceCall, 1),
		expireCalls:  make(chan agentLocationExpireCall, 1),
	}
	lookup := &agentLocationLookupStub{submitErr: errors.New("queue full")}
	observer := newAgentLocationObserverForTest(t, repository, lookup, time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC))

	observer.ObserveReadyConnection(&agentdomain.Agent{ID: 7, ObservedSourceIP: "8.8.8.8", ObservedIPGeneration: 1})
	select {
	case call := <-repository.replaceCalls:
		t.Fatalf("submission failure unexpectedly replaced a location: %#v", call)
	case call := <-repository.expireCalls:
		t.Fatalf("submission failure unexpectedly expired a location: %#v", call)
	case <-time.After(10 * time.Millisecond):
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := observer.Shutdown(shutdownContext); err != nil {
		t.Fatalf("Shutdown after submission error: %v", err)
	}
}

func TestAgentLocationObserverShutdownCancelsAcceptedWaitersAndClosesAdmission(t *testing.T) {
	lookup := &agentLocationLookupStub{}
	observer := newAgentLocationObserverForTest(t, &agentLocationRepositoryStub{}, lookup, time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC))
	observer.ObserveReadyConnection(&agentdomain.Agent{ID: 7, ObservedSourceIP: "8.8.8.8", ObservedIPGeneration: 1})
	lookup.waitSubmission(t)

	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := observer.Shutdown(shutdownContext); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	before := lookup.submissionCount()
	observer.ObserveReadyConnection(&agentdomain.Agent{ID: 8, ObservedSourceIP: "1.1.1.1", ObservedIPGeneration: 1})
	if after := lookup.submissionCount(); after != before {
		t.Fatalf("post-shutdown submission count = %d, want %d", after, before)
	}
}

func TestAgentLocationObserverRejectsMismatchedProviderSource(t *testing.T) {
	repository := &agentLocationRepositoryStub{replaceCalls: make(chan agentLocationReplaceCall, 1)}
	lookup := &agentLocationLookupStub{}
	observer := newAgentLocationObserverForTest(t, repository, lookup, time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC))
	observer.ObserveReadyConnection(&agentdomain.Agent{ID: 7, ObservedSourceIP: "8.8.8.8", ObservedIPGeneration: 1})
	lookup.waitSubmission(t).completion(AgentLocationLookupOutcome{Location: &AgentLocationResolution{
		ObservedIP:  "1.1.1.1",
		Latitude:    1,
		Longitude:   2,
		ProviderKey: "freeipapi",
		ResolvedAt:  time.Now().UTC(),
	}})
	select {
	case call := <-repository.replaceCalls:
		t.Fatalf("mismatched source unexpectedly persisted: %#v", call)
	case <-time.After(10 * time.Millisecond):
	}
}

func newAgentLocationObserverForTest(
	t *testing.T,
	repository agentdomain.AgentLocationRepository,
	lookup AgentLocationLookup,
	now time.Time,
) *AgentLocationObserver {
	t.Helper()
	observer, err := NewAgentLocationObserver(repository, lookup, fixedClock{now: now})
	if err != nil {
		t.Fatalf("NewAgentLocationObserver: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := observer.Shutdown(ctx); err != nil {
			t.Errorf("observer cleanup: %v", err)
		}
	})
	return observer
}

type agentLocationRepositoryStub struct {
	replaceCalls chan agentLocationReplaceCall
	expireCalls  chan agentLocationExpireCall
	replaceErr   error
	expireErr    error
}

type agentLocationReplaceCall struct {
	snapshot   agentdomain.AgentLocationSnapshot
	generation int64
}

type agentLocationExpireCall struct {
	agentID    int
	sourceIP   string
	generation int64
}

func (repository *agentLocationRepositoryStub) GetLocation(context.Context, int) (*agentdomain.AgentLocationSnapshot, error) {
	return nil, nil
}

func (repository *agentLocationRepositoryStub) ReplaceLocationIfObservationMatches(_ context.Context, snapshot agentdomain.AgentLocationSnapshot, generation int64) (bool, error) {
	if repository.replaceCalls != nil {
		repository.replaceCalls <- agentLocationReplaceCall{snapshot: snapshot, generation: generation}
	}
	return repository.replaceErr == nil, repository.replaceErr
}

func (repository *agentLocationRepositoryStub) MarkLocationExpiredIfObservationMatches(_ context.Context, agentID int, sourceIP string, generation int64) (bool, error) {
	if repository.expireCalls != nil {
		repository.expireCalls <- agentLocationExpireCall{agentID: agentID, sourceIP: sourceIP, generation: generation}
	}
	return repository.expireErr == nil, repository.expireErr
}

type agentLocationLookupStub struct {
	mu          sync.Mutex
	submissions []*agentLocationLookupSubmission
	submitErr   error
}

type agentLocationLookupSubmission struct {
	sourceIP   string
	completion func(AgentLocationLookupOutcome)
}

func (lookup *agentLocationLookupStub) SubmitIP(ctx context.Context, sourceIP string, completion func(AgentLocationLookupOutcome)) error {
	lookup.mu.Lock()
	if lookup.submitErr != nil {
		err := lookup.submitErr
		lookup.mu.Unlock()
		return err
	}
	var completionOnce sync.Once
	complete := func(outcome AgentLocationLookupOutcome) {
		completionOnce.Do(func() { completion(outcome) })
	}
	submission := &agentLocationLookupSubmission{sourceIP: sourceIP, completion: complete}
	lookup.submissions = append(lookup.submissions, submission)
	lookup.mu.Unlock()
	go func() {
		<-ctx.Done()
		complete(AgentLocationLookupOutcome{Canceled: true, CompletedAt: time.Now().UTC()})
	}()
	return nil
}

func (lookup *agentLocationLookupStub) waitSubmission(t *testing.T) *agentLocationLookupSubmission {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		lookup.mu.Lock()
		if len(lookup.submissions) > 0 {
			submission := lookup.submissions[0]
			lookup.submissions = lookup.submissions[1:]
			lookup.mu.Unlock()
			return submission
		}
		lookup.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for Agent location lookup submission")
		}
		time.Sleep(time.Millisecond)
	}
}

func (lookup *agentLocationLookupStub) submissionCount() int {
	lookup.mu.Lock()
	defer lookup.mu.Unlock()
	return len(lookup.submissions)
}

func (lookup *agentLocationLookupStub) snapshotSubmissions() []*agentLocationLookupSubmission {
	lookup.mu.Lock()
	defer lookup.mu.Unlock()
	return append([]*agentLocationLookupSubmission(nil), lookup.submissions...)
}
