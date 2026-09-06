package agentdata

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	nucleipocdomain "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// contractRuntimeExchangeStream is a deterministic in-process bidi stream.
// It deliberately exposes transport boundaries that are difficult to trigger
// through a normal generated client (half-close, disconnect and cancellation).
type contractRuntimeExchangeStream struct {
	ctx       context.Context
	requests  []*agentdatav1.ExchangeRuntimeArtifactRequest
	recvErr   error
	recvAt    int
	blockAt   int
	block     <-chan struct{}
	entered   chan struct{}
	responses []*agentdatav1.ExchangeRuntimeArtifactResponse
	sendErr   error
	sent      chan struct{}
	mu        sync.Mutex
}

func (stream *contractRuntimeExchangeStream) Context() context.Context { return stream.ctx }

func (stream *contractRuntimeExchangeStream) Send(response *agentdatav1.ExchangeRuntimeArtifactResponse) error {
	if stream.sendErr != nil {
		return stream.sendErr
	}
	stream.mu.Lock()
	stream.responses = append(stream.responses, proto.Clone(response).(*agentdatav1.ExchangeRuntimeArtifactResponse))
	stream.mu.Unlock()
	if stream.sent != nil {
		select {
		case stream.sent <- struct{}{}:
		default:
		}
	}
	return nil
}

func (stream *contractRuntimeExchangeStream) Recv() (*agentdatav1.ExchangeRuntimeArtifactRequest, error) {
	if stream.recvAt == stream.blockAt && stream.block != nil {
		if stream.entered != nil {
			select {
			case stream.entered <- struct{}{}:
			default:
			}
		}
		select {
		case <-stream.block:
		case <-stream.ctx.Done():
			return nil, stream.ctx.Err()
		}
	}
	if stream.recvAt < len(stream.requests) {
		request := stream.requests[stream.recvAt]
		stream.recvAt++
		return request, nil
	}
	if stream.recvErr != nil {
		err := stream.recvErr
		stream.recvErr = nil
		return nil, err
	}
	return nil, io.EOF
}

func (stream *contractRuntimeExchangeStream) SetHeader(metadata.MD) error  { return nil }
func (stream *contractRuntimeExchangeStream) SendHeader(metadata.MD) error { return nil }
func (stream *contractRuntimeExchangeStream) SetTrailer(metadata.MD)       {}
func (stream *contractRuntimeExchangeStream) SendMsg(any) error            { return nil }
func (stream *contractRuntimeExchangeStream) RecvMsg(any) error            { return io.EOF }

func contractExchangeService(t *testing.T, source *nucleiTemplateSourceStub) (*ExecutionArtifactService, *serverExecutionArtifactResolver, *agentdatav1.RuntimeArtifactExchangeBegin) {
	t.Helper()
	resolver, begin := runtimeArtifactResolverForTest(t, source)
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	return service, resolver, begin
}

func contractBeginRequest(begin *agentdatav1.RuntimeArtifactExchangeBegin) *agentdatav1.ExchangeRuntimeArtifactRequest {
	return &agentdatav1.ExchangeRuntimeArtifactRequest{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_Begin{Begin: begin}}
}

func contractMissingEnd(sequence, count uint64) *agentdatav1.ExchangeRuntimeArtifactRequest {
	return &agentdatav1.ExchangeRuntimeArtifactRequest{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_MissingEnd{MissingEnd: &agentdatav1.MissingBlobDigestsEnd{Sequence: sequence, DigestCount: count}}}
}

func contractPublicationAck(digest string, count uint64) *agentdatav1.ExchangeRuntimeArtifactRequest {
	return &agentdatav1.ExchangeRuntimeArtifactRequest{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_PublicationAck{PublicationAck: &agentdatav1.RuntimeArtifactPublicationAck{SnapshotDigest: digest, EntryCount: count}}}
}

func TestExchangeRuntimeArtifactAllowsEmptyMissingList(t *testing.T) {
	source := &nucleiTemplateSourceStub{templates: []nucleipocdomain.POC{nucleiTestPOC("empty-missing", "id: empty-missing\n")}}
	service, resolver, begin := contractExchangeService(t, source)
	exchange, err := resolver.AuthorizeRuntimeArtifactExchange(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, begin)
	if err != nil {
		t.Fatal(err)
	}
	stream := &contractRuntimeExchangeStream{ctx: agentAuthContext(), requests: []*agentdatav1.ExchangeRuntimeArtifactRequest{
		contractBeginRequest(begin),
		contractMissingEnd(1, 0),
		contractPublicationAck(exchange.SnapshotDigest, uint64(len(exchange.Entries))),
	}}
	if err := service.ExchangeRuntimeArtifact(stream); err != nil {
		t.Fatalf("empty missing exchange returned %v", err)
	}
	if len(stream.responses) != 4 || stream.responses[0].GetManifestFrame() == nil || stream.responses[1].GetManifestEnd() == nil || stream.responses[2].GetExchangeEnd() == nil || stream.responses[3].GetExchangeComplete() == nil {
		t.Fatalf("empty missing response sequence = %#v", stream.responses)
	}
	if got := stream.responses[2].GetExchangeEnd().GetSequence(); got != 1 {
		t.Fatalf("empty missing ExchangeEnd sequence = %d, want 1", got)
	}
}

func TestExchangeRuntimeArtifactMaximumCatalogUsesBoundedFrames(t *testing.T) {
	const maxEntries = nucleiTemplateManifestMaxEntries
	digest := "sha256:" + strings.Repeat("a", 64)
	entries := make([]RuntimeArtifactManifestEntry, maxEntries)
	for index := range entries {
		entries[index] = RuntimeArtifactManifestEntry{RelativePath: fmt.Sprintf("http/%06d.yaml", index), Digest: digest, SizeBytes: 1}
	}
	stream := &contractRuntimeExchangeStream{ctx: agentAuthContext()}
	exchange := &AuthorizedRuntimeArtifactExchange{
		Entries:        append([]RuntimeArtifactManifestEntry(nil), entries...),
		SnapshotDigest: digest,
		boundary:       &runtimeArtifactExchangeBoundary{entries: entries, byDigest: map[string][]byte{digest: []byte("x")}, snapshotDigest: digest},
	}
	service := &ExecutionArtifactService{}
	if err := service.sendRuntimeManifest(stream, exchange); err != nil {
		t.Fatal(err)
	}
	if len(stream.responses) < 2 || stream.responses[len(stream.responses)-1].GetManifestEnd() == nil {
		t.Fatalf("maximum catalog response tail = %#v", stream.responses[len(stream.responses)-1:])
	}
	var frameEntries int
	var expectedSequence uint64 = 1
	for _, response := range stream.responses {
		if size := proto.Size(response); size > runtimeExchangeFrameMaxBytes {
			t.Fatalf("serialized manifest frame size = %d, exceeds %d", size, runtimeExchangeFrameMaxBytes)
		}
		if frame := response.GetManifestFrame(); frame != nil {
			if frame.GetSequence() != expectedSequence || len(frame.GetEntries()) == 0 {
				t.Fatalf("manifest frame sequence/entries = %d/%d, want sequence %d", frame.GetSequence(), len(frame.GetEntries()), expectedSequence)
			}
			expectedSequence++
			frameEntries += len(frame.GetEntries())
		}
	}
	end := stream.responses[len(stream.responses)-1].GetManifestEnd()
	if frameEntries != maxEntries || end.GetEntryCount() != maxEntries || end.GetTotalSizeBytes() != maxEntries || end.GetSequence() != expectedSequence {
		t.Fatalf("maximum catalog accounting = entries=%d end=%d bytes=%d sequence=%d/%d", frameEntries, end.GetEntryCount(), end.GetTotalSizeBytes(), end.GetSequence(), expectedSequence)
	}
}

func TestExchangeRuntimeArtifactAdmissionFailsFastAndReleasesAfterCancellation(t *testing.T) {
	source := &nucleiTemplateSourceStub{templates: []nucleipocdomain.POC{nucleiTestPOC("admission", "id: admission\n")}}
	service, resolver, begin := contractExchangeService(t, source)
	if err := service.SetRuntimeExchangeAdmissionLimit(1); err != nil {
		t.Fatal(err)
	}
	firstContext, cancelFirst := context.WithCancel(agentAuthContext())
	defer cancelFirst()
	first := &contractRuntimeExchangeStream{ctx: firstContext, requests: []*agentdatav1.ExchangeRuntimeArtifactRequest{contractBeginRequest(begin)}, blockAt: 1, block: make(chan struct{}), entered: make(chan struct{}, 1)}
	firstDone := make(chan error, 1)
	go func() { firstDone <- service.ExchangeRuntimeArtifact(first) }()
	select {
	case <-first.entered:
	case <-time.After(time.Second):
		t.Fatal("first exchange did not reach its bounded receive")
	}
	second := &contractRuntimeExchangeStream{ctx: agentAuthContext(), requests: []*agentdatav1.ExchangeRuntimeArtifactRequest{contractBeginRequest(begin)}}
	if err := service.ExchangeRuntimeArtifact(second); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("second exchange error = %v, want RESOURCE_EXHAUSTED", err)
	}
	if source.calls != 1 {
		t.Fatalf("catalog was read while admission was exhausted: calls=%d, want 1", source.calls)
	}
	cancelFirst()
	select {
	case err := <-firstDone:
		if status.Code(err) != codes.Canceled {
			t.Fatalf("canceled first exchange error = %v, want CANCELED", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled first exchange did not finish")
	}
	// The admission slot must be returned even when the stream is canceled.
	exchange, err := resolver.AuthorizeRuntimeArtifactExchange(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, begin)
	if err != nil {
		t.Fatal(err)
	}
	third := &contractRuntimeExchangeStream{ctx: agentAuthContext(), requests: []*agentdatav1.ExchangeRuntimeArtifactRequest{contractBeginRequest(begin), contractMissingEnd(1, 0), contractPublicationAck(exchange.SnapshotDigest, uint64(len(exchange.Entries)))}}
	if err := service.ExchangeRuntimeArtifact(third); err != nil {
		t.Fatalf("exchange after cancellation was not admitted: %v", err)
	}
}

func TestExchangeRuntimeArtifactKeepsOneBoundaryDuringConcurrentCatalogMutation(t *testing.T) {
	const original = "id: concurrent\n"
	source := &nucleiTemplateSourceStub{templates: []nucleipocdomain.POC{nucleiTestPOC("concurrent", original)}}
	service, resolver, begin := contractExchangeService(t, source)
	exchange, err := resolver.AuthorizeRuntimeArtifactExchange(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, begin)
	if err != nil {
		t.Fatal(err)
	}
	baselineReads := source.calls
	block := make(chan struct{})
	stream := &contractRuntimeExchangeStream{
		ctx:     agentAuthContext(),
		blockAt: 1,
		block:   block,
		entered: make(chan struct{}, 1),
		requests: []*agentdatav1.ExchangeRuntimeArtifactRequest{
			contractBeginRequest(begin),
			{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_MissingBlobDigests{MissingBlobDigests: &agentdatav1.MissingBlobDigests{Sequence: 1, Digests: []string{exchange.Entries[0].Digest}}}},
			contractMissingEnd(2, 1),
			contractPublicationAck(exchange.SnapshotDigest, uint64(len(exchange.Entries))),
		},
		sent: make(chan struct{}, 4),
	}
	done := make(chan error, 1)
	go func() { done <- service.ExchangeRuntimeArtifact(stream) }()
	// Manifest and ManifestEnd must have crossed the stream before the
	// receive side is released. This is the exact interval in which a catalog
	// mutation would otherwise create a manifest/blob split.
	for sent := 0; sent < 2; sent++ {
		select {
		case <-stream.sent:
		case <-time.After(time.Second):
			t.Fatal("exchange did not publish its complete manifest")
		}
	}
	source.templates[0] = nucleiTestPOC("concurrent", "id: concurrent\ninfo:\n  name: changed\n")
	close(block)
	if err := <-done; err != nil {
		t.Fatalf("exchange after concurrent catalog mutation = %v", err)
	}
	var blob string
	for _, response := range stream.responses {
		if group := response.GetBlobGroup(); group != nil {
			blob += string(group.GetData())
		}
	}
	if blob != original {
		t.Fatalf("boundary blob after concurrent mutation = %q, want %q", blob, original)
	}
	if source.calls != baselineReads+1 {
		t.Fatalf("catalog reads during one service exchange = %d, want baseline %d + 1", source.calls, baselineReads)
	}
}

func TestExchangeRuntimeArtifactTransportTerminationAndTerminalErrorsFailClosed(t *testing.T) {
	source := &nucleiTemplateSourceStub{templates: []nucleipocdomain.POC{nucleiTestPOC("termination", "id: termination\n")}}
	service, resolver, begin := contractExchangeService(t, source)
	exchange, err := resolver.AuthorizeRuntimeArtifactExchange(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, begin)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		recvErr  error
		requests []*agentdatav1.ExchangeRuntimeArtifactRequest
		want     codes.Code
	}{
		{name: "half-close before MissingEnd", recvErr: io.EOF, requests: []*agentdatav1.ExchangeRuntimeArtifactRequest{contractBeginRequest(begin)}, want: codes.DataLoss},
		{name: "disconnect", recvErr: errors.New("peer disconnected"), requests: []*agentdatav1.ExchangeRuntimeArtifactRequest{contractBeginRequest(begin)}, want: codes.Unavailable},
		{name: "missing terminal sequence", requests: []*agentdatav1.ExchangeRuntimeArtifactRequest{contractBeginRequest(begin), contractMissingEnd(2, 0)}, want: codes.InvalidArgument},
		{name: "publication digest mismatch", requests: []*agentdatav1.ExchangeRuntimeArtifactRequest{contractBeginRequest(begin), contractMissingEnd(1, 0), contractPublicationAck("sha256:"+strings.Repeat("f", 64), uint64(len(exchange.Entries)))}, want: codes.DataLoss},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stream := &contractRuntimeExchangeStream{ctx: agentAuthContext(), requests: test.requests, recvErr: test.recvErr}
			if got := status.Code(service.ExchangeRuntimeArtifact(stream)); got != test.want {
				t.Fatalf("status = %s, want %s", got, test.want)
			}
		})
	}
}

func TestExchangeRuntimeArtifactBoundaryIsStableAcrossCatalogMutation(t *testing.T) {
	oldContent := "id: stable\n"
	source := &nucleiTemplateSourceStub{templates: []nucleipocdomain.POC{nucleiTestPOC("stable", oldContent)}}
	resolver, begin := runtimeArtifactResolverForTest(t, source)
	exchange, err := resolver.AuthorizeRuntimeArtifactExchange(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, begin)
	if err != nil {
		t.Fatal(err)
	}
	oldDigest := exchange.Entries[0].Digest
	source.templates[0] = nucleiTestPOC("stable", "id: stable\ninfo:\n  name: changed\n")
	blob, ok := exchange.blob(oldDigest)
	if !ok || string(blob) != oldContent {
		t.Fatalf("exchange boundary changed after catalog mutation: ok=%t blob=%q", ok, blob)
	}
	if source.calls != 1 {
		t.Fatalf("catalog read count = %d, want one read boundary", source.calls)
	}
}

func TestExchangeRuntimeArtifactBoundariesAreIndependentAcrossTasks(t *testing.T) {
	source := &nucleiTemplateSourceStub{templates: []nucleipocdomain.POC{nucleiTestPOC("independent", "id: independent\n")}}
	resolverA, beginA := runtimeArtifactResolverForTest(t, source)
	resolverB, beginB := runtimeArtifactResolverForTest(t, source)
	lease := AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}
	left, err := resolverA.AuthorizeRuntimeArtifactExchange(context.Background(), lease, beginA)
	if err != nil {
		t.Fatal(err)
	}
	right, err := resolverB.AuthorizeRuntimeArtifactExchange(context.Background(), lease, beginB)
	if err != nil {
		t.Fatal(err)
	}
	if left == right || left.boundary == right.boundary {
		t.Fatal("independent tasks share an exchange boundary")
	}
	left.close()
	if blob, ok := right.blob(right.Entries[0].Digest); !ok || len(blob) == 0 {
		t.Fatal("closing one task exchange invalidated another task boundary")
	}
	if source.calls != 2 {
		t.Fatalf("catalog reads = %d, want one per task exchange", source.calls)
	}
}
