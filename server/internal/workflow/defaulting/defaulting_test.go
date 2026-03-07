package workflowdefaulting

import "testing"

func TestNormalizeRootConfigFillsDefaultsForEnabledTool(t *testing.T) {
	root := map[string]any{
		"subdomain_discovery": map[string]any{
			"recon": map[string]any{
				"enabled": true,
				"tools": map[string]any{
					"subfinder": map[string]any{
						"enabled": true,
					},
				},
			},
			"bruteforce":  map[string]any{"enabled": false, "tools": map[string]any{"subdomain-bruteforce": map[string]any{"enabled": false}}},
			"permutation": map[string]any{"enabled": false, "tools": map[string]any{"subdomain-permutation-resolve": map[string]any{"enabled": false}}},
			"resolve":     map[string]any{"enabled": false, "tools": map[string]any{"subdomain-resolve": map[string]any{"enabled": false}}},
		},
	}

	normalized, err := NormalizeRootConfig(root, []string{"subdomain_discovery"})
	if err != nil {
		t.Fatalf("NormalizeRootConfig failed: %v", err)
	}
	workflowConfig := normalized["subdomain_discovery"].(map[string]any)
	recon := workflowConfig["recon"].(map[string]any)
	tools := recon["tools"].(map[string]any)
	subfinder := tools["subfinder"].(map[string]any)
	if subfinder["timeout-runtime"] != 3600 {
		t.Fatalf("expected timeout default 3600, got %#v", subfinder["timeout-runtime"])
	}
	if subfinder["threads-cli"] != 10 {
		t.Fatalf("expected threads default 10, got %#v", subfinder["threads-cli"])
	}
}

func TestNormalizeRootConfigDoesNotCreateMissingStageOrTool(t *testing.T) {
	root := map[string]any{
		"subdomain_discovery": map[string]any{
			"recon": map[string]any{
				"enabled": true,
				"tools":   map[string]any{},
			},
		},
	}
	normalized, err := NormalizeRootConfig(root, []string{"subdomain_discovery"})
	if err != nil {
		t.Fatalf("NormalizeRootConfig failed: %v", err)
	}
	workflowConfig := normalized["subdomain_discovery"].(map[string]any)
	recon := workflowConfig["recon"].(map[string]any)
	tools := recon["tools"].(map[string]any)
	if _, exists := tools["subfinder"]; exists {
		t.Fatalf("expected missing tool not to be auto-created")
	}
	if _, exists := workflowConfig["bruteforce"]; exists {
		t.Fatalf("expected missing stage not to be auto-created")
	}
}

func TestNormalizeRootConfigDoesNotFillDisabledTool(t *testing.T) {
	root := map[string]any{
		"subdomain_discovery": map[string]any{
			"recon": map[string]any{
				"enabled": true,
				"tools": map[string]any{
					"subfinder": map[string]any{
						"enabled": false,
					},
				},
			},
			"bruteforce":  map[string]any{"enabled": false, "tools": map[string]any{"subdomain-bruteforce": map[string]any{"enabled": false}}},
			"permutation": map[string]any{"enabled": false, "tools": map[string]any{"subdomain-permutation-resolve": map[string]any{"enabled": false}}},
			"resolve":     map[string]any{"enabled": false, "tools": map[string]any{"subdomain-resolve": map[string]any{"enabled": false}}},
		},
	}
	normalized, err := NormalizeRootConfig(root, []string{"subdomain_discovery"})
	if err != nil {
		t.Fatalf("NormalizeRootConfig failed: %v", err)
	}
	workflowConfig := normalized["subdomain_discovery"].(map[string]any)
	recon := workflowConfig["recon"].(map[string]any)
	tools := recon["tools"].(map[string]any)
	subfinder := tools["subfinder"].(map[string]any)
	if _, exists := subfinder["timeout-runtime"]; exists {
		t.Fatalf("expected disabled tool not to receive defaults")
	}
}
