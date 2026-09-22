package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/upgrader"
)

const (
	dispatcherOperationID = "11111111-1111-4111-8111-111111111111"
	dispatcherDigest      = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func TestHostUpgradeDispatcherUsesOnlyExplicitV1CapabilityFallback(t *testing.T) {
	socketPath, requests := newDispatcherSocket(t, func(upgrader.Request) upgrader.Response {
		return upgrader.Response{SchemaVersion: upgrader.RequestSchema, Error: "unsupported upgrader request schema version 2"}
	})
	dispatcher, err := NewHostUpgradeDispatcher(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dispatcher.PlanUpgradeScope(context.Background(), application.HostUpgradeScopePlanRequest{
		OperationID: dispatcherOperationID, ManifestDigest: dispatcherDigest,
	})
	if !errors.Is(err, application.ErrHostUpgradeScopePlanningUnsupported) {
		t.Fatalf("legacy capability result error = %v", err)
	}
	request := receiveDispatcherRequest(t, requests)
	if request.SchemaVersion != upgrader.ScopedRequestSchema || request.Action != upgrader.ActionCapabilities {
		t.Fatalf("capability request = %#v", request)
	}
}

func TestHostUpgradeDispatcherRejectsMalformedV2Capabilities(t *testing.T) {
	socketPath, _ := newDispatcherSocket(t, func(upgrader.Request) upgrader.Response {
		return upgrader.Response{
			SchemaVersion: upgrader.ScopedRequestSchema,
			Accepted:      true,
			Capabilities:  &upgrader.HostCapabilities{SchemaVersions: []int{upgrader.ScopedRequestSchema, upgrader.ScopedRequestSchema}},
		}
	})
	dispatcher, err := NewHostUpgradeDispatcher(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dispatcher.PlanUpgradeScope(context.Background(), application.HostUpgradeScopePlanRequest{
		OperationID: dispatcherOperationID, ManifestDigest: dispatcherDigest,
	})
	if err == nil || errors.Is(err, application.ErrHostUpgradeScopePlanningUnsupported) {
		t.Fatalf("malformed v2 capability error = %v", err)
	}
}

func TestHostUpgradeDispatcherCandidateAvailabilityUsesSchemaV3(t *testing.T) {
	tests := []struct {
		name         string
		availability upgrader.CandidateAvailability
		wantDecision application.HostCandidateAvailabilityDecision
		wantError    error
	}{
		{
			name: "available",
			availability: upgrader.CandidateAvailability{
				ManifestDigest: dispatcherDigest, Decision: upgrader.CandidateAvailabilityAvailable,
				BaselineStateDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ConfirmedDeploymentVersion: "1.0.0",
			},
			wantDecision: application.HostCandidateAvailabilityAvailable,
		},
		{
			name: "already applied",
			availability: upgrader.CandidateAvailability{
				ManifestDigest: dispatcherDigest, Decision: upgrader.CandidateAvailabilityAlreadyApplied,
				BaselineStateDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ConfirmedDeploymentVersion: "1.2.3",
			},
			wantDecision: application.HostCandidateAvailabilityAlreadyApplied,
		},
		{
			name: "not newer",
			availability: upgrader.CandidateAvailability{
				ManifestDigest: dispatcherDigest, Decision: upgrader.CandidateAvailabilityNotNewer,
				BaselineStateDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ConfirmedDeploymentVersion: "1.3.0",
			},
			wantDecision: application.HostCandidateAvailabilityNotNewer,
		},
		{
			name: "fallback",
			availability: upgrader.CandidateAvailability{
				ManifestDigest: dispatcherDigest, Decision: upgrader.CandidateAvailabilityFallback,
			},
			wantDecision: application.HostCandidateAvailabilityFallback,
		},
		{
			name: "conflict fails closed",
			availability: upgrader.CandidateAvailability{
				ManifestDigest: dispatcherDigest, Decision: upgrader.CandidateAvailabilityConflict,
				BaselineStateDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ConfirmedDeploymentVersion: "1.2.3",
			},
			wantError: application.ErrHostDeploymentStateConflict,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			socketPath, requests := newDispatcherSocket(t, func(request upgrader.Request) upgrader.Response {
				switch request.Action {
				case upgrader.ActionCapabilities:
					capabilities := upgrader.DefaultHostCapabilities()
					return upgrader.Response{SchemaVersion: upgrader.RequestSchemaV3, Accepted: true, Capabilities: &capabilities}
				case upgrader.ActionCandidateAvailability:
					return upgrader.Response{SchemaVersion: upgrader.RequestSchemaV3, Accepted: true, CandidateAvailability: &test.availability}
				default:
					return upgrader.Response{SchemaVersion: upgrader.RequestSchemaV3, Error: "unexpected action"}
				}
			})
			dispatcher, err := NewHostUpgradeDispatcher(socketPath)
			if err != nil {
				t.Fatal(err)
			}
			result, err := dispatcher.CandidateAvailability(context.Background(), dispatcherDigest)
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Fatalf("CandidateAvailability() error = %v, want %v", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Decision != test.wantDecision {
				t.Fatalf("CandidateAvailability() = %#v, want decision %q", result, test.wantDecision)
			}
			first := receiveDispatcherRequest(t, requests)
			second := receiveDispatcherRequest(t, requests)
			if first.SchemaVersion != upgrader.RequestSchemaV3 || first.Action != upgrader.ActionCapabilities || second.SchemaVersion != upgrader.RequestSchemaV3 || second.Action != upgrader.ActionCandidateAvailability || second.OperationID != "" || second.ManifestDigest != dispatcherDigest {
				t.Fatalf("candidate availability wire sequence = %#v, %#v", first, second)
			}
		})
	}
}

func TestHostUpgradeDispatcherCandidateAvailabilityAllowsOnlyExactV1Fallback(t *testing.T) {
	t.Run("old host", func(t *testing.T) {
		socketPath, requests := newDispatcherSocket(t, func(upgrader.Request) upgrader.Response {
			return upgrader.Response{SchemaVersion: upgrader.RequestSchema, Error: "unsupported upgrader request schema version 3"}
		})
		dispatcher, err := NewHostUpgradeDispatcher(socketPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dispatcher.CandidateAvailability(context.Background(), dispatcherDigest); !errors.Is(err, application.ErrHostCandidateAvailabilityUnsupported) {
			t.Fatalf("old-host availability error = %v", err)
		}
		request := receiveDispatcherRequest(t, requests)
		if request.SchemaVersion != upgrader.RequestSchemaV3 || request.Action != upgrader.ActionCapabilities {
			t.Fatalf("old-host capability request = %#v", request)
		}
	})

	t.Run("malformed v3 capability is not a fallback", func(t *testing.T) {
		socketPath, _ := newDispatcherSocket(t, func(upgrader.Request) upgrader.Response {
			return upgrader.Response{
				SchemaVersion: upgrader.RequestSchemaV3,
				Accepted:      true,
				Capabilities: &upgrader.HostCapabilities{
					SchemaVersions:                 []int{upgrader.RequestSchema, upgrader.RequestSchemaV3, upgrader.RequestSchemaV3},
					CandidateInventoryAvailability: true,
				},
			}
		})
		dispatcher, err := NewHostUpgradeDispatcher(socketPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dispatcher.CandidateAvailability(context.Background(), dispatcherDigest); err == nil || errors.Is(err, application.ErrHostCandidateAvailabilityUnsupported) {
			t.Fatalf("malformed v3 capability error = %v", err)
		}
	})

	t.Run("v3 schema without inventory capability is unsupported", func(t *testing.T) {
		socketPath, requests := newDispatcherSocket(t, func(upgrader.Request) upgrader.Response {
			capabilities := upgrader.DefaultHostCapabilities()
			capabilities.CandidateInventoryAvailability = false
			return upgrader.Response{SchemaVersion: upgrader.RequestSchemaV3, Accepted: true, Capabilities: &capabilities}
		})
		dispatcher, err := NewHostUpgradeDispatcher(socketPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dispatcher.CandidateAvailability(context.Background(), dispatcherDigest); !errors.Is(err, application.ErrHostCandidateAvailabilityUnsupported) {
			t.Fatalf("incomplete v3 capability error = %v", err)
		}
		request := receiveDispatcherRequest(t, requests)
		if request.Action != upgrader.ActionCapabilities {
			t.Fatalf("incomplete v3 capability request = %#v", request)
		}
	})

	t.Run("rejected inventory comparison fails closed", func(t *testing.T) {
		socketPath, _ := newDispatcherSocket(t, func(request upgrader.Request) upgrader.Response {
			if request.Action == upgrader.ActionCapabilities {
				capabilities := upgrader.DefaultHostCapabilities()
				return upgrader.Response{SchemaVersion: upgrader.RequestSchemaV3, Accepted: true, Capabilities: &capabilities}
			}
			return upgrader.Response{SchemaVersion: upgrader.RequestSchemaV3, Error: "inventory comparison failed"}
		})
		dispatcher, err := NewHostUpgradeDispatcher(socketPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dispatcher.CandidateAvailability(context.Background(), dispatcherDigest); err == nil || errors.Is(err, application.ErrHostCandidateAvailabilityUnsupported) {
			t.Fatalf("rejected inventory comparison error = %v", err)
		}
	})
}

func TestHostUpgradeDispatcherMapsBoundFullScopePlan(t *testing.T) {
	plan := dispatcherFullPlan(t, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	socketPath, requests := newDispatcherSocket(t, func(request upgrader.Request) upgrader.Response {
		switch request.Action {
		case upgrader.ActionCapabilities:
			capabilities := dispatcherV2Capabilities()
			return upgrader.Response{SchemaVersion: upgrader.ScopedRequestSchema, Accepted: true, Capabilities: &capabilities}
		case upgrader.ActionPlan:
			return upgrader.Response{SchemaVersion: upgrader.ScopedRequestSchema, Accepted: true, ScopePlan: &plan}
		default:
			return upgrader.Response{SchemaVersion: upgrader.ScopedRequestSchema, Error: "unexpected action"}
		}
	})
	dispatcher, err := NewHostUpgradeDispatcher(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	result, err := dispatcher.PlanUpgradeScope(context.Background(), application.HostUpgradeScopePlanRequest{
		OperationID: dispatcherOperationID, ManifestDigest: dispatcherDigest, RequireFull: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionMode != domain.ExecutionModeFull || result.PlanDigest != plan.PlanDigest || result.BaselineDeploymentDigest != "" || result.ConfirmedDeploymentVersion != "" {
		t.Fatalf("mapped scope plan = %#v", result)
	}
	first := receiveDispatcherRequest(t, requests)
	second := receiveDispatcherRequest(t, requests)
	if first.Action != upgrader.ActionCapabilities || second.Action != upgrader.ActionPlan || !second.RequireFull {
		t.Fatalf("scope plan wire sequence = %#v, %#v", first, second)
	}
}

func TestHostUpgradeDispatcherFallsBackToV1FullWhenCompositionIsUnavailable(t *testing.T) {
	plan := dispatcherFullPlan(t, "")
	socketPath, requests := newDispatcherSocket(t, func(request upgrader.Request) upgrader.Response {
		switch request.Action {
		case upgrader.ActionCapabilities:
			capabilities := dispatcherV2Capabilities()
			return upgrader.Response{SchemaVersion: upgrader.ScopedRequestSchema, Accepted: true, Capabilities: &capabilities}
		case upgrader.ActionPlan:
			return upgrader.Response{SchemaVersion: upgrader.ScopedRequestSchema, Accepted: true, ScopePlan: &plan}
		default:
			return upgrader.Response{SchemaVersion: upgrader.ScopedRequestSchema, Error: "unexpected action"}
		}
	})
	dispatcher, err := NewHostUpgradeDispatcher(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dispatcher.PlanUpgradeScope(context.Background(), application.HostUpgradeScopePlanRequest{
		OperationID: dispatcherOperationID, ManifestDigest: dispatcherDigest,
	})
	if !errors.Is(err, application.ErrHostUpgradeScopePlanningFullOnly) {
		t.Fatalf("composition-unavailable plan error = %v", err)
	}
	first := receiveDispatcherRequest(t, requests)
	second := receiveDispatcherRequest(t, requests)
	if first.Action != upgrader.ActionCapabilities || second.Action != upgrader.ActionPlan {
		t.Fatalf("composition fallback wire sequence = %#v, %#v", first, second)
	}
}

func TestHostUpgradeDispatcherSendsV2PlanBoundStart(t *testing.T) {
	plan := dispatcherFullPlan(t, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	socketPath, requests := newDispatcherSocket(t, func(request upgrader.Request) upgrader.Response {
		now := time.Now().UTC()
		journal := upgrader.Journal{
			SchemaVersion:              upgrader.ScopedJournalSchema,
			OperationID:                request.OperationID,
			ManifestDigest:             request.ManifestDigest,
			Stage:                      upgrader.StageQueued,
			ExecutionMode:              request.ExecutionMode,
			PlanDigest:                 request.PlanDigest,
			BaselineStateDigest:        request.BaselineStateDigest,
			TouchedServices:            append([]string(nil), request.TouchedServices...),
			ConfirmedDeploymentVersion: request.ConfirmedDeploymentVersion,
			StartedAt:                  now,
			UpdatedAt:                  now,
			StageUpdatedAt:             now,
		}
		return upgrader.Response{SchemaVersion: upgrader.ScopedRequestSchema, Accepted: true, Journal: journal}
	})
	dispatcher, err := NewHostUpgradeDispatcher(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	err = dispatcher.Dispatch(context.Background(), application.HostUpgradeRequest{
		OperationID: dispatcherOperationID, ManifestDigest: dispatcherDigest, Action: application.HostUpgradeActionStart,
		ExecutionMode: domain.ExecutionModeFull, PlanDigest: plan.PlanDigest, TouchedServices: append([]string(nil), plan.TouchedServices...),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := receiveDispatcherRequest(t, requests)
	if request.SchemaVersion != upgrader.ScopedRequestSchema || request.Action != upgrader.ActionStart || request.ExecutionMode != upgrader.ExecutionModeFull || request.PlanDigest != plan.PlanDigest {
		t.Fatalf("plan-bound start request = %#v", request)
	}
}

func dispatcherFullPlan(t *testing.T, compositionDigest string) upgrader.ScopePlan {
	t.Helper()
	store, err := upgrader.NewJournalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	candidate := upgrader.CandidateDeployment{
		ReleaseVersion:    "1.2.3",
		CompositionDigest: compositionDigest,
		Components: []upgrader.DeploymentComponent{
			{ID: "runtime.agent", Digest: dispatcherDigest},
			{ID: "runtime.bootstrap", Digest: dispatcherDigest},
			{ID: "runtime.frontend", Digest: dispatcherDigest},
			{ID: "runtime.nginx", Digest: dispatcherDigest},
			{ID: "runtime.server", Digest: dispatcherDigest},
		},
	}
	planner := upgrader.NewHostScopePlanner(store, upgrader.CandidateDeploymentSourceFunc(func(context.Context, string) (upgrader.CandidateDeployment, error) {
		return candidate, nil
	}), nil)
	plan, err := planner.Plan(context.Background(), dispatcherOperationID, dispatcherDigest, true)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func dispatcherV2Capabilities() upgrader.HostCapabilities {
	capabilities := upgrader.DefaultHostCapabilities()
	capabilities.SchemaVersions = []int{upgrader.RequestSchema, upgrader.ScopedRequestSchema}
	capabilities.CandidateInventoryAvailability = false
	return capabilities
}

func newDispatcherSocket(t *testing.T, responseFor func(upgrader.Request) upgrader.Response) (string, <-chan upgrader.Request) {
	t.Helper()
	// macOS caps Unix-domain socket paths below the nested testing.TempDir
	// length. A short /tmp directory keeps this protocol test portable.
	directory, err := os.MkdirTemp("/tmp", "lf-up-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	socketPath := filepath.Join(directory, "u.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	requests := make(chan upgrader.Request, 4)
	go func() {
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			var request upgrader.Request
			decodeErr := json.NewDecoder(connection).Decode(&request)
			if decodeErr == nil {
				requests <- request
				_ = json.NewEncoder(connection).Encode(responseFor(request))
			}
			_ = connection.Close()
		}
	}()
	t.Cleanup(func() { _ = listener.Close() })
	return socketPath, requests
}

func receiveDispatcherRequest(t *testing.T, requests <-chan upgrader.Request) upgrader.Request {
	t.Helper()
	select {
	case request := <-requests:
		return request
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for host dispatcher request")
		return upgrader.Request{}
	}
}
