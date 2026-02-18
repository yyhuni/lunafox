package steps

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/buildinfo"
	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/docker"
	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
	"github.com/yyhuni/lunafox/tools/installer/internal/ui"
)

const (
	TotalSteps    = 7
	sslVolumeName = "lunafox_ssl"
)

type Installer struct {
	options   cli.Options
	runner    execx.Runner
	printer   *ui.Printer
	toolchain docker.Toolchain
	version   string
	tlsConfig *tls.Config
}

type Step interface {
	Title() string
	Run(ctx context.Context, installer *Installer) error
}

func NewInstaller(options cli.Options, runner execx.Runner, printer *ui.Printer) *Installer {
	return &Installer{options: options, runner: runner, printer: printer}
}

func (installer *Installer) Run(ctx context.Context) error {
	if err := installer.ensureProjectStructure(); err != nil {
		return err
	}
	if err := installer.resolveVersion(); err != nil {
		return err
	}
	installer.printPathContext()

	installer.printer.Banner(installer.options.Mode, installer.version)

	steps := []Step{
		stepSystem{},
		stepTLS{},
		stepImages{},
		stepEnv{},
		stepCompose{},
		stepHealth{},
		stepAgent{},
	}
	for idx, step := range steps {
		installer.printer.Step(idx+1, len(steps), step.Title())
		if err := step.Run(ctx, installer); err != nil {
			return err
		}
	}

	installer.printer.Summary(installer.version, installer.options.ComposeFile, installer.options.PublicURL, strings.Join(installer.toolchain.ComposeCmd, " "))
	return nil
}

func (installer *Installer) printPathContext() {
	installer.printer.Info("ROOT_DIR: %s", installer.options.RootDir)
	installer.printer.Info("DOCKER_DIR: %s", installer.options.DockerDir)
	installer.printer.Info("ENV_FILE: %s", installer.options.EnvFile)
}

func (installer *Installer) ensureProjectStructure() error {
	if stat, err := os.Stat(installer.options.DockerDir); err != nil || !stat.IsDir() {
		return fmt.Errorf("未找到 docker 目录: %s", installer.options.DockerDir)
	}
	if _, err := os.Stat(installer.options.ComposeProd); err != nil {
		return fmt.Errorf("未找到 compose 文件: %s", installer.options.ComposeProd)
	}
	if _, err := os.Stat(installer.options.ComposeDev); err != nil {
		return fmt.Errorf("未找到 compose 文件: %s", installer.options.ComposeDev)
	}
	if _, err := os.Stat(installer.options.ComposeFile); err != nil {
		return fmt.Errorf("未找到当前模式 compose 文件: %s", installer.options.ComposeFile)
	}
	if err := ensureEnvPathWritable(installer.options.EnvFile); err != nil {
		return err
	}
	if err := ensureSSLDirReady(installer.options.DockerDir); err != nil {
		return err
	}
	return nil
}

func ensureEnvPathWritable(envFile string) error {
	envDir := filepath.Dir(strings.TrimSpace(envFile))
	if envDir == "" {
		return fmt.Errorf("ENV_FILE 路径无效: %s", envFile)
	}
	if stat, err := os.Stat(envDir); err != nil || !stat.IsDir() {
		return fmt.Errorf("未找到 .env 所在目录: %s", envDir)
	}
	if _, err := os.Stat(envFile); err == nil {
		file, openErr := os.OpenFile(envFile, os.O_WRONLY|os.O_APPEND, 0o644)
		if openErr != nil {
			return fmt.Errorf(".env 不可写: %s", envFile)
		}
		_ = file.Close()
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取 .env 状态失败: %w", err)
	}
	if err := checkDirWritable(envDir, ".env 所在目录"); err != nil {
		return err
	}
	return nil
}

func ensureSSLDirReady(dockerDir string) error {
	sslPath := sslDir(dockerDir)
	if stat, err := os.Stat(sslPath); err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("证书目录路径不是目录: %s", sslPath)
		}
		if _, readErr := os.ReadDir(sslPath); readErr != nil {
			return fmt.Errorf("证书目录不可访问: %s", sslPath)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取证书目录失败: %w", err)
	}

	parent := filepath.Dir(sslPath)
	if stat, err := os.Stat(parent); err != nil || !stat.IsDir() {
		return fmt.Errorf("未找到证书目录父目录: %s", parent)
	}
	if err := checkDirWritable(parent, "证书目录父目录"); err != nil {
		return err
	}
	return nil
}

func checkDirWritable(dir string, label string) error {
	file, err := os.CreateTemp(dir, ".lunafox-write-check-*")
	if err != nil {
		return fmt.Errorf("%s 不可写: %s", label, dir)
	}
	path := file.Name()
	_ = file.Close()
	_ = os.Remove(path)
	return nil
}

func (installer *Installer) resolveVersion() error {
	if installer.options.Mode == cli.ModeDev {
		installer.version = "dev"
		return nil
	}

	if version := strings.TrimSpace(installer.options.Version); version != "" {
		installer.version = version
		return nil
	}

	if version := strings.TrimSpace(buildinfo.Version); version != "" && version != "dev" && version != "unknown" {
		installer.version = version
		return nil
	}

	return fmt.Errorf("生产模式缺少版本号，请通过 --version 传入或使用注入的构建版本")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func sslDir(dockerDir string) string {
	return filepath.Join(dockerDir, "nginx", "ssl")
}
