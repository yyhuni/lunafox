package subdomaindiscoveryruntime

import (
	"strings"
	"testing"

	"github.com/d3mondev/puredns/v2/pkg/wildcarder"
	"github.com/d3mondev/resolvermt"
	"github.com/stretchr/testify/require"
)

type wildcardBoundaryResolver struct{}

func (wildcardBoundaryResolver) Resolve(domains []string) []wildcarder.DNSAnswer {
	const wildcardAnswer = "192.0.2.10"
	const rootAnswer = "192.0.2.20"

	var answers []wildcarder.DNSAnswer
	for _, domain := range domains {
		switch {
		case domain == "p.example.com":
			answers = append(answers, wildcarder.DNSAnswer{Type: resolvermt.TypeA, Answer: rootAnswer})
		case domain == "something-p.example.com", domain == "api.p.example.com":
			answers = append(answers, wildcarder.DNSAnswer{Type: resolvermt.TypeA, Answer: wildcardAnswer})
		case strings.HasSuffix(domain, ".p.example.com"):
			answers = append(answers, wildcarder.DNSAnswer{Type: resolvermt.TypeA, Answer: wildcardAnswer})
		}
	}
	return answers
}

func (wildcardBoundaryResolver) QueryCount() int { return 0 }

func TestPureDNSWildcardFilteringRespectsDNSLabelBoundary(t *testing.T) {
	filter := wildcarder.New(1, 3, wildcarder.WithResolver(wildcardBoundaryResolver{}))

	domains, roots := filter.Filter(strings.NewReader("something-p.example.com\napi.p.example.com\n"))

	require.ElementsMatch(t, []string{"something-p.example.com"}, domains)
	require.ElementsMatch(t, []string{"p.example.com"}, roots)
}
