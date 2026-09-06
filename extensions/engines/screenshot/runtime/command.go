package screenshotruntime

import (
	"fmt"
	"strconv"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/screenshot/contract"
)

type httpxCommand struct {
	args    []string
	timeout time.Duration
}

func validateScreenshotConfig(config enginecontract.CaptureConfig) error {
	if config.PageTimeout < 1 || config.PageTimeout > 120 {
		return fmt.Errorf("page-timeout must be between 1 and 120 seconds")
	}
	if config.Concurrency < 1 || config.Concurrency > 20 {
		return fmt.Errorf("concurrency must be between 1 and 20")
	}
	if config.Retries < 0 || config.Retries > 3 {
		return fmt.Errorf("retries must be between 0 and 3")
	}
	return nil
}

func buildHTTPXCommand(candidatePath, workspace string, config enginecontract.CaptureConfig) (httpxCommand, error) {
	if err := validateScreenshotConfig(config); err != nil {
		return httpxCommand{}, err
	}
	if candidatePath == "" || workspace == "" {
		return httpxCommand{}, fmt.Errorf("HTTPX candidate path and workspace are required")
	}
	args := []string{
		"-list", candidatePath,
		"-json", "-ss", "-no-screenshot-full-page", "-system-chrome",
		"-screenshot-timeout", strconv.FormatInt(config.PageTimeout, 10) + "s",
		"-sid", "1s",
		"-threads", strconv.FormatInt(config.Concurrency, 10),
		"-retries", strconv.FormatInt(config.Retries, 10),
		"-ho", "window-size=1280,720",
		"-ho", "force-device-scale-factor=1",
		"-ob", "-esb", "-ehb",
		"-srd", workspace,
	}
	return httpxCommand{args: args}, nil
}
