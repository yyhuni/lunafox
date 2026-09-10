# Public deployment contract

> **GENERATED / READ-ONLY** - edit the private source and regenerate this
> projection through the protected release workflow.

This repository publishes a small Compose-first deployment closure. The root
`compose.yaml`, `.env.example`, `resources/`, lifecycle scripts, release
indexes, notices, secretless validators, and the approved generated/read-only
Frontend, Server, Contracts, `engine-go`, Proto, first-party Extensions, Nginx,
and Bootstrap source/build closure are public. Agent source, private release
credentials/signing material, development Compose, installer implementation,
and local evidence remain private.

The public Runtime closure contains source, configuration, module metadata,
generated API bindings, workflow definitions, static resources, and approved
tests for each declared component. The support page's current Alipay, WeChat,
and contact QR JPGs are explicitly approved public assets. The reviewed
Dockerfiles and ignore rules are public so the canonical repository can build
Server, Frontend, Nginx, and Bootstrap. Dependency directories,
framework/build output, coverage, reports, screenshots, logs, test-results,
test plans, caches, certificates, and scratch files are excluded local
evidence, not deployment inputs.

Engine Runtime Images and Engine Packages are public artifacts. First-party
Engine source is GPL-3.0-only. Included third-party components remain governed
by their applicable licenses and attribution terms.

## Release channels

Source tags/main and mutable `release-channel` metadata are published
separately. On a clean work tree, the install command fetches
`channels/<channel>.env` and its `manifests/<version>.yaml` from the fixed
`https://raw.githubusercontent.com/yyhuni/lunafox/release-channel/` source,
validates schema-v3, version, path, and SHA-256, and only then enters the
Compose lifecycle. A manual fetch or checkout of `release-channel` is not
required.

When complete local `channels/` and `manifests/` directories already exist,
install reuses and revalidates them without network access or overwriting. A
single directory, incomplete files, download failure, or validation failure is
fail-closed. An exact release-tag checkout must also match channel `VERSION`.
If an interrupted operation leaves partial metadata, start from a clean work
tree and remove both incomplete directories only after confirming that no
deployment state exists.

The default channel is stable. Until a stable alias is governed and published,
omitting `--channel` explicitly directs the operator to canary and never
silently selects a prerelease. Channel records remain append-only; corrections
use a new prerelease tag and never rebind an existing digest, manifest, channel
record, or public tree.

Schema v2, alpha.46, legacy image variables, installer assets, and
`checksums.txt` are retired. No automatic migration from an old private
installation is provided.

## Supported hosts

The official first-release matrix is Ubuntu Server 22.04 or 24.04, native
Linux `amd64` or `arm64`, rootful Docker Engine, Docker client/daemon API
`>=1.45`, and Docker Compose v2 plugin `>=2.24.0`. Docker Desktop, Windows
containers, rootless Docker, remote daemons, and unlisted distributions are
experimental only. The scripts warn on an unlisted distribution but still
enforce actual Docker, network, image-pull, volume, port, and health checks.

Lifecycle commands use the caller's already authorized default local Docker
context. They do not run implicit `sudo`, change contexts, install plugins, or
modify systemd, daemon, socket, group, or firewall configuration. An enabled
Loki Docker logging plugin and named-volume/`volume-subpath` capability are
required before resource mutation.

## Compose and registries

Every product image, bootstrap image, and Engine Runtime Image is selected from
the release manifest and channel as an immutable digest. Docker Hub
(`docker.io/yyhuni`) is the default. `--registry ghcr` atomically selects the
matching `ghcr.io/yyhuni` candidate for every artifact. Per-service overrides,
mixed registries, automatic fallback, third-party registries, and offline
`docker image load` archives are not supported.

The `lunafox-bootstrap` Dockerfile and entrypoint are public GPL-3.0-only build
inputs, while the Agent it starts remains closed. The bootstrap service is
one-shot and has `restart: "no"`. It imports the release Engine Package inventory and default resources, checks
the Agent Engine mount boundary, registers required Engines, and exits zero
only after its work succeeds. It runs only during a fresh install or confirmed
reset. Its source and build context are part of the generated public Runtime
closure; the Agent it starts remains a private artifact.

This is a self-hosted deployment path for infrastructure the operator owns or
controls. Provider-hosted operation and redistribution of the closed Agent
artifact remain outside the default grant.

## Public source validation

The canonical public repository validates the generated Runtime source without
private credentials on export PRs, including non-publishing Docker builds.
After the protected generated PR is merged to public `main`, the workflow
builds and publishes `lunafox-server`, `lunafox-frontend`, `lunafox-nginx`, and
`lunafox-bootstrap` from the reviewed contexts with SBOM and provenance
attestations. The private release first builds, tests, and signs the closed
Agent binary bundle and exports its exact seven-file directory under
`agent/bin/<release-tag>/`; the public workflow verifies that merged Git tree,
packages `lunafox-agent`, and publishes its immutable digest. The public
workflow never checks out private source or
Agent credentials. It owns final digest promotion, Runtime manifest, channel,
and GitHub Release; missing or mismatched evidence stops release before the
final manifest. Recovery and rollback use only a previously verified immutable
digest.

## Configuration and certificates

On first install the host script generates `.env` with the operating-system
CSPRNG, writes it through a restricted temporary file, and sets mode `0600`.
Normal start/restart reuses that file. Only `--reset --confirm` may regenerate
credentials. Secrets are never printed by the lifecycle scripts.

Before Nginx starts, the script creates the external `lunafox_ssl` volume and a
self-signed certificate with a SAN for the configured DNS name or IP address.
There is no user-certificate override, ACME flow, or automatic renewal. Missing,
invalid, mismatched, or expired material stops the command and preserves state;
recovery is confirmed reset followed by a fresh install.

## Lifecycle and recovery

`install.sh` accepts only an empty LunaFox state. Existing containers, named
volumes, `.env`, state directory, or receipt stop installation until reset is
explicitly confirmed. A bootstrap failure remains visible in container logs;
there is no retry/repair/resume command.

`uninstall.sh` removes Compose containers, networks, and orchestration resources
while preserving data, configuration, certificates, and the completion receipt.
`uninstall.sh --purge --confirm` first invalidates the receipt and then removes
only project-owned volumes and host state.

The bootstrap-created Agent is a resident project container outside the
declarative Compose service list because its credential is created only inside
the bootstrap process. `stop`, `start`, and `restart` manage that existing
container together with the Compose services, but never recreate, re-register,
or otherwise repair it. If it is absent, the supported recovery is a confirmed
reset followed by a fresh install.

Completion is recorded in the separate external `lunafox_receipt` volume. Only
the short-lived invalidator and finalizer helpers can write it; the verifier is
read-only, and runtime/bootstrap services never mount it. Explicit
`start`/`restart`/`status` checks both the receipt and current service/HTTPS
health. Docker daemon or host restart may recover resident services through
their restart policy, but does not rerun bootstrap or claim a newly verified
completion.

## Deferred security and licensing

This release does not add Agent TLS certificate-chain, hostname, or mTLS
identity verification. The existing credential remains a long-lived
eight-character hexadecimal bearer without stronger replay, rotation, or
revocation guarantees.

Deployment files, the exported Runtime source, and first-party Engine source
are `GPL-3.0-only`. The separate closed Agent artifact notice applies only to
the closed Agent artifact; it does not govern Engine Runtime Images or Engine
Packages. Included third-party components remain governed by their applicable
licenses and attribution terms. The notice grants bounded internal self-hosting
and backup use only for the closed Agent artifact; source access, modification,
public redistribution, external registry mirroring, hosted/SaaS operation,
white-label use, and resale require the stated written exception.
