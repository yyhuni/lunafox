package web

import (
	"encoding/json"
	"net/http"

	"github.com/yyhuni/lunafox/tools/installer/internal/installapp"
)

func (router *Router) handleStart(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeAPIError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "仅支持 POST", nil)
		return
	}

	var req startRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "BAD_REQUEST", "请求体解析失败", nil)
		return
	}

	options, err := buildInstallOptions(router.baseOptions, req)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "INVALID_OPTIONS", err.Error(), nil)
		return
	}

	jobID, err := router.service.Start(options)
	if err != nil {
		if runningJobID, ok := installapp.IsJobRunning(err); ok {
			writeAPIError(writer, http.StatusConflict, "JOB_ALREADY_RUNNING", "已有安装任务正在运行", map[string]any{"jobId": runningJobID})
			return
		}
		writeAPIError(writer, http.StatusInternalServerError, "INTERNAL", "安装任务启动失败", nil)
		return
	}

	writeJSON(writer, http.StatusAccepted, startResponse{
		JobID: jobID,
		State: string(installapp.StateRunning),
	})
}
