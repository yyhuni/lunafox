package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	"github.com/yyhuni/lunafox/server/internal/grpc/agentdata"
	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type providerConfigSource interface {
	GetExecutionSubfinderProviderConfig(context.Context) (string, error)
}

func catalogExecutionProviderSource(store catalogapp.ExecutionSubfinderProviderSettingsStore) *catalogapp.ExecutionProviderConfigSource {
	return catalogapp.NewExecutionProviderConfigSource(store)
}

type smokeProviderSettingsStore struct{}

func (*smokeProviderSettingsStore) GetInstanceContext(ctx context.Context) (*catalogdomain.SubfinderProviderSettings, error) {
	if ctx == nil {
		return nil, errors.New("provider settings context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &catalogdomain.SubfinderProviderSettings{Providers: catalogdomain.SubfinderProviderConfigs{}}, nil
}

type smokeArtifactResolver struct {
	bridge         *smokeTaskBridge
	providerConfig providerConfigSource
}

type smokeArtifactRecord struct {
	taskID    int
	scanID    int
	targetID  int
	planBytes []byte
	wordlists map[string][]byte
}

func (resolver *smokeArtifactResolver) record(lease agentdata.AgentExecutionLease, task, execution string) (smokeArtifactRecord, *agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	if resolver == nil || resolver.bridge == nil || lease.AgentID != smokeAgentID {
		return smokeArtifactRecord{}, nil, errors.New("artifact lease is invalid")
	}
	resolver.bridge.authority.mu.RLock()
	currentSession := resolver.bridge.authority.agent.SessionID
	currentEpoch := resolver.bridge.authority.agent.SessionEpoch
	resolver.bridge.authority.mu.RUnlock()
	if currentSession != lease.SessionID || currentEpoch != lease.SessionEpoch {
		return smokeArtifactRecord{}, nil, errors.New("artifact lease is stale")
	}
	resolver.bridge.mu.Lock()
	record := resolver.bridge.records[task]
	if record == nil || !record.assignmentDelivered || record.terminal != nil || record.claimSessionID != lease.SessionID || record.claimSessionEpoch != lease.SessionEpoch {
		resolver.bridge.mu.Unlock()
		return smokeArtifactRecord{}, nil, errors.New("artifact task scope is not an active claim")
	}
	artifactRecord := smokeArtifactRecord{
		taskID: record.taskID, scanID: record.scanID, targetID: record.targetID,
		planBytes: append([]byte(nil), record.planBytes...),
		wordlists: make(map[string][]byte, len(record.wordlists)),
	}
	for name, content := range record.wordlists {
		artifactRecord.wordlists[name] = append([]byte(nil), content...)
	}
	resolver.bridge.mu.Unlock()
	plan, err := decodeSmokePlanBytes(artifactRecord.planBytes)
	if err != nil {
		return smokeArtifactRecord{}, nil, err
	}
	if plan.GetTask() != task || plan.GetExecution() != execution {
		return smokeArtifactRecord{}, nil, errors.New("artifact task scope is unknown")
	}
	return artifactRecord, plan, nil
}

func smokeExecutionTarget(plan *agentexecutionv1.ResolvedEngineExecutionPlan) (scanapp.ExecutionTarget, error) {
	if plan == nil || plan.GetTarget() == nil {
		return scanapp.ExecutionTarget{}, errors.New("saved plan Target is required")
	}
	targetType := ""
	switch plan.GetTarget().GetType() {
	case agentexecutionv1.TargetType_TARGET_TYPE_DOMAIN:
		targetType = scanapp.ExecutionTargetTypeDomain
	case agentexecutionv1.TargetType_TARGET_TYPE_IP:
		targetType = scanapp.ExecutionTargetTypeIP
	case agentexecutionv1.TargetType_TARGET_TYPE_CIDR:
		targetType = scanapp.ExecutionTargetTypeCIDR
	default:
		return scanapp.ExecutionTarget{}, errors.New("saved plan Target type is unsupported")
	}
	return scanapp.ExecutionTarget{Resource: plan.GetTarget().GetResource(), Type: targetType, Value: plan.GetTarget().GetValue()}, nil
}

type smokeEmptyHostPortCursor struct{}

func (smokeEmptyHostPortCursor) ForEachHostPortByScanID(ctx context.Context, _ int, _ func(scanapp.HostPortEvidence) error) error {
	if ctx == nil {
		return errors.New("HostPort cursor context is required")
	}
	return ctx.Err()
}

type smokeEmptyDNSCursor struct{}

func (smokeEmptyDNSCursor) ForEachDNSNameByScanID(ctx context.Context, _ int, _ func(string) error) error {
	if ctx == nil {
		return errors.New("DNS cursor context is required")
	}
	return ctx.Err()
}

type smokeEmptyWebsiteURLCursor struct{}

func (smokeEmptyWebsiteURLCursor) ForEachWebsiteURLByScanID(ctx context.Context, _ int, _ func(string) error) error {
	if ctx == nil {
		return errors.New("Website URL cursor context is required")
	}
	return ctx.Err()
}

func (smokeEmptyWebsiteURLCursor) ForEachEndpointURLByScanID(ctx context.Context, _ int, _ func(string) error) error {
	if ctx == nil {
		return errors.New("Endpoint URL cursor context is required")
	}
	return ctx.Err()
}

type smokeEmptyBlacklistSnapshotStore struct{}

func (smokeEmptyBlacklistSnapshotStore) LoadBlacklistSnapshot(ctx context.Context, scanID int) ([]string, error) {
	if ctx == nil || scanID <= 0 {
		return nil, errors.New("smoke blacklist snapshot inputs are required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []string{}, nil
}

func newSmokeExecutionInputBlacklistSnapshotSource(store scanapp.ScanBlacklistSnapshotStore) agentdata.ExecutionInputBlacklistSnapshotSource {
	return agentdata.NewExecutionInputBlacklistSnapshotSource(store)
}

func newSmokeEmptyExecutionInputBlacklistSnapshotSource() agentdata.ExecutionInputBlacklistSnapshotSource {
	return newSmokeExecutionInputBlacklistSnapshotSource(smokeEmptyBlacklistSnapshotStore{})
}

func newSmokeEmptyExecutionInputBlacklistFilter() (*scanapp.ExecutionInputBlacklistFilter, error) {
	matcher, err := blacklistdomain.CompileMatcher([]string{})
	if err != nil {
		return nil, err
	}
	return scanapp.NewExecutionInputBlacklistFilter(matcher)
}

func (resolver *smokeArtifactResolver) AuthorizeExecutionInput(ctx context.Context, lease agentdata.AgentExecutionLease, request *agentdatav1.StreamExecutionInputRequest) (agentdata.AuthorizedExecutionArtifact, error) {
	if ctx == nil || request == nil {
		return agentdata.AuthorizedExecutionArtifact{}, errors.New("execution input request is required")
	}
	record, _, err := resolver.record(lease, request.GetTask(), request.GetExecution())
	if err != nil {
		return agentdata.AuthorizedExecutionArtifact{}, err
	}
	switch request.GetRole() {
	case executionartifact.RoleSubdomains:
		blacklist, err := newSmokeEmptyExecutionInputBlacklistFilter()
		if err != nil {
			return agentdata.AuthorizedExecutionArtifact{}, err
		}
		producer, err := scanapp.NewSubdomainsProducer(ctx, record.scanID, smokeEmptyDNSCursor{}, blacklist)
		if err != nil {
			return agentdata.AuthorizedExecutionArtifact{}, err
		}
		return agentdata.AuthorizedExecutionArtifact{
			ContentType: executionartifact.ContentTypeSubdomains,
			Produce:     resolver.trackedLineProducer(record.taskID, "subdomains", producer),
		}, nil
	case executionartifact.RoleHostPorts:
		blacklist, err := newSmokeEmptyExecutionInputBlacklistFilter()
		if err != nil {
			return agentdata.AuthorizedExecutionArtifact{}, err
		}
		producer, err := scanapp.NewHostPortsProducer(ctx, record.scanID, smokeEmptyHostPortCursor{}, blacklist)
		if err != nil {
			return agentdata.AuthorizedExecutionArtifact{}, err
		}
		return agentdata.AuthorizedExecutionArtifact{
			ContentType: executionartifact.ContentTypeHostPorts,
			Produce:     resolver.trackedHostPortProducer(record.taskID, "hostPorts", producer),
		}, nil
	case executionartifact.RoleWebsiteURLs:
		blacklist, err := newSmokeEmptyExecutionInputBlacklistFilter()
		if err != nil {
			return agentdata.AuthorizedExecutionArtifact{}, err
		}
		producer, err := scanapp.NewWebsiteURLsProducer(ctx, record.scanID, smokeEmptyWebsiteURLCursor{}, blacklist)
		if err != nil {
			return agentdata.AuthorizedExecutionArtifact{}, err
		}
		return agentdata.AuthorizedExecutionArtifact{ContentType: executionartifact.ContentTypeWebsiteURLs, Produce: resolver.trackedWebsiteURLsProducer(record.taskID, "websiteURLs", producer)}, nil
	default:
		return agentdata.AuthorizedExecutionArtifact{}, errors.New("execution input role is unsupported")
	}
}

func (resolver *smokeArtifactResolver) AuthorizeConfigResource(ctx context.Context, lease agentdata.AgentExecutionLease, request *agentdatav1.StreamConfigResourceRequest) (agentdata.AuthorizedExecutionArtifact, error) {
	if ctx == nil || request == nil {
		return agentdata.AuthorizedExecutionArtifact{}, errors.New("config resource request is required")
	}
	record, plan, err := resolver.record(lease, request.GetTask(), request.GetExecution())
	if err != nil {
		return agentdata.AuthorizedExecutionArtifact{}, err
	}
	for _, binding := range plan.GetConfigResourceBindings() {
		if binding.GetSectionId() != request.GetSectionId() || binding.GetParamKey() != request.GetParamKey() {
			continue
		}
		if binding.GetContentType() != executionartifact.ContentTypeWordlist || binding.GetWordlist() == nil {
			return agentdata.AuthorizedExecutionArtifact{}, errors.New("wordlist binding is invalid")
		}
		resourceName := binding.GetWordlist().GetResource()
		_, parseErr := resourcenames.ParseWordlist(resourceName)
		if parseErr != nil {
			return agentdata.AuthorizedExecutionArtifact{}, errors.New("wordlist binding resource is invalid")
		}
		content := record.wordlists[resourceName]
		if len(content) == 0 {
			return agentdata.AuthorizedExecutionArtifact{}, errors.New("wordlist fixture is unavailable")
		}
		count := binding.GetWordlist().GetLineCount()
		return agentdata.AuthorizedExecutionArtifact{
			ContentType: binding.GetContentType(),
			ExpectedIntegrity: &agentdatav1.ExecutionArtifactIntegrity{
				SizeBytes: uint64(len(content)), Sha256Digest: binding.GetWordlist().GetSha256Digest(), RecordCount: &count,
			},
			Produce: resolver.trackedByteProducer(record.taskID, "config", content, count),
		}, nil
	}
	return agentdata.AuthorizedExecutionArtifact{}, errors.New("config resource binding is not declared by plan")
}

func (resolver *smokeArtifactResolver) AuthorizePlatformResource(ctx context.Context, lease agentdata.AgentExecutionLease, request *agentdatav1.StreamPlatformResourceContentRequest) (agentdata.AuthorizedExecutionArtifact, error) {
	if ctx == nil || request == nil {
		return agentdata.AuthorizedExecutionArtifact{}, errors.New("platform resource request is required")
	}
	record, plan, err := resolver.record(lease, request.GetTask(), request.GetExecution())
	if err != nil {
		return agentdata.AuthorizedExecutionArtifact{}, err
	}
	for _, binding := range plan.GetPlatformResourceBindings() {
		if binding.GetResourceId() != request.GetResourceId() {
			continue
		}
		if binding.GetResourceId() != engineexecution.PlatformResourceSubfinderProviderConfig || binding.GetContentType() != executionartifact.ContentTypeSubfinderProviderConfig {
			return agentdata.AuthorizedExecutionArtifact{}, errors.New("platform resource binding is invalid")
		}
		if resolver.providerConfig == nil {
			return agentdata.AuthorizedExecutionArtifact{}, errors.New("provider config source is unavailable")
		}
		content, err := resolver.providerConfig.GetExecutionSubfinderProviderConfig(ctx)
		if err != nil {
			return agentdata.AuthorizedExecutionArtifact{}, err
		}
		if strings.TrimSpace(content) == "" {
			return agentdata.AuthorizedExecutionArtifact{}, errors.New("provider config source returned empty content")
		}
		return agentdata.AuthorizedExecutionArtifact{
			ContentType: binding.GetContentType(),
			Produce:     resolver.trackedByteProducer(record.taskID, "platform", []byte(content), 0),
		}, nil
	}
	return agentdata.AuthorizedExecutionArtifact{}, errors.New("platform resource binding is not declared by plan")
}

type smokeCountingWriter struct {
	writer io.Writer
	bytes  uint64
}

func (writer *smokeCountingWriter) Write(value []byte) (int, error) {
	written, err := writer.writer.Write(value)
	writer.bytes += uint64(written)
	return written, err
}

func (resolver *smokeArtifactResolver) trackedLineProducer(taskID int, kind string, producer func(func(string) error) error) agentdata.ExecutionArtifactProducer {
	return func(ctx context.Context, writer io.Writer) (uint64, error) {
		counter := &smokeCountingWriter{writer: writer}
		recordCount, err := scanapp.WriteCanonicalLineStream(ctx, counter, producer)
		if err == nil {
			resolver.bridge.observeArtifact(taskID, kind, recordCount, counter.bytes)
		}
		return recordCount, err
	}
}

// trackedWebsiteURLsProducer keeps the smoke lane on the same raw URL product
// contract as production. The generic line producer is intentionally reserved
// for canonical fact products such as Subdomains.
func (resolver *smokeArtifactResolver) trackedWebsiteURLsProducer(taskID int, kind string, producer scanapp.WebsiteURLsProducer) agentdata.ExecutionArtifactProducer {
	return func(ctx context.Context, writer io.Writer) (uint64, error) {
		counter := &smokeCountingWriter{writer: writer}
		recordCount, err := scanapp.WriteObservedURLLineStream(ctx, counter, producer)
		if err == nil {
			resolver.bridge.observeArtifact(taskID, kind, recordCount, counter.bytes)
		}
		return recordCount, err
	}
}

func (resolver *smokeArtifactResolver) trackedHostPortProducer(taskID int, kind string, producer scanapp.HostPortsProducer) agentdata.ExecutionArtifactProducer {
	return func(ctx context.Context, writer io.Writer) (uint64, error) {
		counter := &smokeCountingWriter{writer: writer}
		recordCount, err := scanapp.WriteHostPortJSONL(ctx, counter, producer)
		if err == nil {
			resolver.bridge.observeArtifact(taskID, kind, recordCount, counter.bytes)
		}
		return recordCount, err
	}
}

func (resolver *smokeArtifactResolver) trackedByteProducer(taskID int, kind string, content []byte, recordCount uint64) agentdata.ExecutionArtifactProducer {
	content = append([]byte(nil), content...)
	return func(ctx context.Context, writer io.Writer) (uint64, error) {
		if ctx == nil {
			return 0, errors.New("artifact stream context is required")
		}
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		written, err := writer.Write(content)
		if err != nil {
			return 0, err
		}
		if written != len(content) {
			return 0, io.ErrShortWrite
		}
		resolver.bridge.observeArtifact(taskID, kind, recordCount, uint64(written))
		return recordCount, nil
	}
}

type smokeWordlists struct {
	mu       sync.RWMutex
	root     string
	entries  map[string]scanapp.PlanTaskWordlist
	contents map[string][]byte
	paths    map[string]string
}

func (wordlists *smokeWordlists) ResolveConfigResource(_ context.Context, request scanapp.ConfigResourceResolveRequest) (scanapp.PlanTaskWordlist, error) {
	if wordlists == nil {
		return scanapp.PlanTaskWordlist{}, scanapp.NewConfigResourceInternalError(
			request.Field, request.ResourceKind, request.ResourceName, errors.New("smoke wordlist source is unavailable"),
		)
	}
	if request.ResourceKind != engineexecution.ConfigResourceKindWordlist {
		return scanapp.PlanTaskWordlist{}, scanapp.NewConfigResourceInternalError(
			request.Field, request.ResourceKind, request.ResourceName, fmt.Errorf("unsupported config resource kind %q", request.ResourceKind),
		)
	}
	wordlistID, err := resourcenames.ParseWordlist(strings.TrimSpace(request.ResourceName))
	if err != nil || resourcenames.Wordlist(wordlistID) != strings.TrimSpace(request.ResourceName) {
		return scanapp.PlanTaskWordlist{}, &scanapp.ConfigResourceInputError{
			Field: request.Field, ResourceKind: request.ResourceKind, ResourceName: request.ResourceName,
			Err: fmt.Errorf("wordlist resource must be canonical: %w", err),
		}
	}
	wordlists.mu.RLock()
	defer wordlists.mu.RUnlock()
	resourceName := resourcenames.Wordlist(wordlistID)
	value, ok := wordlists.entries[resourceName]
	if !ok {
		return scanapp.PlanTaskWordlist{}, scanapp.NewConfigResourceUnavailableError(
			request.Field, request.ResourceKind, resourceName, fmt.Errorf("wordlist %q is unavailable", resourceName),
		)
	}
	return value, nil
}

func (wordlists *smokeWordlists) GetByResourceName(ctx context.Context, resourceName string) (*catalogapp.Wordlist, error) {
	if wordlists == nil || ctx == nil {
		return nil, errors.New("smoke wordlist source is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	wordlistID, err := resourcenames.ParseWordlist(resourceName)
	if err != nil || resourcenames.Wordlist(wordlistID) != strings.TrimSpace(resourceName) {
		return nil, catalogapp.ErrWordlistNotFound
	}
	wordlists.mu.RLock()
	entry, ok := wordlists.entries[resourceName]
	path := wordlists.paths[resourceName]
	wordlists.mu.RUnlock()
	if !ok || path == "" {
		return nil, catalogapp.ErrWordlistNotFound
	}
	return &catalogapp.Wordlist{
		ID: wordlistID, FileName: entry.Basename, FilePath: path, FileSize: entry.SizeBytes,
		LineCount: int(entry.LineCount), FileHash: strings.TrimPrefix(entry.SHA256, "sha256:"),
	}, nil
}

func (wordlists *smokeWordlists) GetFilePathByResourceName(ctx context.Context, resourceName string) (string, error) {
	if wordlists == nil || ctx == nil {
		return "", errors.New("smoke wordlist source is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	wordlistID, err := resourcenames.ParseWordlist(resourceName)
	if err != nil || resourcenames.Wordlist(wordlistID) != strings.TrimSpace(resourceName) {
		return "", catalogapp.ErrWordlistNotFound
	}
	wordlists.mu.RLock()
	path := wordlists.paths[resourceName]
	wordlists.mu.RUnlock()
	if path == "" {
		return "", catalogapp.ErrWordlistNotFound
	}
	return path, nil
}

func newSmokeWordlists() (*smokeWordlists, error) {
	root, err := os.MkdirTemp("", "lunafox-engine-cut-wordlists-")
	if err != nil {
		return nil, err
	}
	result := &smokeWordlists{root: root, entries: map[string]scanapp.PlanTaskWordlist{}, contents: map[string][]byte{}, paths: map[string]string{}}
	fixtures := []struct {
		id       int
		fileName string
		content  []byte
	}{
		{id: 1, fileName: "subdomains-top1million-110000.txt", content: []byte("api\nwww\n")},
		{id: 2, fileName: "resolvers.txt", content: []byte("1.1.1.1\n8.8.8.8\n")},
	}
	for _, fixture := range fixtures {
		path := filepath.Join(root, fixture.fileName)
		resourceName := resourcenames.Wordlist(fixture.id)
		content := fixture.content
		if err := os.WriteFile(path, content, 0o640); err != nil {
			_ = os.RemoveAll(root)
			return nil, err
		}
		sum := sha256.Sum256(content)
		result.entries[resourceName] = scanapp.PlanTaskWordlist{Resource: resourceName, Basename: fixture.fileName, SizeBytes: int64(len(content)), LineCount: int64(countCanonicalLines(content)), SHA256: "sha256:" + hex.EncodeToString(sum[:])}
		result.contents[resourceName] = append([]byte(nil), content...)
		result.paths[resourceName] = path
	}
	return result, nil
}

func (wordlists *smokeWordlists) cleanup() {
	if wordlists == nil || wordlists.root == "" {
		return
	}
	_ = os.RemoveAll(wordlists.root)
}

func countCanonicalLines(content []byte) uint64 {
	if len(content) == 0 {
		return 0
	}
	var count uint64
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

var (
	_ agentdata.AgentFinder             = (*smokeAuthority)(nil)
	_ scanapp.ConfigResourceResolver    = (*smokeWordlists)(nil)
	_ agentdata.ExecutionWordlistSource = (*smokeWordlists)(nil)
)
