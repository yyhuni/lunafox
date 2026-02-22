package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/docker"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	"github.com/yyhuni/lunafox/agent/internal/protocol"
	"go.uber.org/zap"
)

const (
	logErrorDockerUnavailable = "docker_unavailable"
	logErrorContainerNotFound = "container_not_found"
	logErrorDockerAPI         = "docker_api_error"
)

var errLogWSUnavailable = errors.New("websocket send failed")

type wsMessageSender interface {
	Send(payload []byte) bool
}

type logStreamManager struct {
	dockerClient *docker.Client
	sender       wsMessageSender

	mu       sync.Mutex
	sessions map[string]context.CancelFunc
}

func newLogStreamManager(dockerClient *docker.Client, sender wsMessageSender) *logStreamManager {
	return &logStreamManager{
		dockerClient: dockerClient,
		sender:       sender,
		sessions:     make(map[string]context.CancelFunc),
	}
}

func (manager *logStreamManager) Close() {
	if manager == nil {
		return
	}
	manager.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(manager.sessions))
	for requestID, cancel := range manager.sessions {
		delete(manager.sessions, requestID)
		cancels = append(cancels, cancel)
	}
	manager.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

func (manager *logStreamManager) HandleOpen(payload protocol.LogOpenPayload) {
	requestID := strings.TrimSpace(payload.RequestID)
	containerName := strings.TrimSpace(payload.Container)
	if requestID == "" || containerName == "" {
		return
	}
	if manager.dockerClient == nil {
		manager.sendLogError(requestID, logErrorDockerUnavailable, "docker client is unavailable")
		manager.sendLogEnd(requestID, "error")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	manager.mu.Lock()
	if prev, ok := manager.sessions[requestID]; ok {
		prev()
	}
	manager.sessions[requestID] = cancel
	manager.mu.Unlock()

	go manager.runStream(ctx, payload)
}

func (manager *logStreamManager) HandleCancel(payload protocol.LogCancelPayload) {
	requestID := strings.TrimSpace(payload.RequestID)
	if requestID == "" {
		return
	}

	manager.mu.Lock()
	cancel, ok := manager.sessions[requestID]
	if ok {
		delete(manager.sessions, requestID)
	}
	manager.mu.Unlock()

	if ok {
		cancel()
	}
}

func (manager *logStreamManager) runStream(ctx context.Context, payload protocol.LogOpenPayload) {
	requestID := strings.TrimSpace(payload.RequestID)
	defer manager.removeSession(requestID)

	if !manager.sendMessage(protocol.MessageTypeLogStarted, protocol.LogStartedPayload{RequestID: requestID}) {
		return
	}

	err := manager.dockerClient.StreamLogs(ctx, docker.StreamLogsOptions{
		Container:  payload.Container,
		Tail:       payload.Tail,
		Follow:     payload.Follow,
		Since:      payload.Since,
		Timestamps: payload.Timestamps,
	}, func(chunk docker.StreamLogChunk) error {
		ok := manager.sendMessage(protocol.MessageTypeLogChunk, protocol.LogChunkPayload{
			RequestID: requestID,
			TS:        chunk.TS.UTC(),
			Stream:    chunk.Stream,
			Line:      chunk.Line,
			Truncated: chunk.Truncated,
		})
		if ok {
			return nil
		}
		return errLogWSUnavailable
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			manager.sendLogEnd(requestID, "cancelled")
			return
		}
		if errors.Is(err, errLogWSUnavailable) {
			return
		}

		code := logErrorDockerAPI
		if docker.IsContainerNotFoundError(err) {
			code = logErrorContainerNotFound
		}
		message := docker.TruncateErrorMessage(strings.TrimSpace(err.Error()))
		if message == "" {
			message = "failed to stream container logs"
		}
		manager.sendLogError(requestID, code, message)
		manager.sendLogEnd(requestID, "error")
		return
	}

	manager.sendLogEnd(requestID, "eof")
}

func (manager *logStreamManager) removeSession(requestID string) {
	if manager == nil || requestID == "" {
		return
	}
	manager.mu.Lock()
	delete(manager.sessions, requestID)
	manager.mu.Unlock()
}

func (manager *logStreamManager) sendLogError(requestID, code, message string) {
	manager.sendMessage(protocol.MessageTypeLogError, protocol.LogErrorPayload{
		RequestID: requestID,
		Code:      code,
		Message:   message,
	})
}

func (manager *logStreamManager) sendLogEnd(requestID, reason string) {
	manager.sendMessage(protocol.MessageTypeLogEnd, protocol.LogEndPayload{
		RequestID: requestID,
		Reason:    reason,
	})
}

func (manager *logStreamManager) sendMessage(messageType string, payload any) bool {
	if manager == nil || manager.sender == nil {
		return false
	}
	message := protocol.Message{
		Type:      messageType,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}
	encoded, err := json.Marshal(message)
	if err != nil {
		logger.Log.Warn("failed to marshal websocket message",
			zap.String("type", messageType),
			zap.Error(err),
		)
		return false
	}
	return manager.sender.Send(encoded)
}
