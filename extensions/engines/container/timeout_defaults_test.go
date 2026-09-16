package containercontract_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
)

func TestExecutionTimeoutDefaultsAndExplicitOverrides(t *testing.T) {
	cases := map[string]map[string]int{
		"subdomain_discovery":   {"recon": 7200, "bruteforce": 28800, "resolve": 14400},
		"port_scan":             {"naabu_active": 14400, "naabu_passive": 3600},
		"website_discovery":     {"httpx": 14400},
		"url_collection":        {"waymore": 28800, "katana": 28800, "uro": 3600, "httpx": 14400},
		"fingerprint_detection": {"observer_ward": 14400},
		"nuclei_vulnerability":  {"nuclei": 28800},
		"directory_scan":        {"ffuf": 86400},
	}
	for name, expected := range cases {
		t.Run(name, func(t *testing.T) {
			payload, err := os.ReadFile(filepath.Join("..", name, "engine.json"))
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := enginecontract.DecodeEngineDefinition(payload, name)
			if err != nil {
				t.Fatal(err)
			}
			for _, override := range []bool{false, true} {
				raw := map[string]any{}
				for section := range expected {
					values := map[string]any{"enabled": true}
					if override {
						values["timeout"] = 3600
					}
					raw[section] = values
				}
				normalized, err := engineexecution.NormalizeAndValidateConfig(raw, manifest.Execution)
				if err != nil {
					t.Fatal(err)
				}
				for section, want := range expected {
					if override {
						want = 3600
					}
					got := normalized[section].(map[string]any)["timeout"]
					if fmt.Sprint(got) != fmt.Sprint(want) {
						t.Fatalf("%s timeout = %v, want %d (override=%t)", section, got, want, override)
					}
				}
			}
		})
	}
}
