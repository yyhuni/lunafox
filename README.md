# LunaFox public deployment

> **GENERATED / READ-ONLY PROJECTION**

This repository is the anonymous production deployment projection generated
from the private source repository
[`yyhuni/lunafox-private`](https://github.com/yyhuni/lunafox-private). It
contains the Compose closure, host lifecycle scripts, release indexes,
deployment resources, and the approved generated/read-only source closure for
Frontend, Server, Contracts, `engine-go`, Proto, first-party Extensions,
Nginx, and Bootstrap. Agent source, private release/signing material, and
standalone installer source are not published here.

## Install

On a clean supported host, use a public `main` or release-tag work tree and run:

```bash
./install.sh --channel canary --public-host example.example
```

During install, the lifecycle fetches the selected schema-v3 channel record and
manifest from the fixed public `release-channel` source, then writes them to
local `channels/` and `manifests/` only after validation. No manual fetch or
checkout of another branch is required. When the work tree is an exact release
tag, that tag must match the channel version. A complete local metadata set is
reused without network access; partial metadata, network errors, or digest and
version failures stop installation without fallback. After an interrupted
attempt, start from a clean work tree, or remove incomplete `channels/` and
`manifests/` only after confirming that no deployment state exists.

The default channel remains stable. Until a stable alias is governed and
published, omitting `--channel` gives explicit canary guidance and never
silently selects a prerelease. Docker Hub is the default image source;
`--registry ghcr` switches the complete immutable image and Engine Package
closure, with no mixed-source or automatic fallback.

## Lifecycle

```bash
./start.sh
./restart.sh
./start.sh --status
./stop.sh
./uninstall.sh
./uninstall.sh --purge --confirm
```

`install.sh` creates the host-owned `.env` and a self-signed SAN certificate
atomically, then runs the public-source `lunafox-bootstrap` image once before
starting resident services. Re-running install, replacing configuration, or
recovering from a failed bootstrap requires the explicit confirmed reset:

```bash
./install.sh --reset --confirm --channel canary --public-host example.example
```

The bootstrap-managed Agent is stopped, started, and restarted together with
the Compose services; normal lifecycle commands never recreate or re-register
it. Normal uninstall removes Compose containers, the Agent container, and the
network but preserves data, configuration, certificates, and the completion
receipt. `--purge` removes only project-owned state after confirmation.
Daemon/host recovery uses resident restart policies; it does not rerun
bootstrap or manufacture a completion receipt.

See [`docs/public-deployment.md`](docs/public-deployment.md) for the supported
host matrix, registry contract, certificate limits, and recovery boundaries.

This is a self-hosted deployment projection: operators run it on infrastructure
they own or control. Provider-hosted operation and redistribution of the closed
Agent artifact require the separate written authorization described in the
notice.

## Release identity

Product images use the fixed `yyhuni` namespace and repositories
`lunafox-server`, `lunafox-frontend`, `lunafox-nginx`, `lunafox-agent`, and
`lunafox-bootstrap`. Engine Runtime Image/Package repositories follow
`lunafox-engine-runtime-<name>`. Every release reference is immutable and
digest-qualified. Public protected `main` builds the reviewed Server, Frontend,
Nginx, Bootstrap, and Agent Runtime Images with immutable digest, SBOM,
provenance, and attestation evidence. Private CI builds, tests, and signs the
closed Agent binary bundle, then places its exact seven-file directory under
`agent/bin/<release-tag>/` in the generated public tree. The public workflow
verifies that merged directory, packages Agent, promotes all accepted digests
to Docker Hub and GHCR, and owns the final release manifest, channel, and
GitHub Release.
The private run ends after it requests protected auto-merge for the generated
export PR; a green private run means `public handoff requested`, not that the
canonical public Release is complete. Check the public repository Actions run
and GitHub Release for final status.

## Runtime source boundary

The exported Runtime source closure includes the TypeScript/React frontend,
Go Server, Contracts, `engine-go`, Proto, first-party Extensions, Nginx
configuration, Bootstrap Dockerfile/entrypoint, and their reviewed tests and
build metadata. It is generated from a frozen private revision and remains
read-only: the private repository is the sole development authority for the
closed Agent source and binary handoff, while the public repository owns final
product publication. Public pull requests do not automatically synchronize
back.

Public pull-request CI validates this closure without secrets and performs
non-publishing Docker builds. Only a validated generated export merged through
protected `main` can publish the four public Runtime Images. Dependency
directories, framework output, coverage, screenshots, logs, test-results,
test plans, caches, TLS material, Agent source, and private release inputs
remain outside the projection.

Engine Runtime Images and Engine Packages are public artifacts. First-party
Engine source is GPL-3.0-only. Included third-party components remain governed
by their applicable licenses and attribution terms.

## Security scope

The current Agent credential remains the existing long-lived eight-character
hexadecimal bearer. This projection does not claim Agent control-plane TLS
identity verification, replay protection, rotation, or revocation. TLS and
credential hardening require a separate security change.

## License

The exported first-party Runtime source, tests, deployment files,
documentation, and indexes are licensed under SPDX `GPL-3.0-only`. Agent
source, Agent credentials, private signing/release material, and the resulting
closed Agent artifact remain outside the public source boundary. The artifact
terms in [`NOTICE-CLOSED-ARTIFACTS.md`](NOTICE-CLOSED-ARTIFACTS.md) apply to
the closed Agent artifact only; they do not govern Engine Runtime Images or
Engine Packages.
