package websocket

import (
	"testing"
	"time"
)

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	timeout := time.After(200 * time.Millisecond)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("timeout waiting for condition")
		case <-ticker.C:
			if cond() {
				return
			}
		}
	}
}

func TestHubRegisterSendTo(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		AgentID: 1,
		Send:    make(chan []byte, 1),
		Hub:     hub,
	}
	hub.Register(client)
	waitFor(t, func() bool { return hub.IsConnected(1) })

	if ok := hub.SendToWithResult(1, []byte("ping")); !ok {
		t.Fatalf("expected send to succeed")
	}

	select {
	case msg := <-client.Send:
		if string(msg) != "ping" {
			t.Fatalf("unexpected message: %s", string(msg))
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected message to be delivered")
	}

	hub.Unregister(client)
	waitFor(t, func() bool { return !hub.IsConnected(1) })

	select {
	case _, ok := <-client.Send:
		if ok {
			t.Fatalf("expected channel to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected channel close")
	}
}

func TestHubSendToMissingClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	if ok := hub.SendToWithResult(99, []byte("noop")); ok {
		t.Fatalf("expected send to fail")
	}
}

func TestHubSendToDropsOnFullChannel(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		AgentID: 2,
		Send:    make(chan []byte, 1),
		Hub:     hub,
	}
	hub.Register(client)
	waitFor(t, func() bool { return hub.IsConnected(2) })

	client.Send <- []byte("full")
	if ok := hub.SendToWithResult(2, []byte("next")); ok {
		t.Fatalf("expected send to fail due to full channel")
	}
	if hub.IsConnected(2) {
		t.Fatalf("expected client to be dropped")
	}

	msg, ok := <-client.Send
	if !ok || string(msg) != "full" {
		t.Fatalf("expected buffered message before close")
	}
	if _, ok := <-client.Send; ok {
		t.Fatalf("expected channel to be closed")
	}
}
