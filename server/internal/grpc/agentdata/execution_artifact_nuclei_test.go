package agentdata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"reflect"
	"testing"

	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	nucleipocdomain "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type nucleiTemplateSourceStub struct {
	templates []nucleipocdomain.POC
	err       error
	calls     int
}

func (stub *nucleiTemplateSourceStub) ListEnabledForExecution(context.Context) ([]nucleipocdomain.POC, error) {
	stub.calls++
	if stub.err != nil {
		return nil, stub.err
	}
	return append([]nucleipocdomain.POC(nil), stub.templates...), nil
}

func nucleiTestPOC(id, content string) nucleipocdomain.POC {
	digest := sha256.Sum256([]byte(content))
	return nucleipocdomain.POC{TemplateID: id, RelativePath: "http/" + id + ".yaml", Content: content, ContentSHA256: hex.EncodeToString(digest[:]), IsEnabled: true}
}

func runtimeArtifactResolverForTest(t *testing.T, source *nucleiTemplateSourceStub) (*serverExecutionArtifactResolver, *agentdatav1.RuntimeArtifactExchangeBegin) {
	t.Helper()
	plan := validExecutionArtifactPlan()
	plan.EngineRelease.Engine = "engine.lunafox.nuclei_vulnerability"
	plan.WorkflowStep.StageId = "nuclei_vulnerability"
	plan.WorkflowStep.StepId = "nuclei_vulnerability"
	plan.RuntimeArtifactBindings = []*agentexecutionv1.RuntimeArtifactBinding{{ArtifactId: "nucleiTemplates", ContentType: "application/vnd.lunafox.nucleiTemplates.v1"}}
	resolver := newExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
	concrete := resolver.(*serverExecutionArtifactResolver)
	concrete.nucleiTemplates = source
	return concrete, &agentdatav1.RuntimeArtifactExchangeBegin{Task: plan.GetTask(), Execution: plan.GetExecution(), ArtifactId: "nucleiTemplates", CompatibilityRevision: testCompatibilityRevision()}
}

func TestBuildNucleiTemplateExchangeIsDeterministicAndDigestAddressed(t *testing.T) {
	left, err := buildNucleiTemplateExchange([]nucleipocdomain.POC{nucleiTestPOC("z", "id: z\n"), nucleiTestPOC("a", "id: a\n")})
	if err != nil {
		t.Fatal(err)
	}
	right, err := buildNucleiTemplateExchange([]nucleipocdomain.POC{nucleiTestPOC("a", "id: a\n"), nucleiTestPOC("z", "id: z\n")})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left.entries, right.entries) || left.snapshotDigest != right.snapshotDigest {
		t.Fatalf("exchange identity differs: %#v/%#v digest=%q/%q", left.entries, right.entries, left.snapshotDigest, right.snapshotDigest)
	}
	if got := left.entries[0].RelativePath; got != "http/a.yaml" {
		t.Fatalf("entries are not sorted by relative path: %q", got)
	}
	if len(left.byDigest) != 2 {
		t.Fatalf("blob count = %d, want 2", len(left.byDigest))
	}
}

func TestBuildNucleiTemplateExchangeRejectsInvalidCatalogRows(t *testing.T) {
	tests := []struct {
		name string
		rows []nucleipocdomain.POC
	}{
		{name: "empty", rows: nil},
		{name: "digest drift", rows: []nucleipocdomain.POC{{TemplateID: "x", RelativePath: "http/x.yaml", Content: "id: x\n", ContentSHA256: "00", IsEnabled: true}}},
		{name: "unsafe path", rows: []nucleipocdomain.POC{{TemplateID: "x", RelativePath: "../x.yaml", Content: "id: x\n", ContentSHA256: digestForNucleiTest("id: x\n"), IsEnabled: true}}},
		{name: "multiple documents", rows: []nucleipocdomain.POC{{TemplateID: "x", RelativePath: "http/x.yaml", Content: "id: x\n---\nid: x\n", ContentSHA256: digestForNucleiTest("id: x\n---\nid: x\n"), IsEnabled: true}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := buildNucleiTemplateExchange(test.rows); err == nil {
				t.Fatal("invalid catalog was accepted")
			}
		})
	}
}

func TestAuthorizeRuntimeArtifactExchangeRequiresPlanBinding(t *testing.T) {
	source := &nucleiTemplateSourceStub{templates: []nucleipocdomain.POC{nucleiTestPOC("x", "id: x\n")}}
	plan := validExecutionArtifactPlan()
	resolver := newExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
	concrete := resolver.(*serverExecutionArtifactResolver)
	concrete.nucleiTemplates = source
	_, err := concrete.AuthorizeRuntimeArtifactExchange(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, &agentdatav1.RuntimeArtifactExchangeBegin{Task: plan.GetTask(), Execution: plan.GetExecution(), ArtifactId: "nucleiTemplates", CompatibilityRevision: testCompatibilityRevision()})
	if status.Code(err) != codes.PermissionDenied && !errors.Is(err, ErrExecutionArtifactPermissionDenied) {
		t.Fatalf("authorization error = %v, want permission denied", err)
	}
	if source.calls != 0 {
		t.Fatalf("catalog was read %d times without a plan binding", source.calls)
	}
}

func TestExchangeRuntimeArtifactCompletesManifestBlobAndPublicationAck(t *testing.T) {
	source := &nucleiTemplateSourceStub{templates: []nucleipocdomain.POC{nucleiTestPOC("x", "id: x\n")}}
	resolver, begin := runtimeArtifactResolverForTest(t, source)
	exchange, err := resolver.AuthorizeRuntimeArtifactExchange(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, begin)
	if err != nil {
		t.Fatal(err)
	}
	stream := &runtimeExchangeTestStream{ctx: agentAuthContext()}
	stream.requests = []*agentdatav1.ExchangeRuntimeArtifactRequest{
		{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_Begin{Begin: begin}},
		{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_MissingBlobDigests{MissingBlobDigests: &agentdatav1.MissingBlobDigests{Sequence: 1, Digests: []string{exchange.Entries[0].Digest}}}},
		{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_MissingEnd{MissingEnd: &agentdatav1.MissingBlobDigestsEnd{Sequence: 2, DigestCount: 1}}},
		{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_PublicationAck{PublicationAck: &agentdatav1.RuntimeArtifactPublicationAck{SnapshotDigest: exchange.SnapshotDigest, EntryCount: 1}}},
	}
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	if err := service.ExchangeRuntimeArtifact(stream); err != nil {
		t.Fatal(err)
	}
	if len(stream.responses) != 6 {
		t.Fatalf("response count = %d, want manifest, end, blob, blob end, exchange end, complete", len(stream.responses))
	}
	if stream.responses[0].GetManifestFrame() == nil || stream.responses[1].GetManifestEnd() == nil || stream.responses[2].GetBlobGroup() == nil || stream.responses[3].GetBlobEnd() == nil || stream.responses[4].GetExchangeEnd() == nil || stream.responses[5].GetExchangeComplete() == nil {
		t.Fatalf("unexpected exchange response sequence: %#v", stream.responses)
	}
}

func TestExchangeRuntimeArtifactRejectsDuplicateMissingDigest(t *testing.T) {
	source := &nucleiTemplateSourceStub{templates: []nucleipocdomain.POC{nucleiTestPOC("x", "id: x\n")}}
	resolver, begin := runtimeArtifactResolverForTest(t, source)
	exchange, err := resolver.AuthorizeRuntimeArtifactExchange(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, begin)
	if err != nil {
		t.Fatal(err)
	}
	stream := &runtimeExchangeTestStream{ctx: agentAuthContext(), requests: []*agentdatav1.ExchangeRuntimeArtifactRequest{
		{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_Begin{Begin: begin}},
		{Payload: &agentdatav1.ExchangeRuntimeArtifactRequest_MissingBlobDigests{MissingBlobDigests: &agentdatav1.MissingBlobDigests{Sequence: 1, Digests: []string{exchange.Entries[0].Digest, exchange.Entries[0].Digest}}}},
	}}
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	err = service.ExchangeRuntimeArtifact(stream)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("duplicate digest error = %v, want invalid argument", err)
	}
}

func digestForNucleiTest(content string) string {
	digest := sha256.Sum256([]byte(content))
	return hex.EncodeToString(digest[:])
}

func testCompatibilityRevision() string {
	return engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision
}

type runtimeExchangeTestStream struct {
	ctx       context.Context
	requests  []*agentdatav1.ExchangeRuntimeArtifactRequest
	requestAt int
	responses []*agentdatav1.ExchangeRuntimeArtifactResponse
}

func (stream *runtimeExchangeTestStream) Context() context.Context { return stream.ctx }
func (stream *runtimeExchangeTestStream) Send(response *agentdatav1.ExchangeRuntimeArtifactResponse) error {
	stream.responses = append(stream.responses, proto.Clone(response).(*agentdatav1.ExchangeRuntimeArtifactResponse))
	return nil
}
func (stream *runtimeExchangeTestStream) Recv() (*agentdatav1.ExchangeRuntimeArtifactRequest, error) {
	if stream.requestAt >= len(stream.requests) {
		return nil, io.EOF
	}
	request := stream.requests[stream.requestAt]
	stream.requestAt++
	return request, nil
}
func (stream *runtimeExchangeTestStream) SetHeader(metadata.MD) error  { return nil }
func (stream *runtimeExchangeTestStream) SendHeader(metadata.MD) error { return nil }
func (stream *runtimeExchangeTestStream) SetTrailer(metadata.MD)       {}
func (stream *runtimeExchangeTestStream) SendMsg(any) error            { return nil }
func (stream *runtimeExchangeTestStream) RecvMsg(any) error            { return io.EOF }
