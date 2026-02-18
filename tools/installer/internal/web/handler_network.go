package web

import (
	"net/http"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

func (router *Router) handleNetworkReachability(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeAPIError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "仅支持 GET", nil)
		return
	}

	hostInput := strings.TrimSpace(request.URL.Query().Get("host"))
	portInput := strings.TrimSpace(request.URL.Query().Get("port"))

	host, err := parsePublicHostInput(hostInput)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "INVALID_HOST", "公网主机格式不合法", map[string]any{"cause": err.Error()})
		return
	}
	if host == "" {
		writeAPIError(writer, http.StatusBadRequest, "INVALID_HOST", "公网主机格式不合法", map[string]any{"cause": "必须填写公网主机（IP 或域名）"})
		return
	}
	if err := validatePort(portInput); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "INVALID_PORT", "公网端口格式不合法", map[string]any{"cause": err.Error()})
		return
	}

	publicURL := buildPublicURL(host, portInput)
	if router.baseOptions.Mode == cli.ModeProd && isLoopbackHost(host) {
		writeJSON(writer, http.StatusOK, networkReachabilityResponse{
			OK:        false,
			Level:     "error",
			Message:   "生产模式不能使用 localhost/127.0.0.1，请改为服务器 IP 或域名",
			PublicURL: publicURL,
		})
		return
	}

	candidates, _ := listNetworkCandidates()
	local, lookupErr := isLocalHost(host, candidates)
	if local {
		if !isTCPPortAvailable(portInput) {
			writeJSON(writer, http.StatusOK, networkReachabilityResponse{
				OK:        false,
				Level:     "error",
				Message:   "端口可能已被占用，请更换端口后重试",
				PublicURL: publicURL,
			})
			return
		}
		writeJSON(writer, http.StatusOK, networkReachabilityResponse{
			OK:        true,
			Level:     "success",
			Message:   "地址与端口预检查通过",
			PublicURL: publicURL,
		})
		return
	}

	message := "该地址不是本机网卡地址，无法在安装器本地完全预检，将在安装步骤继续校验"
	if lookupErr != nil {
		message = "域名暂时无法解析，已跳过本地预检，将在安装步骤继续校验"
	}
	writeJSON(writer, http.StatusOK, networkReachabilityResponse{
		OK:        true,
		Level:     "warning",
		Message:   message,
		PublicURL: publicURL,
	})
}
