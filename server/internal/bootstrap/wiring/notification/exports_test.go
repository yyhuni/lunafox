package notificationwiring

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewNotificationModuleFailsBeforeWorkerConstructionWhenReconciliationCannotRun(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	module, err := NewNotificationModule(db, "notification-test-owner")
	if err == nil {
		t.Fatal("NewNotificationModule() error = nil, want reconciliation failure")
	}
	if module != nil {
		t.Fatalf("NewNotificationModule() module = %#v, want nil before workers are constructed", module)
	}
}
