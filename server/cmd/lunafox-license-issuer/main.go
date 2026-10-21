// Command lunafox-license-issuer serves a publisher-only local signing interface.
// It is independent of the product server and is never included in its image.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/yyhuni/lunafox/server/internal/agentlicense"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "签发工具启动失败：%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flags := flag.NewFlagSet("lunafox-license-issuer", flag.ContinueOnError)
	keyPath := flags.String("key-file", "", "作者私钥文件；默认保存在当前用户配置目录")
	exportPath := flags.String("export-public-key", "", "导出产品需要的公钥到文件，然后退出")
	port := flags.Int("port", 0, "本机监听端口，0表示自动选择空闲端口")
	noOpen := flags.Bool("no-open", false, "不自动打开浏览器")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("不支持位置参数")
	}
	if *port < 0 || *port > 65535 {
		return fmt.Errorf("port必须在0到65535之间")
	}
	if strings.TrimSpace(*keyPath) == "" {
		configPath, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("读取用户配置目录: %w", err)
		}
		*keyPath = filepath.Join(configPath, "lunafox-license-issuer", "issuer-key")
	}
	absoluteKeyPath, err := filepath.Abs(*keyPath)
	if err != nil {
		return err
	}
	if *exportPath != "" {
		absoluteExportPath, err := filepath.Abs(*exportPath)
		if err != nil {
			return err
		}
		if absoluteExportPath == absoluteKeyPath {
			return fmt.Errorf("公钥导出路径不能覆盖私钥文件")
		}
	}
	var privateKey ed25519.PrivateKey
	if *exportPath != "" {
		privateKey, err = loadOrCreatePrivateKey(absoluteKeyPath)
	} else {
		privateKey, err = loadPrivateKey(absoluteKeyPath)
	}
	if err != nil {
		return err
	}
	publicKey := base64.RawURLEncoding.EncodeToString(privateKey.Public().(ed25519.PublicKey))
	if *exportPath != "" {
		if err := os.WriteFile(*exportPath, []byte(publicKey+"\n"), 0o644); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, "公钥已导出：%s\n作者私钥仍仅保存在：%s\n", *exportPath, absoluteKeyPath)
		return nil
	}
	if err := validatePublisherPrivateKey(privateKey); err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()
	session := make([]byte, 32)
	if _, err := rand.Read(session); err != nil {
		return err
	}
	origin := "http://" + listener.Addr().String()
	issuer := &localIssuer{privateKey: privateKey, publicKey: publicKey, keyPath: absoluteKeyPath, origin: origin, session: base64.RawURLEncoding.EncodeToString(session), now: time.Now}
	server := &http.Server{Handler: issuer.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	url := origin + "/#" + issuer.session
	_, _ = fmt.Fprintf(os.Stdout, "本机签发界面：%s\n作者私钥：%s\n关闭此终端或按 Ctrl+C 停止工具。\n", url, absoluteKeyPath)
	if !*noOpen {
		if err := openBrowser(url); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "浏览器未自动打开，请复制上面的本机地址打开。")
		}
	}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func validatePublisherPrivateKey(privateKey ed25519.PrivateKey) error {
	verifier, err := agentlicense.NewDefaultVerifier()
	if err != nil {
		return err
	}
	claims, err := agentlicense.NewClaims(4, 1, time.Now())
	if err != nil {
		return err
	}
	probe, err := agentlicense.Sign(privateKey, claims)
	if err != nil {
		return err
	}
	if _, err := verifier.Verify(probe); err != nil {
		return fmt.Errorf("作者私钥与当前产品公钥不匹配，请恢复原作者私钥；不要重新生成密钥")
	}
	return nil
}

func openBrowser(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	if err := command.Start(); err != nil {
		return err
	}
	// Browser launchers may wait for a window to close; serving must begin
	// immediately so that opening the UI cannot stall its own HTTP server.
	go func() { _ = command.Wait() }()
	return nil
}
