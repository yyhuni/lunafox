package domain

import "testing"

func TestSyncTaskCancellationStatesHaveExpectedLifecycleSemantics(t *testing.T) {
	if !SyncTaskCancelling.Valid() {
		t.Fatal("CANCELLING must be a valid task state")
	}
	if SyncTaskCancelling.Terminal() {
		t.Fatal("CANCELLING must remain active until cleanup completes")
	}
	if !SyncTaskCancelled.Valid() {
		t.Fatal("CANCELLED must be a valid task state")
	}
	if !SyncTaskCancelled.Terminal() {
		t.Fatal("CANCELLED must release the active task slot")
	}
}
