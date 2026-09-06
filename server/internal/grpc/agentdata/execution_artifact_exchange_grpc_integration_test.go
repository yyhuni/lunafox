package agentdata

import (
	"context"
	"io"
	"net"
	"testing"

	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	nucleipocdomain "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const runtimeArtifactIntegrationBuffer = 256 * 1024

// TestExchangeRuntimeArtifactGeneratedGRPCBoundary exercises the generated
// client/server stubs rather than the in-process stream doubles.  In
// particular, it proves that authentication metadata, half-close and the
// manifest/blob terminal ordering survive the actual gRPC transport.
func TestExchangeRuntimeArtifactGeneratedGRPCBoundary(t *testing.T) {
	source := &nucleiTemplateSourceStub{templates: []nucleipocdomain.POC{nucleiTestPOC("grpc", "id: grpc\n")}}
	resolver, begin := runtimeArtifactResolverForTest(t, source)
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	listener := bufconn.Listen(runtimeArtifactIntegrationBuffer)
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(runtimeExchangeFrameMaxBytes),
		grpc.MaxSendMsgSize(runtimeExchangeFrameMaxBytes),
	)
	agentdatav1.RegisterExecutionArtifactServiceServer(grpcServer, service)
	serveErr := make(chan error, 1)
	go func() { serveErr <- grpcServer.Serve(listener) }()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
		select {
		case <-serveErr:
		default:
		}
	})

	conn, err := grpc.NewClient(
		"passthrough:///runtime-artifact-integration",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := agentdatav1.NewExecutionArtifactServiceClient(conn)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token",
		grpcauth.AgentSessionIDMetadataKey, testAgentSessionID,
		grpcauth.AgentSessionEpochMetadataKey, "11",
	))

	stream, err := client.ExchangeRuntimeArtifact(ctx,
		grpc.MaxCallRecvMsgSize(runtimeExchangeFrameMaxBytes),
		grpc.MaxCallSendMsgSize(runtimeExchangeFrameMaxBytes),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(contractBeginRequest(begin)); err != nil {
		t.Fatal(err)
	}
	manifest := recvRuntimeIntegrationResponse(t, stream)
	if manifest.GetManifestFrame() == nil {
		t.Fatalf("first response = %T, want manifest frame", manifest.GetPayload())
	}
	manifestEnd := recvRuntimeIntegrationResponse(t, stream)
	if manifestEnd.GetManifestEnd() == nil {
		t.Fatalf("second response = %T, want manifest end", manifestEnd.GetPayload())
	}
	entry := manifest.GetManifestFrame().GetEntries()[0]
	oldDigest := entry.GetSha256Digest()
	oldContent := "id: grpc\n"
	// Mutate the source after the manifest has crossed the transport. The
	// requested blob must still come from the exchange's original boundary.
	source.templates[0] = nucleiTestPOC("grpc", "id: grpc\ninfo:\n  name: changed\n")
	if err := stream.Send(&agentdatav1.ExchangeRuntimeArtifactRequest{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_MissingBlobDigests{MissingBlobDigests: &agentdatav1.MissingBlobDigests{Sequence: 1, Digests: []string{oldDigest}}}}); err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(contractMissingEnd(2, 1)); err != nil {
		t.Fatal(err)
	}
	blob := recvRuntimeIntegrationResponse(t, stream)
	if blob.GetBlobGroup() == nil || string(blob.GetBlobGroup().GetData()) != oldContent {
		t.Fatalf("boundary blob = %#v, want original content %q", blob.GetPayload(), oldContent)
	}
	blobEnd := recvRuntimeIntegrationResponse(t, stream)
	if blobEnd.GetBlobEnd() == nil || blobEnd.GetBlobEnd().GetSha256Digest() != oldDigest {
		t.Fatalf("blob terminal = %#v", blobEnd.GetPayload())
	}
	exchangeEnd := recvRuntimeIntegrationResponse(t, stream)
	if exchangeEnd.GetExchangeEnd() == nil || exchangeEnd.GetExchangeEnd().GetSequence() != 2 {
		t.Fatalf("exchange terminal = %#v", exchangeEnd.GetPayload())
	}
	if err := stream.Send(contractPublicationAck(manifestEnd.GetManifestEnd().GetSnapshotDigest(), 1)); err != nil {
		t.Fatal(err)
	}
	complete := recvRuntimeIntegrationResponse(t, stream)
	if complete.GetExchangeComplete() == nil || complete.GetExchangeComplete().GetSnapshotDigest() != manifestEnd.GetManifestEnd().GetSnapshotDigest() {
		t.Fatalf("exchange complete = %#v", complete.GetPayload())
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatal(err)
	}
	if source.calls != 1 {
		t.Fatalf("catalog reads during one generated exchange = %d, want 1", source.calls)
	}

	// A second exchange starts a fresh boundary and may observe the mutation.
	second, err := client.ExchangeRuntimeArtifact(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Send(contractBeginRequest(begin)); err != nil {
		t.Fatal(err)
	}
	secondManifest := recvRuntimeIntegrationResponse(t, second)
	secondEnd := recvRuntimeIntegrationResponse(t, second)
	if secondManifest.GetManifestFrame() == nil || secondEnd.GetManifestEnd() == nil || secondManifest.GetManifestFrame().GetEntries()[0].GetSha256Digest() == oldDigest {
		t.Fatalf("second exchange did not observe the new catalog boundary: %#v/%#v", secondManifest.GetPayload(), secondEnd.GetPayload())
	}
	if err := second.Send(contractMissingEnd(1, 0)); err != nil {
		t.Fatal(err)
	}
	if got := recvRuntimeIntegrationResponse(t, second).GetExchangeEnd(); got == nil || got.GetSequence() != 1 {
		t.Fatalf("second exchange end = %#v", got)
	}
	if err := second.Send(contractPublicationAck(secondEnd.GetManifestEnd().GetSnapshotDigest(), 1)); err != nil {
		t.Fatal(err)
	}
	if got := recvRuntimeIntegrationResponse(t, second).GetExchangeComplete(); got == nil {
		t.Fatal("second exchange did not complete")
	}
	if err := second.CloseSend(); err != nil {
		t.Fatal(err)
	}
	if source.calls != 2 {
		t.Fatalf("catalog reads across independent exchanges = %d, want 2", source.calls)
	}
}

func recvRuntimeIntegrationResponse(t *testing.T, stream grpc.BidiStreamingClient[agentdatav1.ExchangeRuntimeArtifactRequest, agentdatav1.ExchangeRuntimeArtifactResponse]) *agentdatav1.ExchangeRuntimeArtifactResponse {
	t.Helper()
	response, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			t.Fatal("runtime artifact exchange ended before terminal frame")
		}
		t.Fatalf("receive runtime artifact response: %v (status %s)", err, status.Code(err))
	}
	return response
}
