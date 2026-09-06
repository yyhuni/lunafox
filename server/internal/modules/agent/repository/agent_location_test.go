package repository

import (
	"context"
	"math"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func TestAgentLocationRepositoryReplacesCompleteSnapshotOnlyForMatchingObservation(t *testing.T) {
	db := openAgentConnectionTestDatabase(t)
	connectionRepository := &agentRepository{db: db}
	locationRepository := &agentLocationRepository{db: db}
	ctx := context.Background()
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	observation, err := connectionRepository.RecordConnection(ctx, 7, "8.8.8.8", base)
	if err != nil {
		t.Fatalf("RecordConnection: %v", err)
	}
	radius := 12.5
	snapshot := agentdomain.AgentLocationSnapshot{
		AgentID:          7,
		Latitude:         37.4219999,
		Longitude:        -122.0840575,
		AccuracyRadiusKM: &radius,
		SourceObservedIP: "8.8.8.8",
		ProviderKey:      "freeipapi",
		ResolvedAt:       base.Add(time.Second),
		UpdatedAt:        base.Add(2 * time.Second),
	}
	replaced, err := locationRepository.ReplaceLocationIfObservationMatches(ctx, snapshot, observation.Generation)
	if err != nil || !replaced {
		t.Fatalf("ReplaceLocationIfObservationMatches() = (%t, %v), want success", replaced, err)
	}
	stored, err := locationRepository.GetLocation(ctx, 7)
	if err != nil {
		t.Fatalf("GetLocation: %v", err)
	}
	if stored == nil || stored.SourceObservedIP != "8.8.8.8" || stored.ProviderKey != "freeipapi" || stored.ForcedExpired || !stored.ResolvedAt.Equal(snapshot.ResolvedAt) || stored.AccuracyRadiusKM == nil || *stored.AccuracyRadiusKM != radius {
		t.Fatalf("stored location = %#v", stored)
	}

	if _, err := connectionRepository.RecordConnection(ctx, 7, "1.1.1.1", base.Add(3*time.Second)); err != nil {
		t.Fatalf("RecordConnection(change): %v", err)
	}
	if _, err := connectionRepository.RecordConnection(ctx, 7, "8.8.8.8", base.Add(4*time.Second)); err != nil {
		t.Fatalf("RecordConnection(return): %v", err)
	}
	stale := snapshot
	stale.Latitude = 50
	stale.ResolvedAt = base.Add(5 * time.Second)
	stale.UpdatedAt = base.Add(5 * time.Second)
	replaced, err = locationRepository.ReplaceLocationIfObservationMatches(ctx, stale, observation.Generation)
	if err != nil || replaced {
		t.Fatalf("stale A-to-B-to-A replacement = (%t, %v), want fenced", replaced, err)
	}
	stored, err = locationRepository.GetLocation(ctx, 7)
	if err != nil || stored == nil || stored.Latitude != snapshot.Latitude || !stored.ForcedExpired || !stored.ResolvedAt.Equal(snapshot.ResolvedAt) {
		t.Fatalf("stale write changed prior provenance: location=%#v err=%v", stored, err)
	}
}

func TestAgentLocationRepositoryExpiresOnlyMatchingGenerationWithoutRewritingProvenance(t *testing.T) {
	db := openAgentConnectionTestDatabase(t)
	connectionRepository := &agentRepository{db: db}
	locationRepository := &agentLocationRepository{db: db}
	ctx := context.Background()
	base := time.Date(2026, 8, 4, 13, 0, 0, 0, time.UTC)
	observation, err := connectionRepository.RecordConnection(ctx, 7, "8.8.8.8", base)
	if err != nil {
		t.Fatalf("RecordConnection: %v", err)
	}
	insertAgentLocationSnapshot(t, db, 7, "8.8.8.8", base)

	marked, err := locationRepository.MarkLocationExpiredIfObservationMatches(ctx, 7, "8.8.8.8", observation.Generation)
	if err != nil || !marked {
		t.Fatalf("MarkLocationExpiredIfObservationMatches() = (%t, %v), want success", marked, err)
	}
	stored, err := locationRepository.GetLocation(ctx, 7)
	if err != nil || stored == nil || !stored.ForcedExpired || stored.SourceObservedIP != "8.8.8.8" || !stored.ResolvedAt.Equal(base) {
		t.Fatalf("expired snapshot provenance = %#v, err=%v", stored, err)
	}

	if _, err := connectionRepository.RecordConnection(ctx, 7, "1.1.1.1", base.Add(time.Minute)); err != nil {
		t.Fatalf("RecordConnection(change): %v", err)
	}
	marked, err = locationRepository.MarkLocationExpiredIfObservationMatches(ctx, 7, "8.8.8.8", observation.Generation)
	if err != nil || marked {
		t.Fatalf("stale expiration = (%t, %v), want fenced", marked, err)
	}
}

func TestAgentLocationRepositoryFastFailsIncompleteSuccessSnapshots(t *testing.T) {
	db := openAgentConnectionTestDatabase(t)
	repository := &agentLocationRepository{db: db}
	base := time.Date(2026, 8, 4, 14, 0, 0, 0, time.UTC)
	valid := agentdomain.AgentLocationSnapshot{
		AgentID:          7,
		Latitude:         1,
		Longitude:        2,
		SourceObservedIP: "8.8.8.8",
		ProviderKey:      "freeipapi",
		ResolvedAt:       base,
		UpdatedAt:        base,
	}
	tests := []struct {
		name       string
		generation int64
		mutate     func(*agentdomain.AgentLocationSnapshot)
	}{
		{name: "missing agent", generation: 1, mutate: func(snapshot *agentdomain.AgentLocationSnapshot) { snapshot.AgentID = 0 }},
		{name: "missing generation", generation: 0, mutate: func(*agentdomain.AgentLocationSnapshot) {}},
		{name: "private source", generation: 1, mutate: func(snapshot *agentdomain.AgentLocationSnapshot) { snapshot.SourceObservedIP = "10.0.0.1" }},
		{name: "non-normalized source", generation: 1, mutate: func(snapshot *agentdomain.AgentLocationSnapshot) { snapshot.SourceObservedIP = " 8.8.8.8 " }},
		{name: "invalid latitude", generation: 1, mutate: func(snapshot *agentdomain.AgentLocationSnapshot) { snapshot.Latitude = math.Inf(1) }},
		{name: "invalid longitude", generation: 1, mutate: func(snapshot *agentdomain.AgentLocationSnapshot) { snapshot.Longitude = 181 }},
		{name: "negative radius", generation: 1, mutate: func(snapshot *agentdomain.AgentLocationSnapshot) { radius := -1.0; snapshot.AccuracyRadiusKM = &radius }},
		{name: "missing provider", generation: 1, mutate: func(snapshot *agentdomain.AgentLocationSnapshot) { snapshot.ProviderKey = "" }},
		{name: "missing resolution time", generation: 1, mutate: func(snapshot *agentdomain.AgentLocationSnapshot) { snapshot.ResolvedAt = time.Time{} }},
		{name: "forced expired success", generation: 1, mutate: func(snapshot *agentdomain.AgentLocationSnapshot) { snapshot.ForcedExpired = true }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid
			test.mutate(&snapshot)
			if replaced, err := repository.ReplaceLocationIfObservationMatches(context.Background(), snapshot, test.generation); err == nil || replaced {
				t.Fatalf("invalid replacement = (%t, %v), want fast-fail", replaced, err)
			}
		})
	}
}

func TestAgentLocationRepositoryReturnsUnknownWhenNoSuccessExists(t *testing.T) {
	db := openAgentConnectionTestDatabase(t)
	connectionRepository := &agentRepository{db: db}
	repository := &agentLocationRepository{db: db}
	observation, err := connectionRepository.RecordConnection(context.Background(), 7, "8.8.8.8", time.Date(2026, 8, 4, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("RecordConnection: %v", err)
	}
	marked, err := repository.MarkLocationExpiredIfObservationMatches(context.Background(), 7, "8.8.8.8", observation.Generation)
	if err != nil || marked {
		t.Fatalf("failure without prior success = (%t, %v), want no row", marked, err)
	}
	location, err := repository.GetLocation(context.Background(), 7)
	if err != nil || location != nil {
		t.Fatalf("GetLocation without success = (%#v, %v), want nil", location, err)
	}
}
