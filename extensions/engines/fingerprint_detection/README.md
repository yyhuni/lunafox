# Fingerprint Detection Engine

`engine.lunafox.fingerprint_detection` is an independently runnable
first-party Engine. It applies the platform-managed FingerprintHub Web corpus
with Observer Ward and updates only the current Website technology projection.
It is not a Website Discovery sub-stage and may be the only enabled applicable
Workflow Step.

## Inputs And Candidate Composition

The manifest declares `domain`, `ip`, and `cidr` applicability, not an input
allow-set. The Engine requests the Server-produced `websiteURLs` role through
`Execution.Input.WebsiteURLs.Path(ctx)` when it builds candidates, then constructs its own Target
baseline first (`http://<target>`, then `https://<target>`), expands IPv4 CIDRs
in inclusive numeric order, and then appends `websiteURLs` in their finalized
input order, including exact baseline overlaps and repeated facts. It never
reads historical Website assets or mutates the input product.

Candidate URL spelling remains intact during input preparation. Observer Ward's
non-empty `target` result field is the result identity. `input_target` is an
optional diagnostic field and is never used as an attribution key, membership
gate, duplicate gate, or fallback. Final result submission uses the observed
`target` exactly as emitted by Observer Ward.

Candidate lines and Observer Ward JSONL rows are consumed incrementally. The
task workspace keeps only scalar counters and the candidate input file; it does
not retain an identity ledger or parsed-result set. Duplicate, unknown, empty,
or malformed `input_target` values do not reject an otherwise valid observed
`target`.
Valid zero-record input remains a baseline-only run, and write, close, or
cancellation failures are reported before Observer Ward starts. Candidate
artifacts are removed on both successful and failed terminal paths;
ordinary request failures do not create an Engine replay set.

## FingerprintHub And Runtime

The Engine accepts only the platform resource
`fingerprintLibraryFingerPrintHub`, mounted as
`/run/lunafox/resources/platform/fingerprintLibraryFingerPrintHub/fingerprinthub_web.json`.
Agent rejects a zero-record artifact before container startup. Runtime also
requires a non-empty JSON aggregate containing usable native FingerprintHub Web
template shapes before Observer Ward starts. It supplies no bundled fallback
corpus and never invokes Observer Ward's updater; a cleared library is valid to
manage but cannot execute successfully until a valid import exists.

The Runtime Image builds Observer Ward from the LunaFox fork release
`yyhuni/observer_ward_for_luna` tag `v2026.6.28-lunafox.1` and verifies its
peeled commit is `65801cf6d4b3dd4bb07ea7a1cf6e849713c42ea6`. Its source-owned
conformance checks run the real pinned binary, not only the version wrapper,
on every released architecture.

An Observer Ward upgrade must change both pins and revalidate the JSON protocol,
trusted-status classification, fixed CLI mapping, observed-target handling,
FingerprintHub loading, empty-corpus behavior, and `linux/amd64` plus
`linux/arm64` image conformance.

## Configuration

The sole default-enabled section is `observer_ward`:

| Parameter | Default | Allowed range | Effect |
| --- | --- | --- | --- |
| `timeout` | `3600` | `60..604800` seconds | Whole Engine execution timeout |
| `request-timeout` | `10` | `1..120` seconds | Observer Ward `--timeout` |
| `threads` | `25` | `1..200` | Observer Ward `--thread` |

The command surface is closed: HTTP mode, the platform probe path, JSON output,
silent output, and no-color output are fixed. Proxy, user-agent, raw record,
certificate, plugin, Nuclei, updater, and arbitrary argument controls are not
available.

## Observations And Result Boundary

A candidate is trustworthy only when an associated
`matched[].result.status` is a concrete HTTP status. Top-level `success` is not
a trust signal. Trustworthy observations collect only
`matched[].result.name`; a trustworthy zero match submits `tech:[]`. A request
without trusted status submits nothing and retains its previous technology
value. Process, protocol, submission, cancellation, cleanup, or required
resource errors fail the Task; ordinary request failures do not trigger an
automatic retry.

Results use the generated `Execution.Results.WebsiteTechnologies` port and the
closed `asset.website_technology.v1` contract, whose complete item shape is
exactly `{url,tech}`. The Engine forwards collected names without trimming,
deduplication, or sorting. Server validates the complete batch, filters it to
the authenticated Target, chooses the last valid exact-URL duplicate, and
canonicalizes only the non-URL winning technology set. It creates a missing
Website from the raw URL and derived Host, or replaces only `Website.tech` on
an existing Website. This
current-only result creates no Website Snapshot, scan summary, finding, or
fingerprint-history record.

After normal completion, the Engine emits one aggregate-only completion message
with candidate, trusted-response, request-failure, match, zero-match, and
acknowledged-submission counters. It never includes URL, response, redirect, or
fingerprint evidence in progress text.

## Runtime Image

Build from the repository root with the canonical Engine and contracts contexts:

```sh
docker buildx build \
  --platform linux/arm64 \
  --build-arg ENGINE_IMAGE_VERSION=0.0.0-dev \
  --build-context contracts=./contracts \
  --build-context engine-go=./engine-go \
  -f extensions/engines/fingerprint_detection/Dockerfile \
  extensions/engines
```

The Runtime Image starts the generated Engine API v2 Facade. The root
`Dockerfile` is production image source; `tests/container/` contains only
development and release conformance assets and is not retained in the final
image. Shared runtime-image and release rules are in
[`../container/README.md`](../container/README.md).
