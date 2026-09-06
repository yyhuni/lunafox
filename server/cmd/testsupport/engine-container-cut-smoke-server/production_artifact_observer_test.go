package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	"github.com/yyhuni/lunafox/server/internal/grpc/agentdata"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type productionArtifactObserverTaskReader struct {
	mu         sync.Mutex
	lease      *scandomain.SavedExecutionPlanLease
	leaseCalls int
}

func (reader *productionArtifactObserverTaskReader) GetSavedExecutionPlanLease(_ context.Context, taskID int) (*scandomain.SavedExecutionPlanLease, error) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	reader.leaseCalls++
	if reader.lease == nil || reader.lease.TaskID != taskID {
		return nil, scandomain.ErrSavedExecutionPlanLeaseNotFound
	}
	copy := *reader.lease
	copy.ResolvedExecutionPlan = append([]byte(nil), reader.lease.ResolvedExecutionPlan...)
	return &copy, nil
}

func (reader *productionArtifactObserverTaskReader) calls() int {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	return reader.leaseCalls
}

type productionArtifactObserverSessionReader struct {
	session agentcontrol.ActiveControlSession
}

func (reader productionArtifactObserverSessionReader) CurrentLeaseSession(agentID int) (agentcontrol.ActiveControlSession, bool) {
	return reader.session, reader.session.AgentID == agentID
}

type productionArtifactObserverDNSCursor struct{}

func (productionArtifactObserverDNSCursor) ForEachDNSNameByScanID(context.Context, int, func(string) error) error {
	return nil
}

type productionArtifactObserverHostPortCursor struct{}

func (productionArtifactObserverHostPortCursor) ForEachHostPortByScanID(context.Context, int, func(scanapp.HostPortEvidence) error) error {
	return nil
}

type productionArtifactObserverWebsiteURLCursor struct{}

func (productionArtifactObserverWebsiteURLCursor) ForEachWebsiteURLByScanID(context.Context, int, func(string) error) error {
	return nil
}

func (productionArtifactObserverWebsiteURLCursor) ForEachEndpointURLByScanID(context.Context, int, func(string) error) error {
	return nil
}

type productionArtifactObserverWordlists struct {
	wordlist *catalogapp.Wordlist
	path     string
}

func (source *productionArtifactObserverWordlists) GetByResourceName(ctx context.Context, resourceName string) (*catalogapp.Wordlist, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if source == nil || source.wordlist == nil || resourceName != "wordlists/7" {
		return nil, catalogapp.ErrWordlistNotFound
	}
	copy := *source.wordlist
	return &copy, nil
}

func (source *productionArtifactObserverWordlists) GetFilePathByResourceName(ctx context.Context, resourceName string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if source == nil || source.wordlist == nil || resourceName != "wordlists/7" || source.path == "" {
		return "", catalogapp.ErrWordlistNotFound
	}
	return source.path, nil
}

type productionArtifactObserverProvider struct{ content string }

func (source productionArtifactObserverProvider) GetExecutionSubfinderProviderConfig(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return source.content, nil
}

type productionArtifactObserverFixture struct {
	observer *productionArtifactObserver
	bridge   *smokeTaskBridge
	plan     *agentexecutionv1.ResolvedEngineExecutionPlan
	lease    agentdata.AgentExecutionLease
	tasks    *productionArtifactObserverTaskReader
	wordlist []byte
}

func newProductionArtifactObserverFixture(t *testing.T, scenario smokeScenario, resources bool) productionArtifactObserverFixture {
	t.Helper()
	plan := validSmokePlanForTest()
	if scenario == smokeScenarioWebsiteEmpty {
		plan.Target.Type = agentexecutionv1.TargetType_TARGET_TYPE_CIDR
		plan.Target.Value = "192.0.2.0/30"
		plan.WorkflowStep.StageId = "websites"
		plan.WorkflowStep.StepId = "website_discovery"
		plan.EngineRelease.Engine = engineWebsiteID
	}
	wordlistContent := []byte("one\ntwo\n")
	wordlistPath := filepath.Join(t.TempDir(), "observer-words.txt")
	if err := os.WriteFile(wordlistPath, wordlistContent, 0o600); err != nil {
		t.Fatalf("write observer wordlist: %v", err)
	}
	wordlistDigest := sha256.Sum256(wordlistContent)
	wordlistDigestHex := hex.EncodeToString(wordlistDigest[:])
	if resources {
		plan.ConfigResourceBindings = []*agentexecutionv1.ConfigResourceBinding{{
			SectionId: "naabu_active", ParamKey: "wordlist", ContentType: executionartifact.ContentTypeWordlist,
			Wordlist: &agentexecutionv1.WordlistDescriptor{
				Resource: "wordlists/7", Basename: filepath.Base(wordlistPath),
				SizeBytes: uint64(len(wordlistContent)), Sha256Digest: "sha256:" + wordlistDigestHex, LineCount: 2,
			},
		}}
		plan.PlatformResourceBindings = []*agentexecutionv1.PlatformResourceBinding{{
			ResourceId:  engineexecution.PlatformResourceSubfinderProviderConfig,
			ContentType: executionartifact.ContentTypeSubfinderProviderConfig,
		}}
	}
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		t.Fatalf("encode observer plan: %v", err)
	}
	agentID := smokeAgentID
	sessionID := smokeTestSessionID
	sessionEpoch := int64(7)
	tasks := &productionArtifactObserverTaskReader{lease: &scandomain.SavedExecutionPlanLease{
		TaskID: 1, ScanID: 1, InputSource: scandomain.InputSourceScanSnapshot, Status: "running", AgentID: &agentID,
		AssignedSessionID: &sessionID, AssignedSessionEpoch: &sessionEpoch,
		ResolvedExecutionPlan: encoded,
	}}
	authority := newSmokeAuthority("cut-smoke-agent-token-artifact-observer", nil)
	record := &smokePlanRecord{
		scenario: scenario, scanID: 1, taskID: 1, targetID: 1,
		engineID: plan.GetEngineRelease().GetEngine(), task: plan.GetTask(), planBytes: encoded,
	}
	bridge := newSmokeTaskBridge(authority, []*smokePlanRecord{record})
	observer, err := newProductionArtifactObserver(agentdata.ServerExecutionArtifactResolverDependencies{
		Tasks: tasks,
		Sessions: productionArtifactObserverSessionReader{session: agentcontrol.ActiveControlSession{
			AgentID: smokeAgentID, SessionID: sessionID, SessionEpoch: sessionEpoch,
		}},
		DNSNames:           productionArtifactObserverDNSCursor{},
		HostPorts:          productionArtifactObserverHostPortCursor{},
		WebsiteURLs:        productionArtifactObserverWebsiteURLCursor{},
		EndpointURLs:       productionArtifactObserverWebsiteURLCursor{},
		BlacklistSnapshots: newSmokeEmptyExecutionInputBlacklistSnapshotSource(),
		Wordlists: &productionArtifactObserverWordlists{
			wordlist: &catalogapp.Wordlist{
				ID: 7, FileName: filepath.Base(wordlistPath), FilePath: wordlistPath,
				FileSize: int64(len(wordlistContent)), LineCount: 2, FileHash: wordlistDigestHex,
			},
			path: wordlistPath,
		},
		ProviderConfig:  productionArtifactObserverProvider{content: "alienvault: []\n"},
		NucleiTemplates: smokeNucleiTemplateSource{},
	}, bridge)
	if err != nil {
		t.Fatalf("newProductionArtifactObserver: %v", err)
	}
	return productionArtifactObserverFixture{
		observer: observer, bridge: bridge, plan: plan, tasks: tasks, wordlist: wordlistContent,
		lease: agentdata.AgentExecutionLease{AgentID: smokeAgentID, SessionID: sessionID, SessionEpoch: sessionEpoch},
	}
}

func TestProductionArtifactObserverStreamsEmptyHostPortsFacts(t *testing.T) {
	fixture := newProductionArtifactObserverFixture(t, smokeScenarioWebsiteEmpty, false)
	request := &agentdatav1.StreamExecutionInputRequest{
		Task: fixture.plan.GetTask(), Execution: fixture.plan.GetExecution(),
		Role: executionartifact.RoleHostPorts,
	}
	authorized, err := fixture.observer.AuthorizeExecutionInput(context.Background(), fixture.lease, request)
	if err != nil {
		t.Fatalf("authorize CIDR HostPorts: %v", err)
	}
	var output bytes.Buffer
	records, err := authorized.Produce(context.Background(), &output)
	want := ""
	if err != nil || records != 0 || output.String() != want {
		t.Fatalf("CIDR production HostPorts = records %d bytes %d content %q err %v", records, output.Len(), output.String(), err)
	}
	snapshot, ok := fixture.bridge.recordForTaskID(1)
	if !ok || snapshot.webArtifact.streams != 1 || snapshot.webArtifact.records != 0 || snapshot.webArtifact.bytes != 0 {
		t.Fatalf("CIDR production HostPorts evidence = %#v", snapshot.webArtifact)
	}
}

func TestProductionArtifactObserverSubdomainsUnavailableOnceUnderConcurrency(t *testing.T) {
	fixture := newProductionArtifactObserverFixture(t, smokeScenarioPortSuccess, false)
	request := &agentdatav1.StreamExecutionInputRequest{
		Task: fixture.plan.GetTask(), Execution: fixture.plan.GetExecution(),
		Role: executionartifact.RoleSubdomains,
	}
	authorized, err := fixture.observer.AuthorizeExecutionInput(context.Background(), fixture.lease, request)
	if err != nil {
		t.Fatalf("authorize first Subdomains attempt: %v", err)
	}
	if _, err := authorized.Produce(context.Background(), io.Discard); status.Code(err) != codes.Unavailable {
		t.Fatalf("first Subdomains Produce status = %s, want Unavailable: %v", status.Code(err), err)
	}

	const concurrentAttempts = 12
	errorsByAttempt := make(chan error, concurrentAttempts)
	var group sync.WaitGroup
	for range concurrentAttempts {
		group.Add(1)
		go func() {
			defer group.Done()
			source, err := fixture.observer.AuthorizeExecutionInput(context.Background(), fixture.lease, request)
			if err != nil {
				errorsByAttempt <- err
				return
			}
			var output bytes.Buffer
			records, err := source.Produce(context.Background(), &output)
			if err == nil && (records != 0 || output.Len() != 0) {
				err = errors.New("production Subdomains output changed")
			}
			errorsByAttempt <- err
		}()
	}
	group.Wait()
	close(errorsByAttempt)
	for err := range errorsByAttempt {
		if err != nil {
			t.Fatalf("concurrent Subdomains Produce: %v", err)
		}
	}

	snapshot, ok := fixture.bridge.recordForTaskID(1)
	if !ok {
		t.Fatal("Subdomains observation record is missing")
	}
	wantBytes := uint64(0)
	if snapshot.hostArtifact.attempts != concurrentAttempts+1 || snapshot.hostArtifact.unavailableFailures != 1 ||
		snapshot.hostArtifact.streams != concurrentAttempts || snapshot.hostArtifact.records != 0 ||
		snapshot.hostArtifact.bytes != wantBytes {
		t.Fatalf("Subdomains observation = %#v", snapshot.hostArtifact)
	}
	if fixture.tasks.calls() != concurrentAttempts+1 {
		t.Fatalf("production resolver lease reads = %d, want %d", fixture.tasks.calls(), concurrentAttempts+1)
	}
}

func TestProductionArtifactObserverDelegatesResourcesAndCausesWordlistDataLoss(t *testing.T) {
	fixture := newProductionArtifactObserverFixture(t, smokeScenarioArtifactIntegrity, true)

	platform, err := fixture.observer.AuthorizePlatformResource(context.Background(), fixture.lease, &agentdatav1.StreamPlatformResourceContentRequest{
		Task: fixture.plan.GetTask(), Execution: fixture.plan.GetExecution(), ResourceId: engineexecution.PlatformResourceSubfinderProviderConfig,
	})
	if err != nil {
		t.Fatalf("authorize production platform resource: %v", err)
	}
	var provider bytes.Buffer
	if records, err := platform.Produce(context.Background(), &provider); err != nil || records != 0 || provider.String() != "alienvault: []\n" {
		t.Fatalf("platform Produce = records %d content %q err %v", records, provider.String(), err)
	}

	service := agentdata.NewExecutionArtifactService(fixture.bridge.authority, fixture.observer)
	request := &agentdatav1.StreamConfigResourceRequest{
		Task: fixture.plan.GetTask(), Execution: fixture.plan.GetExecution(), SectionId: "naabu_active", ParamKey: "wordlist",
	}
	first := &productionArtifactObserverFrameStream{ctx: productionArtifactObserverAuthContext(fixture.bridge.authority.token, fixture.lease)}
	if err := service.StreamConfigResource(request, first); status.Code(err) != codes.DataLoss {
		t.Fatalf("corrupted wordlist status = %s, want DataLoss: %v", status.Code(err), err)
	}
	if len(first.frames) < 2 || first.frames[0].GetHeader() == nil || productionArtifactObserverHasTrailer(first.frames) {
		t.Fatalf("corrupted wordlist frames = %#v", first.frames)
	}
	if got := productionArtifactObserverFrameContent(first.frames); got != string(fixture.wordlist)+"!" {
		t.Fatalf("corrupted wordlist bytes = %q", got)
	}

	second := &productionArtifactObserverFrameStream{ctx: productionArtifactObserverAuthContext(fixture.bridge.authority.token, fixture.lease)}
	if err := service.StreamConfigResource(request, second); err != nil {
		t.Fatalf("second wordlist stream should be clean: %v", err)
	}
	if !productionArtifactObserverHasTrailer(second.frames) || productionArtifactObserverFrameContent(second.frames) != string(fixture.wordlist) {
		t.Fatalf("clean wordlist frames = %#v", second.frames)
	}

	snapshot, ok := fixture.bridge.recordForTaskID(1)
	if !ok {
		t.Fatal("resource observation record is missing")
	}
	if snapshot.platformArtifact.attempts != 1 || snapshot.platformArtifact.streams != 1 || snapshot.platformArtifact.bytes != uint64(len("alienvault: []\n")) {
		t.Fatalf("platform observation = %#v", snapshot.platformArtifact)
	}
	wantConfigBytes := uint64(len(fixture.wordlist)*2 + 1)
	if snapshot.configArtifact.attempts != 2 || snapshot.configArtifact.integrityCorruptions != 1 ||
		snapshot.configArtifact.streams != 2 || snapshot.configArtifact.records != 4 || snapshot.configArtifact.bytes != wantConfigBytes {
		t.Fatalf("config observation = %#v", snapshot.configArtifact)
	}
}

func productionArtifactObserverAuthContext(token string, lease agentdata.AgentExecutionLease) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcauth.AgentAuthenticationTokenMetadataKey, token,
		grpcauth.AgentSessionIDMetadataKey, lease.SessionID,
		grpcauth.AgentSessionEpochMetadataKey, strconv.FormatInt(lease.SessionEpoch, 10),
	))
}

type productionArtifactObserverFrameStream struct {
	ctx    context.Context
	frames []*agentdatav1.ExecutionArtifactFrame
}

func (stream *productionArtifactObserverFrameStream) Send(frame *agentdatav1.ExecutionArtifactFrame) error {
	stream.frames = append(stream.frames, frame)
	return nil
}

func (*productionArtifactObserverFrameStream) SetHeader(metadata.MD) error  { return nil }
func (*productionArtifactObserverFrameStream) SendHeader(metadata.MD) error { return nil }
func (*productionArtifactObserverFrameStream) SetTrailer(metadata.MD)       {}
func (stream *productionArtifactObserverFrameStream) Context() context.Context {
	return stream.ctx
}
func (*productionArtifactObserverFrameStream) SendMsg(any) error { return nil }
func (*productionArtifactObserverFrameStream) RecvMsg(any) error { return io.EOF }

func productionArtifactObserverHasTrailer(frames []*agentdatav1.ExecutionArtifactFrame) bool {
	for _, frame := range frames {
		if frame.GetTrailer() != nil {
			return true
		}
	}
	return false
}

func productionArtifactObserverFrameContent(frames []*agentdatav1.ExecutionArtifactFrame) string {
	var content bytes.Buffer
	for _, frame := range frames {
		if chunk := frame.GetChunk(); chunk != nil {
			content.Write(chunk.GetData())
		}
	}
	return content.String()
}
