package ui

import (
	"fmt"
	"io"
	"strings"
)

type Printer struct {
	Out io.Writer
	Err io.Writer
}

func NewPrinter(out, err io.Writer) *Printer {
	return &Printer{Out: out, Err: err}
}

func (printer *Printer) Info(format string, args ...any) {
	fmt.Fprintf(printer.Out, "ℹ  %s\n", fmt.Sprintf(format, args...))
}

func (printer *Printer) Warn(format string, args ...any) {
	fmt.Fprintf(printer.Err, "⚠  %s\n", fmt.Sprintf(format, args...))
}

func (printer *Printer) Error(format string, args ...any) {
	fmt.Fprintf(printer.Err, "✗  %s\n", fmt.Sprintf(format, args...))
}

func (printer *Printer) Success(format string, args ...any) {
	fmt.Fprintf(printer.Out, "✓  %s\n", fmt.Sprintf(format, args...))
}

func (printer *Printer) Step(current, total int, title string) {
	fmt.Fprintf(printer.Out, "\n[%d/%d] %s\n", current, total, strings.TrimSpace(title))
}

func (printer *Printer) Banner(mode, version string) {
	fmt.Fprintf(printer.Out, "\nLunaFox 安装器 | 模式: %s | 版本: %s\n\n", mode, version)
}

func (printer *Printer) Summary(imageTag, composeFile, publicURL, composeCmd string) {
	fmt.Fprintln(printer.Out, "\n安装完成!")
	fmt.Fprintf(printer.Out, "访问地址: %s\n", publicURL)
	fmt.Fprintln(printer.Out, "默认账号: admin")
	fmt.Fprintln(printer.Out, "默认密码: admin")
	fmt.Fprintf(printer.Out, "镜像标签: %s\n", imageTag)
	fmt.Fprintf(printer.Out, "Compose 文件: %s\n", composeFile)
	fmt.Fprintf(printer.Out, "查看日志: %s -f %s logs -f\n", composeCmd, composeFile)
}
