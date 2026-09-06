package agentcontrol

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type connectionSourceTestAddress string

func (address connectionSourceTestAddress) Network() string { return "tcp" }
func (address connectionSourceTestAddress) String() string  { return string(address) }

func TestConnectionSourceResolverUsesSingleForwardedValueOrDirectPeer(t *testing.T) {
	tests := []struct {
		name      string
		peer      net.Addr
		forwarded []string
		want      ConnectionSourceObservation
	}{
		{name: "missing peer", want: ConnectionSourceObservation{}},
		{name: "direct public IPv4", peer: connectionSourceTestAddress("8.8.8.8:443"), want: ConnectionSourceObservation{ConnectionIP: "8.8.8.8", ObservedSourceIP: "8.8.8.8"}},
		{name: "direct private Docker peer", peer: connectionSourceTestAddress("172.20.0.5:443"), want: ConnectionSourceObservation{ConnectionIP: "172.20.0.5"}},
		{name: "direct IPv6", peer: connectionSourceTestAddress("[2606:4700:4700::1111]:443"), want: ConnectionSourceObservation{ConnectionIP: "2606:4700:4700::1111", ObservedSourceIP: "2606:4700:4700::1111"}},
		{name: "single public forwarded value wins", peer: connectionSourceTestAddress("172.20.0.5:443"), forwarded: []string{"8.8.4.4"}, want: ConnectionSourceObservation{ConnectionIP: "8.8.4.4", ObservedSourceIP: "8.8.4.4"}},
		{name: "single private forwarded value wins", peer: connectionSourceTestAddress("172.20.0.5:443"), forwarded: []string{"10.2.3.4"}, want: ConnectionSourceObservation{ConnectionIP: "10.2.3.4"}},
		{name: "mapped forwarded value is normalized", peer: connectionSourceTestAddress("172.20.0.5:443"), forwarded: []string{"::ffff:8.8.8.8"}, want: ConnectionSourceObservation{ConnectionIP: "8.8.8.8", ObservedSourceIP: "8.8.8.8"}},
		{name: "missing forwarded value falls back to peer", peer: connectionSourceTestAddress("172.20.0.5:443"), want: ConnectionSourceObservation{ConnectionIP: "172.20.0.5"}},
		{name: "repeated forwarded value falls back to peer", peer: connectionSourceTestAddress("172.20.0.5:443"), forwarded: []string{"8.8.8.8", "1.1.1.1"}, want: ConnectionSourceObservation{ConnectionIP: "172.20.0.5"}},
		{name: "forwarded chain falls back to peer", peer: connectionSourceTestAddress("172.20.0.5:443"), forwarded: []string{"8.8.8.8, 1.1.1.1"}, want: ConnectionSourceObservation{ConnectionIP: "172.20.0.5"}},
		{name: "malformed forwarded value falls back to peer", peer: connectionSourceTestAddress("172.20.0.5:443"), forwarded: []string{"not-an-address"}, want: ConnectionSourceObservation{ConnectionIP: "172.20.0.5"}},
		{name: "special forwarded value falls back to peer", peer: connectionSourceTestAddress("172.20.0.5:443"), forwarded: []string{"192.0.2.1"}, want: ConnectionSourceObservation{ConnectionIP: "172.20.0.5"}},
		{name: "invalid direct peer does not use header alone", peer: connectionSourceTestAddress("not-an-address"), forwarded: []string{"8.8.8.8"}, want: ConnectionSourceObservation{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.peer != nil {
				ctx = peer.NewContext(ctx, &peer.Peer{Addr: test.peer})
			}
			if len(test.forwarded) > 0 {
				ctx = metadata.NewIncomingContext(ctx, metadata.MD{forwardedForMetadataKey: test.forwarded})
			}
			resolver := NewConnectionSourceResolver()
			got := resolver.ResolveObservation(ctx)
			if got != test.want {
				t.Fatalf("ResolveObservation() = %#v, want %#v", got, test.want)
			}
			if connection := resolver.Resolve(ctx); connection != test.want.ConnectionIP {
				t.Fatalf("Resolve() = %q, want %q", connection, test.want.ConnectionIP)
			}
		})
	}
}

func TestConnectionSourceResolverRejectsUnsupportedConnectionCandidates(t *testing.T) {
	resolver := NewConnectionSourceResolver()
	for _, raw := range []string{
		"192.0.2.1", "198.18.0.1", "240.0.0.1", "224.0.0.1", "0.0.0.0",
		"2001:db8::1", "100::1", "64:ff9b::808:808", "fec0::1", "ff02::1", "::",
	} {
		t.Run(raw, func(t *testing.T) {
			ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: connectionSourceTestAddress(raw)})
			if got := resolver.ResolveObservation(ctx); got != (ConnectionSourceObservation{}) {
				t.Fatalf("ResolveObservation(%s) = %#v, want empty", raw, got)
			}
		})
	}
}

func TestConnectionSourceResolverKeepsPrivateTransportAddressesOutOfGeoIP(t *testing.T) {
	resolver := NewConnectionSourceResolver()
	for _, raw := range []string{"10.0.0.1", "127.0.0.1", "169.254.1.1", "100.64.0.1", "fd00::1", "::1", "fe80::1"} {
		t.Run(raw, func(t *testing.T) {
			ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: connectionSourceTestAddress(raw)})
			got := resolver.ResolveObservation(ctx)
			if got.ConnectionIP == "" || got.ObservedSourceIP != "" {
				t.Fatalf("ResolveObservation(%s) = %#v, want private connection with no GeoIP source", raw, got)
			}
		})
	}
}
