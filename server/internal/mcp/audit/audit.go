package audit

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

// Event is the complete allowlist for MCP audit metadata. Payloads, secrets,
// SQL, stacks, and arbitrary error values intentionally have no fields here.
type Event struct {
	RequestID      string
	UserID         int
	KeyRecordID    int
	Method         string
	Tool           string
	ResultCategory string
	ErrorCode      string
	Duration       time.Duration
	ResponseBytes  int
}

// Emitter is deliberately best-effort and side-effect free for MCP results.
type Emitter interface {
	Emit(Event)
}

// Logger emits allowlisted fields through the existing structured Server logger.
type Logger struct{}

// NewLogger creates the default Server logger emitter.
func NewLogger() *Logger { return &Logger{} }

// Emit writes one bounded audit event and never serializes a bearer or payload.
func (*Logger) Emit(event Event) {
	fields := []zap.Field{
		pkg.RequestIDField(event.RequestID),
		zap.String("mcp.method", event.Method),
		zap.String("mcp.result_category", event.ResultCategory),
		zap.Int64("mcp.duration_ms", event.Duration.Milliseconds()),
		zap.Int("mcp.response_bytes", event.ResponseBytes),
	}
	if event.UserID > 0 {
		fields = append(fields, zap.Int("mcp.user_id", event.UserID))
	}
	if event.KeyRecordID > 0 {
		fields = append(fields, zap.Int("mcp.key_record_id", event.KeyRecordID))
	}
	if event.Tool != "" {
		fields = append(fields, zap.String("mcp.tool", event.Tool))
	}
	if event.ErrorCode != "" {
		fields = append(fields, zap.String("mcp.error_code", event.ErrorCode))
	}
	pkg.Info("MCP audit event", fields...)
}
