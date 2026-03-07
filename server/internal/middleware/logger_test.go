package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestLoggerMiddleware_LogsSemanticHTTPFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logs := withObservedLogger(t)
	router := gin.New()
	router.Use(Logger())
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusCreated, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping?q=1", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	entries := logs.FilterMessage("Request completed").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 request log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	for _, key := range []string{
		"request.id",
		"http.request.method",
		"http.response.status_code",
		"url.path",
		"url.query",
		"client.address",
		"user_agent.original",
		"http.server.request.duration_ms",
		"http.response.body.size",
	} {
		if _, ok := ctx[key]; !ok {
			t.Fatalf("expected %s field, got %v", key, ctx)
		}
	}
	for _, key := range []string{
		"request_id",
		"status",
		"method",
		"path",
		"query",
		"ip",
		"user_agent",
		"latency",
		"body_size",
	} {
		if _, ok := ctx[key]; ok {
			t.Fatalf("expected legacy %s field removed, got %v", key, ctx)
		}
	}
}

func TestLoggerMiddleware_UsesRequestIdHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Logger())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Request-Id", "req-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Header().Get("Request-Id"); got != "req-123" {
		t.Fatalf("expected Request-Id response header req-123, got %q", got)
	}
	if got := w.Header().Get("X-Request-ID"); got != "" {
		t.Fatalf("expected legacy X-Request-ID header removed, got %q", got)
	}
}

func TestLoggerMiddleware_GeneratesRequestIdHeaderWhenAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Logger())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Header().Get("Request-Id"); got == "" {
		t.Fatal("expected Request-Id response header to be generated")
	}
	if got := w.Header().Get("X-Request-ID"); got != "" {
		t.Fatalf("expected legacy X-Request-ID header removed, got %q", got)
	}
}

func TestRecoveryMiddleware_LogsSemanticRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logs := withObservedLogger(t)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		setRequestID(c, "req-500")
		c.Next()
	})
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	entries := logs.FilterMessage("Panic recovered").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 recovery log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	if _, ok := ctx["request.id"]; !ok {
		t.Fatalf("expected request.id field, got %v", ctx)
	}
	if _, ok := ctx["request_id"]; ok {
		t.Fatalf("expected legacy request_id field removed, got %v", ctx)
	}
}

func TestRecoveryMiddleware_ReturnsCamelCaseRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		setRequestID(c, "req-500")
		c.Next()
	})
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal recovery body: %v", err)
	}
	if _, ok := body["requestId"]; !ok {
		t.Fatalf("expected requestId field, got %v", body)
	}
	if _, ok := body["request_id"]; ok {
		t.Fatalf("expected legacy request_id field removed, got %v", body)
	}
}
