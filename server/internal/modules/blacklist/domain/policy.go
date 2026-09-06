// Package domain owns the canonical blacklist rule language and immutable
// policy value objects. Storage and transport must not invent a second parser.
package domain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
)

const (
	MaxPolicyPatterns       = 10_000
	MaxPolicyPatternBytes   = 253
	MaxPolicyJSONBytes      = 1 << 20
	MaxEffectivePatterns    = 20_000
	MaxEffectiveJSONBytes   = 2 << 20
	blacklistWildcardPrefix = "*."
)

var (
	ErrInvalidPattern         = errors.New("invalid blacklist pattern")
	ErrPolicyLimitExceeded    = errors.New("blacklist policy limit exceeded")
	ErrEffectiveLimitExceeded = errors.New("blacklist effective policy limit exceeded")
	ErrNonCanonicalPatterns   = errors.New("blacklist patterns are not canonical")
)

// Scope identifies the singleton policy owner.
type Scope string

const (
	ScopeGlobal Scope = "global"
	ScopeTarget Scope = "target"
)

// Policy is the five-field persistence concept. ETag is derived from Patterns
// and deliberately is not a persisted field.
type Policy struct {
	ID        int
	Scope     Scope
	TargetID  *int
	Patterns  []string
	UpdatedAt time.Time
}

// CanonicalizePolicyPatterns accepts user input and returns one stable,
// deduplicated pattern array that satisfies the fixed per-policy limits.
func CanonicalizePolicyPatterns(patterns []string) ([]string, error) {
	return canonicalizePatterns(patterns, MaxPolicyPatterns, MaxPolicyJSONBytes, ErrPolicyLimitExceeded)
}

// CanonicalizeEffectivePatterns returns the non-weakening union used by one
// scan snapshot. Callers must validate each persisted source first so corrupt
// policy rows cannot be silently repaired during a scan create transaction.
func CanonicalizeEffectivePatterns(patternSets ...[]string) ([]string, error) {
	total := 0
	for _, patterns := range patternSets {
		total += len(patterns)
	}
	combined := make([]string, 0, total)
	for _, patterns := range patternSets {
		combined = append(combined, patterns...)
	}
	return canonicalizePatterns(combined, MaxEffectivePatterns, MaxEffectiveJSONBytes, ErrEffectiveLimitExceeded)
}

// ValidateCanonicalPolicyPatterns rejects persisted rows that do not already
// use the canonical representation. Runtime callers must fail closed instead
// of normalizing database corruption into a different policy.
func ValidateCanonicalPolicyPatterns(patterns []string) error {
	return validateCanonicalPatterns(patterns, MaxPolicyPatterns, MaxPolicyJSONBytes, ErrPolicyLimitExceeded)
}

// ValidateCanonicalEffectivePatterns validates a frozen scan snapshot.
func ValidateCanonicalEffectivePatterns(patterns []string) error {
	return validateCanonicalPatterns(patterns, MaxEffectivePatterns, MaxEffectiveJSONBytes, ErrEffectiveLimitExceeded)
}

func validateCanonicalPatterns(patterns []string, maxCount, maxJSONBytes int, limitError error) error {
	canonical, err := canonicalizePatterns(patterns, maxCount, maxJSONBytes, limitError)
	if err != nil {
		return err
	}
	if len(canonical) != len(patterns) {
		return ErrNonCanonicalPatterns
	}
	for index := range canonical {
		if canonical[index] != patterns[index] {
			return ErrNonCanonicalPatterns
		}
	}
	return nil
}

func canonicalizePatterns(patterns []string, maxCount, maxJSONBytes int, limitError error) ([]string, error) {
	if len(patterns) > maxCount {
		return nil, limitError
	}
	seen := make(map[string]struct{}, len(patterns))
	canonical := make([]string, 0, len(patterns))
	for _, raw := range patterns {
		pattern, err := canonicalizePattern(raw)
		if err != nil {
			return nil, err
		}
		if len(pattern) > MaxPolicyPatternBytes || !isASCII(pattern) {
			return nil, limitError
		}
		if _, exists := seen[pattern]; exists {
			continue
		}
		seen[pattern] = struct{}{}
		canonical = append(canonical, pattern)
	}
	sort.Strings(canonical)
	if len(canonical) > maxCount {
		return nil, limitError
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("encode canonical blacklist patterns: %w", err)
	}
	if len(payload) > maxJSONBytes {
		return nil, limitError
	}
	return canonical, nil
}

func canonicalizePattern(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ErrInvalidPattern
	}
	if strings.HasPrefix(value, blacklistWildcardPrefix) {
		base := strings.TrimPrefix(value, blacklistWildcardPrefix)
		if strings.Contains(base, "*") {
			return "", ErrInvalidPattern
		}
		canonical, ok := contractresults.NormalizeSubdomainDNSName(base)
		if !ok {
			return "", ErrInvalidPattern
		}
		return blacklistWildcardPrefix + canonical, nil
	}
	if strings.Contains(value, "*") {
		return "", ErrInvalidPattern
	}
	if prefix, err := netip.ParsePrefix(value); err == nil {
		if !prefix.Addr().Is4() {
			return "", ErrInvalidPattern
		}
		return prefix.Masked().String(), nil
	}
	if address, err := netip.ParseAddr(value); err == nil {
		if !address.Is4() {
			return "", ErrInvalidPattern
		}
		return address.String(), nil
	}
	if canonical, ok := contractresults.NormalizeSubdomainDNSName(value); ok {
		return canonical, nil
	}
	return "", ErrInvalidPattern
}

func isASCII(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] > 0x7f {
			return false
		}
	}
	return true
}

// CanonicalPatternsJSON returns the exact compact bytes from which an ETag is
// derived. It validates that its input is already canonical.
func CanonicalPatternsJSON(patterns []string) ([]byte, error) {
	if err := ValidateCanonicalEffectivePatterns(patterns); err != nil {
		return nil, err
	}
	return json.Marshal(patterns)
}

// ETag returns the opaque lower-case SHA-256 hex digest of canonical JSON.
func ETag(patterns []string) (string, error) {
	payload, err := CanonicalPatternsJSON(patterns)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

// Matcher is immutable after construction and provides O(1) exact lookups plus
// at most 33 prefix checks for an IPv4 candidate.
type Matcher struct {
	exactDomains  map[string]struct{}
	wildcardBases map[string]struct{}
	exactIPv4     map[uint32]struct{}
	cidrs         [33]map[uint32]struct{}
}

// CompileMatcher validates and compiles a frozen canonical pattern array.
func CompileMatcher(patterns []string) (*Matcher, error) {
	if err := ValidateCanonicalEffectivePatterns(patterns); err != nil {
		return nil, err
	}
	matcher := &Matcher{
		exactDomains:  make(map[string]struct{}),
		wildcardBases: make(map[string]struct{}),
		exactIPv4:     make(map[uint32]struct{}),
	}
	for _, pattern := range patterns {
		switch {
		case strings.HasPrefix(pattern, blacklistWildcardPrefix):
			matcher.wildcardBases[strings.TrimPrefix(pattern, blacklistWildcardPrefix)] = struct{}{}
		case strings.Contains(pattern, "/"):
			prefix, err := netip.ParsePrefix(pattern)
			if err != nil || !prefix.Addr().Is4() {
				return nil, ErrNonCanonicalPatterns
			}
			bits := prefix.Bits()
			if matcher.cidrs[bits] == nil {
				matcher.cidrs[bits] = make(map[uint32]struct{})
			}
			matcher.cidrs[bits][ipv4Uint32(prefix.Masked().Addr())] = struct{}{}
		case isIPv4Pattern(pattern):
			address, err := netip.ParseAddr(pattern)
			if err != nil || !address.Is4() {
				return nil, ErrNonCanonicalPatterns
			}
			matcher.exactIPv4[ipv4Uint32(address)] = struct{}{}
		default:
			matcher.exactDomains[pattern] = struct{}{}
		}
	}
	return matcher, nil
}

func isIPv4Pattern(pattern string) bool {
	address, err := netip.ParseAddr(pattern)
	return err == nil && address.Is4()
}

// MatchesHostname checks a canonical DNS hostname or canonical IPv4 literal.
// It deliberately returns false for malformed values instead of changing them.
func (matcher *Matcher) MatchesHostname(host string) bool {
	if matcher == nil || host == "" {
		return false
	}
	if address, err := netip.ParseAddr(host); err == nil && address.Is4() {
		return matcher.matchesIPv4(address)
	}
	if _, found := matcher.exactDomains[host]; found {
		return true
	}
	for base := range matcher.wildcardBases {
		if host != base && strings.HasSuffix(host, "."+base) {
			return true
		}
	}
	return false
}

// MatchesURL considers only a Host derived from the raw HTTP(S) authority.
// It intentionally leaves path, query, fragment, and percent syntax opaque so
// security-test payloads do not become unmatched merely because net/url would
// reject their later components.
func (matcher *Matcher) MatchesURL(rawURL string) bool {
	if matcher == nil {
		return false
	}
	host, err := contractresults.DeriveObservedAssetURLHost(rawURL)
	if err != nil {
		return false
	}
	return matcher.MatchesHostname(host)
}

// MatchesHostPort treats the persisted host and IPv4 identities independently.
func (matcher *Matcher) MatchesHostPort(host, ip string) bool {
	return matcher.MatchesHostname(host) || matcher.MatchesHostname(ip)
}

func (matcher *Matcher) matchesIPv4(address netip.Addr) bool {
	value := ipv4Uint32(address)
	if _, found := matcher.exactIPv4[value]; found {
		return true
	}
	for bits := 0; bits <= 32; bits++ {
		entries := matcher.cidrs[bits]
		if len(entries) == 0 {
			continue
		}
		if _, found := entries[value&ipv4Mask(bits)]; found {
			return true
		}
	}
	return false
}

func ipv4Uint32(address netip.Addr) uint32 {
	bytes := address.As4()
	return binary.BigEndian.Uint32(bytes[:])
}

func ipv4Mask(bits int) uint32 {
	if bits == 0 {
		return 0
	}
	return ^uint32(0) << (32 - bits)
}
