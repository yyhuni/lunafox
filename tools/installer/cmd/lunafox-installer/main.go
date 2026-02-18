package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
	"github.com/yyhuni/lunafox/tools/installer/internal/web"
)

func main() {
	options, err := cli.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "✗ 参数错误: %v\n", err)
		os.Exit(1)
	}

	runner := execx.NewOSRunner()
	server := web.NewServer(options, runner, os.Stdout, os.Stderr)
	if err := server.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Web 安装页面启动失败: %v\n", err)
		os.Exit(1)
	}
}
