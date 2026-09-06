package screenshotruntime

import (
	"reflect"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/screenshot/contract"
)

func TestBuildHTTPXCommandUsesFixedScreenshotContract(t *testing.T) {
	command, err := buildHTTPXCommand("/workspace/httpx-candidates.txt", "/workspace", enginecontract.CaptureConfig{PageTimeout: 30, Concurrency: 8, Retries: 2})
	if err != nil {
		t.Fatalf("buildHTTPXCommand() error = %v", err)
	}
	want := []string{
		"-list", "/workspace/httpx-candidates.txt", "-json", "-ss", "-no-screenshot-full-page", "-system-chrome",
		"-screenshot-timeout", "30s", "-sid", "1s", "-threads", "8", "-retries", "2",
		"-ho", "window-size=1280,720", "-ho", "force-device-scale-factor=1",
		"-ob", "-esb", "-ehb", "-srd", "/workspace",
	}
	if !reflect.DeepEqual(command.args, want) {
		t.Fatalf("HTTPX args = %#v, want %#v", command.args, want)
	}
	if command.timeout != 0 {
		t.Fatalf("production HTTPX command unexpectedly has a total timeout: %s", command.timeout)
	}
}

func TestValidateScreenshotConfigBounds(t *testing.T) {
	valid := enginecontract.CaptureConfig{PageTimeout: 15, Concurrency: 5, Retries: 1}
	if err := validateScreenshotConfig(valid); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	cases := []enginecontract.CaptureConfig{
		{PageTimeout: 0, Concurrency: 5, Retries: 1},
		{PageTimeout: 121, Concurrency: 5, Retries: 1},
		{PageTimeout: 15, Concurrency: 0, Retries: 1},
		{PageTimeout: 15, Concurrency: 21, Retries: 1},
		{PageTimeout: 15, Concurrency: 5, Retries: -1},
		{PageTimeout: 15, Concurrency: 5, Retries: 4},
	}
	for _, config := range cases {
		if err := validateScreenshotConfig(config); err == nil {
			t.Fatalf("invalid config accepted: %+v", config)
		}
	}
}
