package websitediscoveryruntime

import (
	"fmt"
	"strconv"
	"time"

	websitediscoverycontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
)

type httpxCommand struct {
	args    []string
	timeout time.Duration
}

func buildHTTPXCommand(urlsFilePath, outputFilePath string, config websitediscoverycontract.HTTPXConfig) (httpxCommand, error) {
	args := []string{
		"-list", urlsFilePath,
		"-json",
		"-silent",
		"-no-color",
		"-status-code",
		"-content-type",
		"-content-length",
		"-location",
		"-title",
		"-server",
		"-tech-detect",
		"-cdn",
		"-vhost",
		"-include-response",
		"-rstr", "2000",
		"-random-agent",
		"-o", outputFilePath,
	}
	if config.Threads > 0 {
		args = append(args, "-threads", strconv.FormatInt(config.Threads, 10))
	}
	if config.RateLimit > 0 {
		args = append(args, "-rate-limit", strconv.FormatInt(config.RateLimit, 10))
	}
	if config.RequestTimeout > 0 {
		args = append(args, "-timeout", strconv.FormatInt(config.RequestTimeout, 10))
	}
	if config.Retries >= 0 {
		args = append(args, "-retries", strconv.FormatInt(config.Retries, 10))
	}
	if config.Timeout <= 0 {
		return httpxCommand{}, fmt.Errorf("httpx timeout must be positive")
	}
	return httpxCommand{args: args, timeout: time.Duration(config.Timeout) * time.Second}, nil
}
