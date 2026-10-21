package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type agentLicenseStoreStub struct {
	license AgentLicense
	err     error
	key     string
	at      time.Time
	calls   int
}

func (stub *agentLicenseStoreStub) CurrentAt(_ context.Context, now time.Time) (AgentLicense, error) {
	stub.at = now
	stub.calls++
	return stub.license, stub.err
}

func (stub *agentLicenseStoreStub) ActivateAt(_ context.Context, key string, now time.Time) (AgentLicense, error) {
	stub.key = key
	stub.at = now
	stub.calls++
	return stub.license, stub.err
}

func TestAgentLicenseServiceActivatesTrimmedKeyAtOneInstant(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.FixedZone("test", 8*60*60))
	store := &agentLicenseStoreStub{license: AgentLicense{State: "active", AgentLimit: 10}}
	service, err := NewAgentLicenseService(store, fixedClock{now: now})
	if err != nil {
		t.Fatal(err)
	}
	license, err := service.Activate(context.Background(), "  signed-key \n")
	if err != nil || license.AgentLimit != 10 || store.calls != 1 || store.key != "signed-key" || !store.at.Equal(now.UTC()) || store.at.Location() != time.UTC {
		t.Fatalf("Activate license=%#v key=%q at=%v calls=%d err=%v", license, store.key, store.at, store.calls, err)
	}
	store.err = errors.New("storage unavailable")
	if _, err := service.Activate(context.Background(), "signed-key"); !errors.Is(err, store.err) {
		t.Fatal(err)
	}
}

func TestAgentLicenseServiceRejectsMissingAndOversizeKeysBeforePersistence(t *testing.T) {
	store := &agentLicenseStoreStub{}
	service, err := NewAgentLicenseService(store, fixedClock{now: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		key  string
		want error
	}{
		{" \n", agentdomain.ErrAgentLicenseKeyRequired},
		{strings.Repeat("x", 4097), agentdomain.ErrAgentLicenseKeyInvalid},
	} {
		if _, err := service.Activate(context.Background(), sample.key); !errors.Is(err, sample.want) {
			t.Fatal(err)
		}
	}
	if store.calls != 0 {
		t.Fatal("invalid inputs reached persistence")
	}
	if _, err := NewAgentLicenseService(nil, fixedClock{}); err == nil {
		t.Fatal("nil store accepted")
	}
	if _, err := NewAgentLicenseService(store, nil); err == nil {
		t.Fatal("nil clock accepted")
	}
}
