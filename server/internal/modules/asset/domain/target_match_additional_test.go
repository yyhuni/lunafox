package domain

import "testing"

func TestIsURLMatchTargetSupportsIPAndCIDR(t *testing.T) {
	ipTarget := TargetRef{Name: "192.168.1.10", Type: TargetTypeIP}
	if !IsURLMatchTarget("https://192.168.1.10/admin", ipTarget) {
		t.Fatal("expected IP target to match exact host")
	}
	if IsURLMatchTarget("https://192.168.1.11/admin", ipTarget) {
		t.Fatal("expected different IP host not to match")
	}

	cidrTarget := TargetRef{Name: "10.0.0.0/24", Type: TargetTypeCIDR}
	if !IsURLMatchTarget("https://10.0.0.5/api", cidrTarget) {
		t.Fatal("expected CIDR target to match contained IP")
	}
	if IsURLMatchTarget("https://example.com/api", cidrTarget) {
		t.Fatal("expected hostname not to match CIDR target")
	}
}

func TestIsURLAndSubdomainMatchTargetRejectInvalidInputs(t *testing.T) {
	if IsURLMatchTarget("", TargetRef{Name: "example.com", Type: TargetTypeDomain}) {
		t.Fatal("expected empty URL not to match")
	}
	if IsURLMatchTarget("://bad-url", TargetRef{Name: "example.com", Type: TargetTypeDomain}) {
		t.Fatal("expected invalid URL not to match")
	}
	if IsURLMatchTarget("https://example.com", TargetRef{Name: "10.0.0.0/invalid", Type: TargetTypeCIDR}) {
		t.Fatal("expected invalid CIDR target not to match")
	}
	if IsURLMatchTarget("https://example.com", TargetRef{Name: "example.com", Type: "unknown"}) {
		t.Fatal("expected unknown target type not to match")
	}

	if !IsSubdomainMatchTarget("api.example.com", TargetRef{Name: "example.com", Type: TargetTypeDomain}) {
		t.Fatal("expected subdomain to match domain target")
	}
	if IsSubdomainMatchTarget("bad host", TargetRef{Name: "example.com", Type: TargetTypeDomain}) {
		t.Fatal("expected invalid DNS name not to match")
	}
	if IsSubdomainMatchTarget("api.example.com", TargetRef{Name: "192.168.1.10", Type: TargetTypeIP}) {
		t.Fatal("expected non-domain target not to match subdomain")
	}
	if IsURLMatchTarget("/relative/path", TargetRef{Name: "example.com", Type: TargetTypeDomain}) {
		t.Fatal("expected URL without hostname not to match")
	}
	if IsURLMatchTarget("https://10.0.0.5", TargetRef{Name: "10.0.0.0/invalid", Type: TargetTypeCIDR}) {
		t.Fatal("expected invalid CIDR to reject valid host IP")
	}
	if IsSubdomainMatchTarget("", TargetRef{Name: "example.com", Type: TargetTypeDomain}) {
		t.Fatal("expected blank subdomain not to match")
	}
	if IsSubdomainMatchTarget("api.example.com", TargetRef{Name: "", Type: TargetTypeDomain}) {
		t.Fatal("expected blank target name not to match")
	}
}
