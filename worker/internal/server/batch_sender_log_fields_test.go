package server

import (
	"context"
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func withObservedLogger(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zap.DebugLevel)
	prevLogger := pkg.Logger
	pkg.Logger = zap.New(core)
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})
	return logs
}

func TestBatchSenderQueueOverflowLogsSemanticFields(t *testing.T) {
	logs := withObservedLogger(t)
	client := &captureClient{
		failCount: 1,
		err:       &HTTPError{StatusCode: 500, Body: "server error"},
	}
	sender := NewBatchSenderWithQueueLimit(context.Background(), client, 1, 2, "subdomain", 2, 2)

	if err := sender.Add("a"); err != nil {
		t.Fatalf("add first item: %v", err)
	}
	if err := sender.Add("b"); err == nil {
		t.Fatalf("expected second add to fail due to send error")
	}
	if err := sender.Add("c"); err == nil {
		t.Fatalf("expected third add to fail due to queue overflow")
	}

	entries := logs.FilterMessage("Batch queue limit exceeded; dropping new item").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 queue overflow log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	for _, key := range []string{"queue.length", "queue.max_items", "drop.total"} {
		if _, ok := ctx[key]; !ok {
			t.Fatalf("expected %s field, got %v", key, ctx)
		}
	}
	for _, key := range []string{"queueLength", "maxQueuedItems", "droppedTotal"} {
		if _, ok := ctx[key]; ok {
			t.Fatalf("expected legacy field %s removed, got %v", key, ctx)
		}
	}
}

func TestBatchSenderFailureAndSuccessLogsSemanticFields(t *testing.T) {
	logs := withObservedLogger(t)
	nonRetryable := &captureClient{failCount: 1, err: &HTTPError{StatusCode: 400, Body: "bad request"}}
	sender := NewBatchSender(context.Background(), nonRetryable, 1, 2, "subdomain", 2)

	if err := sender.Add("a"); err != nil {
		t.Fatalf("add first item: %v", err)
	}
	if err := sender.Add("b"); err == nil {
		t.Fatalf("expected second add to fail due to non-retryable error")
	}

	failureEntries := logs.FilterMessage("Non-retryable error sending batch (data validation issue)").All()
	if len(failureEntries) != 1 {
		t.Fatalf("expected 1 non-retryable failure log, got %d", len(failureEntries))
	}
	failureCtx := failureEntries[0].ContextMap()
	if _, ok := failureCtx["http.response.status_code"]; !ok {
		t.Fatalf("expected http.response.status_code field, got %v", failureCtx)
	}
	if _, ok := failureCtx["statusCode"]; ok {
		t.Fatalf("expected legacy statusCode removed, got %v", failureCtx)
	}

	logs.TakeAll()
	successClient := &captureClient{}
	successSender := NewBatchSender(context.Background(), successClient, 1, 2, "subdomain", 2)
	if err := successSender.Add("a"); err != nil {
		t.Fatalf("success add first item: %v", err)
	}
	if err := successSender.Add("b"); err != nil {
		t.Fatalf("success add second item: %v", err)
	}

	successEntries := logs.FilterMessage("Batch sent").All()
	if len(successEntries) != 1 {
		t.Fatalf("expected 1 success log, got %d", len(successEntries))
	}
	successCtx := successEntries[0].ContextMap()
	for _, key := range []string{"send.item_total", "send.batch_total"} {
		if _, ok := successCtx[key]; !ok {
			t.Fatalf("expected %s field, got %v", key, successCtx)
		}
	}
	for _, key := range []string{"totalSent", "totalBatches"} {
		if _, ok := successCtx[key]; ok {
			t.Fatalf("expected legacy field %s removed, got %v", key, successCtx)
		}
	}
}
