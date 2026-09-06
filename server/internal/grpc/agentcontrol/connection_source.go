package agentcontrol

import (
	"context"
	"net"
	"net/netip"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/geolocation"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const forwardedForMetadataKey = "x-forwarded-for"

// ConnectionSourceObservation is the server-owned result of resolving one
// active control connection. ConnectionIP is the sole current-address value;
// ObservedSourceIP is its strict-public subset for GeoIP only.
type ConnectionSourceObservation struct {
	ConnectionIP     string
	ObservedSourceIP string
}

// ConnectionSourceResolver resolves a display-only source observation at the
// gRPC boundary. It deliberately has no proxy trust configuration: this data
// never participates in authentication, authorization, or business decisions.
type ConnectionSourceResolver struct{}

// NewConnectionSourceResolver constructs the display-only source resolver.
func NewConnectionSourceResolver() *ConnectionSourceResolver {
	return &ConnectionSourceResolver{}
}

// Resolve returns the normalized current connection address. It is retained as
// a narrow compatibility helper; callers needing GeoIP input should use
// ResolveObservation and consume ObservedSourceIP only for that purpose.
func (*ConnectionSourceResolver) Resolve(ctx context.Context) string {
	return (&ConnectionSourceResolver{}).ResolveObservation(ctx).ConnectionIP
}

// ResolveObservation selects a single valid forwarded address only after a
// usable direct peer is present. Any malformed, repeated, or chained header
// falls back to that peer rather than implying a recoverable true source.
func (*ConnectionSourceResolver) ResolveObservation(ctx context.Context) ConnectionSourceObservation {
	directPeer, ok := directPeerAddress(ctx)
	if !ok {
		return ConnectionSourceObservation{}
	}

	connectionIP, ok := normalizeConnectionAddress(directPeer)
	if !ok {
		return ConnectionSourceObservation{}
	}
	if forwarded, ok := singleForwardedConnectionAddress(ctx); ok {
		connectionIP = forwarded
	}
	observedSourceIP, _ := geolocation.NormalizePublicIP(connectionIP)
	return ConnectionSourceObservation{
		ConnectionIP:     connectionIP,
		ObservedSourceIP: observedSourceIP,
	}
}

func directPeerAddress(ctx context.Context) (netip.Addr, bool) {
	if ctx == nil {
		return netip.Addr{}, false
	}
	peerInfo, ok := peer.FromContext(ctx)
	if !ok || peerInfo == nil || peerInfo.Addr == nil {
		return netip.Addr{}, false
	}
	return parsePeerAddress(peerInfo.Addr.String())
}

func parsePeerAddress(raw string) (netip.Addr, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return netip.Addr{}, false
	}
	if addressPort, err := netip.ParseAddrPort(trimmed); err == nil {
		return normalizeAddress(addressPort.Addr()), true
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		return parseAddress(host)
	}
	return parseAddress(trimmed)
}

func singleForwardedConnectionAddress(ctx context.Context) (string, bool) {
	metadataValues, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	values := metadataValues.Get(forwardedForMetadataKey)
	if len(values) != 1 {
		return "", false
	}
	value := strings.TrimSpace(values[0])
	if value == "" || strings.Contains(value, ",") {
		return "", false
	}
	address, ok := parseAddress(value)
	if !ok {
		return "", false
	}
	return normalizeConnectionAddress(address)
}

func parseAddress(raw string) (netip.Addr, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return netip.Addr{}, false
	}
	address, err := netip.ParseAddr(trimmed)
	if err != nil {
		return netip.Addr{}, false
	}
	return normalizeAddress(address), true
}

func normalizeAddress(address netip.Addr) netip.Addr {
	return address.Unmap().WithZone("")
}

// normalizeConnectionAddress accepts valid current transport addresses,
// including private/loopback/link-local/CGNAT values, but rejects unspecified,
// multicast, documentation, benchmarking, reserved and other special-use
// ranges. Public validation remains owned by geolocation.NormalizePublicIP.
func normalizeConnectionAddress(address netip.Addr) (string, bool) {
	address = normalizeAddress(address)
	if !address.IsValid() || address.IsUnspecified() || address.IsMulticast() {
		return "", false
	}
	normalized := address.String()
	if _, ok := geolocation.NormalizePublicIP(normalized); ok {
		return normalized, true
	}
	if address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || isCarrierGradeNAT(address) {
		return normalized, true
	}
	return "", false
}

func isCarrierGradeNAT(address netip.Addr) bool {
	return netip.MustParsePrefix("100.64.0.0/10").Contains(address)
}
