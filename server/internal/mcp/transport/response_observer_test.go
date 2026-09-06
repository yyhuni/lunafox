package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseObserverSuppressesWritesAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	recorder := httptest.NewRecorder()
	observer := &responseObserver{ResponseWriter: recorder, ctx: ctx}

	cancel()
	observer.WriteHeader(http.StatusOK)
	if written, err := observer.Write([]byte("secret response")); !errors.Is(err, context.Canceled) || written != 0 {
		t.Fatalf("cancelled write = (%d, %v), want (0, context.Canceled)", written, err)
	}
	observer.Flush()

	if recorder.Code != 200 && recorder.Code != 0 {
		t.Fatalf("cancelled observer wrote status %d", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("cancelled observer wrote %d response bytes", recorder.Body.Len())
	}
}
