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
when a protected workflow matrix child needs a receipt shard. One Buildx
invocation per selected Engine publishes one OCI image index. Production always
builds `linux/amd64` and `linux/arm64`;
development defaults to the host Docker daemon platform and accepts an explicit
supported-platform override for cross-platform verification. The publisher
resolves the actual index bytes, verifies their digest and exact requested
platform set, and records a digest-qualified build receipt. Production copies
that already-built OCI graph to the second Registry; it does not rebuild it.
Every normal and raw `imagetools inspect` uses the same explicitly configured
Buildx builder as the build, including development builders configured for an
HTTP Registry transport.

`validate-build-results` re-discovers the source set and rejects missing,
extra, reordered, or mismatched records. Package generation can consume only a
receipt that matches the current validated Dockerfile/build-context/repository
inputs. It never invents a fallback ref.

The protected public workflow dynamically discovers matrix membership, bounds
multi-platform publication to three fail-fast children, and runs
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
