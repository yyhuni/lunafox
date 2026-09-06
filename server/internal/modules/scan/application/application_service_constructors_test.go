package application

import "testing"

func TestScanApplicationConstructorsExist(t *testing.T) {
	if NewTaskProgressLogApplicationService(nil, nil, nil) == nil {
		t.Fatalf("expected task progress log application service")
	}
}
