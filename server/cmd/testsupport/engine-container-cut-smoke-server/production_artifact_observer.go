package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	"github.com/yyhuni/lunafox/server/internal/grpc/agentdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type productionArtifactObserver struct {
	inner  agentdata.ExecutionArtifactResolver
	bridge *smokeTaskBridge
}

// newProductionArtifactObserver constructs the production resolver itself so
// the smoke can observe or fault its output without substituting authorization.
func newProductionArtifactObserver(
	dependencies agentdata.ServerExecutionArtifactResolverDependencies,
	bridge *smokeTaskBridge,
) (*productionArtifactObserver, error) {
	if bridge == nil {
		return nil, errors.New("smoke artifact observer bridge is required")
	}
	production, err := agentdata.NewServerExecutionArtifactResolver(dependencies)
	if err != nil {
		return nil, fmt.Errorf("initialize production execution artifact resolver: %w", err)
	}
	return &productionArtifactObserver{inner: production, bridge: bridge}, nil
}

func (observer *productionArtifactObserver) AuthorizeExecutionInput(
	ctx context.Context,
	lease agentdata.AgentExecutionLease,
	request *agentdatav1.StreamExecutionInputRequest,
) (agentdata.AuthorizedExecutionArtifact, error) {
	source, err := observer.requireInner().AuthorizeExecutionInput(ctx, lease, request)
	if err != nil {
		return agentdata.AuthorizedExecutionArtifact{}, err
	}
	taskID, err := observedArtifactTaskID(request.GetTask())
	if err != nil {
		return agentdata.AuthorizedExecutionArtifact{}, err
	}
	kind := request.GetRole()
	if kind == "" {
		return agentdata.AuthorizedExecutionArtifact{}, errors.New("production resolver authorized an unknown execution input")
	}
	// Observation counters retain their historical host/web buckets while the
	// production resolver is addressed by the new domain-fact roles.
	switch kind {
	case executionartifact.RoleSubdomains:
		kind = "host"
	case executionartifact.RoleHostPorts, executionartifact.RoleWebsiteURLs:
		kind = "web"
	}
	return observer.observe(source, taskID, kind), nil
}

func (observer *productionArtifactObserver) AuthorizeConfigResource(
	ctx context.Context,
	lease agentdata.AgentExecutionLease,
	request *agentdatav1.StreamConfigResourceRequest,
) (agentdata.AuthorizedExecutionArtifact, error) {
	source, err := observer.requireInner().AuthorizeConfigResource(ctx, lease, request)
	if err != nil {
		return agentdata.AuthorizedExecutionArtifact{}, err
	}
	taskID, err := observedArtifactTaskID(request.GetTask())
	if err != nil {
		return agentdata.AuthorizedExecutionArtifact{}, err
	}
	return observer.observe(source, taskID, "config"), nil
}

func (observer *productionArtifactObserver) AuthorizePlatformResource(
	ctx context.Context,
	lease agentdata.AgentExecutionLease,
	request *agentdatav1.StreamPlatformResourceContentRequest,
) (agentdata.AuthorizedExecutionArtifact, error) {
	source, err := observer.requireInner().AuthorizePlatformResource(ctx, lease, request)
	if err != nil {
		return agentdata.AuthorizedExecutionArtifact{}, err
	}
	taskID, err := observedArtifactTaskID(request.GetTask())
	if err != nil {
		return agentdata.AuthorizedExecutionArtifact{}, err
	}
	return observer.observe(source, taskID, "platform"), nil
}

func (observer *productionArtifactObserver) AuthorizeRuntimeArtifactExchange(
	ctx context.Context,
	lease agentdata.AgentExecutionLease,
	begin *agentdatav1.RuntimeArtifactExchangeBegin,
) (*agentdata.AuthorizedRuntimeArtifactExchange, error) {
	resolver, ok := observer.requireInner().(agentdata.RuntimeArtifactExchangeResolver)
	if !ok {
		return nil, errors.New("production runtime artifact exchange resolver is unavailable")
	}
	exchange, err := resolver.AuthorizeRuntimeArtifactExchange(ctx, lease, begin)
	if err != nil {
		return nil, err
	}
	if exchange == nil {
		return nil, errors.New("production runtime artifact exchange returned an empty boundary")
	}
	taskID, err := observedArtifactTaskID(begin.GetTask())
	if err != nil {
		return nil, err
	}
	// The bidirectional exchange has no ExecutionArtifactProducer callback.
	// Record its completed manifest at authorization time so the smoke keeps
	// the same bounded evidence shape as the older producer observations.
	if _, ok := observer.bridge.beginObservedArtifactProduce(taskID, "nuclei_template"); !ok {
		return nil, errors.New("production Nuclei template task is not registered by the smoke")
	}
	var bytes uint64
	for _, entry := range exchange.Entries {
		if ^uint64(0)-bytes < entry.SizeBytes {
			return nil, errors.New("production Nuclei template observation size overflow")
		}
		bytes += entry.SizeBytes
	}
	observer.bridge.observeArtifact(taskID, "nuclei_template", uint64(len(exchange.Entries)), bytes)
	return exchange, nil
}

func (observer *productionArtifactObserver) requireInner() agentdata.ExecutionArtifactResolver {
	if observer == nil || observer.inner == nil {
		return unavailableProductionArtifactResolver{}
	}
	return observer.inner
}

func observedArtifactTaskID(task string) (int, error) {
	_, taskID, err := resourcenames.ParseTask(task)
	if err != nil {
		return 0, fmt.Errorf("observe production artifact task %q: %w", task, err)
	}
	return taskID, nil
}

type smokeArtifactFault uint8

const (
	smokeArtifactFaultNone smokeArtifactFault = iota
	smokeArtifactFaultUnavailable
	smokeArtifactFaultIntegrity
)

func (observer *productionArtifactObserver) observe(
	source agentdata.AuthorizedExecutionArtifact,
	taskID int,
	kind string,
) agentdata.AuthorizedExecutionArtifact {
	producer := source.Produce
	if producer == nil {
		return source
	}
	source.Produce = func(ctx context.Context, writer io.Writer) (uint64, error) {
		fault, ok := observer.bridge.beginObservedArtifactProduce(taskID, kind)
		if !ok {
			return 0, errors.New("production artifact task is not registered by the smoke")
		}
		if fault == smokeArtifactFaultUnavailable {
			return 0, status.Error(codes.Unavailable, "smoke Subdomains transport interruption")
		}

		counting := &productionArtifactCountingWriter{writer: writer}
		recordCount, err := producer(ctx, counting)
		if err != nil {
			return recordCount, err
		}
		if fault == smokeArtifactFaultIntegrity {
			if _, err := counting.Write([]byte("!")); err != nil {
				return recordCount, err
			}
		}
		observer.bridge.observeArtifact(taskID, kind, recordCount, counting.bytes)
		return recordCount, nil
	}
	return source
}

func (bridge *smokeTaskBridge) beginObservedArtifactProduce(taskID int, kind string) (smokeArtifactFault, bool) {
	if bridge == nil || taskID <= 0 {
		return smokeArtifactFaultNone, false
	}
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	record := bridge.byTaskID[taskID]
	if record == nil {
		return smokeArtifactFaultNone, false
	}
	observation := observedArtifactRole(record, kind)
	if observation == nil {
		return smokeArtifactFaultNone, false
	}
	observation.attempts++
	if kind == "host" && record.scenario == smokeScenarioPortSuccess && observation.unavailableFailures == 0 {
		observation.unavailableFailures++
		return smokeArtifactFaultUnavailable, true
	}
	if kind == "config" && record.scenario == smokeScenarioArtifactIntegrity && observation.integrityCorruptions == 0 {
		observation.integrityCorruptions++
		return smokeArtifactFaultIntegrity, true
	}
	return smokeArtifactFaultNone, true
}

func observedArtifactRole(record *smokePlanRecord, kind string) *smokeArtifactObservation {
	if record == nil {
		return nil
	}
	switch kind {
	case "host":
		return &record.hostArtifact
	case "web":
		return &record.webArtifact
	case "config":
		return &record.configArtifact
	case "platform":
		return &record.platformArtifact
	case "nuclei_template":
		return &record.nucleiTemplateArtifact
	default:
		return nil
	}
}

type productionArtifactCountingWriter struct {
	writer io.Writer
	bytes  uint64
}

func (writer *productionArtifactCountingWriter) Write(payload []byte) (int, error) {
	if writer == nil || writer.writer == nil {
		return 0, errors.New("production artifact observer writer is required")
	}
	written, err := writer.writer.Write(payload)
	if written < 0 || written > len(payload) {
		return 0, errors.New("production artifact writer returned an invalid byte count")
	}
	if ^uint64(0)-writer.bytes < uint64(written) {
		return written, errors.New("production artifact observation size overflow")
	}
	writer.bytes += uint64(written)
	if written != len(payload) && err == nil {
		err = io.ErrShortWrite
	}
	return written, err
}

// unavailableProductionArtifactResolver keeps nil receiver failures explicit;
// it never authorizes or produces an artifact.
type unavailableProductionArtifactResolver struct{}

func (unavailableProductionArtifactResolver) AuthorizeExecutionInput(context.Context, agentdata.AgentExecutionLease, *agentdatav1.StreamExecutionInputRequest) (agentdata.AuthorizedExecutionArtifact, error) {
	return agentdata.AuthorizedExecutionArtifact{}, errors.New("production execution artifact resolver is unavailable")
}

func (unavailableProductionArtifactResolver) AuthorizeConfigResource(context.Context, agentdata.AgentExecutionLease, *agentdatav1.StreamConfigResourceRequest) (agentdata.AuthorizedExecutionArtifact, error) {
	return agentdata.AuthorizedExecutionArtifact{}, errors.New("production execution artifact resolver is unavailable")
}

func (unavailableProductionArtifactResolver) AuthorizePlatformResource(context.Context, agentdata.AgentExecutionLease, *agentdatav1.StreamPlatformResourceContentRequest) (agentdata.AuthorizedExecutionArtifact, error) {
	return agentdata.AuthorizedExecutionArtifact{}, errors.New("production execution artifact resolver is unavailable")
}

var _ agentdata.ExecutionArtifactResolver = (*productionArtifactObserver)(nil)
var _ agentdata.RuntimeArtifactExchangeResolver = (*productionArtifactObserver)(nil)
