package web

import (
	"net/http"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/installapp"
)

type InstallService interface {
	Start(options cli.Options) (string, error)
	Snapshot(jobID string) (installapp.InstallStateSnapshot, error)
	Subscribe(jobID string, afterID int64) (<-chan installapp.InstallEvent, func(), error)
}

type Router struct {
	baseOptions cli.Options
	service     InstallService
}

func NewRouter(baseOptions cli.Options, service InstallService) *Router {
	return &Router{
		baseOptions: baseOptions,
		service:     service,
	}
}

func (router *Router) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", router.handleIndex)
	mux.HandleFunc("/api/start", router.handleStart)
	mux.HandleFunc("/api/state", router.handleState)
	mux.HandleFunc("/api/events", router.handleEvents)
	mux.HandleFunc("/api/network/reachability", router.handleNetworkReachability)
	return mux
}
