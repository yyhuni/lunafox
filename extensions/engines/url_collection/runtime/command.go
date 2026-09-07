package urlcollectionruntime

import (
	"fmt"
	"strconv"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

// toolCommand is the complete, fixed invocation of one image-local tool.
// Arguments are intentionally constructed from the generated typed config;
// no user-provided shell fragment, executable, or arbitrary flag is accepted.
type toolCommand struct {
	name    string
	args    []string
	timeout time.Duration
}

func buildWaymoreCommand(target enginecontract.Target, outputPath string, config enginecontract.WaymoreConfig) (toolCommand, error) {
	if target.Type != enginecontract.TargetTypeDomain || target.Value == "" {
		return toolCommand{}, fmt.Errorf("waymore requires a domain Target")
	}
	if outputPath == "" || config.Timeout <= 0 {
		return toolCommand{}, fmt.Errorf("waymore output path and timeout are required")
	}
	return toolCommand{
		name:    "waymore",
		args:    []string{"-i", target.Value, "-mode", "U", "-oU", outputPath},
		timeout: time.Duration(config.Timeout) * time.Second,
	}, nil
}

func buildKatanaCommand(seedPath, outputPath string, config enginecontract.KatanaConfig) (toolCommand, error) {
	if seedPath == "" || outputPath == "" || config.Timeout <= 0 {
		return toolCommand{}, fmt.Errorf("katana seed path, output path, and timeout are required")
	}
	for label, value := range map[string]int64{
		"depth": config.Depth, "concurrency": config.Concurrency, "rate-limit": config.RateLimit,
		"request-timeout": config.RequestTimeout, "retries": config.Retries, "delay": config.Delay,
	} {
		if value < 0 || (label != "retries" && label != "delay" && value == 0) {
			return toolCommand{}, fmt.Errorf("katana %s is invalid", label)
		}
	}
	return toolCommand{
		name: "katana",
		args: []string{
			"-list", seedPath, "-o", outputPath, "-silent",
			"-d", strconv.FormatInt(config.Depth, 10),
			"-c", strconv.FormatInt(config.Concurrency, 10),
			"-rl", strconv.FormatInt(config.RateLimit, 10),
			"-timeout", strconv.FormatInt(config.RequestTimeout, 10),
			"-retry", strconv.FormatInt(config.Retries, 10),
			"-rd", strconv.FormatInt(config.Delay, 10),
			"-fs", "rdn", "-dr",
		},
		timeout: time.Duration(config.Timeout) * time.Second,
	}, nil
}

func buildUroCommand(inputPath, outputPath string, config enginecontract.UroConfig) (toolCommand, error) {
	if inputPath == "" || outputPath == "" || config.Timeout <= 0 {
		return toolCommand{}, fmt.Errorf("uro input path, output path, and timeout are required")
	}
	args := []string{"-i", inputPath, "-o", outputPath}
	if len(config.Whitelist) > 0 {
		args = append(args, "-w")
		args = append(args, config.Whitelist...)
	}
	if len(config.Blacklist) > 0 {
		args = append(args, "-b")
		args = append(args, config.Blacklist...)
	}
	if len(config.Filters) > 0 {
		args = append(args, "-f")
		args = append(args, config.Filters...)
	}
	return toolCommand{name: "uro", args: args, timeout: time.Duration(config.Timeout) * time.Second}, nil
}

func buildHTTPXCommand(inputPath, outputPath string, config enginecontract.HTTPXConfig) (toolCommand, error) {
	if inputPath == "" || outputPath == "" || config.Timeout <= 0 {
		return toolCommand{}, fmt.Errorf("httpx input path, output path, and timeout are required")
	}
	for label, value := range map[string]int64{
		"threads": config.Threads, "rate-limit": config.RateLimit, "request-timeout": config.RequestTimeout, "retries": config.Retries,
	} {
		if value < 0 || (label != "retries" && value == 0) {
			return toolCommand{}, fmt.Errorf("httpx %s is invalid", label)
		}
	}
	return toolCommand{
		name: "httpx",
		args: []string{
			"-list", inputPath, "-json", "-silent", "-no-color",
			"-status-code", "-content-type", "-content-length", "-location", "-title", "-server",
			"-tech-detect", "-cdn", "-vhost", "-include-response", "-rstr", "2000", "-random-agent",
			"-threads", strconv.FormatInt(config.Threads, 10),
			"-rate-limit", strconv.FormatInt(config.RateLimit, 10),
			"-timeout", strconv.FormatInt(config.RequestTimeout, 10),
			"-retries", strconv.FormatInt(config.Retries, 10),
			"-o", outputPath,
		},
		timeout: time.Duration(config.Timeout) * time.Second,
	}, nil
}
