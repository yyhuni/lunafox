package main

import (
	"context"
	"errors"
	"reflect"
	"testing"

	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
)

type recordingInputResolver struct {
	paths map[string]string
	calls []string
	err   error
}

func (resolver *recordingInputResolver) path(ctx context.Context, role string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	resolver.calls = append(resolver.calls, role)
	if resolver.err != nil {
		return "", resolver.err
	}
	return resolver.paths[role], nil
}

func (resolver *recordingInputResolver) SubdomainsPath(ctx context.Context) (string, error) {
	return resolver.path(ctx, "subdomains")
}

func (resolver *recordingInputResolver) HostPortsPath(ctx context.Context) (string, error) {
	return resolver.path(ctx, "hostPorts")
}

func (resolver *recordingInputResolver) WebsiteURLsPath(ctx context.Context) (string, error) {
	return resolver.path(ctx, "websiteURLs")
}

func (resolver *recordingInputResolver) EndpointURLsPath(ctx context.Context) (string, error) {
	return resolver.path(ctx, "endpointURLs")
}

func TestGeneratedInputHandlesMaterializeOnlyWhenPathIsCalled(t *testing.T) {
	resolver := &recordingInputResolver{paths: map[string]string{
		"subdomains":   "/run/lunafox/inputs/subdomains.txt",
		"hostPorts":    "/run/lunafox/inputs/host-ports.jsonl",
		"websiteURLs":  "/run/lunafox/inputs/website-urls.txt",
		"endpointURLs": "/run/lunafox/inputs/endpoint-urls.txt",
	}}
	input, err := lunafoxProjectInput(resolver)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolver.calls) != 0 {
		t.Fatalf("constructing typed handles made resolver calls: %v", resolver.calls)
	}
	path, err := input.HostPorts.Path(context.Background())
	if err != nil || path != resolver.paths["hostPorts"] {
		t.Fatalf("HostPorts.Path() = %q, %v", path, err)
	}
	if len(resolver.calls) != 1 || resolver.calls[0] != "hostPorts" {
		t.Fatalf("resolver calls = %v, want one hostPorts call", resolver.calls)
	}
}

func TestGeneratedInputHandlesPropagateContextAndErrors(t *testing.T) {
	want := errors.New("materialization failed")
	resolver := &recordingInputResolver{err: want}
	input, err := lunafoxProjectInput(resolver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := input.HostPorts.Path(context.Background()); !errors.Is(err, want) {
		t.Fatalf("Path() error = %v, want %v", err, want)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := input.WebsiteURLs.Path(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Path() error = %v", err)
	}
}

func TestGeneratedConfigPreservesRequiredHeaderArray(t *testing.T) {
	enabled := true
	section := &engineexecutionpb.ConfigSection{SectionId: "httpx", Enabled: &enabled}
	for _, key := range []string{"timeout", "threads", "rate-limit", "request-timeout", "retries"} {
		section.Params = append(section.Params, &engineexecutionpb.ConfigValue{ParamKey: key, Value: &engineexecutionpb.ConfigValue_IntegerValue{IntegerValue: 60}})
	}
	if _, err := lunafoxProjectHTTPXConfig(section); err == nil {
		t.Fatal("complete config without headers was accepted")
	}
	headers := []string{"Cookie: a=1; b=two", `X-Fields: a,b:c; "quoted"`}
	section.Params = append(section.Params, &engineexecutionpb.ConfigValue{ParamKey: "headers", Value: &engineexecutionpb.ConfigValue_StringArrayValue{StringArrayValue: &engineexecutionpb.StringArrayValue{Values: headers}}})
	projected, err := lunafoxProjectHTTPXConfig(section)
	if err != nil || !reflect.DeepEqual(projected.Headers, headers) {
		t.Fatalf("projected headers = %#v, %v", projected.Headers, err)
	}
	projected.Headers[0] = "changed"
	if headers[0] != "Cookie: a=1; b=two" {
		t.Fatal("generated config aliased the transport values")
	}
}
