package ui

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

type Printer struct {
	Out      io.Writer
	Err      io.Writer
	colorOut bool
	colorErr bool
}

func NewPrinter(out, err io.Writer) *Printer {
	return &Printer{
		Out:      out,
		Err:      err,
		colorOut: shouldEnableColor(out),
		colorErr: shouldEnableColor(err),
	}
}

func (printer *Printer) Info(format string, args ...any) {
	fmt.Fprintf(printer.Out, "%s  %s\n", printer.styleOut("ℹ", ansiCyan), fmt.Sprintf(format, args...))
}

func (printer *Printer) Warn(format string, args ...any) {
	fmt.Fprintf(printer.Err, "%s  %s\n", printer.styleErr("⚠", ansiYellow), fmt.Sprintf(format, args...))
}

func (printer *Printer) Error(format string, args ...any) {
	fmt.Fprintf(printer.Err, "%s  %s\n", printer.styleErr("✗", ansiRed, ansiBold), fmt.Sprintf(format, args...))
}

func (printer *Printer) Success(format string, args ...any) {
	fmt.Fprintf(printer.Out, "%s  %s\n", printer.styleOut("✓", ansiGreen, ansiBold), fmt.Sprintf(format, args...))
}

func (printer *Printer) Step(current, total int, title string) {
	stepLabel := fmt.Sprintf("[%d/%d]", current, total)
	fmt.Fprintf(printer.Out, "\n%s %s\n", printer.styleOut(stepLabel, ansiBlue, ansiBold), strings.TrimSpace(title))
}

func (printer *Printer) Banner(mode, version string) {
	title := printer.styleOut("LunaFox 安装器", ansiBlue, ansiBold)
	modeLabel := printer.styleOut(mode, ansiMagenta, ansiBold)
	versionLabel := printer.styleOut(version, ansiMagenta, ansiBold)
	fmt.Fprintf(printer.Out, "\n%s | 模式: %s | 版本: %s\n\n", title, modeLabel, versionLabel)
}

func (printer *Printer) Summary(imageTag, composeFile, publicURL, composeCmd string) {
	line := strings.Repeat("=", 100)
	sep := strings.Repeat("-", 100)

	fmt.Fprintf(printer.Out, "\n%s\n", printer.styleOut(line, ansiBlue))
	fmt.Fprintln(printer.Out, printer.styleOut("安装完成", ansiGreen, ansiBold))
	fmt.Fprintf(printer.Out, "%s\n", printer.styleOut(sep, ansiBlue))
	fmt.Fprintf(printer.Out, "%s %s\n", printer.styleOut("访问地址   ", ansiBold), publicURL)
	fmt.Fprintf(printer.Out, "%s admin\n", printer.styleOut("默认账号   ", ansiBold))
	fmt.Fprintf(printer.Out, "%s admin\n", printer.styleOut("默认密码   ", ansiBold))
	fmt.Fprintf(printer.Out, "%s %s\n", printer.styleOut("镜像标签   ", ansiBold), imageTag)
	fmt.Fprintf(printer.Out, "%s %s\n", printer.styleOut("Compose 文件", ansiBold), composeFile)
	fmt.Fprintf(printer.Out, "%s\n", printer.styleOut(sep, ansiBlue))
	fmt.Fprintln(printer.Out, printer.styleOut("常用命令", ansiCyan, ansiBold))
	fmt.Fprintf(printer.Out, "  %s %s -f %s logs -f\n", printer.styleOut("日志:", ansiDim), composeCmd, composeFile)
	fmt.Fprintf(printer.Out, "  %s %s -f %s down\n", printer.styleOut("停止:", ansiDim), composeCmd, composeFile)
	fmt.Fprintf(printer.Out, "%s\n", printer.styleOut(line, ansiBlue))
}

const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiRed     = "\x1b[31m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiBlue    = "\x1b[34m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[36m"
)

func (printer *Printer) styleOut(text string, styles ...string) string {
	return styleText(text, printer.colorOut, styles...)
}

func (printer *Printer) styleErr(text string, styles ...string) string {
	return styleText(text, printer.colorErr, styles...)
}

func styleText(text string, enabled bool, styles ...string) string {
	if !enabled || len(styles) == 0 {
		return text
	}
	return strings.Join(styles, "") + text + ansiReset
}

func shouldEnableColor(writer io.Writer) bool {
	if colorDisabled() {
		return false
	}
	if colorForced() {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb") {
		return false
	}
	file, ok := writer.(*os.File)
	if !ok || file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	if (info.Mode() & os.ModeCharDevice) == 0 {
		return false
	}

	// Avoid raw ANSI on legacy Windows consoles without VT support.
	if runtime.GOOS == "windows" &&
		strings.TrimSpace(os.Getenv("WT_SESSION")) == "" &&
		strings.TrimSpace(os.Getenv("ANSICON")) == "" &&
		!strings.EqualFold(strings.TrimSpace(os.Getenv("ConEmuANSI")), "ON") {
		return false
	}
	return true
}

func colorDisabled() bool {
	if strings.TrimSpace(os.Getenv("LUNAFOX_NO_COLOR")) == "1" {
		return true
	}
	return strings.TrimSpace(os.Getenv("NO_COLOR")) != ""
}

func colorForced() bool {
	return strings.TrimSpace(os.Getenv("LUNAFOX_FORCE_COLOR")) == "1"
}
