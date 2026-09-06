package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type hostPortCommandStoreStub struct {
	batchUpsertIn    []assetdomain.HostPort
	deletedIPs       []string
	batchUpsertCalls int
	deleteByIPsCalls int
	upsertErr        error
	deleteErr        error
}

func (stub *hostPortCommandStoreStub) BatchUpsertContext(_ context.Context, items []assetdomain.HostPort) (int64, error) {
	stub.batchUpsertCalls++
	if stub.upsertErr != nil {
		return 0, stub.upsertErr
	}
	stub.batchUpsertIn = append([]assetdomain.HostPort(nil), items...)
	return int64(len(items)), nil
}

func (stub *hostPortCommandStoreStub) DeleteByIPs(ips []string) (int64, error) {
	stub.deleteByIPsCalls++
	if stub.deleteErr != nil {
		return 0, stub.deleteErr
	}
	stub.deletedIPs = append([]string(nil), ips...)
	return int64(len(ips)), nil
}

type hostPortCommandTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
	ctx     context.Context
}

func (stub *hostPortCommandTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	target, ok := stub.targets[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *target
	return &copyTarget, nil
}

func (stub *hostPortCommandTargetLookupStub) GetActiveByIDContext(ctx context.Context, id int) (*assetdomain.TargetRef, error) {
	stub.ctx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetActiveByID(id)
}

func TestHostPortCommandServiceBatchUpsertAndDelete(t *testing.T) {
	store := &hostPortCommandStoreStub{}
	lookup := &hostPortCommandTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}}}
	service := NewHostPortCommandService(store, lookup)
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "caller")

	affected, err := service.BatchUpsert(ctx, 1, []HostPortItem{{Host: "a.example.com", IP: "1.1.1.1", Port: 443}})
	if err != nil {
		t.Fatalf("batch upsert failed: %v", err)
	}
	if affected != 1 || len(store.batchUpsertIn) != 1 {
		t.Fatalf("unexpected upsert result affected=%d size=%d", affected, len(store.batchUpsertIn))
	}
	if lookup.ctx != ctx || lookup.ctx.Value(contextKey{}) != "caller" {
		t.Fatal("target lookup did not preserve caller context")
	}

	deleted, err := service.BatchDeleteByIPs(context.Background(), []string{"1.1.1.1", "2.2.2.2"})
	if err != nil {
		t.Fatalf("batch delete failed: %v", err)
	}
	if deleted != 2 || len(store.deletedIPs) != 2 {
		t.Fatalf("unexpected delete result deleted=%d ips=%v", deleted, store.deletedIPs)
	}
}

func TestHostPortCommandServiceBatchUpsertIgnoresIPv6Items(t *testing.T) {
	store := &hostPortCommandStoreStub{}
	lookup := &hostPortCommandTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}}}
	service := NewHostPortCommandService(store, lookup)

	affected, err := service.BatchUpsert(context.Background(), 1, []HostPortItem{
		{Host: "www.example.com", IP: "2606:4700::6812:1034", Port: 443},
		{Host: "api.example.com", IP: "192.0.2.10", Port: 443},
	})
	if err != nil {
		t.Fatalf("batch upsert failed: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected one affected IPv4 item, got %d", affected)
	}
	if len(store.batchUpsertIn) != 1 || store.batchUpsertIn[0].IP != "192.0.2.10" {
		t.Fatalf("expected only IPv4 host-port asset, got %+v", store.batchUpsertIn)
	}
}

func TestHostPortCommandServiceTargetNotFound(t *testing.T) {
	service := NewHostPortCommandService(&hostPortCommandStoreStub{}, &hostPortCommandTargetLookupStub{err: gorm.ErrRecordNotFound})

	_, err := service.BatchUpsert(context.Background(), 9, []HostPortItem{{Host: "a.example.com", IP: "1.1.1.1", Port: 80}})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestHostPortCommandServiceBatchDeleteByIPsEmpty(t *testing.T) {
	store := &hostPortCommandStoreStub{}
	service := NewHostPortCommandService(store, &hostPortCommandTargetLookupStub{})

	deleted, err := service.BatchDeleteByIPs(context.Background(), nil)
	if err != nil {
		t.Fatalf("batch delete failed: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected deleted count 0, got %d", deleted)
	}
	if store.deleteByIPsCalls != 0 {
		t.Fatalf("expected store delete not to be called, got %d calls", store.deleteByIPsCalls)
	}
}

func TestHostPortCommandServiceBatchUpsertEmptyAndLookupError(t *testing.T) {
	t.Run("empty items", func(t *testing.T) {
		store := &hostPortCommandStoreStub{}
		service := NewHostPortCommandService(store, &hostPortCommandTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}}})

		affected, err := service.BatchUpsert(context.Background(), 1, nil)
		if err != nil {
			t.Fatalf("batch upsert failed: %v", err)
		}
		if affected != 0 {
			t.Fatalf("expected affected count 0, got %d", affected)
		}
		if store.batchUpsertCalls != 0 {
			t.Fatalf("expected store upsert not to be called, got %d calls", store.batchUpsertCalls)
		}
	})

	t.Run("generic lookup error", func(t *testing.T) {
		store := &hostPortCommandStoreStub{}
		service := NewHostPortCommandService(store, &hostPortCommandTargetLookupStub{err: errors.New("lookup failed")})

		if _, err := service.BatchUpsert(context.Background(), 1, []HostPortItem{{Host: "a.example.com", IP: "1.1.1.1", Port: 443}}); err == nil || err.Error() != "lookup failed" {
			t.Fatalf("expected lookup error, got %v", err)
		}
		if store.batchUpsertCalls != 0 {
			t.Fatalf("expected store upsert not to be called, got %d calls", store.batchUpsertCalls)
		}
	})
}
