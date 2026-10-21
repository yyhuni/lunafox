package directoryscanruntime

import (
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// targetVariantSuffixes is the fixed backup-file suffix set for target-derived
// variants. It is intentionally closed: the variant count is bounded by the
// fixed name and suffix sets, not by a configurable limit.
var targetVariantSuffixes = []string{
	".zip",
	".tar.gz",
	".tar",
	".rar",
	".7z",
	".sql",
	".bak",
	"_bak.zip",
}

// targetVariantPayloads returns deterministic target-derived payloads for a
// Website candidate. Only the candidate host contributes names; paths, ports,
// queries, and fragments never do. IP/CIDR/non-host candidates yield nothing.
func targetVariantPayloads(candidate string) []string {
	parsed, err := url.Parse(candidate)
	if err != nil {
		return nil
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" || net.ParseIP(host) != nil {
		return nil
	}

	names := make([]string, 0, 5)
	appendName := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		for _, existing := range names {
			if existing == name {
				return
			}
		}
		names = append(names, name)
	}

	appendName(host)
	if registrable, err := publicsuffix.EffectiveTLDPlusOne(host); err == nil {
		appendName(registrable)
	}
	if stripped := strings.TrimPrefix(host, "www."); stripped != host {
		appendName(stripped)
	}
	labels := strings.Split(host, ".")
	if len(labels) >= 1 {
		appendName(labels[0])
	}
	if len(labels) >= 2 {
		appendName(labels[1])
	}

	payloads := make([]string, 0, len(names)*len(targetVariantSuffixes))
	seen := make(map[string]struct{}, len(names)*len(targetVariantSuffixes))
	for _, name := range names {
		for _, suffix := range targetVariantSuffixes {
			payload := name + suffix
			if _, duplicate := seen[payload]; duplicate {
				continue
			}
			seen[payload] = struct{}{}
			payloads = append(payloads, payload)
		}
	}
	return payloads
}
