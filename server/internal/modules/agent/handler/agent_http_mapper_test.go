package handler

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func TestToAgentOutputIncludesAgentVersionWithoutWorkerIdentity(t *testing.T) {
	now := time.Now().UTC()
	agent := &agentdomain.Agent{
		ID:           1,
		InstanceID:   "agt-1",
		DisplayName:  "agent-1",
		Status:       "online",
		AgentVersion: "v2.0.0",
		CreatedAt:    now,
	}

	resp := toAgentOutput(agent, nil)
	if resp.InstanceID != "agt-1" {
		t.Fatalf("expected instanceId field populated")
	}
	if resp.DisplayName != "agent-1" {
		t.Fatalf("expected displayName field populated")
	}
	if resp.AgentVersion != "v2.0.0" {
		t.Fatalf("expected agentVersion field populated")
	}
	if _, ok := reflect.TypeOf(resp).FieldByName("WorkerVersion"); ok {
		t.Fatal("Agent API response must not expose WorkerVersion")
	}
}

func TestAgentDetailOutputExcludesAdministrativeDiagnosticsAndActualAddressClaims(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	response := toAgentDetailOutput(&agentdomain.Agent{
		ID:                   7,
		InstanceID:           "agt-7",
		DisplayName:          "Agent 7",
		Status:               "online",
		ObservedSourceIP:     "8.8.8.8",
		ObservedIPGeneration: 2,
		Location: &agentdomain.AgentLocationSnapshot{
			AgentID:          7,
			Latitude:         1,
			Longitude:        2,
			SourceObservedIP: "8.8.8.8",
			ProviderKey:      "freeipapi",
			ResolvedAt:       now,
		},
		CreatedAt: now,
	}, nil, now)
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal Agent detail response: %v", err)
	}
	var document any
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatalf("Unmarshal Agent detail response: %v", err)
	}
	assertJSONFieldsAbsent(t, document, map[string]struct{}{
		"country": {}, "countryName": {}, "region": {}, "regionName": {}, "city": {},
		"postalCode": {}, "timezone": {}, "asn": {}, "isp": {}, "proxy": {}, "isProxy": {},
		"providerName": {}, "rawResponse": {}, "lastAttempt": {}, "lastAttemptAt": {},
		"attemptState": {}, "failureClass": {}, "failureReason": {}, "nextEligibleAt": {},
		"realIp": {}, "actualIp": {}, "agentPublicIp": {},
	})
}

func TestToAgentDetailOutputPreservesCurrentLocationPrecisionAndNullableRadius(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	agent := &agentdomain.Agent{
		ID:                   7,
		InstanceID:           "agt-7",
		DisplayName:          "Agent 7",
		Status:               "online",
		ObservedSourceIP:     "8.8.8.8",
		ObservedIPGeneration: 3,
		Location: &agentdomain.AgentLocationSnapshot{
			AgentID:          7,
			Latitude:         1.123456789012345,
			Longitude:        2.987654321098765,
			SourceObservedIP: "8.8.8.8",
			ProviderKey:      "freeipapi",
			ResolvedAt:       now.Add(-time.Hour),
		},
		CreatedAt: now,
	}
	response := toAgentDetailOutput(agent, nil, now)
	if response.ObservedSourceIP != "8.8.8.8" || response.ObservedIPGeneration != 3 || response.LocationState != "current" {
		t.Fatalf("observation/freshness projection = %#v", response)
	}
	if response.Location == nil || response.Location.Latitude != 1.123456789012345 || response.Location.Longitude != 2.987654321098765 || response.Location.AccuracyRadiusKM != nil {
		t.Fatalf("current location was rounded or given a synthetic radius: %#v", response.Location)
	}
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal Agent detail: %v", err)
	}
	var serialized map[string]any
	if err := json.Unmarshal(payload, &serialized); err != nil {
		t.Fatalf("Unmarshal Agent detail: %v", err)
	}
	location, ok := serialized["location"].(map[string]any)
	if !ok || location["latitude"] != 1.123456789012345 || location["longitude"] != 2.987654321098765 {
		t.Fatalf("serialized coordinates lost precision: %s", payload)
	}
	if radius, exists := location["accuracyRadiusKm"]; !exists || radius != nil {
		t.Fatalf("nullable accuracyRadiusKm must be explicit null: %s", payload)
	}
}

func TestToAgentDetailOutputKeepsExpiredProvenanceAfterSourceChangeOrFailure(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	radius := 12.5
	agent := &agentdomain.Agent{
		ID:                   7,
		InstanceID:           "agt-7",
		DisplayName:          "Agent 7",
		Status:               "online",
		ConnectionIP:         "172.20.0.5",
		ObservedSourceIP:     "1.1.1.1",
		ObservedIPGeneration: 4,
		Location: &agentdomain.AgentLocationSnapshot{
			AgentID:          7,
			Latitude:         1.123456789,
			Longitude:        2.987654321,
			AccuracyRadiusKM: &radius,
			SourceObservedIP: "8.8.8.8",
			ProviderKey:      "freeipapi",
			ResolvedAt:       now.Add(-time.Hour),
			ForcedExpired:    true,
		},
		CreatedAt: now,
	}
	response := toAgentDetailOutput(agent, nil, now)
	if response.ObservedSourceIP != "1.1.1.1" || response.ObservedIPGeneration != 4 || response.LocationState != "expired" {
		t.Fatalf("observation/freshness projection = %#v", response)
	}
	if response.Location == nil || response.Location.SourceObservedIP != "8.8.8.8" || response.Location.ProviderKey != "freeipapi" || !response.Location.ResolvedAt.Equal(now.Add(-time.Hour)) || response.Location.AccuracyRadiusKM == nil || *response.Location.AccuracyRadiusKM != radius {
		t.Fatalf("last-success provenance was rebound or discarded: %#v", response.Location)
	}
	if response.ConnectionIP != "172.20.0.5" {
		t.Fatalf("connection IP projection = %q, want private control address", response.ConnectionIP)
	}
	if _, exists := reflect.TypeOf(response).FieldByName("IPAddress"); exists {
		t.Fatal("Agent detail response must not expose legacy IPAddress")
	}
}

func TestToAgentDetailOutputUsesExplicitUnknownWithoutPlaceholder(t *testing.T) {
	response := toAgentDetailOutput(&agentdomain.Agent{ID: 8, DisplayName: "Agent 8", CreatedAt: time.Now().UTC()}, nil, time.Now().UTC())
	if response.LocationState != "unknown" || response.Location != nil {
		t.Fatalf("unknown location projection = %#v", response)
	}
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal Agent detail: %v", err)
	}
	if !strings.Contains(string(payload), `"locationState":"unknown"`) || !strings.Contains(string(payload), `"location":null`) {
		t.Fatalf("unknown location is not explicit: %s", payload)
	}
}

func assertJSONFieldsAbsent(t *testing.T, value any, forbidden map[string]struct{}) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if _, found := forbidden[key]; found {
				t.Errorf("Agent detail exposed forbidden field %q", key)
			}
			assertJSONFieldsAbsent(t, nested, forbidden)
		}
	case []any:
		for _, nested := range typed {
			assertJSONFieldsAbsent(t, nested, forbidden)
		}
	}
}
