package main

import (
	"context"
	"errors"
	"testing"
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

func TestGeneratedInputHandlesExposeCompleteRegistryWithoutEagerCalls(t *testing.T) {
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
	if input.Subdomains == nil || input.HostPorts == nil || input.WebsiteURLs == nil || input.EndpointURLs == nil {
		t.Fatal("generated Input did not expose the complete typed Registry")
	}
	if len(resolver.calls) != 0 {
		t.Fatalf("constructing typed handles made resolver calls: %v", resolver.calls)
	}
	if path, err := input.WebsiteURLs.Path(context.Background()); err != nil || path != resolver.paths["websiteURLs"] {
		t.Fatalf("WebsiteURLs.Path() = %q, %v", path, err)
	}
	if len(resolver.calls) != 1 || resolver.calls[0] != "websiteURLs" {
		t.Fatalf("resolver calls = %v, want one websiteURLs call", resolver.calls)
	}
}

func TestGeneratedInputHandlesPropagateErrors(t *testing.T) {
	want := errors.New("materialization failed")
	resolver := &recordingInputResolver{err: want}
	input, err := lunafoxProjectInput(resolver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := input.Subdomains.Path(context.Background()); !errors.Is(err, want) {
		t.Fatalf("Path() error = %v, want %v", err, want)
	}
}
