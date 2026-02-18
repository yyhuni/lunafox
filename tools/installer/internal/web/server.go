package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
	"github.com/yyhuni/lunafox/tools/installer/internal/installapp"
)

type Server struct {
	baseOptions cli.Options
	out         io.Writer

	httpServer *http.Server
}

func NewServer(baseOptions cli.Options, runner execx.Runner, out io.Writer, err io.Writer) *Server {
	installService := installapp.NewService(runner, out, err)
	router := NewRouter(baseOptions, installService)

	return &Server{
		baseOptions: baseOptions,
		out:         out,
		httpServer: &http.Server{
			Addr:    listenAddress(baseOptions.ListenAddr),
			Handler: router.Handler(),
		},
	}
}

func (server *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.httpServer.Shutdown(shutdownCtx)
	}()

	fmt.Fprintf(server.out, "ℹ  Web 安装页面已启动: http://%s\n", server.httpServer.Addr)
	fmt.Fprintf(server.out, "ℹ  打开浏览器后点击“开始安装”即可执行\n")

	err := server.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func listenAddress(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return cli.DefaultListenAddr
	}
	return trimmed
}
