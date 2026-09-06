package portscanruntime

import (
	"reflect"
	"strings"
	"testing"
	"time"

	portscancontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

func TestBuildNaabuActiveCommandUsesTopPortMode(t *testing.T) {
	cmd, err := buildNaabuActiveCommand("/tmp/subdomains.txt", "/tmp/active.jsonl", portscancontract.NaabuActiveConfig{
		Enabled:  true,
		Timeout:  120,
		Threads:  50,
		Ports:    "80,443",
		PortMode: "top",
		TopPorts: "full",
		Rate:     500,
	})
	if err != nil {
		t.Fatalf("buildNaabuActiveCommand failed: %v", err)
	}

	want := []string{"-exclude-cdn", "-warm-up-time", "5", "-verify", "-list", "/tmp/subdomains.txt", "-json", "-silent", "-o", "/tmp/active.jsonl", "-c", "50", "-top-ports", "full", "-rate", "500"}
	if !reflect.DeepEqual(cmd.args, want) || cmd.timeout != 120*time.Second {
		t.Fatalf("unexpected active command: %+v", cmd)
	}
}

func TestBuildNaabuActiveCommandUsesCustomPortMode(t *testing.T) {
	cmd, err := buildNaabuActiveCommand("/tmp/subdomains.txt", "/tmp/active.jsonl", portscancontract.NaabuActiveConfig{
		Enabled:  true,
		Timeout:  120,
		Threads:  50,
		Ports:    "80,443",
		PortMode: "custom",
		TopPorts: "full",
		Rate:     500,
	})
	if err != nil {
		t.Fatalf("buildNaabuActiveCommand failed: %v", err)
	}

	want := []string{"-exclude-cdn", "-warm-up-time", "5", "-verify", "-list", "/tmp/subdomains.txt", "-json", "-silent", "-o", "/tmp/active.jsonl", "-c", "50", "-p", "80,443", "-rate", "500"}
	if !reflect.DeepEqual(cmd.args, want) || cmd.timeout != 120*time.Second {
		t.Fatalf("unexpected active command: %+v", cmd)
	}
}

func TestBuildNaabuActiveCommandUsesTopAndCustomPortMode(t *testing.T) {
	cmd, err := buildNaabuActiveCommand("/tmp/subdomains.txt", "/tmp/active.jsonl", portscancontract.NaabuActiveConfig{
		Enabled:  true,
		Timeout:  120,
		Threads:  50,
		Ports:    "80,443",
		PortMode: "top-and-custom",
		TopPorts: "full",
		Rate:     500,
	})
	if err != nil {
		t.Fatalf("buildNaabuActiveCommand failed: %v", err)
	}

	want := []string{"-exclude-cdn", "-warm-up-time", "5", "-verify", "-list", "/tmp/subdomains.txt", "-json", "-silent", "-o", "/tmp/active.jsonl", "-c", "50", "-p", "80,443", "-top-ports", "full", "-rate", "500"}
	if !reflect.DeepEqual(cmd.args, want) || cmd.timeout != 120*time.Second {
		t.Fatalf("unexpected active command: %+v", cmd)
	}
}

func TestBuildNaabuActiveCommandUsesTopAndCustomWithoutCustomPorts(t *testing.T) {
	cmd, err := buildNaabuActiveCommand("/tmp/subdomains.txt", "/tmp/active.jsonl", portscancontract.NaabuActiveConfig{
		Enabled:  true,
		Timeout:  120,
		Threads:  50,
		Ports:    "  ",
		PortMode: "top-and-custom",
		TopPorts: "100",
		Rate:     500,
	})
	if err != nil {
		t.Fatalf("buildNaabuActiveCommand failed: %v", err)
	}

	want := []string{"-exclude-cdn", "-warm-up-time", "5", "-verify", "-list", "/tmp/subdomains.txt", "-json", "-silent", "-o", "/tmp/active.jsonl", "-c", "50", "-top-ports", "100", "-rate", "500"}
	if !reflect.DeepEqual(cmd.args, want) || cmd.timeout != 120*time.Second {
		t.Fatalf("unexpected active command: %+v", cmd)
	}
}

func TestBuildNaabuActiveCommandRejectsOldTopPlusCustomMode(t *testing.T) {
	_, err := buildNaabuActiveCommand("/tmp/subdomains.txt", "/tmp/active.jsonl", portscancontract.NaabuActiveConfig{
		Enabled:  true,
		Timeout:  120,
		Ports:    "80,443",
		PortMode: "top-plus-custom",
		TopPorts: "100",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported port-mode") {
		t.Fatalf("expected old port mode to be rejected, got %v", err)
	}
}

func TestBuildNaabuActiveCommandRejectsCustomModeWithoutPorts(t *testing.T) {
	_, err := buildNaabuActiveCommand("/tmp/subdomains.txt", "/tmp/active.jsonl", portscancontract.NaabuActiveConfig{
		Enabled:  true,
		Timeout:  120,
		PortMode: "custom",
		TopPorts: "100",
		Ports:    "  ",
	})
	if err == nil || !strings.Contains(err.Error(), "ports is required") {
		t.Fatalf("expected missing ports error, got %v", err)
	}
}

func TestBuildNaabuPassiveCommandOmitsActiveOnlyArgs(t *testing.T) {
	cmd, err := buildNaabuPassiveCommand("/tmp/subdomains.txt", "/tmp/passive.jsonl", portscancontract.NaabuPassiveConfig{Enabled: true, Timeout: 60})
	if err != nil {
		t.Fatalf("buildNaabuPassiveCommand failed: %v", err)
	}

	want := []string{"-list", "/tmp/subdomains.txt", "-passive", "-json", "-silent", "-o", "/tmp/passive.jsonl"}
	if !reflect.DeepEqual(cmd.args, want) || cmd.timeout != 60*time.Second {
		t.Fatalf("unexpected passive command: %+v", cmd)
	}
}
