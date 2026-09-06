package ocidistribution

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

const testDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestBuildCloudflareAccelerationSeparatesSignatureIdentityFromDownloadTransport(t *testing.T) {
	references := []string{
		"docker.io/yyhuni/lunafox-engine-port-scan@" + testDigest,
		"ghcr.io/yyhuni/lunafox-engine-port-scan@" + testDigest,
	}

	acceleration, err := BuildCloudflareAcceleration(references)
	if err != nil {
		t.Fatalf("BuildCloudflareAcceleration() error = %v", err)
	}
	if got, want := acceleration.SignatureReference.String(), references[1]; got != want {
		t.Fatalf("SignatureReference = %q, want original GHCR %q", got, want)
	}
	wantDownloads := []string{
		"docker.lunafox.cc.cd/yyhuni/lunafox-engine-port-scan@" + testDigest,
		references[0],
		references[1],
	}
	if got := acceleration.DownloadReferenceStrings(); !reflect.DeepEqual(got, wantDownloads) {
		t.Fatalf("DownloadReferenceStrings() = %#v, want %#v", got, wantDownloads)
	}
	parsed, err := ParseCloudflareAcceleration(wantDownloads)
	if err != nil || !reflect.DeepEqual(parsed, acceleration) {
		t.Fatalf("ParseCloudflareAcceleration() = %#v, %v; want %#v", parsed, err, acceleration)
	}
}

func TestBuildCloudflareAccelerationRejectsAnyNonReleasePair(t *testing.T) {
	validDocker := "docker.io/yyhuni/lunafox-engine-port-scan@" + testDigest
	validGHCR := "ghcr.io/yyhuni/lunafox-engine-port-scan@" + testDigest
	tests := []struct {
		name string
		refs []string
		want string
	}{
		{name: "wrong count", refs: []string{validDocker}, want: "exactly Docker Hub and GHCR"},
		{name: "wrong order", refs: []string{validGHCR, validDocker}, want: "Docker Hub followed by GHCR"},
		{name: "different repository", refs: []string{validDocker, "ghcr.io/yyhuni/lunafox-engine-subdomain-discovery@" + testDigest}, want: "one repository and digest"},
		{name: "different digest", refs: []string{validDocker, "ghcr.io/yyhuni/lunafox-engine-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, want: "one repository and digest"},
		{name: "outside prefix", refs: []string{"docker.io/yyhuni/other@" + testDigest, "ghcr.io/yyhuni/other@" + testDigest}, want: "not a first-party LunaFox repository"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildCloudflareAcceleration(test.refs)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("BuildCloudflareAcceleration(%#v) error = %v, want %q", test.refs, err, test.want)
			}
		})
	}
}

func TestCanAdvanceAfterFailureAllowsOnlyCFTransportFailures(t *testing.T) {
	cloudflare := ociartifact.DigestReference{Registry: CloudflareRegistry, Repository: "yyhuni/lunafox-engine-port-scan", Digest: testDigest}
	official := ociartifact.DigestReference{Registry: "docker.io", Repository: cloudflare.Repository, Digest: testDigest}

	for _, reason := range []ociartifact.CandidateFailureReason{
		ociartifact.CandidateFailureReasonDNS,
		ociartifact.CandidateFailureReasonTCP,
		ociartifact.CandidateFailureReasonTLS,
		ociartifact.CandidateFailureReasonAttemptTimeout,
		ociartifact.CandidateFailureReasonRateLimited,
		ociartifact.CandidateFailureReasonTransientServer,
	} {
		if !CanAdvanceAfterFailure(cloudflare, ociartifact.NewCandidateFailure(reason, errors.New("transport"))) {
			t.Fatalf("CF reason %q did not allow transport fallback", reason)
		}
	}
	for _, reason := range []ociartifact.CandidateFailureReason{
		ociartifact.CandidateFailureReasonUnauthorized,
		ociartifact.CandidateFailureReasonForbidden,
		ociartifact.CandidateFailureReasonNotFound,
		ociartifact.CandidateFailureReasonManifestUnknown,
		ociartifact.CandidateFailureReasonBlobUnknown,
	} {
		if CanAdvanceAfterFailure(cloudflare, ociartifact.NewCandidateFailure(reason, errors.New("policy or integrity"))) {
			t.Fatalf("CF reason %q unexpectedly allowed fallback", reason)
		}
	}
	if !CanAdvanceAfterFailure(official, ociartifact.NewCandidateFailure(ociartifact.CandidateFailureReasonNotFound, errors.New("official unavailable"))) {
		t.Fatal("official candidate availability behavior changed")
	}
}
