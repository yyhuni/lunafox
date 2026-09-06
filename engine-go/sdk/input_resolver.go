package sdk

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/grpc/codes"
)

// inputResolver is deliberately scoped to one Run invocation. Its successful
// path cache and flights therefore cannot cross an execution or lease.
type inputResolver struct {
	reporter *reporter
	service  protocol.EngineExecutionInputServiceClient

	mu      sync.Mutex
	paths   map[string]string
	flights map[string]*inputFlight
}

type inputFlight struct {
	done    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	path    string
	err     error
	waiters int
	// abandoned fences a zero-waiter flight while its RPC cleanup is still
	// running. A later explicit call waits for that cleanup, then starts a
	// fresh request instead of inheriting the canceled result.
	abandoned bool
	closed    bool
}

func newInputResolver(reporter *reporter, service protocol.EngineExecutionInputServiceClient) *inputResolver {
	return &inputResolver{
		reporter: reporter,
		service:  service,
		paths:    make(map[string]string),
		flights:  make(map[string]*inputFlight),
	}
}

// path materializes one Registry role through the private input service. The
// resolver coalesces overlapping calls for this execution and caches only a
// successfully returned canonical path. A canceled waiter never cancels
// shared work while another waiter remains.
func (resolver *inputResolver) path(ctx context.Context, role string) (string, error) {
	if resolver == nil || resolver.reporter == nil || resolver.service == nil {
		return "", errors.New("engine input resolver is unavailable")
	}
	if ctx == nil {
		return "", errors.New("engine input context is required")
	}
	if role == "" || role != strings.TrimSpace(role) || strings.ContainsAny(role, "\x00\r\n") {
		return "", &Error{Code: protocolInvalidArgumentCode(), Reason: protocol.ReasonInvalidReportingRequest}
	}
	if err := ctx.Err(); err != nil {
		return "", mapContextError(err)
	}

	for {
		resolver.mu.Lock()
		if path, ok := resolver.paths[role]; ok {
			resolver.mu.Unlock()
			return path, nil
		}
		flight := resolver.flights[role]
		if flight == nil {
			// Preserve trace/request values for the shared RPC, but deliberately
			// strip the first waiter's cancellation and deadline: those govern
			// only that waiter and cannot cancel another valid waiter.
			flightCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
			flight = &inputFlight{done: make(chan struct{}), ctx: flightCtx, cancel: cancel}
			resolver.flights[role] = flight
			go resolver.runFlight(role, flight)
		}
		if flight.abandoned {
			done := flight.done
			resolver.mu.Unlock()
			select {
			case <-done:
				// The abandoned flight has removed itself. Re-check the
				// success cache before creating a fresh request.
				continue
			case <-ctx.Done():
				return "", mapContextError(ctx.Err())
			}
		}
		flight.waiters++
		done := flight.done
		resolver.mu.Unlock()

		select {
		case <-done:
			resolver.mu.Lock()
			path, err := flight.path, flight.err
			resolver.mu.Unlock()
			return path, err
		case <-ctx.Done():
			resolver.leaveFlight(role, flight)
			return "", mapContextError(ctx.Err())
		}
	}
}

// These methods are the sole bridge supplied to generated adapters. The
// role string remains inside SDK transport plumbing so Engine business code
// cannot turn a typed handle into arbitrary Registry or RPC access.
func (resolver *inputResolver) SubdomainsPath(ctx context.Context) (string, error) {
	return resolver.path(ctx, protocol.ExecutionInputRoleSubdomains)
}

func (resolver *inputResolver) HostPortsPath(ctx context.Context) (string, error) {
	return resolver.path(ctx, protocol.ExecutionInputRoleHostPorts)
}

func (resolver *inputResolver) WebsiteURLsPath(ctx context.Context) (string, error) {
	return resolver.path(ctx, protocol.ExecutionInputRoleWebsiteURLs)
}

func (resolver *inputResolver) EndpointURLsPath(ctx context.Context) (string, error) {
	return resolver.path(ctx, protocol.ExecutionInputRoleEndpointURLs)
}

func (resolver *inputResolver) leaveFlight(role string, flight *inputFlight) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	if flight.closed || flight.waiters == 0 {
		return
	}
	flight.waiters--
	if flight.waiters == 0 {
		flight.abandoned = true
		flight.cancel()
	}
}

func (resolver *inputResolver) runFlight(role string, flight *inputFlight) {
	path, err := resolver.materialize(flight.ctx, role)
	resolver.mu.Lock()
	flight.path = path
	flight.err = err
	flight.closed = true
	if err == nil {
		resolver.paths[role] = path
	}
	if resolver.flights[role] == flight {
		delete(resolver.flights, role)
	}
	close(flight.done)
	resolver.mu.Unlock()
	flight.cancel()
}

func (resolver *inputResolver) materialize(ctx context.Context, role string) (string, error) {
	callContext, cleanup, err := resolver.reporter.callContext(ctx)
	if err != nil {
		return "", err
	}
	defer cleanup()
	response, err := resolver.service.MaterializeExecutionInput(callContext, &protocol.MaterializeExecutionInputRequest{Role: role})
	if err != nil {
		return "", resolver.reporter.mapCallError(err)
	}
	if response == nil || !validPublishedInputPath(response.GetPath()) {
		return "", &Error{Code: protocolInternalCode(), Reason: protocol.ReasonReportingInternalInvariant}
	}
	return response.GetPath(), nil
}

func validPublishedInputPath(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || !filepath.IsAbs(value) || filepath.Clean(value) != value {
		return false
	}
	return pathWithin(protocol.InputsDirectoryPath, value)
}

// Keep status-code construction in one place so this private file does not
// expose gRPC details through the generated Engine contract.
func protocolInvalidArgumentCode() codes.Code { return codes.InvalidArgument }
func protocolInternalCode() codes.Code        { return codes.Internal }
