package application

import (
	"context"
	"testing"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	pkglogger "github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func withObservedLogger(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	previousLogger := pkglogger.Logger
	previousSugar := pkglogger.Sugar
	pkglogger.Logger = logger
	pkglogger.Sugar = logger.Sugar()
	t.Cleanup(func() {
		pkglogger.Logger = previousLogger
		pkglogger.Sugar = previousSugar
	})
	return logs
}

func TestLifecycleServiceStopActiveForDelete_LogsSemanticFields(t *testing.T) {
	logs := withObservedLogger(t)
	store := &lifecycleScanStoreStub{}
	service := NewLifecycleService(store, &lifecycleScanStopStoreStub{err: scandomain.ErrScanCannotStop}, nil)

	_, err := service.stopActiveForDelete(context.Background(), &QueryScan{ID: 1001, Status: string(scandomain.ScanStatusRunning)})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	entries := logs.FilterMessage("ignoring stop error during scan deletion").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 lifecycle warning log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	if _, ok := ctx["scan.id"]; !ok {
		t.Fatalf("expected scan.id field, got %v", ctx)
	}
	if _, ok := ctx["scan.status"]; !ok {
		t.Fatalf("expected scan.status field, got %v", ctx)
	}
	if _, ok := ctx["scan_id"]; ok {
		t.Fatalf("expected legacy scan_id field removed, got %v", ctx)
	}
	if _, ok := ctx["scan_status"]; ok {
		t.Fatalf("expected legacy scan_status field removed, got %v", ctx)
	}
}
