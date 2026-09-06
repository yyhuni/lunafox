package agentdata

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

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	fingerprintapp "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/application"
	fingerprintdomain "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	nucleipocdomain "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

// ExecutionArtifactTaskReader exposes the persisted saved-plan lease for an
// already Workflow-authorized task. Workflow owns predecessor evaluation;
// artifact production only verifies the current task's lease and plan scope.
type ExecutionArtifactTaskReader interface {
	GetSavedExecutionPlanLease(ctx context.Context, taskID int) (*scandomain.SavedExecutionPlanLease, error)
}

// ExecutionArtifactSessionReader returns the fresh lease authority for an
// already-confirmed process session. A short detached-recoverable interval may
// keep existing task operations alive but can never admit a scheduler claim.
type ExecutionArtifactSessionReader interface {
	CurrentLeaseSession(agentID int) (agentcontrol.ActiveControlSession, bool)
}

// ExecutionWordlistSource keeps plan-pinned catalog reads under the artifact
// request context so cancellation and deadline remain scoped to the authorized
// pre-start stream.
type ExecutionWordlistSource interface {
	GetByResourceName(ctx context.Context, resourceName string) (*catalogapp.Wordlist, error)
	GetFilePathByResourceName(ctx context.Context, resourceName string) (string, error)
}

// ExecutionProviderConfigSource produces the complete late-bound provider
// content for one stream attempt.
type ExecutionProviderConfigSource interface {
	GetExecutionSubfinderProviderConfig(ctx context.Context) (string, error)
}

type ExecutionFingerprintArtifactSource interface {
	OpenCurrent(context.Context, fingerprintdomain.Library) (fingerprintapp.ArtifactHandle, error)
}

type ExecutionNucleiTemplateSource interface {
	ListEnabledForExecution(context.Context) ([]nucleipocdomain.POC, error)
}

// NucleiTemplateTaskFromContext returns the canonical task scope attached by
// the resolver while it performs a late-bound template read.  The helper is
// intentionally read-only: a catalog source may use it to select fixture or
// audit data, but it cannot change the authorized task or digest.
func NucleiTemplateTaskFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	value, ok := ctx.Value(nucleiTemplateTaskContextKey{}).(string)
	return value, ok && value != ""
}

type nucleiTemplateTaskContextKey struct{}

// ServerExecutionArtifactResolverDependencies are the authoritative Server
// sources used after a task has been claimed. None are package or workflow
// authoring readers.
type ServerExecutionArtifactResolverDependencies struct {
	Tasks                 ExecutionArtifactTaskReader
	Sessions              ExecutionArtifactSessionReader
	DNSNames              scanapp.DNSNameCursor
	HostPorts             scanapp.HostPortCursor
	WebsiteURLs           scanapp.WebsiteURLCursor
	EndpointURLs          scanapp.EndpointURLCursor
	InventoryDNSNames     scanapp.TargetDNSNameCursor
	InventoryHostPorts    scanapp.TargetHostPortCursor
	InventoryWebsiteURLs  scanapp.TargetWebsiteURLCursor
	InventoryEndpointURLs scanapp.TargetEndpointURLCursor
	BlacklistSnapshots    ExecutionInputBlacklistSnapshotSource
	Wordlists             ExecutionWordlistSource
	ProviderConfig        ExecutionProviderConfigSource
	FingerprintArtifacts  ExecutionFingerprintArtifactSource
	NucleiTemplates       ExecutionNucleiTemplateSource
}

type serverExecutionArtifactResolver struct {
	tasks                 ExecutionArtifactTaskReader
	sessions              ExecutionArtifactSessionReader
	dnsNames              scanapp.DNSNameCursor
	hostPorts             scanapp.HostPortCursor
	websiteURLs           scanapp.WebsiteURLCursor
	endpointURLs          scanapp.EndpointURLCursor
	inventoryDNSNames     scanapp.TargetDNSNameCursor
	inventoryHostPorts    scanapp.TargetHostPortCursor
	inventoryWebsiteURLs  scanapp.TargetWebsiteURLCursor
	inventoryEndpointURLs scanapp.TargetEndpointURLCursor
	blacklistSnapshots    ExecutionInputBlacklistSnapshotSource
	wordlists             ExecutionWordlistSource
	providerConfig        ExecutionProviderConfigSource
	fingerprintArtifacts  ExecutionFingerprintArtifactSource
	nucleiTemplates       ExecutionNucleiTemplateSource
}

// NewServerExecutionArtifactResolver builds the single production resolver
// shared by all typed artifact operations.
func NewServerExecutionArtifactResolver(dependencies ServerExecutionArtifactResolverDependencies) (ExecutionArtifactResolver, error) {
	if dependencies.Tasks == nil || dependencies.Sessions == nil || dependencies.DNSNames == nil || dependencies.HostPorts == nil || dependencies.WebsiteURLs == nil || dependencies.EndpointURLs == nil || dependencies.BlacklistSnapshots == nil || dependencies.Wordlists == nil || dependencies.ProviderConfig == nil || dependencies.NucleiTemplates == nil {
		return nil, fmt.Errorf("execution artifact resolver dependencies are incomplete")
	}
	resolver := &serverExecutionArtifactResolver{
		tasks: dependencies.Tasks, sessions: dependencies.Sessions,
		dnsNames:           availabilityDNSNameCursor{inner: dependencies.DNSNames},
		hostPorts:          availabilityHostPortCursor{inner: dependencies.HostPorts},
		websiteURLs:        dependencies.WebsiteURLs,
		endpointURLs:       dependencies.EndpointURLs,
		blacklistSnapshots: dependencies.BlacklistSnapshots,
		wordlists:          dependencies.Wordlists, providerConfig: dependencies.ProviderConfig, fingerprintArtifacts: dependencies.FingerprintArtifacts, nucleiTemplates: dependencies.NucleiTemplates,
	}
	if dependencies.InventoryDNSNames != nil {
		resolver.inventoryDNSNames = availabilityTargetDNSNameCursor{inner: dependencies.InventoryDNSNames}
	}
	if dependencies.InventoryHostPorts != nil {
		resolver.inventoryHostPorts = availabilityTargetHostPortCursor{inner: dependencies.InventoryHostPorts}
	}
	if dependencies.InventoryWebsiteURLs != nil {
		resolver.inventoryWebsiteURLs = availabilityTargetWebsiteURLCursor{inner: dependencies.InventoryWebsiteURLs}
	}
	if dependencies.InventoryEndpointURLs != nil {
		resolver.inventoryEndpointURLs = availabilityTargetEndpointURLCursor{inner: dependencies.InventoryEndpointURLs}
	}
	return resolver, nil
}

type artifactVisitorError struct{ err error }

func (failure artifactVisitorError) Error() string { return failure.err.Error() }
func (failure artifactVisitorError) Unwrap() error { return failure.err }

type availabilityDNSNameCursor struct{ inner scanapp.DNSNameCursor }

func (cursor availabilityDNSNameCursor) ForEachDNSNameByScanID(ctx context.Context, scanID int, visit func(string) error) error {
	err := cursor.inner.ForEachDNSNameByScanID(ctx, scanID, func(value string) error {
		if err := visit(value); err != nil {
			return artifactVisitorError{err: err}
		}
		return nil
	})
	var visitorFailure artifactVisitorError
	if errors.As(err, &visitorFailure) {
		return visitorFailure.err
	}
	if err != nil {
		return unavailableError("read current-scan DNS evidence", err)
	}
	return nil
}

type availabilityHostPortCursor struct {
	inner scanapp.HostPortCursor
}

type availabilityTargetDNSNameCursor struct{ inner scanapp.TargetDNSNameCursor }

func (cursor availabilityTargetDNSNameCursor) ForEachDNSNameByTargetID(ctx context.Context, targetID int, visit func(string) error) error {
	err := cursor.inner.ForEachDNSNameByTargetID(ctx, targetID, func(value string) error {
		if err := visit(value); err != nil {
			return artifactVisitorError{err: err}
		}
		return nil
	})
	var visitorFailure artifactVisitorError
	if errors.As(err, &visitorFailure) {
		return visitorFailure.err
	}
	if err != nil {
		return unavailableError("read current Target DNS evidence", err)
	}
	return nil
}

type availabilityTargetHostPortCursor struct{ inner scanapp.TargetHostPortCursor }

func (cursor availabilityTargetHostPortCursor) ForEachHostPortByTargetID(ctx context.Context, targetID int, visit func(scanapp.HostPortEvidence) error) error {
	err := cursor.inner.ForEachHostPortByTargetID(ctx, targetID, func(value scanapp.HostPortEvidence) error {
		if err := visit(value); err != nil {
			return artifactVisitorError{err: err}
		}
		return nil
	})
	var visitorFailure artifactVisitorError
	if errors.As(err, &visitorFailure) {
		return visitorFailure.err
	}
	if err != nil {
		return unavailableError("read current Target HostPort evidence", err)
	}
	return nil
}

type availabilityTargetWebsiteURLCursor struct {
	inner scanapp.TargetWebsiteURLCursor
}

type availabilityTargetEndpointURLCursor struct {
	inner scanapp.TargetEndpointURLCursor
}

func (cursor availabilityTargetEndpointURLCursor) ForEachEndpointURLByTargetID(ctx context.Context, targetID int, visit func(string) error) error {
	err := cursor.inner.ForEachEndpointURLByTargetID(ctx, targetID, func(value string) error {
		if err := visit(value); err != nil {
			return artifactVisitorError{err: err}
		}
		return nil
	})
	var visitorFailure artifactVisitorError
	if errors.As(err, &visitorFailure) {
		return visitorFailure.err
	}
	if err != nil {
		return unavailableError("read current Target Endpoint evidence", err)
	}
	return nil
}

func (cursor availabilityTargetWebsiteURLCursor) ForEachWebsiteURLByTargetID(ctx context.Context, targetID int, visit func(string) error) error {
	err := cursor.inner.ForEachWebsiteURLByTargetID(ctx, targetID, func(value string) error {
		if err := visit(value); err != nil {
			return artifactVisitorError{err: err}
		}
		return nil
	})
	var visitorFailure artifactVisitorError
	if errors.As(err, &visitorFailure) {
		return visitorFailure.err
	}
	if err != nil {
		return unavailableError("read current Target website evidence", err)
	}
	return nil
}

func (cursor availabilityHostPortCursor) ForEachHostPortByScanID(ctx context.Context, scanID int, visit func(scanapp.HostPortEvidence) error) error {
	err := cursor.inner.ForEachHostPortByScanID(ctx, scanID, func(value scanapp.HostPortEvidence) error {
		if err := visit(value); err != nil {
			return artifactVisitorError{err: err}
		}
		return nil
	})
	var visitorFailure artifactVisitorError
	if errors.As(err, &visitorFailure) {
		return visitorFailure.err
	}
	if err != nil {
		return unavailableError("read current-scan HostPort evidence", err)
	}
	return nil
}

type authorizedExecutionPlan struct {
	taskID      int
	scanID      int
	targetID    int
	inputSource scandomain.InputSource
	plan        *agentexecutionv1.ResolvedEngineExecutionPlan
}

func (resolver *serverExecutionArtifactResolver) AuthorizeExecutionInput(ctx context.Context, lease AgentExecutionLease, request *agentdatav1.StreamExecutionInputRequest) (AuthorizedExecutionArtifact, error) {
	role, _, err := validateExecutionInputRequest(request)
	if err != nil {
		return AuthorizedExecutionArtifact{}, err
	}
	descriptor, ok := executionartifact.LookupRoleID(request.GetRole())
	if !ok {
		return AuthorizedExecutionArtifact{}, permissionDeniedError("execution input role is not authorized")
	}
	authorized, err := resolver.authorizePlan(ctx, lease, request.GetTask(), request.GetExecution())
	if err != nil {
		return AuthorizedExecutionArtifact{}, err
	}
	switch role {
	case artifactRoleSubdomains:
		blacklist, err := resolver.resolveExecutionInputBlacklistFilter(ctx, authorized.scanID)
		if err != nil {
			return AuthorizedExecutionArtifact{}, err
		}
		producer, err := resolver.newSubdomainsProducer(ctx, authorized, blacklist)
		if err != nil {
			return AuthorizedExecutionArtifact{}, dataLossError(err)
		}
		return AuthorizedExecutionArtifact{
			ContentType:                  descriptor.ContentType,
			executionInputBlacklistStats: executionInputBlacklistStatsFromFilter(blacklist),
			Produce: func(ctx context.Context, writer io.Writer) (uint64, error) {
				return scanapp.WriteCanonicalLineStream(ctx, writer, producer)
			},
		}, nil
	case artifactRoleHostPorts:
		blacklist, err := resolver.resolveExecutionInputBlacklistFilter(ctx, authorized.scanID)
		if err != nil {
			return AuthorizedExecutionArtifact{}, err
		}
		producer, err := resolver.newHostPortsProducer(ctx, authorized, blacklist)
		if err != nil {
			return AuthorizedExecutionArtifact{}, dataLossError(err)
		}
		return AuthorizedExecutionArtifact{
			ContentType:                  descriptor.ContentType,
			executionInputBlacklistStats: executionInputBlacklistStatsFromFilter(blacklist),
			Produce: func(ctx context.Context, writer io.Writer) (uint64, error) {
				return scanapp.WriteHostPortJSONL(ctx, writer, producer)
			},
		}, nil
	case artifactRoleWebsiteURLs:
		blacklist, err := resolver.resolveExecutionInputBlacklistFilter(ctx, authorized.scanID)
		if err != nil {
			return AuthorizedExecutionArtifact{}, err
		}
		producer, err := resolver.newWebsiteURLsProducer(ctx, authorized, blacklist)
		if err != nil {
			return AuthorizedExecutionArtifact{}, dataLossError(err)
		}
		return AuthorizedExecutionArtifact{
			ContentType:                  descriptor.ContentType,
			executionInputBlacklistStats: executionInputBlacklistStatsFromFilter(blacklist),
			Produce: func(ctx context.Context, writer io.Writer) (uint64, error) {
				return scanapp.WriteObservedURLLineStream(ctx, writer, producer)
			},
		}, nil
	case artifactRoleEndpointURLs:
		blacklist, err := resolver.resolveExecutionInputBlacklistFilter(ctx, authorized.scanID)
		if err != nil {
			return AuthorizedExecutionArtifact{}, err
		}
		producer, err := resolver.newEndpointURLsProducer(ctx, authorized, blacklist)
		if err != nil {
			return AuthorizedExecutionArtifact{}, dataLossError(err)
		}
		return AuthorizedExecutionArtifact{
			ContentType:                  descriptor.ContentType,
			executionInputBlacklistStats: executionInputBlacklistStatsFromFilter(blacklist),
			Produce: func(ctx context.Context, writer io.Writer) (uint64, error) {
				return scanapp.WriteObservedURLLineStream(ctx, writer, producer)
			},
		}, nil
	default:
		return AuthorizedExecutionArtifact{}, permissionDeniedError("execution input role is not authorized")
	}
}

func (resolver *serverExecutionArtifactResolver) newSubdomainsProducer(ctx context.Context, authorized authorizedExecutionPlan, blacklist *scanapp.ExecutionInputBlacklistFilter) (scanapp.SubdomainsProducer, error) {
	switch authorized.inputSource {
	case scandomain.InputSourceScanSnapshot:
		return scanapp.NewSubdomainsProducer(ctx, authorized.scanID, resolver.dnsNames, blacklist)
	case scandomain.InputSourceTargetInventory:
		if resolver.inventoryDNSNames == nil {
			return nil, unavailableError("read current Target DNS evidence", errors.New("Target inventory cursor is unavailable"))
		}
		return scanapp.NewTargetInventorySubdomainsProducer(ctx, authorized.targetID, resolver.inventoryDNSNames, blacklist)
	default:
		return nil, dataLossError(fmt.Errorf("persisted Scan input source is invalid: %q", authorized.inputSource))
	}
}

func (resolver *serverExecutionArtifactResolver) newHostPortsProducer(ctx context.Context, authorized authorizedExecutionPlan, blacklist *scanapp.ExecutionInputBlacklistFilter) (scanapp.HostPortsProducer, error) {
	switch authorized.inputSource {
	case scandomain.InputSourceScanSnapshot:
		return scanapp.NewHostPortsProducer(ctx, authorized.scanID, resolver.hostPorts, blacklist)
	case scandomain.InputSourceTargetInventory:
		if resolver.inventoryHostPorts == nil {
			return nil, unavailableError("read current Target HostPort evidence", errors.New("Target inventory cursor is unavailable"))
		}
		return scanapp.NewTargetInventoryHostPortsProducer(ctx, authorized.targetID, resolver.inventoryHostPorts, blacklist)
	default:
		return nil, dataLossError(fmt.Errorf("persisted Scan input source is invalid: %q", authorized.inputSource))
	}
}

func (resolver *serverExecutionArtifactResolver) newWebsiteURLsProducer(ctx context.Context, authorized authorizedExecutionPlan, blacklist *scanapp.ExecutionInputBlacklistFilter) (scanapp.WebsiteURLsProducer, error) {
	switch authorized.inputSource {
	case scandomain.InputSourceScanSnapshot:
		return scanapp.NewWebsiteURLsProducer(ctx, authorized.scanID, resolver.websiteURLs, blacklist)
	case scandomain.InputSourceTargetInventory:
		if resolver.inventoryWebsiteURLs == nil {
			return nil, unavailableError("read current Target website evidence", errors.New("Target inventory cursor is unavailable"))
		}
		return scanapp.NewTargetInventoryWebsiteURLsProducer(ctx, authorized.targetID, resolver.inventoryWebsiteURLs, blacklist)
	default:
		return nil, dataLossError(fmt.Errorf("persisted Scan input source is invalid: %q", authorized.inputSource))
	}
}

func (resolver *serverExecutionArtifactResolver) newEndpointURLsProducer(ctx context.Context, authorized authorizedExecutionPlan, blacklist *scanapp.ExecutionInputBlacklistFilter) (scanapp.EndpointURLsProducer, error) {
	switch authorized.inputSource {
	case scandomain.InputSourceScanSnapshot:
		return scanapp.NewEndpointURLsProducer(ctx, authorized.scanID, resolver.endpointURLs, blacklist)
	case scandomain.InputSourceTargetInventory:
		if resolver.inventoryEndpointURLs == nil {
			return nil, unavailableError("read current Target Endpoint evidence", errors.New("Target inventory cursor is unavailable"))
		}
		return scanapp.NewTargetInventoryEndpointURLsProducer(ctx, authorized.targetID, resolver.inventoryEndpointURLs, blacklist)
	default:
		return nil, dataLossError(fmt.Errorf("persisted Scan input source is invalid: %q", authorized.inputSource))
	}
}

func executionInputBlacklistStatsFromFilter(filter *scanapp.ExecutionInputBlacklistFilter) func() executionInputBlacklistStats {
	if filter == nil {
		return nil
	}
	return func() executionInputBlacklistStats {
		stats := filter.Stats()
		return executionInputBlacklistStats{examined: stats.Examined, excluded: stats.Excluded, emitted: stats.Emitted}
	}
}

func (resolver *serverExecutionArtifactResolver) resolveExecutionInputBlacklistFilter(ctx context.Context, scanID int) (*scanapp.ExecutionInputBlacklistFilter, error) {
	if resolver == nil || resolver.blacklistSnapshots == nil {
		return nil, unavailableError("read frozen Scan blacklist snapshot", errors.New("snapshot source is unavailable"))
	}
	filter, err := resolver.blacklistSnapshots.ResolveExecutionInputBlacklistFilter(ctx, scanID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrExecutionArtifactDataLoss) || errors.Is(err, ErrExecutionArtifactUnavailable) {
			return nil, err
		}
		return nil, unavailableError("read frozen Scan blacklist snapshot", err)
	}
	if filter == nil {
		return nil, dataLossError(errors.New("frozen Scan blacklist snapshot returned no matcher"))
	}
	return filter, nil
}

func (resolver *serverExecutionArtifactResolver) AuthorizeConfigResource(ctx context.Context, lease AgentExecutionLease, request *agentdatav1.StreamConfigResourceRequest) (AuthorizedExecutionArtifact, error) {
	if _, err := validateConfigResourceRequest(request); err != nil {
		return AuthorizedExecutionArtifact{}, err
	}
	authorized, err := resolver.authorizePlan(ctx, lease, request.GetTask(), request.GetExecution())
	if err != nil {
		return AuthorizedExecutionArtifact{}, err
	}
	var selected *agentexecutionv1.ConfigResourceBinding
	for _, binding := range authorized.plan.GetConfigResourceBindings() {
		if binding.GetSectionId() == request.GetSectionId() && binding.GetParamKey() == request.GetParamKey() {
			selected = binding
			break
		}
	}
	if selected == nil {
		return AuthorizedExecutionArtifact{}, permissionDeniedError("config resource binding is not authorized")
	}
	if err := validatePlanRoleContent(executionartifact.RoleWordlistConfigResource, selected.GetContentType()); err != nil {
		return AuthorizedExecutionArtifact{}, dataLossError(err)
	}
	producer, err := resolver.resolveWordlistProducer(ctx, selected.GetWordlist())
	if err != nil {
		return AuthorizedExecutionArtifact{}, dataLossError(err)
	}
	recordCount := selected.GetWordlist().GetLineCount()
	return AuthorizedExecutionArtifact{
		ContentType: selected.GetContentType(),
		ExpectedIntegrity: &agentdatav1.ExecutionArtifactIntegrity{
			SizeBytes: selected.GetWordlist().GetSizeBytes(), Sha256Digest: selected.GetWordlist().GetSha256Digest(), RecordCount: &recordCount,
		},
		Produce: producer,
	}, nil
}

func (resolver *serverExecutionArtifactResolver) AuthorizePlatformResource(ctx context.Context, lease AgentExecutionLease, request *agentdatav1.StreamPlatformResourceContentRequest) (AuthorizedExecutionArtifact, error) {
	if _, err := validatePlatformResourceRequest(request); err != nil {
		return AuthorizedExecutionArtifact{}, err
	}
	authorized, err := resolver.authorizePlan(ctx, lease, request.GetTask(), request.GetExecution())
	if err != nil {
		return AuthorizedExecutionArtifact{}, err
	}
	var selected *agentexecutionv1.PlatformResourceBinding
	for _, binding := range authorized.plan.GetPlatformResourceBindings() {
		if binding.GetResourceId() == request.GetResourceId() {
			selected = binding
			break
		}
	}
	if selected == nil {
		return AuthorizedExecutionArtifact{}, permissionDeniedError("platform resource binding is not authorized")
	}
	descriptor, ok := executionartifact.LookupPlatformResource(selected.GetResourceId())
	if !ok {
		return AuthorizedExecutionArtifact{}, dataLossError(fmt.Errorf("unsupported platform resource binding %q", selected.GetResourceId()))
	}
	if err := validatePlanRoleContent(descriptor.Role, selected.GetContentType()); err != nil {
		return AuthorizedExecutionArtifact{}, dataLossError(err)
	}
	if executionartifact.IsFingerprintLibraryRole(descriptor.Role) {
		return resolver.resolveFingerprintArtifact(ctx, descriptor, selected.GetContentType())
	}
	content, err := resolver.providerConfig.GetExecutionSubfinderProviderConfig(ctx)
	if err != nil {
		if errors.Is(err, catalogapp.ErrExecutionProviderConfigInvalid) {
			return AuthorizedExecutionArtifact{}, fmt.Errorf("produce subfinder provider config: %w", err)
		}
		return AuthorizedExecutionArtifact{}, unavailableError("produce subfinder provider config", err)
	}
	if strings.TrimSpace(content) == "" {
		return AuthorizedExecutionArtifact{}, fmt.Errorf("provider config source returned empty content")
	}
	return AuthorizedExecutionArtifact{
		ContentType: selected.GetContentType(),
		Produce: func(ctx context.Context, writer io.Writer) (uint64, error) {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
			_, err := io.WriteString(writer, content)
			return 0, err
		},
	}, nil
}

// AuthorizeRuntimeArtifactExchange performs the complete authorization and
// reads the enabled catalog exactly once for this exchange.  The returned
// boundary is memory-only and is released by the bidi service; Server never
// persists a task snapshot, generation, directory, or spool.
func (resolver *serverExecutionArtifactResolver) AuthorizeRuntimeArtifactExchange(ctx context.Context, lease AgentExecutionLease, begin *agentdatav1.RuntimeArtifactExchangeBegin) (*AuthorizedRuntimeArtifactExchange, error) {
	if err := validateRuntimeArtifactExchangeBegin(begin); err != nil {
		return nil, err
	}
	authorized, err := resolver.authorizePlan(ctx, lease, begin.GetTask(), begin.GetExecution())
	if err != nil {
		return nil, err
	}
	if !planDeclaresRuntimeArtifact(authorized.plan, "nucleiTemplates") {
		return nil, permissionDeniedError("runtime artifact is not declared by the saved plan")
	}
	if resolver == nil || resolver.nucleiTemplates == nil {
		return nil, unavailableError("read enabled Nuclei templates", errors.New("Nuclei template source is unavailable"))
	}
	templateCtx := context.WithValue(ctx, nucleiTemplateTaskContextKey{}, begin.GetTask())
	templates, err := resolver.nucleiTemplates.ListEnabledForExecution(templateCtx)
	if err != nil {
		return nil, unavailableError("read enabled Nuclei templates", err)
	}
	boundary, err := buildNucleiTemplateExchange(templates)
	if err != nil {
		return nil, err
	}
	entries := append([]RuntimeArtifactManifestEntry(nil), boundary.entries...)
	return &AuthorizedRuntimeArtifactExchange{Entries: entries, SnapshotDigest: boundary.snapshotDigest, boundary: boundary}, nil
}

func planDeclaresRuntimeArtifact(plan *agentexecutionv1.ResolvedEngineExecutionPlan, artifactID string) bool {
	if plan == nil || artifactID == "" {
		return false
	}
	for _, binding := range plan.GetRuntimeArtifactBindings() {
		descriptor, known := executionartifact.LookupRuntimeArtifactID(binding.GetArtifactId())
		if known && binding.GetArtifactId() == artifactID && binding.GetContentType() == descriptor.ContentType {
			return true
		}
	}
	return false
}

func (resolver *serverExecutionArtifactResolver) resolveFingerprintArtifact(ctx context.Context, descriptor executionartifact.Descriptor, contentType string) (AuthorizedExecutionArtifact, error) {
	if resolver.fingerprintArtifacts == nil {
		return AuthorizedExecutionArtifact{}, unavailableError("resolve current fingerprint artifact", errors.New("fingerprint artifact source is unavailable"))
	}
	library, err := fingerprintdomain.ParseLibrary(descriptor.Library)
	if err != nil {
		return AuthorizedExecutionArtifact{}, dataLossError(err)
	}
	handle, err := resolver.fingerprintArtifacts.OpenCurrent(ctx, library)
	if err != nil {
		return AuthorizedExecutionArtifact{}, unavailableError("resolve current fingerprint artifact", err)
	}
	recordCount := uint64(handle.Descriptor.RecordCount)
	size := uint64(handle.Descriptor.SizeBytes)
	return AuthorizedExecutionArtifact{
		ContentType:       contentType,
		ExpectedIntegrity: &agentdatav1.ExecutionArtifactIntegrity{SizeBytes: size, Sha256Digest: handle.Descriptor.SHA256Digest, RecordCount: &recordCount},
		Produce: func(ctx context.Context, writer io.Writer) (uint64, error) {
			defer handle.Reader.Close()
			if err := ctx.Err(); err != nil {
				return 0, err
			}
			_, err := io.Copy(writer, handle.Reader)
			return recordCount, err
		},
	}, nil
}

func (resolver *serverExecutionArtifactResolver) authorizePlan(ctx context.Context, caller AgentExecutionLease, task, execution string) (authorizedExecutionPlan, error) {
	if resolver == nil || resolver.tasks == nil || resolver.sessions == nil {
		return authorizedExecutionPlan{}, fmt.Errorf("execution artifact resolver is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return authorizedExecutionPlan{}, err
	}
	scanID, taskID, err := resourcenames.ParseTask(task)
	if err != nil || resourcenames.Task(scanID, taskID) != task {
		return authorizedExecutionPlan{}, permissionDeniedError("task scope is not canonical")
	}
	lease, err := resolver.tasks.GetSavedExecutionPlanLease(ctx, taskID)
	if err != nil {
		if errors.Is(err, scandomain.ErrSavedExecutionPlanLeaseNotFound) {
			return authorizedExecutionPlan{}, permissionDeniedError("task scope has no persisted execution-plan lease")
		}
		// The lease reader validates input_source while decoding the Scan row. A
		// rejected persisted value is immutable-state corruption, not a transient
		// lookup failure that an Agent retry could repair.
		if errors.Is(err, scandomain.ErrInvalidInputSource) {
			return authorizedExecutionPlan{}, dataLossError(fmt.Errorf("read persisted Scan input source: %w", err))
		}
		return authorizedExecutionPlan{}, unavailableError("read saved execution plan lease", err)
	}
	if lease == nil || lease.TaskID != taskID || lease.ScanID != scanID {
		return authorizedExecutionPlan{}, permissionDeniedError("task scope does not match its persisted lease")
	}
	if lease.Status != string(scandomain.TaskStatusRunning) || lease.AssignedSessionID == nil || strings.TrimSpace(*lease.AssignedSessionID) == "" || lease.AssignedSessionEpoch == nil || *lease.AssignedSessionEpoch <= 0 || lease.AgentID == nil {
		return authorizedExecutionPlan{}, inactiveError("task lease is not active")
	}
	if *lease.AgentID != caller.AgentID {
		return authorizedExecutionPlan{}, permissionDeniedError("task is owned by another Agent")
	}
	if *lease.AssignedSessionID != caller.SessionID || *lease.AssignedSessionEpoch != caller.SessionEpoch {
		return authorizedExecutionPlan{}, inactiveError("task lease does not match the calling Agent process session")
	}
	current, ok := resolver.sessions.CurrentLeaseSession(caller.AgentID)
	if !ok || current.AgentID != caller.AgentID || current.SessionID != caller.SessionID || current.SessionEpoch != caller.SessionEpoch {
		return authorizedExecutionPlan{}, inactiveError("task lease does not match the current ready Agent session")
	}
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(lease.ResolvedExecutionPlan)
	if err != nil {
		return authorizedExecutionPlan{}, dataLossError(fmt.Errorf("decode persisted execution plan: %w", err))
	}
	expectedTask := resourcenames.Task(lease.ScanID, lease.TaskID)
	expectedScan := fmt.Sprintf("scans/%d", lease.ScanID)
	if plan.GetTask() != expectedTask || plan.GetWorkflowStep().GetScan() != expectedScan {
		return authorizedExecutionPlan{}, dataLossError(fmt.Errorf("persisted execution plan scope does not match its task lease"))
	}
	if executionID, err := resourcenames.ParseExecution(plan.GetExecution()); err != nil || resourcenames.Execution(executionID) != plan.GetExecution() {
		return authorizedExecutionPlan{}, dataLossError(fmt.Errorf("persisted execution plan execution resource is not canonical"))
	}
	if task != plan.GetTask() || execution != plan.GetExecution() {
		return authorizedExecutionPlan{}, permissionDeniedError("requested execution scope is not authorized by the persisted plan")
	}
	if !lease.InputSource.Valid() {
		return authorizedExecutionPlan{}, dataLossError(fmt.Errorf("persisted Scan input source is invalid: %q", lease.InputSource))
	}
	targetResource := plan.GetTarget().GetResource()
	targetID, err := resourcenames.ParseTarget(targetResource)
	if err != nil || resourcenames.Target(targetID) != targetResource {
		return authorizedExecutionPlan{}, dataLossError(fmt.Errorf("persisted execution plan Target resource is not canonical"))
	}
	return authorizedExecutionPlan{
		taskID:      taskID,
		scanID:      scanID,
		targetID:    targetID,
		inputSource: lease.InputSource,
		plan:        plan,
	}, nil
}

func (resolver *serverExecutionArtifactResolver) resolveWordlistProducer(ctx context.Context, descriptor *agentexecutionv1.WordlistDescriptor) (ExecutionArtifactProducer, error) {
	if descriptor == nil {
		return nil, fmt.Errorf("wordlist descriptor is missing")
	}
	resourceName := strings.TrimSpace(descriptor.GetResource())
	wordlistID, err := resourcenames.ParseWordlist(resourceName)
	if err != nil || resourcenames.Wordlist(wordlistID) != resourceName {
		return nil, fmt.Errorf("wordlist resource is not canonical")
	}
	wordlist, err := resolver.wordlists.GetByResourceName(ctx, resourceName)
	if err != nil {
		if errors.Is(err, catalogapp.ErrWordlistNotFound) {
			return nil, dataLossError(fmt.Errorf("resolve plan-pinned wordlist: %w", err))
		}
		return nil, unavailableError("read plan-pinned wordlist catalog", err)
	}
	if wordlist == nil || wordlist.ID != wordlistID || wordlist.FileSize < 0 || wordlist.LineCount < 0 {
		return nil, fmt.Errorf("wordlist catalog identity or metadata is invalid")
	}
	if uint64(wordlist.FileSize) != descriptor.GetSizeBytes() || uint64(wordlist.LineCount) != descriptor.GetLineCount() || "sha256:"+wordlist.FileHash != descriptor.GetSha256Digest() {
		return nil, fmt.Errorf("wordlist catalog metadata no longer matches the plan-pinned descriptor")
	}
	path, err := resolver.wordlists.GetFilePathByResourceName(ctx, resourceName)
	if err != nil {
		if errors.Is(err, catalogapp.ErrWordlistNotFound) || errors.Is(err, catalogapp.ErrFileNotFound) {
			return nil, dataLossError(fmt.Errorf("resolve plan-pinned wordlist file: %w", err))
		}
		return nil, unavailableError("resolve plan-pinned wordlist file", err)
	}
	if wordlist.FileName != descriptor.GetBasename() || filepath.Base(path) != descriptor.GetBasename() {
		return nil, fmt.Errorf("wordlist basename no longer matches the plan-pinned descriptor")
	}
	if err := validateWordlistRegularFile(path); err != nil {
		return nil, err
	}
	// A cache miss must fail before the stream header when current bytes no
	// longer match the plan. The transfer verifies again to close the race
	// between this preflight observation and the streamed read.
	if err := verifyPlanPinnedWordlistFile(ctx, path, descriptor); err != nil {
		return nil, err
	}
	return func(ctx context.Context, writer io.Writer) (uint64, error) {
		if err := validateWordlistRegularFile(path); err != nil {
			return 0, dataLossError(err)
		}
		file, err := openWordlistRegularFile(path)
		if err != nil {
			return 0, dataLossError(err)
		}
		defer file.Close()
		counter := &artifactLineCounter{}
		buffer := make([]byte, 64*1024)
		for {
			if err := ctx.Err(); err != nil {
				return counter.Count(), err
			}
			count, readErr := file.Read(buffer)
			if count > 0 {
				if err := counter.Add(buffer[:count]); err != nil {
					return counter.Count(), dataLossError(err)
				}
				if _, writeErr := writer.Write(buffer[:count]); writeErr != nil {
					return counter.Count(), writeErr
				}
			}
			if errors.Is(readErr, io.EOF) {
				return counter.Count(), nil
			}
			if readErr != nil {
				return counter.Count(), dataLossError(readErr)
			}
		}
	}, nil
}

func verifyPlanPinnedWordlistFile(ctx context.Context, path string, descriptor *agentexecutionv1.WordlistDescriptor) error {
	file, err := openWordlistRegularFile(path)
	if err != nil {
		return err
	}
	defer file.Close()

	counter := &artifactLineCounter{}
	digest := sha256.New()
	buffer := make([]byte, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		count, readErr := file.Read(buffer)
		if count > 0 {
			chunk := buffer[:count]
			if err := counter.Add(chunk); err != nil {
				return err
			}
			if _, err := digest.Write(chunk); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	actualDigest := "sha256:" + hex.EncodeToString(digest.Sum(nil))
	if counter.size != descriptor.GetSizeBytes() || counter.Count() != descriptor.GetLineCount() || actualDigest != descriptor.GetSha256Digest() {
		return fmt.Errorf("wordlist content no longer matches the plan-pinned descriptor")
	}
	return nil
}

func validateWordlistRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("wordlist content must be a regular file")
	}
	return nil
}

func openWordlistRegularFile(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	closeOnError := func(err error) (*os.File, error) {
		_ = file.Close()
		return nil, err
	}
	openedInfo, err := file.Stat()
	if err != nil {
		return closeOnError(err)
	}
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return closeOnError(err)
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !openedInfo.Mode().IsRegular() || !os.SameFile(openedInfo, pathInfo) {
		return closeOnError(fmt.Errorf("wordlist content must remain the same regular file while opened"))
	}
	return file, nil
}

type artifactLineCounter struct {
	newlines uint64
	size     uint64
	lastByte byte
}

func (counter *artifactLineCounter) Add(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if ^uint64(0)-counter.size < uint64(len(data)) {
		return fmt.Errorf("wordlist byte counter overflow")
	}
	for _, value := range data {
		if value == '\n' {
			if counter.newlines == ^uint64(0) {
				return fmt.Errorf("wordlist record counter overflow")
			}
			counter.newlines++
		}
	}
	counter.size += uint64(len(data))
	counter.lastByte = data[len(data)-1]
	return nil
}

func (counter *artifactLineCounter) Count() uint64 {
	if counter.size == 0 || counter.lastByte == '\n' {
		return counter.newlines
	}
	return counter.newlines + 1
}

func validatePlanRoleContent(role executionartifact.Role, contentType string) error {
	descriptor, ok := executionartifact.Lookup(role)
	if !ok || descriptor.ContentType != contentType {
		return fmt.Errorf("persisted binding content type does not match its typed role")
	}
	return nil
}

func permissionDeniedError(message string) error {
	return fmt.Errorf("%w: %s", ErrExecutionArtifactPermissionDenied, message)
}

func inactiveError(message string) error {
	return fmt.Errorf("%w: %s", ErrExecutionArtifactInactive, message)
}

func unavailableError(operation string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %s: %v", ErrExecutionArtifactUnavailable, operation, err)
}

func dataLossError(err error) error {
	if err == nil {
		return ErrExecutionArtifactDataLoss
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ErrExecutionArtifactDataLoss) || errors.Is(err, ErrExecutionArtifactUnavailable) || errors.Is(err, ErrExecutionArtifactPermissionDenied) || errors.Is(err, ErrExecutionArtifactInactive) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrExecutionArtifactDataLoss, err)
}

var _ ExecutionArtifactResolver = (*serverExecutionArtifactResolver)(nil)
