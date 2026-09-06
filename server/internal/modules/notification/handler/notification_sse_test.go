package handler

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/middleware"
)

type notificationSSEBrokerStub struct {
	updates chan struct{}
}

func (stub *notificationSSEBrokerStub) Subscribe(int) (<-chan struct{}, func()) {
	return stub.updates, func() {}
}

type notificationSSETokenVersionReaderStub struct{}

func (notificationSSETokenVersionReaderStub) GetTokenVersion(context.Context, int) (int, error) {
	return 1, nil
}

func TestNotificationSSEHandlerFlushesIdleHeartbeatWithoutRefresh(t *testing.T) {
	handler := NewNotificationSSEHandler(&notificationSSEBrokerStub{updates: make(chan struct{})})
	handler.heartbeatInterval = 10 * time.Millisecond
	server, token := notificationSSETestServer(t, handler, time.Minute)
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, server.URL+"/stream", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("open notification stream: %v", err)
	}
	defer response.Body.Close()

	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read heartbeat before stream close: %v", err)
	}
	if line != ": heartbeat\n" {
		t.Fatalf("heartbeat line = %q", line)
	}
	blank, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read heartbeat delimiter: %v", err)
	}
	if blank != "\n" {
		t.Fatalf("heartbeat delimiter = %q", blank)
	}
}

func TestNotificationSSEHandlerEndsStreamAtJWTExpiry(t *testing.T) {
	handler := NewNotificationSSEHandler(&notificationSSEBrokerStub{updates: make(chan struct{})})
	server, token := notificationSSETestServer(t, handler, 1500*time.Millisecond)
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, server.URL+"/stream", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("open notification stream: %v", err)
	}
	defer response.Body.Close()

	done := make(chan error, 1)
	go func() {
		_, readErr := io.Copy(io.Discard, response.Body)
		done <- readErr
	}()
	select {
	case readErr := <-done:
		if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) {
			t.Fatalf("read stream through JWT expiry: %v", readErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("notification stream did not end at JWT expiry")
	}
}

func notificationSSETestServer(t *testing.T, streamHandler *NotificationSSEHandler, accessExpire time.Duration) (*httptest.Server, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	manager := auth.NewJWTManager("test-notification-sse-secret-32!", accessExpire, time.Hour)
	token, _, err := manager.GenerateAccessToken(7, "operator", 1)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	engine := gin.New()
	engine.Use(middleware.AuthMiddleware(manager, notificationSSETokenVersionReaderStub{}))
	engine.GET("/stream", streamHandler.Stream)
	return httptest.NewServer(engine), token
}
