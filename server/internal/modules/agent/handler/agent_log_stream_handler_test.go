package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseLogStreamQueryDefaults(t *testing.T) {
	ctx := newLogQueryTestContext("/api/admin/agents/1/logs/stream?container=lunafox-agent")

	query, err := parseLogStreamQuery(ctx)
	if err != nil {
		t.Fatalf("parseLogStreamQuery error: %v", err)
	}
	if query.Container != "lunafox-agent" {
		t.Fatalf("unexpected container: %q", query.Container)
	}
	if query.Tail != defaultLogTail {
		t.Fatalf("expected default tail %d, got %d", defaultLogTail, query.Tail)
	}
	if !query.Follow {
		t.Fatalf("expected default follow=true")
	}
	if !query.Timestamps {
		t.Fatalf("expected default timestamps=true")
	}
}

func TestParseLogStreamQueryInvalidContainer(t *testing.T) {
	ctx := newLogQueryTestContext("/api/admin/agents/1/logs/stream?container=../../etc/passwd")

	if _, err := parseLogStreamQuery(ctx); err == nil {
		t.Fatalf("expected container validation error")
	}
}

func TestParseLogStreamQueryTailOutOfRange(t *testing.T) {
	ctx := newLogQueryTestContext("/api/admin/agents/1/logs/stream?container=lunafox-agent&tail=3001")

	if _, err := parseLogStreamQuery(ctx); err == nil {
		t.Fatalf("expected tail range validation error")
	}
}

func TestParseLogStreamQueryFollowAndTimestamps(t *testing.T) {
	ctx := newLogQueryTestContext("/api/admin/agents/1/logs/stream?container=lunafox-agent&follow=false&timestamps=false&tail=50")

	query, err := parseLogStreamQuery(ctx)
	if err != nil {
		t.Fatalf("parseLogStreamQuery error: %v", err)
	}
	if query.Follow {
		t.Fatalf("expected follow=false")
	}
	if query.Timestamps {
		t.Fatalf("expected timestamps=false")
	}
	if query.Tail != 50 {
		t.Fatalf("expected tail=50, got %d", query.Tail)
	}
}

func newLogQueryTestContext(target string) *gin.Context {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", target, nil)
	return ctx
}
