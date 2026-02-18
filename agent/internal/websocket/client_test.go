package websocket

import (
	"context"
	"testing"
	"time"
)

func TestClientSendBufferFull(t *testing.T) {
	client := &Client{send: make(chan []byte, 1)}
	if !client.Send([]byte("first")) {
		t.Fatalf("expected first send to succeed")
	}
	if client.Send([]byte("second")) {
		t.Fatalf("expected second send to fail when buffer is full")
	}
}

func TestSleepWithContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if sleepWithContext(ctx, 50*time.Millisecond) {
		t.Fatalf("expected sleepWithContext to return false when canceled")
	}
}

func TestSleepWithContextElapsed(t *testing.T) {
	if !sleepWithContext(context.Background(), 5*time.Millisecond) {
		t.Fatalf("expected sleepWithContext to return true after delay")
	}
}
