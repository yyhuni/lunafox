package urlcollectionruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	contractresults "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

func TestSubmitStagedEndpointsUsesDiskBackedLastRecordWins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "endpoints.stage")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, endpoint := range []contractresults.Endpoint{
		{URL: "https://example.com/a", Host: "example.com", Title: "first"},
		{URL: "https://example.com/b", Host: "example.com", Title: "other"},
		{URL: "https://example.com/a", Host: "example.com", Title: "last"},
	} {
		if err := appendStagedEndpoint(file, endpoint, &count); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	submitter := &captureEndpointSubmitter{}
	if err := submitStagedEndpoints(context.Background(), submitter, path); err != nil {
		t.Fatal(err)
	}
	if len(submitter.items) != 2 {
		t.Fatalf("submission captured=%#v", submitter.items)
	}
	if submitter.items[0].URL != "https://example.com/a" || submitter.items[0].Title != "last" {
		t.Fatalf("last record winner = %#v", submitter.items)
	}
}

func TestSubmitStagedEndpointsDoesNotSubmitEmptyStage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "endpoints.stage")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	submitter := &captureEndpointSubmitter{}
	if err := submitStagedEndpoints(context.Background(), submitter, path); err != nil {
		t.Fatal(err)
	}
	if submitter.calls != 0 || len(submitter.items) != 0 {
		t.Fatalf("empty stage must not invoke a submission: calls=%d captured=%#v", submitter.calls, submitter.items)
	}
}

func TestSubmitStagedEndpointsPreservesPartialAcknowledgementOnFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "endpoints.stage")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, endpoint := range []contractresults.Endpoint{
		{URL: "https://example.com/a", Host: "example.com"},
		{URL: "https://example.com/b", Host: "example.com"},
	} {
		if err := appendStagedEndpoint(file, endpoint, &count); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	expected := errors.New("second batch acknowledgement failed")
	submitter := &partialEndpointSubmitter{err: expected}
	err = submitStagedEndpoints(context.Background(), submitter, path)
	if !errors.Is(err, expected) {
		t.Fatalf("partial submission err:%v", err)
	}
}

type captureEndpointSubmitter struct {
	calls int
	items []contractresults.Endpoint
}

type partialEndpointSubmitter struct {
	err error
}

func (submitter *partialEndpointSubmitter) Submit(context.Context, <-chan contractresults.Endpoint) error {
	return submitter.err
}

func (submitter *captureEndpointSubmitter) Submit(_ context.Context, items <-chan contractresults.Endpoint) error {
	submitter.calls++
	for item := range items {
		submitter.items = append(submitter.items, item)
	}
	return nil
}
