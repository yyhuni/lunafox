package steps

import (
	"context"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/docker"
)

type stepSystem struct{}

func (stepSystem) Title() string {
	return "系统环境校验"
}

func (stepSystem) Run(ctx context.Context, installer *Installer) error {
	toolchain, err := docker.Detect(ctx, installer.runner)
	if err != nil {
		return err
	}
	installer.toolchain = toolchain
	installer.printer.Info("使用 compose 命令: %s", strings.Join(toolchain.ComposeCmd, " "))
	return nil
}
