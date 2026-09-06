package agentdata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type executionArtifactResolverStub struct {
	input         AuthorizedExecutionArtifact
	config        AuthorizedExecutionArtifact
	platform      AuthorizedExecutionArtifact
	inputCalls    int
	configCalls   int
	platformCalls int
	lastLease     AgentExecutionLease
}

func (stub *executionArtifactResolverStub) AuthorizeExecutionInput(_ context.Context, lease AgentExecutionLease, _ *agentdatav1.StreamExecutionInputRequest) (AuthorizedExecutionArtifact, error) {
	stub.inputCalls++
	stub.lastLease = lease
	return stub.input, nil
}
func (stub *executionArtifactResolverStub) AuthorizeConfigResource(_ context.Context, lease AgentExecutionLease, _ *agentdatav1.StreamConfigResourceRequest) (AuthorizedExecutionArtifact, error) {
	stub.configCalls++
	stub.lastLease = lease
	return stub.config, nil
}
func (stub *executionArtifactResolverStub) AuthorizePlatformResource(_ context.Context, lease AgentExecutionLease, _ *agentdatav1.StreamPlatformResourceContentRequest) (AuthorizedExecutionArtifact, error) {
	stub.platformCalls++
	stub.lastLease = lease
	return stub.platform, nil
}
type executionArtifactFrameStream struct {
	ctx    context.Context
	frames []*agentdatav1.ExecutionArtifactFrame
	err    error
}

func (stream *executionArtifactFrameStream) Context() context.Context { return stream.ctx }
func (stream *executionArtifactFrameStream) Send(frame *agentdatav1.ExecutionArtifactFrame) error {
	if stream.err != nil {
		return stream.err
	}
	stream.frames = append(stream.frames, proto.Clone(frame).(*agentdatav1.ExecutionArtifactFrame))
	return nil
}
func (stream *executionArtifactFrameStream) SetHeader(metadata.MD) error  { return nil }
func (stream *executionArtifactFrameStream) SendHeader(metadata.MD) error { return nil }
func (stream *executionArtifactFrameStream) SetTrailer(metadata.MD)       {}
func (stream *executionArtifactFrameStream) SendMsg(any) error            { return nil }
func (stream *executionArtifactFrameStream) RecvMsg(any) error            { return io.EOF }

func executionArtifactTestSource(content string, count uint64) AuthorizedExecutionArtifact {
	return executionArtifactTestSourceWithType("application/vnd.lunafox.subdomains.v1", content, count)
}

func executionArtifactTestSourceWithType(contentType, content string, count uint64) AuthorizedExecutionArtifact {
	return AuthorizedExecutionArtifact{
		ContentType: contentType,
		Produce: func(_ context.Context, writer io.Writer) (uint64, error) {
			_, err := io.Copy(writer, strings.NewReader(content))
			return count, err
		},
	}
}

func TestExecutionArtifactServiceStreamsHeaderChunksTrailerWithExactDigest(t *testing.T) {
	content := strings.Repeat("example.com\n", 7000)
	resolver := &executionArtifactResolverStub{input: executionArtifactTestSource(content, 7000)}
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
	request := &agentdatav1.StreamExecutionInputRequest{
		Task: "scans/1/tasks/1", Execution: "executions/1",
		Role: executionartifact.RoleSubdomains,
	}
	if err := service.StreamExecutionInput(request, stream); err != nil {
		t.Fatalf("StreamExecutionInput returned error: %v", err)
	}
	if len(stream.frames) < 3 || stream.frames[0].GetHeader() == nil || stream.frames[len(stream.frames)-1].GetTrailer() == nil {
		t.Fatalf("unexpected frame sequence of length %d", len(stream.frames))
	}
	for index := 1; index < len(stream.frames)-1; index++ {
		chunk := stream.frames[index].GetChunk()
		if chunk == nil || len(chunk.GetData()) == 0 || len(chunk.GetData()) > executionArtifactChunkMaxBytes {
			t.Fatalf("invalid chunk[%d]: %#v", index, stream.frames[index])
		}
	}
	hasher := sha256.Sum256([]byte(content))
	trailer := stream.frames[len(stream.frames)-1].GetTrailer().GetIntegrity()
	if trailer.GetSizeBytes() != uint64(len(content)) || trailer.GetSha256Digest() != "sha256:"+hex.EncodeToString(hasher[:]) || trailer.GetRecordCount() != 7000 {
		t.Fatalf("unexpected trailer: %#v", trailer)
	}
	if resolver.inputCalls != 1 {
		t.Fatalf("resolver input calls = %d, want 1", resolver.inputCalls)
	}
	if resolver.lastLease != (AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}) {
		t.Fatalf("resolver Agent lease = %#v", resolver.lastLease)
	}
}

func TestExecutionArtifactServiceAllowsLegalZeroSubdomainsWithoutChunk(t *testing.T) {
	resolver := &executionArtifactResolverStub{input: executionArtifactTestSourceWithType("application/vnd.lunafox.subdomains.v1", "", 0)}
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
	err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{
		Task: "scans/1/tasks/1", Execution: "executions/1",
		Role: executionartifact.RoleSubdomains,
	}, stream)
	if err != nil {
		t.Fatalf("zero Subdomains returned error: %v", err)
	}
	if len(stream.frames) != 2 || stream.frames[0].GetHeader() == nil || stream.frames[1].GetTrailer() == nil {
		t.Fatalf("zero Subdomains frames = %#v", stream.frames)
	}
	if stream.frames[1].GetTrailer().GetIntegrity().GetSizeBytes() != 0 || stream.frames[1].GetTrailer().GetIntegrity().GetRecordCount() != 0 {
		t.Fatalf("zero Subdomains integrity = %#v", stream.frames[1].GetTrailer().GetIntegrity())
	}
}

func TestExecutionArtifactServiceAllowsLegalZeroHostPortsWithoutChunk(t *testing.T) {
	resolver := &executionArtifactResolverStub{input: executionArtifactTestSourceWithType("application/vnd.lunafox.host-ports.v1", "", 0)}
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
	err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{
		Task: "scans/1/tasks/1", Execution: "executions/1",
		Role: executionartifact.RoleHostPorts,
	}, stream)
	if err != nil {
		t.Fatalf("zero HostPorts returned error: %v", err)
	}
	if len(stream.frames) != 2 || stream.frames[0].GetHeader() == nil || stream.frames[1].GetTrailer() == nil {
		t.Fatalf("zero HostPorts frames = %#v", stream.frames)
	}
}

func TestExecutionArtifactServiceAllowsLegalZeroWebsiteURLsWithoutChunk(t *testing.T) {
	resolver := &executionArtifactResolverStub{input: AuthorizedExecutionArtifact{
		ContentType: executionartifact.ContentTypeWebsiteURLs,
		Produce:     func(context.Context, io.Writer) (uint64, error) { return 0, nil },
	}}
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
	err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{
		Task: "scans/1/tasks/1", Execution: "executions/1",
		Role: executionartifact.RoleWebsiteURLs,
	}, stream)
	if err != nil || len(stream.frames) != 2 || stream.frames[0].GetHeader() == nil || stream.frames[1].GetTrailer() == nil {
		t.Fatalf("zero WebsiteURLs status/frames = %v/%#v", err, stream.frames)
	}
}

func TestExecutionArtifactServiceWordlistExpectedIntegrityMismatchStopsBeforeTrailer(t *testing.T) {
	expected := &agentdatav1.ExecutionArtifactIntegrity{SizeBytes: 3, Sha256Digest: "sha256:" + strings.Repeat("a", 64), RecordCount: pointerUint64(1)}
	resolver := &executionArtifactResolverStub{config: AuthorizedExecutionArtifact{
		ContentType:       "application/vnd.lunafox.wordlist.v1",
		ExpectedIntegrity: expected,
		Produce: func(_ context.Context, writer io.Writer) (uint64, error) {
			_, err := io.WriteString(writer, "abc\n")
			return 1, err
		},
	}}
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
	err := service.StreamConfigResource(&agentdatav1.StreamConfigResourceRequest{Task: "scans/1/tasks/1", Execution: "executions/1", SectionId: "bruteforce", ParamKey: "wordlist"}, stream)
	if status.Code(err) != codes.DataLoss {
		t.Fatalf("status = %s, want DATA_LOSS; err=%v", status.Code(err), err)
	}
	if len(stream.frames) == 0 || stream.frames[len(stream.frames)-1].GetTrailer() != nil {
		t.Fatalf("mismatched wordlist must not publish trailer: %#v", stream.frames)
	}
}

func TestExecutionArtifactServiceWordlistDataLossPreflightStopsBeforeHeaderAndProducer(t *testing.T) {
	preflightCalls := 0
	producerCalls := 0
	resolver := &executionArtifactResolverStub{config: AuthorizedExecutionArtifact{
		ContentType: executionartifact.ContentTypeWordlist,
		ExpectedIntegrity: &agentdatav1.ExecutionArtifactIntegrity{
			SizeBytes: 1, Sha256Digest: "sha256:" + strings.Repeat("a", 64), RecordCount: pointerUint64(1),
		},
		Preflight: func(context.Context) error {
			preflightCalls++
			return ErrExecutionArtifactDataLoss
		},
		Produce: func(context.Context, io.Writer) (uint64, error) {
			producerCalls++
			return 0, nil
		},
	}}
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

	err := service.StreamConfigResource(&agentdatav1.StreamConfigResourceRequest{
		Task: "scans/1/tasks/1", Execution: "executions/1", SectionId: "bruteforce", ParamKey: "wordlist",
	}, stream)
	if status.Code(err) != codes.DataLoss || len(stream.frames) != 0 {
		t.Fatalf("status=%s frames=%d, want DATA_LOSS before header; err=%v", status.Code(err), len(stream.frames), err)
	}
	if preflightCalls != 1 || producerCalls != 0 {
		t.Fatalf("preflight/producer calls = %d/%d, want 1/0", preflightCalls, producerCalls)
	}
}

func TestExecutionArtifactServiceFingerprintLibraryRequiresAndVerifiesIntegrity(t *testing.T) {
	content := "{\"fingerprint\":[]}\n"
	digest := sha256.Sum256([]byte(content))
	recordCount := uint64(0)
	expected := &agentdatav1.ExecutionArtifactIntegrity{
		SizeBytes: uint64(len(content)), Sha256Digest: "sha256:" + hex.EncodeToString(digest[:]), RecordCount: &recordCount,
	}
	resolver := &executionArtifactResolverStub{platform: AuthorizedExecutionArtifact{
		ContentType:       executionartifact.ContentTypeFingerprintLibraryFingerPrintHub,
		ExpectedIntegrity: expected,
		Produce: func(_ context.Context, writer io.Writer) (uint64, error) {
			_, err := io.WriteString(writer, content)
			return 0, err
		},
	}}
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
	err := service.StreamPlatformResourceContent(&agentdatav1.StreamPlatformResourceContentRequest{
		Task: "scans/1/tasks/1", Execution: "executions/1", ResourceId: "fingerprintLibraryFingerPrintHub",
	}, stream)
	if err != nil {
		t.Fatalf("StreamPlatformResourceContent() error = %v", err)
	}
	if len(stream.frames) != 3 || stream.frames[0].GetHeader().GetExpectedIntegrity() == nil || stream.frames[2].GetTrailer() == nil {
		t.Fatalf("fingerprint stream frames = %#v", stream.frames)
	}
	if resolver.platformCalls != 1 {
		t.Fatalf("platform resolver calls = %d, want 1", resolver.platformCalls)
	}

	resolver.platform.ExpectedIntegrity = nil
	stream = &executionArtifactFrameStream{ctx: agentAuthContext()}
	err = service.StreamPlatformResourceContent(&agentdatav1.StreamPlatformResourceContentRequest{
		Task: "scans/1/tasks/1", Execution: "executions/1", ResourceId: "fingerprintLibraryFingerPrintHub",
	}, stream)
	if status.Code(err) != codes.Internal || len(stream.frames) != 0 {
		t.Fatalf("missing fingerprint integrity status/frames = %s/%#v", status.Code(err), stream.frames)
	}
}

func TestExecutionArtifactServiceRejectsMissingSelectorBeforeResolver(t *testing.T) {
	resolver := &executionArtifactResolverStub{input: executionArtifactTestSource("x\n", 1)}
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
	err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{Task: "scans/1/tasks/1", Execution: "executions/1"}, stream)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status = %s, want INVALID_ARGUMENT", status.Code(err))
	}
	if resolver.inputCalls != 0 {
		t.Fatal("resolver must not run for malformed selector")
	}
}

func TestExecutionArtifactServiceRejectsUnknownExecutionInputRolesBeforeResolver(t *testing.T) {
	for _, role := range []string{"unknown", "Subdomains", "subdomains ", "host_ports"} {
		t.Run(role, func(t *testing.T) {
			resolver := &executionArtifactResolverStub{input: executionArtifactTestSource("x\n", 1)}
			service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

			err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{
				Task: "scans/1/tasks/1", Execution: "executions/1", Role: role,
			}, stream)
			if status.Code(err) != codes.InvalidArgument || resolver.inputCalls != 0 || len(stream.frames) != 0 {
				t.Fatalf("role %q status=%s resolver calls=%d frames=%d err=%v", role, status.Code(err), resolver.inputCalls, len(stream.frames), err)
			}
		})
	}
}

func TestExecutionArtifactServiceClassifiesAgentTokenLookupFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		finder AgentFinder
		want   codes.Code
	}{
		{name: "agent not found", finder: &agentFinderStub{err: agentdomain.ErrAgentNotFound}, want: codes.Unauthenticated},
		{name: "nil agent", finder: &agentFinderStub{}, want: codes.Unauthenticated},
		{name: "invalid agent", finder: &agentFinderStub{agent: &agentdomain.Agent{}}, want: codes.Unauthenticated},
		{name: "lookup canceled", finder: &agentFinderStub{err: context.Canceled}, want: codes.Canceled},
		{name: "lookup deadline exceeded", finder: &agentFinderStub{err: context.DeadlineExceeded}, want: codes.DeadlineExceeded},
		{name: "agent store unavailable", finder: &agentFinderStub{err: errors.New("database unavailable")}, want: codes.Unavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolver := &executionArtifactResolverStub{input: executionArtifactTestSource("x\n", 1)}
			service := NewExecutionArtifactService(test.finder, resolver)
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

			err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{
				Task: "scans/1/tasks/1", Execution: "executions/1",
				Role: executionartifact.RoleSubdomains,
			}, stream)
			if status.Code(err) != test.want {
				t.Fatalf("status = %s, want %s; err=%v", status.Code(err), test.want, err)
			}
			if resolver.inputCalls != 0 || len(stream.frames) != 0 {
				t.Fatalf("authentication failure reached resolver/stream: calls=%d frames=%d", resolver.inputCalls, len(stream.frames))
			}
		})
	}
}

func TestExecutionArtifactServiceRejectsMissingOrAmbiguousProcessSessionMetadata(t *testing.T) {
	for _, md := range []metadata.MD{
		metadata.Pairs(grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token"),
		metadata.Pairs(
			grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token",
			grpcauth.AgentSessionIDMetadataKey, testAgentSessionID,
			grpcauth.AgentSessionEpochMetadataKey, "11",
			grpcauth.AgentSessionEpochMetadataKey, "11",
		),
	} {
		resolver := &executionArtifactResolverStub{input: executionArtifactTestSource("x\n", 1)}
		service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
		stream := &executionArtifactFrameStream{ctx: metadata.NewIncomingContext(context.Background(), md)}
		err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{
			Task: "scans/1/tasks/1", Execution: "executions/1",
			Role: executionartifact.RoleSubdomains,
		}, stream)
		if status.Code(err) != codes.FailedPrecondition || resolver.inputCalls != 0 {
			t.Fatalf("session metadata status=%s calls=%d err=%v", status.Code(err), resolver.inputCalls, err)
		}
	}
}

func TestExecutionArtifactChunkAndFrameLimitsAreEnforced(t *testing.T) {
	frame := &agentdatav1.ExecutionArtifactFrame{Payload: &agentdatav1.ExecutionArtifactFrame_Chunk{Chunk: &agentdatav1.ExecutionArtifactChunk{Data: make([]byte, executionArtifactChunkMaxBytes+1)}}}
	if status.Code(sendExecutionArtifactFrame(func(*agentdatav1.ExecutionArtifactFrame) error { return nil }, frame)) != codes.Internal {
		t.Fatal("oversized chunk must be rejected")
	}
	frame = &agentdatav1.ExecutionArtifactFrame{Payload: &agentdatav1.ExecutionArtifactFrame_Header{Header: &agentdatav1.ExecutionArtifactHeader{Task: strings.Repeat("x", executionArtifactFrameMaxBytes)}}}
	if status.Code(sendExecutionArtifactFrame(func(*agentdatav1.ExecutionArtifactFrame) error { return nil }, frame)) != codes.Internal {
		t.Fatal("oversized serialized frame must be rejected")
	}
}

func TestExecutionArtifactErrorMappingRetainsRetryableUnavailableOnlyAtServerBoundary(t *testing.T) {
	if status.Code(mapExecutionArtifactError(ErrExecutionArtifactUnavailable)) != codes.Unavailable {
		t.Fatal("unavailable must map to UNAVAILABLE")
	}
	if status.Code(mapExecutionArtifactError(ErrExecutionArtifactDataLoss)) != codes.DataLoss {
		t.Fatal("data loss must map to DATA_LOSS")
	}
	if status.Code(mapExecutionArtifactError(errors.New("producer"))) != codes.Internal {
		t.Fatal("unknown producer error must map to INTERNAL")
	}
}

func pointerUint64(value uint64) *uint64 { return &value }
