package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/installapp"
)

func (router *Router) handleEvents(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeAPIError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "仅支持 GET", nil)
		return
	}

	jobID := strings.TrimSpace(request.URL.Query().Get("jobId"))
	if jobID == "" {
		writeAPIError(writer, http.StatusBadRequest, "JOB_ID_REQUIRED", "缺少 jobId", nil)
		return
	}

	lastEventID := parseLastEventID(request)
	events, cancel, err := router.service.Subscribe(jobID, lastEventID)
	if err != nil {
		if installapp.IsJobNotFound(err) {
			writeAPIError(writer, http.StatusNotFound, "JOB_NOT_FOUND", "安装任务不存在", map[string]any{"jobId": jobID})
			return
		}
		writeAPIError(writer, http.StatusInternalServerError, "INTERNAL", "订阅事件失败", nil)
		return
	}
	defer cancel()

	flusher, ok := writer.(http.Flusher)
	if !ok {
		writeAPIError(writer, http.StatusInternalServerError, "SSE_UNSUPPORTED", "当前环境不支持 SSE", nil)
		return
	}

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-request.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeSSE(writer, event); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSE(writer http.ResponseWriter, event installapp.InstallEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "id: %d\n", event.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "event: %s\n", event.Type); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", body); err != nil {
		return err
	}
	return nil
}

func parseLastEventID(request *http.Request) int64 {
	raw := strings.TrimSpace(request.Header.Get("Last-Event-ID"))
	if raw == "" {
		raw = strings.TrimSpace(request.URL.Query().Get("lastEventId"))
	}
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}
