package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type hostPortCommandStoreStub struct {
	bulkUpsertIn []assetdomain.HostPort
	deletedIPs   []string
	upsertErr    error
	deleteErr    error
}

func (stub *hostPortCommandStoreStub) BulkUpsert(items []assetdomain.HostPort) (int64, error) {
	if stub.upsertErr != nil {
		return 0, stub.upsertErr
	}
	stub.bulkUpsertIn = append([]assetdomain.HostPort(nil), items...)
	return int64(len(items)), nil
}

func (stub *hostPortCommandStoreStub) DeleteByIPs(ips []string) (int64, error) {
	if stub.deleteErr != nil {
		return 0, stub.deleteErr
	}
	stub.deletedIPs = append([]string(nil), ips...)
	return int64(len(ips)), nil
}

type hostPortCommandTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
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

func TestHostPortCommandServiceBulkUpsertAndDelete(t *testing.T) {
	store := &hostPortCommandStoreStub{}
	lookup := &hostPortCommandTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}}}
	service := NewHostPortCommandService(store, lookup)

	affected, err := service.BulkUpsert(context.Background(), 1, []HostPortItem{{Host: "a.example.com", IP: "1.1.1.1", Port: 443}})
	if err != nil {
		t.Fatalf("bulk upsert failed: %v", err)
	}
	if affected != 1 || len(store.bulkUpsertIn) != 1 {
		t.Fatalf("unexpected upsert result affected=%d size=%d", affected, len(store.bulkUpsertIn))
	}

	deleted, err := service.BulkDeleteByIPs(context.Background(), []string{"1.1.1.1", "2.2.2.2"})
	if err != nil {
		t.Fatalf("bulk delete failed: %v", err)
	}
	if deleted != 2 || len(store.deletedIPs) != 2 {
		t.Fatalf("unexpected delete result deleted=%d ips=%v", deleted, store.deletedIPs)
	}
}

func TestHostPortCommandServiceTargetNotFound(t *testing.T) {
	service := NewHostPortCommandService(&hostPortCommandStoreStub{}, &hostPortCommandTargetLookupStub{err: gorm.ErrRecordNotFound})

	_, err := service.BulkUpsert(context.Background(), 9, []HostPortItem{{Host: "a.example.com", IP: "1.1.1.1", Port: 80}})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}
