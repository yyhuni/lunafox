package application

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

const (
	taskRuntimeLogWriteTimeout = 2 * time.Second
	taskRuntimeLogContentBytes = 16 * 1024
	taskRuntimeLogBatchEntries = 100
)

// TaskRuntimeLogEvent is a Server event whose task ownership has already been
// established by its application or transport boundary.
type TaskRuntimeLogEvent struct {
	ScanID  int
	TaskID  int
	EventID string
	Level   string
	Content string
}

// TaskRuntimeLogAppender adds task evidence without changing a business result.
// Callers must establish task ownership before invoking this internal boundary.
type TaskRuntimeLogAppender interface {
	AppendTaskRuntimeLog(context.Context, TaskRuntimeLogEvent)
	AppendTaskRuntimeLogs(context.Context, []TaskRuntimeLogEvent)
}

type taskRuntimeLogStore interface {
	BatchCreateTaskRuntimeLogs(context.Context, *TaskProgressLogBatchCreateRequest) (int, int, error)
}

type taskRuntimeLogAppender struct {
	store taskRuntimeLogStore
	now   func() time.Time
}

// NewTaskRuntimeLogAppender reuses the existing task-log persistence boundary.
func NewTaskRuntimeLogAppender(store taskRuntimeLogStore) TaskRuntimeLogAppender {
	if store == nil {
		panic("task runtime log store is required")
	}
	return &taskRuntimeLogAppender{store: store, now: time.Now}
}

func (appender *taskRuntimeLogAppender) AppendTaskRuntimeLog(ctx context.Context, event TaskRuntimeLogEvent) {
	if ctx == nil {
		appender.logFailure(event, errors.New("task runtime log context is required"))
		return
	}
	// A single completed operation retains its evidence even when its request
	// was cancelled. Bulk callers provide one independent budget for the batch.
	writeCtx, cancel := context.WithTimeout(context.Background(), taskRuntimeLogWriteTimeout)
	defer cancel()
	appender.AppendTaskRuntimeLogs(writeCtx, []TaskRuntimeLogEvent{event})
}

// AppendTaskRuntimeLogs honors one caller-owned evidence budget for the entire
// operation. A failed write stops the batch so an unavailable database cannot
// add one timeout per cancelled task or superseded lease.
func (appender *taskRuntimeLogAppender) AppendTaskRuntimeLogs(ctx context.Context, events []TaskRuntimeLogEvent) {
	if ctx == nil {
		appender.logFailure(TaskRuntimeLogEvent{}, errors.New("task runtime log context is required"))
		return
	}
	writeCtx, cancel := context.WithTimeout(ctx, taskRuntimeLogWriteTimeout)
	defer cancel()
	for _, event := range events {
		if err := writeCtx.Err(); err != nil {
			appender.logFailure(event, err)
			return
		}
		if !appender.appendTaskRuntimeLog(writeCtx, event) {
			return
		}
	}
}

func (appender *taskRuntimeLogAppender) appendTaskRuntimeLog(writeCtx context.Context, event TaskRuntimeLogEvent) bool {
	if event.ScanID <= 0 || event.TaskID <= 0 || strings.TrimSpace(event.EventID) == "" ||
		event.Content == "" ||
		(event.Level != "info" && event.Level != "warning" && event.Level != "error") {
		appender.logFailure(event, errors.New("invalid task runtime log event"))
		return false
	}
	if !utf8.ValidString(event.Content) || strings.IndexByte(event.Content, 0) >= 0 {
		// PostgreSQL text cannot retain invalid UTF-8 or NUL. Keep every original
		// byte reversible instead of replacing it or losing the whole event.
		event.Content = "[binary/base64] " + base64.StdEncoding.EncodeToString([]byte(event.Content))
	}
	hash := sha256.New()
	for _, value := range []string{event.EventID, event.Level, event.Content} {
		_, _ = io.WriteString(hash, value)
		_, _ = hash.Write([]byte{0})
	}
	requestID := "server-runtime-" + hex.EncodeToString(hash.Sum(nil))
	emittedAt := appender.now().UTC()
	remaining := event.Content
	sequence := int64(1)
	for remaining != "" {
		items := make([]TaskProgressLogCreateItem, 0, taskRuntimeLogBatchEntries)
		for remaining != "" && len(items) < taskRuntimeLogBatchEntries {
			end := len(remaining)
			if end > taskRuntimeLogContentBytes {
				end = taskRuntimeLogContentBytes
				for !utf8.RuneStart(remaining[end]) {
					end--
				}
			}
			items = append(items, TaskProgressLogCreateItem{
				Sequence: sequence, Level: event.Level, Content: remaining[:end], EmittedAt: emittedAt,
			})
			remaining = remaining[end:]
			sequence++
		}
		accepted, duplicates, err := appender.store.BatchCreateTaskRuntimeLogs(writeCtx, &TaskProgressLogBatchCreateRequest{
			ScanID: event.ScanID, TaskID: event.TaskID, RequestID: requestID, Items: items,
		})
		if err == nil && (accepted < 0 || duplicates < 0 || accepted+duplicates != len(items)) {
			err = errors.New("task runtime log acknowledgement is incomplete")
		}
		if err != nil {
			appender.logFailure(event, err)
			return false
		}
	}
	return true
}

func (appender *taskRuntimeLogAppender) logFailure(event TaskRuntimeLogEvent, err error) {
	pkg.Error("task runtime log persistence failed",
		zap.Int("scan.id", event.ScanID), zap.Int("task.id", event.TaskID),
		zap.String("event.id", event.EventID), zap.Error(err),
	)
}
