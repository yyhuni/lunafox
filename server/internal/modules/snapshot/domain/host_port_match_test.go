package domain

import "testing"

func TestIsHostPortMatchTarget(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		ip     string
		target ScanTargetRef
		match  bool
	}{
		{
			name:   "domain subdomain match",
			host:   "a.example.com",
			ip:     "1.1.1.1",
			target: ScanTargetRef{Name: "example.com", Type: "domain"},
			match:  true,
		},
		{
			name:   "domain mismatch",
			host:   "evil.com",
			ip:     "1.1.1.1",
			target: ScanTargetRef{Name: "example.com", Type: "domain"},
			match:  false,
		},
		{
			name:   "ip match",
			host:   "x",
			ip:     "10.0.0.1",
			target: ScanTargetRef{Name: "10.0.0.1", Type: "ip"},
			match:  true,
		},
		{
			name:   "cidr match",
			host:   "x",
			ip:     "10.0.0.42",
			target: ScanTargetRef{Name: "10.0.0.0/24", Type: "cidr"},
			match:  true,
		},
		{
			name:   "cidr invalid target",
			host:   "x",
			ip:     "10.0.0.42",
			target: ScanTargetRef{Name: "invalid", Type: "cidr"},
			match:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual := IsHostPortMatchTarget(testCase.host, testCase.ip, testCase.target)
			if actual != testCase.match {
				t.Fatalf("unexpected match result want=%v got=%v", testCase.match, actual)
			}
		})
	}
}
