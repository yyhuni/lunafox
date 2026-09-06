# Contributing to the public deployment projection

This repository is a generated, read-only deployment and release projection of
[`yyhuni/lunafox-private`](https://github.com/yyhuni/lunafox-private). It is not
a second development authority. Pull requests are reviewed against a generated
snapshot; they do not become private Agent source or a private release input
merely by being merged here.

The private repository remains the only development authority for the closed
Agent source and binary handoff; the public repository owns packaging and the
final product release. Please report a projection defect with the affected
release tag, public path, and reproducible verification output. The generated Frontend,
Server, Contracts, `engine-go`, Proto, first-party Extensions, Nginx, and
Bootstrap source, tests, and build metadata are available for inspection and
reproduction. The support page's current Alipay, WeChat, and contact QR JPGs
are intentionally public; do not treat that as approval for new personal or
credential-bearing assets. Do not add Agent source, credentials, private
signing/release material, development Docker contexts, certificates, or local
build evidence.

Public tests must run against the exported closure. Generated test reports,
screenshots, coverage, browser caches, test-results, test plans, logs,
dependency directories, framework/build output, and TLS material remain
private. The public pull-request workflow is deliberately secretless and
validates source plus `push: false` builds. After a generated export PR is
approved and merged into protected `main`, the public workflow publishes
immutable GHCR digests with SBOM/provenance evidence for Server, Frontend,
Nginx, and Bootstrap. The private release workflow builds and signs the Agent
binary bundle and exports its exact seven-file directory under
`agent/bin/<release-tag>`. The public protected-main workflow verifies that
merged Git tree, packages the Agent image, and owns final digest promotion,
manifest, channel, and GitHub Release publication.
The private release run ends when it requests protected auto-merge for the
generated export PR. Its success means `public handoff requested`; final
publication status comes from the public repository Actions run and GitHub
Release, which may still be pending or may fail independently.
There is no private Runtime Image fallback or bridge manifest; missing evidence
fails the release before final-manifest generation.

Public-to-private reverse synchronization is intentionally not enabled. A
future community-source workflow would require a separately approved change
that defines the public source boundary, transformation, review, conflict
handling, and release ownership. Community pull requests are never silently
imported into the private repository.

Files carrying `GENERATED` or `READ-ONLY` markers are overwritten by the next
approved export. Changes to those files should be made in the private source
and regenerated through the release workflow.
