package domain

import "testing"

func TestTargetBuildersCanonicalizeEverySupportedTargetType(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantName   string
		wantType   string
		buildBatch bool
	}{
		{name: "domain", raw: " Example.COM. ", wantName: "example.com", wantType: TargetTypeDomain},
		{name: "international domain", raw: " b\u00fccher.example ", wantName: "xn--bcher-kva.example", wantType: TargetTypeDomain},
		{name: "IPv4", raw: " 192.0.2.10 ", wantName: "192.0.2.10", wantType: TargetTypeIP},
		{name: "masked IPv4 CIDR", raw: " 192.0.2.3/24 ", wantName: "192.0.2.0/24", wantType: TargetTypeCIDR},
		{name: "batch uses same canonicalizer", raw: " Example.COM. ", wantName: "example.com", wantType: TargetTypeDomain, buildBatch: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				target *Target
				err    error
			)
			if test.buildBatch {
				target, err = BuildBatchTarget(test.raw)
			} else {
				target, err = BuildTarget(test.raw)
			}
			if err != nil {
				t.Fatalf("build target: %v", err)
			}
			if target.Name != test.wantName || target.Type != test.wantType {
				t.Fatalf("target = %#v, want name %q type %q", target, test.wantName, test.wantType)
			}
			if got := DetectTargetType(test.raw); got != test.wantType {
				t.Fatalf("DetectTargetType(%q) = %q, want %q", test.raw, got, test.wantType)
			}
		})
	}
}

func TestTargetBuildersRejectIPv6AtWriteBoundary(t *testing.T) {
	for _, raw := range []string{
		"2001:db8::1",
		"2001:db8::/64",
		"::ffff:192.0.2.10",
		"::ffff:192.0.2.0/120",
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := BuildTarget(raw); err == nil {
				t.Fatalf("BuildTarget(%q) accepted IPv6", raw)
			}
			if _, err := BuildBatchTarget(raw); err == nil {
				t.Fatalf("BuildBatchTarget(%q) accepted IPv6", raw)
			}
			if got := DetectTargetType(raw); got != "" {
				t.Fatalf("DetectTargetType(%q) = %q, want rejection", raw, got)
			}
		})
	}
}

func TestTargetRenameUsesCanonicalIPv4OnlyBoundary(t *testing.T) {
	target := &Target{Name: "example.com", Type: TargetTypeDomain}
	if err := target.Rename("192.0.2.3/30"); err != nil {
		t.Fatalf("Rename canonical CIDR: %v", err)
	}
	if target.Name != "192.0.2.0/30" || target.Type != TargetTypeCIDR {
		t.Fatalf("renamed target = %#v", target)
	}

	beforeName, beforeType := target.Name, target.Type
	if err := target.Rename("2001:db8::1"); err == nil {
		t.Fatal("Rename accepted IPv6")
	}
	if target.Name != beforeName || target.Type != beforeType {
		t.Fatalf("failed Rename mutated target: got %#v want name %q type %q", target, beforeName, beforeType)
	}
}
