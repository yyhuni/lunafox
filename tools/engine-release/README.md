# Engine release tool

`engine-release` owns the build-time handoff from Engine authoring sources to
verified Runtime Image build receipts and generated Engine Package v2 files.
It is release tooling only; Server, Agent, and Engine runtime code do not read
its discovery or receipt JSON.

## Discovery contract

`discover` enumerates direct children of `extensions/engines/` that contain an
`engine.json`. Every definition is decoded by the contracts-owned strict
`engine.v5` decoder before it can enter the release set. A discovered Engine
must also provide a regular, non-symlink `Dockerfile`, and its
`engine.lunafox.<local_name>` identity must match its directory.

The author-owned image inputs are:

- the validated `engineId`, used only to associate the build and derive its
  first-party repository name;
- `<engine>/Dockerfile`, which owns the executable, tools, dependencies, and
  OCI startup contract;
- the `extensions/engines` build context, shared because the built-in Engines
  are one Go module, plus the explicit contracts named context.

Discovery does not read Registry, tag, digest, or image location from
`engine.json`, a legacy runtime manifest, or a source-authored `package.json`.
The release publisher supplies Registry bases and namespaces through its
environment. The shared contracts function derives only
`lunafox-engine-runtime-<local-name>` from `engineId`; there is no per-Engine
release list or repository map.

Phase one is a closed first-party identity set. Each
`engine.lunafox.<local_name>` remains stable across releases and never embeds a
version, package/image digest, hash, or random suffix. Product surfaces use the
localized `displayName` together with `publisher`; duplicate display names do
not identify or merge an Engine. Repository derivation is only a LunaFox
first-party rule. Third-party namespace ownership, rename/transfer, signature
binding, external repositories, and trust onboarding require a separate
security change.

Run discovery without building or publishing images:

```bash
make engine-release-discover
```

`-engine-id` is an optional, exact canonical selection only for `discover` and
`validate-build-results`. It is checked against the complete current discovery
before output is written. Package build and package validation commands reject
it, so a one-Engine receipt cannot become a partial Package release.

## Image receipt flow

`scripts/ci/publish-engine-runtime-images.sh` consumes discovery in canonical
`engineId` order. It normally publishes the complete discovered set, and its
`ENGINE_RELEASE_ENGINE_ID` publisher input selects exactly one canonical Engine
when a protected workflow matrix child needs a receipt shard. The publisher
freezes one Buildx command per selected Engine. `ENGINE_BUILD_ATTEMPTS` defaults
to three, is fixed at three in production, and may only be reduced in
development. Retries reuse the same builder and argument vector; the second and
third attempts wait 2 and 4 seconds respectively, plus 0-999 ms of jitter.
Exhausting the retry budget fails the matrix child and leaves the existing
aggregation and downstream publication gates closed. Production always builds
`linux/amd64` and `linux/arm64`;
development defaults to the host Docker daemon platform and accepts an explicit
supported-platform override for cross-platform verification. The publisher
resolves the actual index bytes, verifies their digest and exact requested
platform set, and records a digest-qualified build receipt. Production copies
that already-built OCI graph to the second Registry; it does not rebuild it.
Every normal and raw `imagetools inspect` uses the same explicitly configured
Buildx builder as the build, including development builders configured for an
HTTP Registry transport.

Before a production build, the publisher derives one mutable cache reference
from that Engine's canonical repository:
`ghcr.io/<owner>/<engine-runtime-repository>:buildcache`. A readable cache is
passed as `cache-from`; a missing or unreadable cache produces a warning and the
complete frozen build proceeds without the import. Cache export uses BuildKit's
`ignore-error=true`, so its diagnostics remain visible without making cache
availability a release gate. Different Engine repositories never share a
writable cache reference.

The cache is acceleration state only. Its ref or digest is never copied to
Docker Hub and never enters Runtime Image identity, build receipts, Registry
evidence, SBOM, provenance, signatures, packages, or manifests. A successful
image still passes the existing digest, platform, independent cold-pull,
payload, receipt, attestation, and evidence checks. Checksum authentication also
remains enabled; the publisher does not set `GOSUMDB=off`. The official-source
[network reliability research note](../../docs/research/go-module-release-network-reliability-2026-09-15.md)
is non-normative rationale for these controls.

`validate-build-results` re-discovers the source set and rejects missing,
extra, reordered, or mismatched records. Package generation can consume only a
receipt that matches the current validated Dockerfile/build-context/repository
inputs. It never invents a fallback ref.

The protected public workflow dynamically discovers matrix membership, bounds
native platform publication to sixteen fail-fast children, and runs
`scripts/ci/aggregate-engine-runtime-image-shards.sh` before signing. The
aggregate compares its discovery artifact to the checked-out source, requires
one valid receipt and matching Registry evidence shard for every discovered
Engine, and restores the existing complete receipt artifact. Signing and all
Package stages consume only that restored complete receipt.

## Package artifact receipt

`build-packages` writes a strict
`lunafox.engine-package-build-results.v1` receipt. The receipt is not trusted
as an index by itself. `validate-package-artifacts` binds every receipt record
one-to-one to current discovery and the verified Runtime Image receipt, then
validates the corresponding expanded directory and `.lfengine.tar.gz`.

The validator requires exactly the discovered package set, canonical archive
paths, exact Runtime Image refs/digests, and the real SHA-256 of each archive.
It parses exactly four regular archive members, delegates package.json,
engine.v5, and both locale payloads to the contracts-owned strict package-v2
decoder, compares expanded and archive bytes to freshly derived publisher
inputs, and rebuilds the canonical archive serialization to reject changed
tar/gzip order, headers, modes, ownership, timestamps, compression metadata,
extra members, trailing bytes, and symlinks. The packages root may not contain
unreported entries.

`validate-package-build-results` performs the closed receipt-schema and
optional previous-release evolution checks without reading artifacts. Node CI
guards call these Go commands instead of maintaining a second package schema.

The production `publish-engine-packages-v2` workflow then publishes each
validated archive as one canonical OCI artifact: an explicit OCI image-manifest
root type, one LunaFox package layer, and the OCI empty `{}` config descriptor.
Its raw-manifest evidence compares the sole layer digest with this receipt's
archive-byte `packageDigest`; the separately signed artifact manifest digest
appears only in the digest-qualified artifact ref. A previously published
non-canonical envelope must be republished and re-signed rather than accepted
as an installation fallback.

The full `make engine-release ENGINE_VERSION=<semver>` command performs real
Registry writes and image builds. Use it only with the publisher environment
documented by `scripts/ci/publish-engine-runtime-images.sh --help`.

## Verification

```bash
cd tools/engine-release && go test ./... -count=1
bash scripts/ci/aggregate-engine-runtime-image-shards-selftest.sh
make verify-engine-release-contract-selftest
node scripts/ci/check-engine-release-contract.mjs
```

### Native production platforms

The public workflow uses `build-engine-runtime-platform.sh` on Ubuntu 24.04
AMD64 and ARM64 hosts. Each discovered Engine builds once per architecture,
publishes by digest and passes image-local inventory checks on that native host.
Caches are separated by architecture; the previous multi-platform cache is a
read-only warm-start input. No platform job writes the final public tag.

`finalize-engine-runtime-platforms.sh` requires exactly two receipts bound to
the same release identity and checked-out discovery. It validates each raw
platform index, assembles the multi-platform index, copies that graph to GHCR,
verifies both raw index digests, and produces a receipt with `buildCount: 2`.
Existing immutable tags must retain their graph. The older single BuildKit
invocation receipt (`buildCount: 1`) remains readable for existing releases
and the local publisher; neither count permits a per-Registry rebuild.
The aggregate, signing, anonymous Registry checks and package gates remain
mandatory. Missing or duplicated platforms fail before publication.

Focused regression: `node --test scripts/ci/finalize-engine-runtime-platforms.test.mjs`.
Official build pattern: https://docs.docker.com/build/ci/github-actions/multi-platform/
