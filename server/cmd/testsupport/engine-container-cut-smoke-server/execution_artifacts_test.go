package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

func TestSmokeTrackedWebsiteURLsProducerPreservesRawLines(t *testing.T) {
	values := []string{
		"HTTPS://Example.COM:443/path?x=%00#fragment",
		"https://example.com/?x=%0d%0aInjected",
		"https://example.com/%zz",
	}
	producer := scanapp.WebsiteURLsProducer(func(emit func(string) error) error {
		for _, value := range values {
			if err := emit(value); err != nil {
				return err
			}
		}
		return nil
	})

	var output bytes.Buffer
	records, err := (&smokeArtifactResolver{}).trackedWebsiteURLsProducer(1, "websiteURLs", producer)(context.Background(), &output)
	if err != nil {
		t.Fatalf("tracked WebsiteURLs producer: %v", err)
	}
	if records != uint64(len(values)) || output.String() != strings.Join(values, "\n")+"\n" {
		t.Fatalf("raw WebsiteURLs product = records %d content %q", records, output.String())
	}
}

func TestSmokeTrackedWebsiteURLsProducerRejectsUnsafeStoredLine(t *testing.T) {
	producer := scanapp.WebsiteURLsProducer(func(emit func(string) error) error {
		return emit("https://example.com/\runsafe")
	})

	var output bytes.Buffer
	if _, err := (&smokeArtifactResolver{}).trackedWebsiteURLsProducer(1, "websiteURLs", producer)(context.Background(), &output); err == nil {
		t.Fatal("tracked WebsiteURLs producer accepted an unsafe stored line")
	}
	if output.Len() != 0 {
		t.Fatalf("unsafe WebsiteURLs line wrote %q", output.String())
	}
}
