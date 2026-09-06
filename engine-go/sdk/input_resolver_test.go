package sdk

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// recordingInputClient is intentionally a transport-shaped fake. These tests
// exercise the resolver's lifecycle without introducing a second retry owner.
type recordingInputClient struct {
	mu            sync.Mutex
	calls         int
	roles         []string
	started       chan struct{}
	release       chan struct{}
	canceled      chan struct{}
	startOne      sync.Once
	cancelOne     sync.Once
	errors        []error
	observedValue any
}

func (client *recordingInputClient) MaterializeExecutionInput(ctx context.Context, request *protocol.MaterializeExecutionInputRequest, _ ...grpc.CallOption) (*protocol.MaterializeExecutionInputResponse, error) {
	client.mu.Lock()
	index := client.calls
	client.calls++
	client.roles = append(client.roles, request.GetRole())
	client.observedValue = ctx.Value(inputContextKey{})
	release := client.release
	canceled := client.canceled
	if client.started != nil {
		client.startOne.Do(func() { close(client.started) })
	}
	var injected error
	if index < len(client.errors) {
		injected = client.errors[index]
	}
	client.mu.Unlock()
	if release != nil {
		select {
		case <-release:
		case <-ctx.Done():
			if canceled != nil {
				client.cancelOne.Do(func() { close(canceled) })
			}
			return nil, ctx.Err()
		}
	}
	if injected != nil {
		return nil, injected
	}
	return &protocol.MaterializeExecutionInputResponse{Path: inputPathForRole(request.GetRole())}, nil
}

type inputContextKey struct{}

func (client *recordingInputClient) callCount() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.calls
}

func (client *recordingInputClient) requestedRoles() []string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return append([]string(nil), client.roles...)
}

func inputPathForRole(role string) string {
	switch role {
	case protocol.ExecutionInputRoleSubdomains:
		return protocol.SubdomainsInputPath
	case protocol.ExecutionInputRoleHostPorts:
		return protocol.HostPortsInputPath
	case protocol.ExecutionInputRoleWebsiteURLs:
		return protocol.WebsiteURLsInputPath
	case protocol.ExecutionInputRoleEndpointURLs:
		return protocol.EndpointURLsInputPath
	default:
		return ""
	}
}

func newInputResolverForTest(t *testing.T, client *recordingInputClient) *inputResolver {
	t.Helper()
	reporter := newReporter(validContext(t.TempDir()), &recordingReportingClient{}, testCredential())
	t.Cleanup(reporter.close)
	return newInputResolver(reporter, client)
}

func TestInputResolverSingleFlightsAndCachesOnlySuccess(t *testing.T) {
	client := &recordingInputClient{started: make(chan struct{}), release: make(chan struct{})}
	resolver := newInputResolverForTest(t, client)

	const callers = 12
	results := make(chan struct {
		path string
		err  error
	}, callers)
	for index := 0; index < callers; index++ {
		go func() {
			path, err := resolver.SubdomainsPath(context.Background())
			results <- struct {
				path string
				err  error
			}{path: path, err: err}
		}()
	}
	select {
	case <-client.started:
	case <-time.After(time.Second):
		t.Fatal("single-flight request did not start")
	}
	close(client.release)
	for index := 0; index < callers; index++ {
		result := <-results
		if result.err != nil || result.path != protocol.SubdomainsInputPath {
			t.Fatalf("Path() result %d = %q, %v", index, result.path, result.err)
		}
	}
	if got := client.callCount(); got != 1 {
		t.Fatalf("materialize calls = %d, want one shared call", got)
	}
	path, err := resolver.SubdomainsPath(context.Background())
	if err != nil || path != protocol.SubdomainsInputPath {
		t.Fatalf("cached Path() = %q, %v", path, err)
	}
	if got := client.callCount(); got != 1 {
		t.Fatalf("cached Path() opened %d calls, want one", got)
	}
}

func TestInputResolverWaiterCancellationDoesNotCancelRemainingWaiter(t *testing.T) {
	client := &recordingInputClient{started: make(chan struct{}), release: make(chan struct{})}
	resolver := newInputResolverForTest(t, client)
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() {
		_, err := resolver.SubdomainsPath(firstCtx)
		firstResult <- err
	}()
	select {
	case <-client.started:
	case <-time.After(time.Second):
		t.Fatal("shared request did not start")
	}
	secondResult := make(chan error, 1)
	go func() {
		_, err := resolver.SubdomainsPath(context.Background())
		secondResult <- err
	}()
	deadline := time.Now().Add(time.Second)
	joined := false
	for time.Now().Before(deadline) {
		resolver.mu.Lock()
		flight := resolver.flights["subdomains"]
		waiters := 0
		if flight != nil {
			waiters = flight.waiters
		}
		resolver.mu.Unlock()
		if waiters == 2 {
			joined = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !joined {
		t.Fatal("second waiter did not join the shared flight")
	}
	cancelFirst()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled waiter error = %v", err)
	}
	close(client.release)
	if err := <-secondResult; err != nil {
		t.Fatalf("remaining waiter error = %v", err)
	}
	if got := client.callCount(); got != 1 {
		t.Fatalf("remaining waiter caused %d requests", got)
	}
}

func TestInputResolverZeroWaiterCancellationAllowsFreshFlight(t *testing.T) {
	client := &recordingInputClient{started: make(chan struct{}), release: make(chan struct{})}
	resolver := newInputResolverForTest(t, client)
	ctx, cancel := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() {
		_, err := resolver.SubdomainsPath(ctx)
		firstResult <- err
	}()
	select {
	case <-client.started:
	case <-time.After(time.Second):
		t.Fatal("cancellable request did not start")
	}
	cancel()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("first canceled waiter error = %v", err)
	}
	// The canceled transport observes its own context and exits. A later
	// explicit call must then create a fresh flight and may succeed.
	close(client.release)
	path, err := resolver.SubdomainsPath(context.Background())
	if err != nil || path != protocol.SubdomainsInputPath {
		t.Fatalf("fresh Path() = %q, %v", path, err)
	}
	if got := client.callCount(); got != 2 {
		t.Fatalf("fresh call count = %d, want canceled plus fresh attempt", got)
	}
}

func TestInputResolverFailureIsNotSticky(t *testing.T) {
	client := &recordingInputClient{errors: []error{status.Error(codes.InvalidArgument, "bad role")}}
	resolver := newInputResolverForTest(t, client)
	if _, err := resolver.SubdomainsPath(context.Background()); err == nil {
		t.Fatal("first failed Path() returned nil")
	}
	path, err := resolver.SubdomainsPath(context.Background())
	if err != nil || path != protocol.SubdomainsInputPath {
		t.Fatalf("retry Path() = %q, %v", path, err)
	}
	if got := client.callCount(); got != 2 {
		t.Fatalf("failure was cached after %d calls, want two", got)
	}
}

func TestInputResolverPropagatesContextValuesWithoutSharingWaiterCancellation(t *testing.T) {
	client := &recordingInputClient{}
	resolver := newInputResolverForTest(t, client)
	ctx := context.WithValue(context.Background(), inputContextKey{}, "trace-1")
	path, err := resolver.SubdomainsPath(ctx)
	if err != nil || path != protocol.SubdomainsInputPath {
		t.Fatalf("Path() = %q, %v", path, err)
	}
	client.mu.Lock()
	observed := client.observedValue
	client.mu.Unlock()
	if observed != "trace-1" {
		t.Fatalf("input RPC context value = %#v, want trace-1", observed)
	}

	blocked := &recordingInputClient{started: make(chan struct{}), release: make(chan struct{}), canceled: make(chan struct{})}
	resolver = newInputResolverForTest(t, blocked)
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() {
		_, err := resolver.SubdomainsPath(firstCtx)
		firstResult <- err
	}()
	select {
	case <-blocked.started:
	case <-time.After(time.Second):
		t.Fatal("context propagation cancellation flight did not start")
	}
	secondResult := make(chan error, 1)
	go func() {
		_, err := resolver.SubdomainsPath(context.Background())
		secondResult <- err
	}()
	deadline := time.Now().Add(time.Second)
	joined := false
	for time.Now().Before(deadline) {
		resolver.mu.Lock()
		flight := resolver.flights["subdomains"]
		waiters := 0
		if flight != nil {
			waiters = flight.waiters
		}
		resolver.mu.Unlock()
		if waiters == 2 {
			joined = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !joined {
		t.Fatal("remaining waiter did not join shared RPC")
	}
	cancelFirst()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled first waiter error = %v", err)
	}
	select {
	case <-blocked.canceled:
		t.Fatal("one departing waiter canceled the shared RPC")
	default:
	}
	close(blocked.release)
	if err := <-secondResult; err != nil {
		t.Fatalf("remaining waiter error = %v", err)
	}
}

func TestInputResolverMapsTransportErrorsToNeutralSDKErrors(t *testing.T) {
	client := &recordingInputClient{errors: []error{status.Error(codes.Internal, "server secret and path")}}
	resolver := newInputResolverForTest(t, client)
	_, err := resolver.SubdomainsPath(context.Background())
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != codes.Internal || typed.Reason != protocol.ReasonReportingInternalInvariant {
		t.Fatalf("transport error = %#v, want neutral internal SDK error", err)
	}
	if strings.Contains(err.Error(), "server secret") || strings.Contains(err.Error(), "path") {
		t.Fatalf("transport error leaked server details: %v", err)
	}
}

func TestInputResolverUsesOnlyRoleSpecificLazyOperations(t *testing.T) {
	client := &recordingInputClient{}
	resolver := newInputResolverForTest(t, client)

	if got := client.callCount(); got != 0 {
		t.Fatalf("constructing the resolver made %d input requests", got)
	}
	for _, test := range []struct {
		name string
		path func(context.Context) (string, error)
		want string
	}{
		{name: "subdomains", path: resolver.SubdomainsPath, want: protocol.SubdomainsInputPath},
		{name: "hostPorts", path: resolver.HostPortsPath, want: protocol.HostPortsInputPath},
		{name: "websiteURLs", path: resolver.WebsiteURLsPath, want: protocol.WebsiteURLsInputPath},
		{name: "endpointURLs", path: resolver.EndpointURLsPath, want: protocol.EndpointURLsInputPath},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.path(context.Background())
			if err != nil || got != test.want {
				t.Fatalf("typed Path() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
	if got, want := client.requestedRoles(), []string{
		protocol.ExecutionInputRoleSubdomains,
		protocol.ExecutionInputRoleHostPorts,
		protocol.ExecutionInputRoleWebsiteURLs,
		protocol.ExecutionInputRoleEndpointURLs,
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("input roles = %q, want %q", got, want)
	}
}

func TestAdapterInputBridgeHasNoGenericRoleOrReadablePathAPI(t *testing.T) {
	bridge := reflect.TypeFor[Adapter]().In(3)
	if bridge.Kind() != reflect.Interface {
		t.Fatalf("adapter input bridge kind = %s, want interface", bridge.Kind())
	}
	expected := []string{"EndpointURLsPath", "HostPortsPath", "SubdomainsPath", "WebsiteURLsPath"}
	if bridge.NumMethod() != len(expected) {
		t.Fatalf("adapter input bridge methods = %d, want %d", bridge.NumMethod(), len(expected))
	}
	contextType := reflect.TypeFor[context.Context]()
	errorType := reflect.TypeFor[error]()
	for _, name := range expected {
		method, ok := bridge.MethodByName(name)
		if !ok {
			t.Fatalf("adapter input bridge is missing %s", name)
		}
		if method.Type.NumIn() != 1 || method.Type.In(0) != contextType ||
			method.Type.NumOut() != 2 || method.Type.Out(0).Kind() != reflect.String || method.Type.Out(1) != errorType {
			t.Fatalf("adapter bridge method %s has unexpected signature %s", name, method.Type)
		}
	}
	for _, forbidden := range []string{"Path", "Ensure", "Materialize", "Resolve"} {
		if _, ok := bridge.MethodByName(forbidden); ok {
			t.Fatalf("adapter input bridge exposes forbidden generic method %s", forbidden)
		}
	}
}
