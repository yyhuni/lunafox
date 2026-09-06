package agentdata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	fingerprintapp "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/application"
	fingerprintdomain "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

type executionArtifactTaskReaderStub struct {
	lease        *scandomain.SavedExecutionPlanLease
	err          error
	keepNilLease bool
}

func (stub *executionArtifactTaskReaderStub) GetSavedExecutionPlanLease(context.Context, int) (*scandomain.SavedExecutionPlanLease, error) {
	return stub.lease, stub.err
}

func TestServerExecutionArtifactResolverStreamsWorkflowAuthorizedCIDRHostPorts(t *testing.T) {
	plan := validExecutionArtifactPlan()
	plan.Target.Type = agentexecutionv1.TargetType_TARGET_TYPE_CIDR
	plan.Target.Value = "192.0.2.0/30"
	plan.WorkflowStep.StageId = "websites"
	plan.WorkflowStep.StepId = "website_discovery"
	plan.EngineRelease.Engine = "engine.lunafox.website_discovery"
	setPlanInputBindingForTest(plan, executionartifact.RoleHostPortsInput, executionartifact.ContentTypeHostPorts)
	resolver := newExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

	err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{
		Task: plan.GetTask(), Execution: plan.GetExecution(),
		Role: executionartifact.RoleHostPorts,
	}, stream)
	if err != nil {
		t.Fatalf("StreamExecutionInput failed: %v", err)
	}
	want := ""
	if got := executionArtifactFrameContent(stream.frames); got != want {
		t.Fatalf("HostPorts content = %q, want %q", got, want)
	}
	if got := stream.frames[len(stream.frames)-1].GetTrailer().GetIntegrity().GetRecordCount(); got != 0 {
		t.Fatalf("HostPorts record count = %d, want 0", got)
	}
}

type executionArtifactSessionReaderStub struct {
	session agentcontrol.ActiveControlSession
	ready   bool
}

func (stub executionArtifactSessionReaderStub) CurrentLeaseSession(int) (agentcontrol.ActiveControlSession, bool) {
	return stub.session, stub.ready
}

type executionArtifactDNSCursorStub struct {
	records []string
	err     error
}

func (stub executionArtifactDNSCursorStub) ForEachDNSNameByScanID(ctx context.Context, _ int, visit func(string) error) error {
	if stub.err != nil {
		return stub.err
	}
	for _, record := range stub.records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

type executionArtifactTargetDNSCursorStub struct {
	records   []string
	err       error
	failAfter int
	calls     int
	targetIDs []int
}

func (stub *executionArtifactTargetDNSCursorStub) ForEachDNSNameByTargetID(ctx context.Context, targetID int, visit func(string) error) error {
	stub.calls++
	stub.targetIDs = append(stub.targetIDs, targetID)
	if stub.err != nil && stub.failAfter == 0 {
		return stub.err
	}
	for index, record := range stub.records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if stub.failAfter > 0 && index >= stub.failAfter {
			return stub.err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

type executionArtifactHostPortCursorStub struct {
	records []scanapp.HostPortEvidence
}

type executionArtifactTargetHostPortCursorStub struct {
	records   []scanapp.HostPortEvidence
	err       error
	calls     int
	targetIDs []int
}

func (stub *executionArtifactTargetHostPortCursorStub) ForEachHostPortByTargetID(ctx context.Context, targetID int, visit func(scanapp.HostPortEvidence) error) error {
	stub.calls++
	stub.targetIDs = append(stub.targetIDs, targetID)
	if stub.err != nil {
		return stub.err
	}
	for _, record := range stub.records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

type executionArtifactWebsiteURLCursorStub struct {
	records []string
}

type executionArtifactEndpointURLCursorStub struct {
	records []string
}

type executionArtifactTargetWebsiteURLCursorStub struct {
	records   []string
	err       error
	calls     int
	targetIDs []int
}

func (stub *executionArtifactTargetWebsiteURLCursorStub) ForEachWebsiteURLByTargetID(ctx context.Context, targetID int, visit func(string) error) error {
	stub.calls++
	stub.targetIDs = append(stub.targetIDs, targetID)
	if stub.err != nil {
		return stub.err
	}
	for _, record := range stub.records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

type executionInputBlacklistSnapshotSourceStub struct {
	patterns []string
	err      error
	calls    int
	scanID   int
}

func (stub *executionInputBlacklistSnapshotSourceStub) ResolveExecutionInputBlacklistFilter(_ context.Context, scanID int) (*scanapp.ExecutionInputBlacklistFilter, error) {
	stub.calls++
	stub.scanID = scanID
	if stub.err != nil {
		return nil, stub.err
	}
	matcher, err := blacklistdomain.CompileMatcher(stub.patterns)
	if err != nil {
		return nil, err
	}
	return scanapp.NewExecutionInputBlacklistFilter(matcher)
}

func (stub executionArtifactWebsiteURLCursorStub) ForEachWebsiteURLByScanID(ctx context.Context, _ int, visit func(string) error) error {
	for _, record := range stub.records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

func (stub executionArtifactEndpointURLCursorStub) ForEachEndpointURLByScanID(ctx context.Context, _ int, visit func(string) error) error {
	for _, record := range stub.records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

type executionArtifactTargetEndpointURLCursorStub struct {
	records   []string
	err       error
	calls     int
	targetIDs []int
}

func (stub *executionArtifactTargetEndpointURLCursorStub) ForEachEndpointURLByTargetID(ctx context.Context, targetID int, visit func(string) error) error {
	stub.calls++
	stub.targetIDs = append(stub.targetIDs, targetID)
	if stub.err != nil {
		return stub.err
	}
	for _, record := range stub.records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

func TestWebsiteURLsAuthorizationUsesOnlySavedPlanAndSnapshotScope(t *testing.T) {
	plan := validExecutionArtifactPlan()
	plan.WorkflowStep.StageId = "url_collection"
	plan.WorkflowStep.StepId = "url_collection"
	plan.EngineRelease.Engine = "engine.lunafox.url_collection"
	setPlanInputBindingForTest(plan, executionartifact.RoleWebsiteURLsInput, executionartifact.ContentTypeWebsiteURLs)
	tasks := &executionArtifactTaskReaderStub{}
	resolver := newExecutionArtifactResolverForTest(t, plan, tasks, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
	artifact, err := resolver.AuthorizeExecutionInput(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, &agentdatav1.StreamExecutionInputRequest{
		Task: plan.GetTask(), Execution: plan.GetExecution(),
		Role: executionartifact.RoleWebsiteURLs,
	})
	if err != nil || artifact.Preflight != nil {
		t.Fatalf("WebsiteURLs authorization = artifact:%#v err:%v", artifact, err)
	}
	// This is an input-integrity gate. The authored workflow scope remains the
	// saved plan value and no resolver code rewrites it.
	if plan.GetWorkflowStep().GetStageId() != "url_collection" {
		t.Fatalf("resolver rewrote workflow topology: %#v", plan.GetWorkflowStep())
	}
}

func (stub executionArtifactHostPortCursorStub) ForEachHostPortByScanID(ctx context.Context, _ int, visit func(scanapp.HostPortEvidence) error) error {
	for _, record := range stub.records {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

type executionProviderConfigSourceStub struct {
	content string
	err     error
	observe func(context.Context)
}

func (stub executionProviderConfigSourceStub) GetExecutionSubfinderProviderConfig(ctx context.Context) (string, error) {
	if stub.observe != nil {
		stub.observe(ctx)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return stub.content, stub.err
}

type executionFingerprintArtifactSourceStub struct {
	contents map[fingerprintdomain.Library][]byte
	calls    int
}

func (stub *executionFingerprintArtifactSourceStub) OpenCurrent(_ context.Context, library fingerprintdomain.Library) (fingerprintapp.ArtifactHandle, error) {
	stub.calls++
	contents, ok := stub.contents[library]
	if !ok {
		return fingerprintapp.ArtifactHandle{}, errors.New("artifact is unavailable")
	}
	digest := sha256.Sum256(contents)
	return fingerprintapp.ArtifactHandle{
		Descriptor: fingerprintapp.ArtifactDescriptor{Library: library, SHA256Digest: "sha256:" + hex.EncodeToString(digest[:]), SizeBytes: int64(len(contents)), RecordCount: 0, Filename: library.CanonicalFilename(), ContentType: library.NativeContentType()},
		Reader:     io.NopCloser(strings.NewReader(string(contents))),
	}, nil
}

func TestServerExecutionArtifactResolverStreamsDeclaredFingerprintLibrariesOnlyAfterAuthorization(t *testing.T) {
	for _, descriptor := range executionartifact.Descriptors() {
		if !executionartifact.IsFingerprintLibraryRole(descriptor.Role) {
			continue
		}
		t.Run(descriptor.PlatformResourceID, func(t *testing.T) {
			plan := validExecutionArtifactPlan()
			plan.PlatformResourceBindings = []*agentexecutionv1.PlatformResourceBinding{{ResourceId: descriptor.PlatformResourceID, ContentType: descriptor.ContentType}}
			library, err := fingerprintdomain.ParseLibrary(descriptor.Library)
			if err != nil {
				t.Fatalf("registry library = %v", err)
			}
			contents := []byte("fingerprint-" + string(library))
			source := &executionFingerprintArtifactSourceStub{contents: map[fingerprintdomain.Library][]byte{library: contents}}
			resolver := newExecutionArtifactResolverForTestWithSources(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{}, &executionWordlistSourceStub{}, executionProviderConfigSourceStub{content: "alienvault: []\n"}, source)
			artifact, err := resolver.AuthorizePlatformResource(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, &agentdatav1.StreamPlatformResourceContentRequest{Task: plan.GetTask(), Execution: plan.GetExecution(), ResourceId: descriptor.PlatformResourceID})
			if err != nil {
				t.Fatalf("AuthorizePlatformResource() error = %v", err)
			}
			if source.calls != 1 || artifact.ContentType != descriptor.ContentType || artifact.ExpectedIntegrity == nil || artifact.ExpectedIntegrity.GetSizeBytes() != uint64(len(contents)) || artifact.ExpectedIntegrity.GetSha256Digest() == "" || artifact.ExpectedIntegrity.GetRecordCount() != 0 {
				t.Fatalf("fingerprint artifact authorization result = %#v, calls=%d", artifact, source.calls)
			}
			var produced strings.Builder
			if records, err := artifact.Produce(context.Background(), &produced); err != nil || records != 0 || produced.String() != string(contents) {
				t.Fatalf("fingerprint artifact production = records %d, content %q, err %v", records, produced.String(), err)
			}
		})
	}
}

func TestServerExecutionArtifactResolverReadsLatestFingerprintCurrentAfterPlanWasSaved(t *testing.T) {
	descriptor, ok := executionartifact.Lookup(executionartifact.RoleFingerprintLibraryFingerPrintHub)
	if !ok {
		t.Fatal("FingerprintHub fingerprint descriptor is missing")
	}
	plan := validExecutionArtifactPlan()
	plan.PlatformResourceBindings = []*agentexecutionv1.PlatformResourceBinding{{ResourceId: descriptor.PlatformResourceID, ContentType: descriptor.ContentType}}
	library, err := fingerprintdomain.ParseLibrary(descriptor.Library)
	if err != nil {
		t.Fatalf("ParseLibrary() error = %v", err)
	}
	// The saved plan is created before this value changes. OpenCurrent runs only
	// after plan/lease authorization at execution start, so it must read latest.
	source := &executionFingerprintArtifactSourceStub{contents: map[fingerprintdomain.Library][]byte{library: []byte("before-task-start")}}
	resolver := newExecutionArtifactResolverForTestWithSources(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{}, &executionWordlistSourceStub{}, executionProviderConfigSourceStub{content: "alienvault: []\n"}, source)
	source.contents[library] = []byte("latest-at-engine-start")
	artifact, err := resolver.AuthorizePlatformResource(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, &agentdatav1.StreamPlatformResourceContentRequest{Task: plan.GetTask(), Execution: plan.GetExecution(), ResourceId: descriptor.PlatformResourceID})
	if err != nil {
		t.Fatalf("AuthorizePlatformResource() error = %v", err)
	}
	var produced strings.Builder
	if _, err := artifact.Produce(context.Background(), &produced); err != nil {
		t.Fatalf("Produce() error = %v", err)
	}
	if got := produced.String(); got != "latest-at-engine-start" || source.calls != 1 {
		t.Fatalf("late-bound fingerprint artifact = %q, calls=%d", got, source.calls)
	}
}

type executionWordlistSourceStub struct {
	wordlist *catalogapp.Wordlist
	filePath string
	err      error
	observe  func(context.Context)
}

func (stub *executionWordlistSourceStub) GetByResourceName(ctx context.Context, _ string) (*catalogapp.Wordlist, error) {
	if stub.observe != nil {
		stub.observe(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if stub.err != nil {
		return nil, stub.err
	}
	if stub.wordlist == nil {
		return nil, catalogapp.ErrWordlistNotFound
	}
	return stub.wordlist, nil
}

func (stub *executionWordlistSourceStub) GetFilePathByResourceName(ctx context.Context, _ string) (string, error) {
	if stub.observe != nil {
		stub.observe(ctx)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if stub.err != nil {
		return "", stub.err
	}
	if stub.filePath == "" {
		return "", catalogapp.ErrWordlistNotFound
	}
	return stub.filePath, nil
}

func TestServerExecutionArtifactResolverStreamsSubdomainsFromGlobalRegistryWithoutSavedPlanMembership(t *testing.T) {
	plan := validExecutionArtifactPlan()
	// Execution input membership is deliberately absent from the saved plan.
	// A current task/execution lease may request any role in the closed Registry.
	resolver := newExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{records: []string{"api.example.com", "example.com", "www.example.com"}}, executionArtifactHostPortCursorStub{})
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

	err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{
		Task: plan.GetTask(), Execution: plan.GetExecution(),
		Role: executionartifact.RoleSubdomains,
	}, stream)
	if err != nil {
		t.Fatalf("StreamExecutionInput failed: %v", err)
	}
	if got := executionArtifactFrameContent(stream.frames); got != "api.example.com\nexample.com\nwww.example.com\n" {
		t.Fatalf("Subdomains content = %q", got)
	}
	if got := stream.frames[len(stream.frames)-1].GetTrailer().GetIntegrity().GetRecordCount(); got != 3 {
		t.Fatalf("Subdomains record count = %d, want 3", got)
	}
}

func TestServerExecutionArtifactResolverStreamsAllGlobalRegistryRolesWithoutSavedPlanMembership(t *testing.T) {
	plan := validExecutionArtifactPlan()
	targetDNS := &executionArtifactTargetDNSCursorStub{records: []string{"api.inventory.example"}}
	targetHostPorts := &executionArtifactTargetHostPortCursorStub{records: []scanapp.HostPortEvidence{{Host: "inventory.example", IP: "198.51.100.10", Port: 443}}}
	targetURLs := &executionArtifactTargetWebsiteURLCursorStub{records: []string{"https://inventory.example/path"}}
	targetEndpointURLs := &executionArtifactTargetEndpointURLCursorStub{records: []string{"https://inventory.example/api"}}
	resolver := newTargetInventoryExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{}, targetDNS, targetHostPorts, targetURLs, targetEndpointURLs)
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)

	for _, test := range []struct {
		name        string
		role        string
		contentType string
		want        string
		count       uint64
	}{
		{name: "subdomains", role: executionartifact.RoleSubdomains, contentType: executionartifact.ContentTypeSubdomains, want: "api.inventory.example\n", count: 1},
		{name: "hostPorts", role: executionartifact.RoleHostPorts, contentType: executionartifact.ContentTypeHostPorts, want: "{\"host\":\"inventory.example\",\"ip\":\"198.51.100.10\",\"port\":443}\n", count: 1},
		{name: "websiteURLs", role: executionartifact.RoleWebsiteURLs, contentType: executionartifact.ContentTypeWebsiteURLs, want: "https://inventory.example/path\n", count: 1},
		{name: "endpointURLs", role: executionartifact.RoleEndpointURLs, contentType: executionartifact.ContentTypeEndpointURLs, want: "https://inventory.example/api\n", count: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
			err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{Task: plan.GetTask(), Execution: plan.GetExecution(), Role: test.role}, stream)
			if err != nil {
				t.Fatalf("StreamExecutionInput() error = %v", err)
			}
			if got := executionArtifactFrameContent(stream.frames); got != test.want {
				t.Fatalf("stream content = %q, want %q", got, test.want)
			}
			header := stream.frames[0].GetHeader()
			if header == nil || header.GetTask() != plan.GetTask() || header.GetExecution() != plan.GetExecution() || header.GetExecutionInput().GetRole() != test.role || header.GetContentType() != test.contentType {
				t.Fatalf("stream header = %#v, want canonical scope and role %q/%q", header, test.role, test.contentType)
			}
			trailer := stream.frames[len(stream.frames)-1].GetTrailer()
			if trailer == nil || trailer.GetIntegrity().GetRecordCount() != test.count || trailer.GetIntegrity().GetSha256Digest() == "" {
				t.Fatalf("stream trailer = %#v, want record count %d", trailer, test.count)
			}
		})
	}
	if targetDNS.calls != 1 || targetHostPorts.calls != 1 || targetURLs.calls != 1 || targetEndpointURLs.calls != 1 {
		t.Fatalf("Target inventory cursor calls = dns:%d hostPorts:%d websiteURLs:%d endpointURLs:%d, want one per role", targetDNS.calls, targetHostPorts.calls, targetURLs.calls, targetEndpointURLs.calls)
	}
	for _, ids := range [][]int{targetDNS.targetIDs, targetHostPorts.targetIDs, targetURLs.targetIDs, targetEndpointURLs.targetIDs} {
		if len(ids) != 1 || ids[0] != 7 {
			t.Fatalf("Target inventory cursor target IDs = %v, want [7]", ids)
		}
	}
	if got := plan.GetTarget().GetResource(); got != "targets/7" {
		t.Fatalf("canonical Target resource changed during inventory resolution: %q", got)
	}
}

func TestServerExecutionArtifactResolverRequeriesTargetInventoryOnRetry(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)
	targetDNS := &executionArtifactTargetDNSCursorStub{records: []string{"before.example"}}
	resolver := newTargetInventoryExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{}, targetDNS, &executionArtifactTargetHostPortCursorStub{}, &executionArtifactTargetWebsiteURLCursorStub{})
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)

	first := &executionArtifactFrameStream{ctx: agentAuthContext()}
	if err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), first); err != nil {
		t.Fatalf("first StreamExecutionInput() error = %v", err)
	}
	if got := executionArtifactFrameContent(first.frames); got != "before.example\n" {
		t.Fatalf("first inventory content = %q", got)
	}

	// The second request represents a retry after the Target changed. The
	// resolver must invoke the Target cursor again instead of replaying bytes.
	targetDNS.records = []string{"after.example"}
	second := &executionArtifactFrameStream{ctx: agentAuthContext()}
	if err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), second); err != nil {
		t.Fatalf("retry StreamExecutionInput() error = %v", err)
	}
	if got := executionArtifactFrameContent(second.frames); got != "after.example\n" {
		t.Fatalf("retry inventory content = %q", got)
	}
	if targetDNS.calls != 2 || len(targetDNS.targetIDs) != 2 || targetDNS.targetIDs[0] != 7 || targetDNS.targetIDs[1] != 7 {
		t.Fatalf("Target inventory retry calls = %d target IDs = %v, want two reads for Target 7", targetDNS.calls, targetDNS.targetIDs)
	}
}

func TestServerExecutionArtifactResolverAllowsEmptyTargetInventoryForAllRoles(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)
	setPlanInputBindingForTest(plan, executionartifact.RoleHostPortsInput, executionartifact.ContentTypeHostPorts)
	setPlanInputBindingForTest(plan, executionartifact.RoleWebsiteURLsInput, executionartifact.ContentTypeWebsiteURLs)
	setPlanInputBindingForTest(plan, executionartifact.RoleEndpointURLsInput, executionartifact.ContentTypeEndpointURLs)
	targetDNS := &executionArtifactTargetDNSCursorStub{}
	targetHostPorts := &executionArtifactTargetHostPortCursorStub{}
	targetURLs := &executionArtifactTargetWebsiteURLCursorStub{}
	resolver := newTargetInventoryExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{}, targetDNS, targetHostPorts, targetURLs)
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)

	for _, role := range []string{executionartifact.RoleSubdomains, executionartifact.RoleHostPorts, executionartifact.RoleWebsiteURLs, executionartifact.RoleEndpointURLs} {
		stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
		if err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{Task: plan.GetTask(), Execution: plan.GetExecution(), Role: role}, stream); err != nil {
			t.Fatalf("role %s StreamExecutionInput() error = %v", role, err)
		}
		if got := executionArtifactFrameContent(stream.frames); got != "" {
			t.Fatalf("role %s empty inventory content = %q", role, got)
		}
		trailer := stream.frames[len(stream.frames)-1].GetTrailer()
		if trailer == nil || trailer.GetIntegrity().GetRecordCount() != 0 || trailer.GetIntegrity().GetSizeBytes() != 0 {
			t.Fatalf("role %s empty inventory trailer = %#v", role, trailer)
		}
	}
	if got := plan.GetTarget().GetResource(); got != "targets/7" {
		t.Fatalf("canonical Target resource changed for empty inventory: %q", got)
	}
}

func TestServerExecutionArtifactResolverDoesNotFallBackWhenTargetInventoryFails(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)
	readErr := errors.New("Target inventory query failed")
	targetDNS := &executionArtifactTargetDNSCursorStub{records: []string{"partial.example", "never-published.example"}, err: readErr, failAfter: 1}
	resolver := newTargetInventoryExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{}, targetDNS, &executionArtifactTargetHostPortCursorStub{}, &executionArtifactTargetWebsiteURLCursorStub{})
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

	err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("status = %s, want UNAVAILABLE; err=%v", status.Code(err), err)
	}
	if len(stream.frames) != 1 || stream.frames[0].GetHeader() == nil || executionArtifactFrameContent(stream.frames) != "" {
		t.Fatalf("failed Target inventory published partial/fallback content: frames=%#v", stream.frames)
	}
	if targetDNS.calls != 1 || len(targetDNS.targetIDs) != 1 || targetDNS.targetIDs[0] != 7 {
		t.Fatalf("Target inventory failure calls = %d target IDs = %v, want one read for Target 7", targetDNS.calls, targetDNS.targetIDs)
	}
}

func TestServerExecutionArtifactResolverMapsSnapshotDependencyFailureToUnavailable(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)
	resolver := newExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{err: errors.New("database unavailable")}, executionArtifactHostPortCursorStub{})
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

	err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("status = %s, want UNAVAILABLE; err=%v", status.Code(err), err)
	}
	if len(stream.frames) != 1 || stream.frames[0].GetHeader() == nil {
		t.Fatalf("authorized stream must contain only its header before source failure: %#v", stream.frames)
	}
}

func TestServerExecutionArtifactResolverFailsClosedForFrozenBlacklistBeforeHeader(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)

	for _, test := range []struct {
		name  string
		store *executionInputBlacklistSnapshotStoreStub
		want  codes.Code
	}{
		{name: "missing", store: &executionInputBlacklistSnapshotStoreStub{err: scanapp.ErrScanBlacklistSnapshotNotFound}, want: codes.DataLoss},
		{name: "corrupt", store: &executionInputBlacklistSnapshotStoreStub{patterns: []string{"Example.COM"}}, want: codes.DataLoss},
		{name: "unavailable", store: &executionInputBlacklistSnapshotStoreStub{err: errors.New("database unavailable")}, want: codes.Unavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshots := NewExecutionInputBlacklistSnapshotSource(test.store)
			resolver := newExecutionArtifactResolverForTestWithSnapshotSource(
				t,
				plan,
				&executionArtifactTaskReaderStub{},
				executionArtifactDNSCursorStub{records: []string{"api.example.com"}},
				executionArtifactHostPortCursorStub{},
				snapshots,
				&executionWordlistSourceStub{},
				executionProviderConfigSourceStub{content: "alienvault: []\n"},
			)
			service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

			err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream)
			if status.Code(err) != test.want || len(stream.frames) != 0 {
				t.Fatalf("status=%s frames=%d want=%s err=%v", status.Code(err), len(stream.frames), test.want, err)
			}
			if test.store.calls != 1 || test.store.scanID != 23 {
				t.Fatalf("snapshot lookup calls=%d scanID=%d, want one lookup for Scan 23", test.store.calls, test.store.scanID)
			}
		})
	}
}

func TestServerExecutionArtifactResolverFiltersDeclaredFactsUsingFrozenSnapshot(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)
	setPlanInputBindingForTest(plan, executionartifact.RoleHostPortsInput, executionartifact.ContentTypeHostPorts)
	setPlanInputBindingForTest(plan, executionartifact.RoleWebsiteURLsInput, executionartifact.ContentTypeWebsiteURLs)
	snapshots := &executionInputBlacklistSnapshotSourceStub{patterns: []string{"*.blocked.example", "192.0.2.0/24", "blocked.example"}}
	resolver := newExecutionArtifactResolverForTestWithExecutionInputSources(
		t,
		plan,
		&executionArtifactTaskReaderStub{},
		executionArtifactDNSCursorStub{records: []string{"blocked.example", "api.blocked.example", "keep.example"}},
		executionArtifactHostPortCursorStub{records: []scanapp.HostPortEvidence{
			{Host: "keep.example", IP: "198.51.100.10", Port: 443},
			{Host: "keep.example", IP: "192.0.2.10", Port: 443},
			{Host: "blocked.example", IP: "198.51.100.11", Port: 443},
		}},
		executionArtifactWebsiteURLCursorStub{records: []string{
			"https://blocked.example:8443/path?q=1#fragment",
			"http://192.0.2.17/admin",
			"https://keep.example/path?q=1",
		}},
		executionArtifactEndpointURLCursorStub{records: []string{
			"https://blocked.example/api",
			"https://keep.example/api",
		}},
		snapshots,
		&executionWordlistSourceStub{},
		executionProviderConfigSourceStub{content: "alienvault: []\n"},
	)
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)

	for _, test := range []struct {
		name  string
		role  string
		want  string
		count uint64
	}{
		{name: "subdomains", role: executionartifact.RoleSubdomains, want: "keep.example\n", count: 1},
		{name: "hostPorts", role: executionartifact.RoleHostPorts, want: "{\"host\":\"keep.example\",\"ip\":\"198.51.100.10\",\"port\":443}\n", count: 1},
		{name: "websiteURLs", role: executionartifact.RoleWebsiteURLs, want: "https://keep.example/path?q=1\n", count: 1},
		{name: "endpointURLs", role: executionartifact.RoleEndpointURLs, want: "https://keep.example/api\n", count: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
			err := service.StreamExecutionInput(&agentdatav1.StreamExecutionInputRequest{Task: plan.GetTask(), Execution: plan.GetExecution(), Role: test.role}, stream)
			if err != nil {
				t.Fatalf("StreamExecutionInput() error = %v", err)
			}
			if got := executionArtifactFrameContent(stream.frames); got != test.want {
				t.Fatalf("stream content = %q, want %q", got, test.want)
			}
			trailer := stream.frames[len(stream.frames)-1].GetTrailer()
			if trailer == nil || trailer.GetIntegrity().GetRecordCount() != test.count {
				t.Fatalf("stream trailer = %#v, want record count %d", trailer, test.count)
			}
		})
	}
	if snapshots.calls != 4 || snapshots.scanID != 23 {
		t.Fatalf("snapshot resolution calls=%d scanID=%d, want one immutable matcher per role for Scan 23", snapshots.calls, snapshots.scanID)
	}
}

func TestServerExecutionArtifactResolverKeepsAllExcludedProductsValidAndRetryDeterministic(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)
	snapshots := &executionInputBlacklistSnapshotSourceStub{patterns: []string{"*.blocked.example"}}
	resolver := newExecutionArtifactResolverForTestWithSnapshotSource(
		t,
		plan,
		&executionArtifactTaskReaderStub{},
		executionArtifactDNSCursorStub{records: []string{"one.blocked.example", "two.blocked.example"}},
		executionArtifactHostPortCursorStub{},
		snapshots,
		&executionWordlistSourceStub{},
		executionProviderConfigSourceStub{content: "alienvault: []\n"},
	)
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	var previousContent string
	var previousDigest string
	for attempt := 0; attempt < 2; attempt++ {
		stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
		if err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream); err != nil {
			t.Fatalf("attempt %d StreamExecutionInput() error = %v", attempt, err)
		}
		content := executionArtifactFrameContent(stream.frames)
		trailer := stream.frames[len(stream.frames)-1].GetTrailer()
		if content != "" || trailer == nil || trailer.GetIntegrity().GetRecordCount() != 0 || trailer.GetIntegrity().GetSizeBytes() != 0 || trailer.GetIntegrity().GetSha256Digest() != "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
			t.Fatalf("attempt %d all-excluded product = content %q trailer %#v", attempt, content, trailer)
		}
		if attempt > 0 && (content != previousContent || trailer.GetIntegrity().GetSha256Digest() != previousDigest) {
			t.Fatalf("retry product changed: content %q/%q digest %q/%q", content, previousContent, trailer.GetIntegrity().GetSha256Digest(), previousDigest)
		}
		previousContent = content
		previousDigest = trailer.GetIntegrity().GetSha256Digest()
	}
	if snapshots.calls != 2 {
		t.Fatalf("snapshot resolutions = %d, want one frozen matcher per retry", snapshots.calls)
	}
}

func TestServerExecutionArtifactResolverCancellationPreventsExecutionInputHeader(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)
	resolver := newExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	ctx, cancel := context.WithCancel(agentAuthContext())
	cancel()
	stream := &executionArtifactFrameStream{ctx: ctx}
	err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream)
	if status.Code(err) != codes.Canceled || len(stream.frames) != 0 {
		t.Fatalf("cancelled status=%s frames=%d err=%v", status.Code(err), len(stream.frames), err)
	}
}

func TestServerExecutionArtifactResolverRequiresFrozenBlacklistSnapshotSource(t *testing.T) {
	resolver, err := NewServerExecutionArtifactResolver(ServerExecutionArtifactResolverDependencies{
		Tasks:          &executionArtifactTaskReaderStub{},
		Sessions:       executionArtifactSessionReaderStub{},
		DNSNames:       executionArtifactDNSCursorStub{},
		HostPorts:      executionArtifactHostPortCursorStub{},
		WebsiteURLs:    executionArtifactWebsiteURLCursorStub{},
		Wordlists:      &executionWordlistSourceStub{},
		ProviderConfig: executionProviderConfigSourceStub{content: "alienvault: []\n"},
	})
	if err == nil || resolver != nil {
		t.Fatalf("resolver without private snapshot source = %#v, %v", resolver, err)
	}
}

func TestServerExecutionArtifactResolverClassifiesLeaseLookupFailureBeforeHeader(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)

	for _, test := range []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "missing task lease", err: scandomain.ErrSavedExecutionPlanLeaseNotFound, want: codes.PermissionDenied},
		{name: "lease store unavailable", err: errors.New("database unavailable"), want: codes.Unavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolver := newExecutionArtifactResolverForTest(t, plan, &executionArtifactTaskReaderStub{err: test.err}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
			service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

			err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream)
			if status.Code(err) != test.want || len(stream.frames) != 0 {
				t.Fatalf("status=%s frames=%d want=%s err=%v", status.Code(err), len(stream.frames), test.want, err)
			}
		})
	}
}

func TestServerExecutionArtifactResolverRejectsOwnershipAndStaleSessionBeforeHeader(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)

	t.Run("different Agent", func(t *testing.T) {
		tasks := &executionArtifactTaskReaderStub{}
		resolver := newExecutionArtifactResolverForTest(t, plan, tasks, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
		tasks.lease.AgentID = intPointerForResolver(99)
		service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
		stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
		err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream)
		if status.Code(err) != codes.PermissionDenied || len(stream.frames) != 0 {
			t.Fatalf("different Agent status=%s frames=%d err=%v", status.Code(err), len(stream.frames), err)
		}
	})

	t.Run("stale session epoch", func(t *testing.T) {
		tasks := &executionArtifactTaskReaderStub{}
		resolver := newExecutionArtifactResolverForTest(t, plan, tasks, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
		tasks.lease.AssignedSessionEpoch = int64PointerForResolver(12)
		service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
		stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
		err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream)
		if status.Code(err) != codes.FailedPrecondition || len(stream.frames) != 0 {
			t.Fatalf("stale epoch status=%s frames=%d err=%v", status.Code(err), len(stream.frames), err)
		}
	})

	t.Run("different process session", func(t *testing.T) {
		tasks := &executionArtifactTaskReaderStub{}
		resolver := newExecutionArtifactResolverForTest(t, plan, tasks, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
		otherSession := "session-other"
		tasks.lease.AssignedSessionID = &otherSession
		service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
		stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
		err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream)
		if status.Code(err) != codes.FailedPrecondition || len(stream.frames) != 0 {
			t.Fatalf("wrong process status=%s frames=%d err=%v", status.Code(err), len(stream.frames), err)
		}
	})
}

func TestServerExecutionArtifactResolverRejectsInactiveOrIncompleteLeaseBeforeHeader(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)

	for _, test := range []struct {
		name   string
		mutate func(*executionArtifactTaskReaderStub)
		want   codes.Code
	}{
		{name: "nil lease", mutate: func(tasks *executionArtifactTaskReaderStub) { tasks.lease = nil; tasks.keepNilLease = true }, want: codes.PermissionDenied},
		{name: "pending lease", mutate: func(tasks *executionArtifactTaskReaderStub) {
			tasks.lease.Status = string(scandomain.TaskStatusPending)
		}, want: codes.FailedPrecondition},
		{name: "missing epoch", mutate: func(tasks *executionArtifactTaskReaderStub) { tasks.lease.AssignedSessionEpoch = nil }, want: codes.FailedPrecondition},
		{name: "missing process session", mutate: func(tasks *executionArtifactTaskReaderStub) { tasks.lease.AssignedSessionID = nil }, want: codes.FailedPrecondition},
		{name: "missing Agent", mutate: func(tasks *executionArtifactTaskReaderStub) { tasks.lease.AgentID = nil }, want: codes.FailedPrecondition},
	} {
		t.Run(test.name, func(t *testing.T) {
			tasks := &executionArtifactTaskReaderStub{}
			resolver := newExecutionArtifactResolverForTest(t, plan, tasks, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
			test.mutate(tasks)
			service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

			err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream)
			if status.Code(err) != test.want || len(stream.frames) != 0 {
				t.Fatalf("status=%s frames=%d want=%s err=%v", status.Code(err), len(stream.frames), test.want, err)
			}
		})
	}
}

func TestServerExecutionArtifactResolverRejectsScopeBindingAndPlanDriftBeforeHeader(t *testing.T) {
	for _, test := range []struct {
		name            string
		mutatePersisted func(*agentexecutionv1.ResolvedEngineExecutionPlan)
		mutateTask      func(*executionArtifactTaskReaderStub)
		mutateReq       func(*agentdatav1.StreamExecutionInputRequest)
		want            codes.Code
	}{
		{
			name: "wrong requested task",
			mutateReq: func(request *agentdatav1.StreamExecutionInputRequest) {
				request.Task = "scans/23/tasks/32"
			},
			want: codes.PermissionDenied,
		},
		{
			name: "wrong requested execution",
			mutateReq: func(request *agentdatav1.StreamExecutionInputRequest) {
				request.Execution = "executions/another-execution"
			},
			want: codes.PermissionDenied,
		},
		{name: "invalid persisted input source", mutateTask: func(tasks *executionArtifactTaskReaderStub) {
			tasks.err = scandomain.ErrInvalidInputSource
		}, want: codes.DataLoss},
		{
			name: "corrupt persisted plan",
			mutateTask: func(tasks *executionArtifactTaskReaderStub) {
				tasks.lease.ResolvedExecutionPlan = []byte("not protobuf")
			},
			want: codes.DataLoss,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan := validExecutionArtifactPlan()
			setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)
			tasks := &executionArtifactTaskReaderStub{}
			resolver := newExecutionArtifactResolverForTest(t, plan, tasks, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{})
			if test.mutatePersisted != nil {
				test.mutatePersisted(plan)
				encoded, err := proto.Marshal(plan)
				if err != nil {
					t.Fatalf("marshal drifted persisted plan: %v", err)
				}
				tasks.lease.ResolvedExecutionPlan = encoded
			}
			if test.mutateTask != nil {
				test.mutateTask(tasks)
			}
			request := subdomainsRequestForPlan(plan)
			if test.mutateReq != nil {
				test.mutateReq(request)
			}
			service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

			err := service.StreamExecutionInput(request, stream)
			if status.Code(err) != test.want || len(stream.frames) != 0 {
				t.Fatalf("status=%s frames=%d want=%s err=%v", status.Code(err), len(stream.frames), test.want, err)
			}
		})
	}
}

func TestServerExecutionArtifactResolverStreamsPlanPinnedWordlist(t *testing.T) {
	content := []byte("one\ntwo\n")
	digest := sha256.Sum256(content)
	path := filepath.Join(t.TempDir(), "dns.txt")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write wordlist: %v", err)
	}
	plan := validExecutionArtifactPlan()
	plan.Config.Sections = []*agentexecutionv1.EngineConfigSection{{SectionId: "bruteforce", Enabled: true}}
	plan.ConfigResourceBindings = []*agentexecutionv1.ConfigResourceBinding{{
		SectionId: "bruteforce", ParamKey: "wordlist", ContentType: executionartifact.ContentTypeWordlist,
		Wordlist: &agentexecutionv1.WordlistDescriptor{
			Resource: "wordlists/7", Basename: "dns.txt", SizeBytes: uint64(len(content)),
			Sha256Digest: "sha256:" + hex.EncodeToString(digest[:]), LineCount: 2,
		},
	}}
	requestCtx := agentAuthContext()
	observedContextCalls := 0
	wordlists := &executionWordlistSourceStub{
		wordlist: &catalogapp.Wordlist{ID: 7, FileName: "dns.txt", FilePath: path, FileSize: int64(len(content)), LineCount: 2, FileHash: hex.EncodeToString(digest[:])},
		filePath: path,
		observe: func(ctx context.Context) {
			observedContextCalls++
			if ctx != requestCtx {
				t.Fatal("wordlist source did not receive the artifact request context")
			}
		},
	}
	resolver := newExecutionArtifactResolverForTestWithSources(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{}, wordlists, executionProviderConfigSourceStub{content: "alienvault: []\n"})
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: requestCtx}

	err := service.StreamConfigResource(&agentdatav1.StreamConfigResourceRequest{
		Task: plan.GetTask(), Execution: plan.GetExecution(), SectionId: "bruteforce", ParamKey: "wordlist",
	}, stream)
	if err != nil {
		t.Fatalf("StreamConfigResource failed: %v", err)
	}
	if got := executionArtifactFrameContent(stream.frames); got != string(content) {
		t.Fatalf("wordlist content = %q, want %q", got, content)
	}
	if expected := stream.frames[0].GetHeader().GetExpectedIntegrity(); expected == nil || expected.GetSha256Digest() != plan.ConfigResourceBindings[0].GetWordlist().GetSha256Digest() || expected.GetRecordCount() != 2 {
		t.Fatalf("unexpected expected integrity: %#v", expected)
	}
	if observedContextCalls != 2 {
		t.Fatalf("wordlist source context calls = %d, want metadata and path reads", observedContextCalls)
	}
}

func TestServerExecutionArtifactResolverClassifiesWordlistSourceFailuresBeforeHeader(t *testing.T) {
	plan := validExecutionArtifactPlan()
	plan.Config.Sections = []*agentexecutionv1.EngineConfigSection{{SectionId: "bruteforce", Enabled: true}}
	plan.ConfigResourceBindings = []*agentexecutionv1.ConfigResourceBinding{{
		SectionId: "bruteforce", ParamKey: "wordlist", ContentType: executionartifact.ContentTypeWordlist,
		Wordlist: &agentexecutionv1.WordlistDescriptor{
			Resource: "wordlists/7", Basename: "dns.txt", SizeBytes: 1,
			Sha256Digest: "sha256:" + strings.Repeat("a", 64), LineCount: 1,
		},
	}}
	for _, test := range []struct {
		name      string
		wordlists ExecutionWordlistSource
		want      codes.Code
	}{
		{name: "catalog unavailable", wordlists: &executionWordlistSourceStub{err: errors.New("database unavailable")}, want: codes.Unavailable},
		{name: "pinned wordlist missing", wordlists: &executionWordlistSourceStub{}, want: codes.DataLoss},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolver := newExecutionArtifactResolverForTestWithSources(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{}, test.wordlists, executionProviderConfigSourceStub{content: "alienvault: []\n"})
			service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
			err := service.StreamConfigResource(&agentdatav1.StreamConfigResourceRequest{
				Task: plan.GetTask(), Execution: plan.GetExecution(), SectionId: "bruteforce", ParamKey: "wordlist",
			}, stream)
			if status.Code(err) != test.want || len(stream.frames) != 0 {
				t.Fatalf("status=%s frames=%d want=%s err=%v", status.Code(err), len(stream.frames), test.want, err)
			}
		})
	}
}

func TestServerExecutionArtifactResolverRejectsPostPlanWordlistDriftBeforeHeader(t *testing.T) {
	content := []byte("one\ntwo\n")
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, *agentexecutionv1.ResolvedEngineExecutionPlan, *executionWordlistSourceStub)
	}{
		{name: "Catalog row deleted", mutate: func(_ *testing.T, _ *agentexecutionv1.ResolvedEngineExecutionPlan, source *executionWordlistSourceStub) {
			source.wordlist = nil
		}},
		{name: "file deleted", mutate: func(t *testing.T, _ *agentexecutionv1.ResolvedEngineExecutionPlan, source *executionWordlistSourceStub) {
			if err := os.Remove(source.filePath); err != nil {
				t.Fatalf("remove wordlist: %v", err)
			}
		}},
		{name: "file unreadable", mutate: func(t *testing.T, _ *agentexecutionv1.ResolvedEngineExecutionPlan, source *executionWordlistSourceStub) {
			if err := os.Chmod(source.filePath, 0); err != nil {
				t.Fatalf("make wordlist unreadable: %v", err)
			}
			file, err := os.Open(source.filePath)
			if err == nil {
				_ = file.Close()
				t.Skip("current user can read mode-000 files")
			}
		}},
		{name: "file is non-regular", mutate: func(t *testing.T, _ *agentexecutionv1.ResolvedEngineExecutionPlan, source *executionWordlistSourceStub) {
			if err := os.Remove(source.filePath); err != nil {
				t.Fatalf("remove wordlist: %v", err)
			}
			if err := os.Mkdir(source.filePath, 0o700); err != nil {
				t.Fatalf("replace wordlist with directory: %v", err)
			}
		}},
		{name: "basename drift", mutate: func(t *testing.T, _ *agentexecutionv1.ResolvedEngineExecutionPlan, source *executionWordlistSourceStub) {
			renamed := filepath.Join(filepath.Dir(source.filePath), "renamed.txt")
			if err := os.Rename(source.filePath, renamed); err != nil {
				t.Fatalf("rename wordlist: %v", err)
			}
			source.filePath = renamed
		}},
		{name: "Catalog size drift", mutate: func(_ *testing.T, _ *agentexecutionv1.ResolvedEngineExecutionPlan, source *executionWordlistSourceStub) {
			source.wordlist.FileSize++
		}},
		{name: "Catalog line count drift", mutate: func(_ *testing.T, _ *agentexecutionv1.ResolvedEngineExecutionPlan, source *executionWordlistSourceStub) {
			source.wordlist.LineCount++
		}},
		{name: "Catalog digest drift", mutate: func(_ *testing.T, _ *agentexecutionv1.ResolvedEngineExecutionPlan, source *executionWordlistSourceStub) {
			source.wordlist.FileHash = strings.Repeat("f", 64)
		}},
		{name: "same-name file bytes drift", mutate: func(t *testing.T, _ *agentexecutionv1.ResolvedEngineExecutionPlan, source *executionWordlistSourceStub) {
			if err := os.WriteFile(source.filePath, []byte("red\nsix\n"), 0o600); err != nil {
				t.Fatalf("replace wordlist bytes: %v", err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, source := planPinnedWordlistFixture(t, content)
			test.mutate(t, plan, source)
			resolver := newExecutionArtifactResolverForTestWithSources(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{}, source, executionProviderConfigSourceStub{content: "alienvault: []\n"})
			service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}

			err := service.StreamConfigResource(&agentdatav1.StreamConfigResourceRequest{
				Task: plan.GetTask(), Execution: plan.GetExecution(), SectionId: "bruteforce", ParamKey: "wordlist",
			}, stream)
			if status.Code(err) != codes.DataLoss || len(stream.frames) != 0 {
				t.Fatalf("status=%s frames=%d, want DATA_LOSS before header; err=%v", status.Code(err), len(stream.frames), err)
			}
		})
	}
}

func planPinnedWordlistFixture(t *testing.T, content []byte) (*agentexecutionv1.ResolvedEngineExecutionPlan, *executionWordlistSourceStub) {
	t.Helper()
	digest := sha256.Sum256(content)
	encodedDigest := hex.EncodeToString(digest[:])
	path := filepath.Join(t.TempDir(), "dns.txt")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write wordlist: %v", err)
	}
	plan := validExecutionArtifactPlan()
	plan.Config.Sections = []*agentexecutionv1.EngineConfigSection{{SectionId: "bruteforce", Enabled: true}}
	plan.ConfigResourceBindings = []*agentexecutionv1.ConfigResourceBinding{{
		SectionId: "bruteforce", ParamKey: "wordlist", ContentType: executionartifact.ContentTypeWordlist,
		Wordlist: &agentexecutionv1.WordlistDescriptor{
			Resource: "wordlists/7", Basename: "dns.txt", SizeBytes: uint64(len(content)),
			Sha256Digest: "sha256:" + encodedDigest, LineCount: 2,
		},
	}}
	return plan, &executionWordlistSourceStub{
		wordlist: &catalogapp.Wordlist{ID: 7, FileName: "dns.txt", FilePath: path, FileSize: int64(len(content)), LineCount: 2, FileHash: encodedDigest},
		filePath: path,
	}
}

func TestServerExecutionArtifactResolverStreamsNonEmptyProviderWithoutExpectedIntegrity(t *testing.T) {
	plan := validExecutionArtifactPlan()
	plan.PlatformResourceBindings = []*agentexecutionv1.PlatformResourceBinding{{
		ResourceId:  engineexecution.PlatformResourceSubfinderProviderConfig,
		ContentType: executionartifact.ContentTypeSubfinderProviderConfig,
	}}
	content := "alienvault: []\nbevigil: []\n"
	requestCtx := agentAuthContext()
	observedContext := false
	resolver := newExecutionArtifactResolverForTestWithSources(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{}, &executionWordlistSourceStub{}, executionProviderConfigSourceStub{
		content: content,
		observe: func(ctx context.Context) {
			observedContext = ctx == requestCtx
		},
	})
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: requestCtx}

	err := service.StreamPlatformResourceContent(&agentdatav1.StreamPlatformResourceContentRequest{
		Task: plan.GetTask(), Execution: plan.GetExecution(), ResourceId: engineexecution.PlatformResourceSubfinderProviderConfig,
	}, stream)
	if err != nil {
		t.Fatalf("StreamPlatformResourceContent failed: %v", err)
	}
	if got := executionArtifactFrameContent(stream.frames); got != content {
		t.Fatalf("provider content = %q, want %q", got, content)
	}
	if stream.frames[0].GetHeader().GetExpectedIntegrity() != nil || stream.frames[len(stream.frames)-1].GetTrailer().GetIntegrity().RecordCount != nil {
		t.Fatal("late-bound provider must omit expected integrity and record count")
	}
	if !observedContext {
		t.Fatal("provider source did not receive the artifact request context")
	}
}

func TestServerExecutionArtifactResolverClassifiesProviderFailuresBeforeHeader(t *testing.T) {
	plan := validExecutionArtifactPlan()
	plan.PlatformResourceBindings = []*agentexecutionv1.PlatformResourceBinding{{
		ResourceId:  engineexecution.PlatformResourceSubfinderProviderConfig,
		ContentType: executionartifact.ContentTypeSubfinderProviderConfig,
	}}
	for _, test := range []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "settings unavailable", err: errors.New("database unavailable"), want: codes.Unavailable},
		{name: "invalid complete mapping", err: catalogapp.ErrExecutionProviderConfigInvalid, want: codes.Internal},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolver := newExecutionArtifactResolverForTestWithSources(t, plan, &executionArtifactTaskReaderStub{}, executionArtifactDNSCursorStub{}, executionArtifactHostPortCursorStub{}, &executionWordlistSourceStub{}, executionProviderConfigSourceStub{err: test.err})
			service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
			stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
			err := service.StreamPlatformResourceContent(&agentdatav1.StreamPlatformResourceContentRequest{
				Task: plan.GetTask(), Execution: plan.GetExecution(), ResourceId: engineexecution.PlatformResourceSubfinderProviderConfig,
			}, stream)
			if status.Code(err) != test.want || len(stream.frames) != 0 {
				t.Fatalf("status=%s frames=%d want=%s err=%v", status.Code(err), len(stream.frames), test.want, err)
			}
		})
	}
}

func TestArtifactLineCounterRejectsOverflow(t *testing.T) {
	lineOverflow := &artifactLineCounter{newlines: ^uint64(0)}
	if err := lineOverflow.Add([]byte{'\n'}); err == nil {
		t.Fatal("expected line counter overflow")
	}
	sizeOverflow := &artifactLineCounter{size: ^uint64(0)}
	if err := sizeOverflow.Add([]byte{'x'}); err == nil {
		t.Fatal("expected byte counter overflow")
	}
}

func newExecutionArtifactResolverForTest(
	t *testing.T,
	plan *agentexecutionv1.ResolvedEngineExecutionPlan,
	tasks *executionArtifactTaskReaderStub,
	dns executionArtifactDNSCursorStub,
	hostPorts executionArtifactHostPortCursorStub,
) ExecutionArtifactResolver {
	t.Helper()
	return newExecutionArtifactResolverForTestWithSources(t, plan, tasks, dns, hostPorts, &executionWordlistSourceStub{}, executionProviderConfigSourceStub{content: "alienvault: []\n"})
}

func newExecutionArtifactResolverForTestWithSources(
	t *testing.T,
	plan *agentexecutionv1.ResolvedEngineExecutionPlan,
	tasks *executionArtifactTaskReaderStub,
	dns executionArtifactDNSCursorStub,
	hostPorts executionArtifactHostPortCursorStub,
	wordlists ExecutionWordlistSource,
	provider ExecutionProviderConfigSource,
	fingerprintSources ...ExecutionFingerprintArtifactSource,
) ExecutionArtifactResolver {
	return newExecutionArtifactResolverForTestWithSnapshotSource(t, plan, tasks, dns, hostPorts, &executionInputBlacklistSnapshotSourceStub{}, wordlists, provider, fingerprintSources...)
}

func newExecutionArtifactResolverForTestWithSnapshotSource(
	t *testing.T,
	plan *agentexecutionv1.ResolvedEngineExecutionPlan,
	tasks *executionArtifactTaskReaderStub,
	dns executionArtifactDNSCursorStub,
	hostPorts executionArtifactHostPortCursorStub,
	blacklistSnapshots ExecutionInputBlacklistSnapshotSource,
	wordlists ExecutionWordlistSource,
	provider ExecutionProviderConfigSource,
	fingerprintSources ...ExecutionFingerprintArtifactSource,
) ExecutionArtifactResolver {
	return newExecutionArtifactResolverForTestWithExecutionInputSources(t, plan, tasks, dns, hostPorts, executionArtifactWebsiteURLCursorStub{}, executionArtifactEndpointURLCursorStub{}, blacklistSnapshots, wordlists, provider, fingerprintSources...)
}

func newExecutionArtifactResolverForTestWithExecutionInputSources(
	t *testing.T,
	plan *agentexecutionv1.ResolvedEngineExecutionPlan,
	tasks *executionArtifactTaskReaderStub,
	dns executionArtifactDNSCursorStub,
	hostPorts executionArtifactHostPortCursorStub,
	websiteURLs executionArtifactWebsiteURLCursorStub,
	endpointURLs executionArtifactEndpointURLCursorStub,
	blacklistSnapshots ExecutionInputBlacklistSnapshotSource,
	wordlists ExecutionWordlistSource,
	provider ExecutionProviderConfigSource,
	fingerprintSources ...ExecutionFingerprintArtifactSource,
) ExecutionArtifactResolver {
	t.Helper()
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		t.Fatalf("encode execution plan: %v", err)
	}
	if tasks.lease == nil && !tasks.keepNilLease {
		sessionID := testAgentSessionID
		tasks.lease = &scandomain.SavedExecutionPlanLease{
			TaskID: 31, ScanID: 23, InputSource: scandomain.InputSourceScanSnapshot, Status: string(scandomain.TaskStatusRunning),
			AgentID: intPointerForResolver(42), AssignedSessionID: &sessionID, AssignedSessionEpoch: int64PointerForResolver(testAgentSessionEpoch), ResolvedExecutionPlan: encoded,
		}
	}
	var fingerprintSource ExecutionFingerprintArtifactSource
	if len(fingerprintSources) > 0 {
		fingerprintSource = fingerprintSources[0]
	}
	resolver, err := NewServerExecutionArtifactResolver(ServerExecutionArtifactResolverDependencies{
		Tasks: tasks,
		Sessions: executionArtifactSessionReaderStub{ready: true, session: agentcontrol.ActiveControlSession{
			AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch, StreamID: 1,
			LastSeenAt: time.Now(), Phase: agentcontrol.ControlSessionPhaseReadyAttached,
		}},
		DNSNames: dns, HostPorts: hostPorts, WebsiteURLs: websiteURLs, EndpointURLs: endpointURLs, BlacklistSnapshots: blacklistSnapshots, Wordlists: wordlists, ProviderConfig: provider, FingerprintArtifacts: fingerprintSource, NucleiTemplates: &nucleiTemplateSourceStub{},
	})
	if err != nil {
		t.Fatalf("NewServerExecutionArtifactResolver failed: %v", err)
	}
	return resolver
}

func newTargetInventoryExecutionArtifactResolverForTest(
	t *testing.T,
	plan *agentexecutionv1.ResolvedEngineExecutionPlan,
	tasks *executionArtifactTaskReaderStub,
	targetDNS *executionArtifactTargetDNSCursorStub,
	targetHostPorts *executionArtifactTargetHostPortCursorStub,
	targetURLs *executionArtifactTargetWebsiteURLCursorStub,
	targetEndpointSources ...*executionArtifactTargetEndpointURLCursorStub,
) ExecutionArtifactResolver {
	t.Helper()
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		t.Fatalf("encode execution plan: %v", err)
	}
	if tasks.lease == nil && !tasks.keepNilLease {
		sessionID := testAgentSessionID
		tasks.lease = &scandomain.SavedExecutionPlanLease{
			TaskID: 31, ScanID: 23, InputSource: scandomain.InputSourceTargetInventory, Status: string(scandomain.TaskStatusRunning),
			AgentID: intPointerForResolver(42), AssignedSessionID: &sessionID, AssignedSessionEpoch: int64PointerForResolver(testAgentSessionEpoch), ResolvedExecutionPlan: encoded,
		}
	}
	targetEndpointURLs := &executionArtifactTargetEndpointURLCursorStub{}
	if len(targetEndpointSources) > 0 {
		targetEndpointURLs = targetEndpointSources[0]
	}
	resolver, err := NewServerExecutionArtifactResolver(ServerExecutionArtifactResolverDependencies{
		Tasks: tasks,
		Sessions: executionArtifactSessionReaderStub{ready: true, session: agentcontrol.ActiveControlSession{
			AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch, StreamID: 1,
			LastSeenAt: time.Now(), Phase: agentcontrol.ControlSessionPhaseReadyAttached,
		}},
		// These values are deliberately distinct from the Target inventory. A
		// targetInventory lease must never silently select a Scan snapshot.
		DNSNames:              executionArtifactDNSCursorStub{records: []string{"snapshot-only.example"}},
		HostPorts:             executionArtifactHostPortCursorStub{records: []scanapp.HostPortEvidence{{Host: "snapshot-only.example", IP: "192.0.2.10", Port: 80}}},
		WebsiteURLs:           executionArtifactWebsiteURLCursorStub{records: []string{"https://snapshot-only.example"}},
		EndpointURLs:          executionArtifactEndpointURLCursorStub{},
		InventoryDNSNames:     targetDNS,
		InventoryHostPorts:    targetHostPorts,
		InventoryWebsiteURLs:  targetURLs,
		InventoryEndpointURLs: targetEndpointURLs,
		BlacklistSnapshots:    &executionInputBlacklistSnapshotSourceStub{},
		Wordlists:             &executionWordlistSourceStub{},
		ProviderConfig:        executionProviderConfigSourceStub{content: "alienvault: []\n"},
		NucleiTemplates:       &nucleiTemplateSourceStub{},
	})
	if err != nil {
		t.Fatalf("NewServerExecutionArtifactResolver failed: %v", err)
	}
	return resolver
}

func validExecutionArtifactPlan() *agentexecutionv1.ResolvedEngineExecutionPlan {
	return &agentexecutionv1.ResolvedEngineExecutionPlan{
		Execution: "executions/scan-23-task-31", Task: "scans/23/tasks/31",
		Target:       &agentexecutionv1.CanonicalTarget{Resource: "targets/7", Type: agentexecutionv1.TargetType_TARGET_TYPE_DOMAIN, Value: "example.com"},
		WorkflowStep: &agentexecutionv1.WorkflowStepScope{Scan: "scans/23", Workflow: "scanWorkflows/default", StageId: "ports", StepId: "scan"},
		EngineRelease: &agentexecutionv1.EngineRelease{
			Engine: "engine.lunafox.port_scan", PackageDigest: "sha256:" + strings.Repeat("a", 64), EngineApiMajor: 2, CompatibilityRevision: "engine-execution-diagnostics-r1",
		},
		RuntimeImage: &agentexecutionv1.RuntimeImage{Refs: []string{"docker.io/lunafox/lunafox-engine-port-scan@sha256:" + strings.Repeat("b", 64)}},
		Config:       &agentexecutionv1.FinalEngineConfig{},
		Limits: &agentexecutionv1.ExecutionLimits{
			MaxExecutionDuration: durationpb.New(time.Hour), ProgressMessageMaxBytes: 4096, ResultBatchMaxItems: 1000, ResultBatchMaxBytes: 1 << 20,
		},
	}
}

func subdomainsRequestForPlan(plan *agentexecutionv1.ResolvedEngineExecutionPlan) *agentdatav1.StreamExecutionInputRequest {
	return &agentdatav1.StreamExecutionInputRequest{
		Task: plan.GetTask(), Execution: plan.GetExecution(),
		Role: executionartifact.RoleSubdomains,
	}
}

// These helpers intentionally do nothing. Input role membership is a global
// Registry fact and is no longer persisted in the saved plan; call sites retain
// them only to keep fixture setup explicit about the removed authorization axis.
func setPlanInputBindingForTest(*agentexecutionv1.ResolvedEngineExecutionPlan, executionartifact.Role, string) {
}

func executionArtifactFrameContent(frames []*agentdatav1.ExecutionArtifactFrame) string {
	var builder strings.Builder
	for _, frame := range frames {
		if chunk := frame.GetChunk(); chunk != nil {
			_, _ = builder.Write(chunk.GetData())
		}
	}
	return builder.String()
}

func intPointerForResolver(value int) *int       { return &value }
func int64PointerForResolver(value int64) *int64 { return &value }

var _ io.Writer = (*strings.Builder)(nil)
