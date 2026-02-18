package web

import (
	"net/http"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/installapp"
)

func (router *Router) handleState(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeAPIError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "仅支持 GET", nil)
		return
	}

	jobID := strings.TrimSpace(request.URL.Query().Get("jobId"))
	if jobID == "" {
		writeAPIError(writer, http.StatusBadRequest, "JOB_ID_REQUIRED", "缺少 jobId", nil)
		return
	}

	snapshot, err := router.service.Snapshot(jobID)
	if err != nil {
		if installapp.IsJobNotFound(err) {
			writeAPIError(writer, http.StatusNotFound, "JOB_NOT_FOUND", "安装任务不存在", map[string]any{"jobId": jobID})
			return
		}
		writeAPIError(writer, http.StatusInternalServerError, "INTERNAL", "读取任务状态失败", nil)
		return
	}

	writeJSON(writer, http.StatusOK, snapshot)
}
