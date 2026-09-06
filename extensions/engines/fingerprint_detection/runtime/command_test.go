package fingerprintdetectionruntime

import (
	"reflect"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/fingerprint_detection/contract"
)

func TestBuildObserverWardCommandUsesClosedHTTPJSONSurface(t *testing.T) {
	command, err := buildObserverWardCommand("/workspace/candidates.txt", "/run/lunafox/resources/platform/fingerprintLibraryFingerPrintHub/fingerprinthub_web.json", enginecontract.ObserverWardConfig{Timeout: 3600, RequestTimeout: 10, Threads: 25})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-l", "/workspace/candidates.txt", "-p", "/run/lunafox/resources/platform/fingerprintLibraryFingerPrintHub/fingerprinthub_web.json", "--mode", "http", "--timeout", "10", "--thread", "25", "--format", "json", "--silent", "--no-color"}
	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("args = %#v, want %#v", command.Args, want)
	}
	if command.Name != "observer-ward" || command.Timeout != time.Hour {
		t.Fatalf("command = %#v", command)
	}
}

func TestBuildObserverWardCommandRejectsUndeclaredRanges(t *testing.T) {
	for _, config := range []enginecontract.ObserverWardConfig{{Timeout: 59, RequestTimeout: 10, Threads: 25}, {Timeout: 3600, RequestTimeout: 121, Threads: 25}, {Timeout: 3600, RequestTimeout: 10, Threads: 201}} {
		if _, err := buildObserverWardCommand("candidates", "probe", config); err == nil {
			t.Fatalf("config %#v unexpectedly accepted", config)
		}
	}
}
