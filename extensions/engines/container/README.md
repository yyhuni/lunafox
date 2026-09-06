# Shared Builtin Engine Image Conformance

This `container/` directory is shared CI/release infrastructure for all
first-party Engine Runtime Images. It is not an Engine implementation tree and
is not copied into an Engine Runtime Image. Each individual Engine keeps its
production source and root `Dockerfile` at its own root, while its
container-only verification assets live in `<engine>/tests/container/`.

All inventory-mapped builtin Engine Runtime Images share one container-visible contract.
The contract is a platform projection, not a second Engine Package manifest;
package v2 binds only the verified digest-qualified image refs and cannot change
the image command, user, paths, or bootstrap names.

Each image has:

- OCI `User` `0:0`, `WorkingDir` `/workspace`, `StopSignal` `SIGTERM`, an
  engine-specific absolute `Entrypoint`, and an empty `Cmd`.
- Engine binaries under `/opt/lunafox-engine/bin/` and scanner tools under
  `/opt/lunafox-tools/bin/` on `PATH`.
- The same runtime roots: `/run/lunafox/context`, `/run/lunafox/credential`,
  `/run/lunafox/inputs`, `/run/lunafox/resources/config`,
  `/run/lunafox/resources/platform`, `/run/lunafox/socket`, and writable
  `/workspace`.

## Dockerfile Authoring Boundary

The scaffold generates the shared platform skeleton for standard Go Engines:
builder/module-cache setup, named `contracts` and `engine-go` contexts, runtime
base and fixed directories, OCI metadata, root/signal/entrypoint defaults, and
the verify/runtime conformance stages. Standard Dockerfiles are marked with
`# lunafox:dockerfile-mode=standard` and the generated region is refreshed by
`make engine-refresh` and checked by `make engine-generated-check`.

Tool downloads, checksums, extra system packages, multi-language builders, and
Engine-specific runtime files stay between the stable
`lunafox:engine-owned:*` markers. They remain ordinary Dockerfile source and
are not described by a new `tools.yaml` or `build.yaml` schema.

Engines that need Rust, Python, Chromium, native compilation, or another
non-standard stage topology may keep their complete Dockerfile and mark it
`# lunafox:dockerfile-mode=custom`. Refresh and check flows preserve custom
Dockerfiles. The current custom engines are `screenshot`,
`fingerprint_detection`, `subdomain_discovery`, and `url_collection`; they
retain their special build/runtime logic while following this same runtime
image contract.

`ENGINE_IMAGE_VERSION` is a required build argument. Local additive builds use
`0.0.0-dev`; a publisher must pass the actual release version. The build fails
instead of emitting an empty version label when the argument is missing.

Agent attaches fixed task-scoped mounts without bootstrap environment variables:
`/run/lunafox/context/execution.pb`, `/run/lunafox/socket/engine.sock`,
`/run/lunafox/credential/token`, and one read-only `/run/lunafox/inputs`
directory. It never mounts input files individually. The fixed paths, binary Context format, and
workspace ABI are owned by `engine-go/protocol`; production images and container
creation must not override them through `LUNAFOX_` environment values. The image
does not declare `VOLUME`; Agent attaches task-scoped mounts with the appropriate
read-only or writable mode. The images launch the generated Engine API v2 facade
as their main process.

Each generated Engine receives all four typed input handles (`Subdomains`,
`HostPorts`, `WebsiteURLs`, and `EndpointURLs`). Calling a handle's
`Path(ctx)` requests only that Registry role through the independent input UDS
service; Agent owns scoped authorization, retry, same-execution single-flight,
hidden sibling staging, and atomic publication. Config and platform resources
remain eager mounts. An old Engine image, package, plan, or adapter is rejected
before handler execution; rollback requires one matching revision and a fresh
install rather than migration or task draining.

## Runtime Image Conformance

`container/cmd/engine-image-conformance` is the common real-image runner. It
accepts either one verified image-first build receipt or one explicit
`DIRECTORY=DIGEST_REF` value for every inventory Engine. The target platform is always explicit. Run the
same receipt once for each platform and retain each JSON report; a receipt's
declared `platforms[]` is input validation, not execution evidence.

```sh
make verify-engine-image-conformance \
  ENGINE_IMAGE_CONFORMANCE_PLATFORM=linux/amd64 \
  ENGINE_IMAGE_CONFORMANCE_OUTPUT=dist/engine-runtime-images/conformance-amd64.json

make verify-engine-image-conformance \
  ENGINE_IMAGE_CONFORMANCE_PLATFORM=linux/arm64 \
  ENGINE_IMAGE_CONFORMANCE_OUTPUT=dist/engine-runtime-images/conformance-arm64.json
```

The runner first executes the protocol-direct fixture from
`contracts/engineapi/conformance`. For every image it then pulls and inspects
the requested platform, copies that Engine's source-owned
`tests/container/container-conformance.sh` below the configured
daemon-visible temporary root, and mounts it read-only into a disposable
network-isolated container. The script runs through the image shell against
the canonical Engine binary path; it is not retained in the pulled Runtime
Image. The runner rejects
Worker/Agent/Docker CLI fallback, and starts the image's unmodified default
Entrypoint with canonical Context, credential, input, config-resource,
platform-resource, socket, and workspace mounts. The fixture socket is created
inside a disposable daemon-owned named volume by the repository-local
`engine-image-conformance-proxy`; this preserves a real filesystem UDS when
the Docker daemon runs behind Docker Desktop/OrbStack, where a host-process
AF_UNIX listener cannot be exported through a bind mount. The proxy is only
test infrastructure and does not change the Engine bootstrap endpoint or
production Agent/Engine topology. Container inspect must show
the default root identity, empty Cmd, SIGTERM, default nonprivileged/security
and network profile, read-only platform files, and writable workspace.

The lifecycle cases cover legal empty-input exit 0, invalid-bootstrap exit 1,
status-only gRPC acknowledgement ordering, and a real `docker stop` for every
Engine on the selected platform. Each per-Engine JSON record carries its own
`sigtermVerified` evidence. Every Engine's result fixture holds the
`SubmitResultBatch` response, requires the container to remain running, then
releases the empty acknowledgement and requires exit 0. The signal fixture uses
non-empty inputs and offline scanner stubs, blocks at the first progress
acknowledgement, and requires SIGTERM-derived cancellation to exit 0. Separate
offline scanner fixtures are mounted read-only over
canonical tool paths to lock argv, workspace output, typed result submission,
and tool subprocess ownership without public network access or a real SYN scan.
The image-local checks run before those overrides and remain the proof that the
real scanner payloads exist, have the pinned versions, and execute successfully
on the selected architecture; native help probes must return exit 0 rather than
merely print an execution error. A non-native platform run is accepted only when
Docker can actually pull and execute it (for example through configured
emulation); the daemon architecture itself is never treated as proof for the
requested image platform.

Fingerprint Detection is one such inventory row. Its image builds Observer Ward
from the LunaFox fork `yyhuni/observer_ward_for_luna` tag
`v2026.6.28-lunafox.1` at the verified peeled commit
`65801cf6d4b3dd4bb07ea7a1cf6e849713c42ea6`, and its Engine-local conformance
must execute the unwrapped binary help probe on both `linux/amd64` and
`linux/arm64`. Changing that fork pin requires revalidating the Observer Ward
JSON protocol, `input_target` attribution, trusted-status behavior, fixed
command mapping, managed FingerprintHub loading, empty-corpus fail-closed
behavior, redirect attribution, and both architecture reports before
publication.

The CLI derives its root context from `SIGINT`/`SIGTERM`. Docker pull, ordinary
operations, image-local runs, and container waits each have an explicit bounded
deadline. Known containers are removed with a separate bounded cleanup context,
so caller cancellation does not skip teardown.

Direct refs use the CLI entrypoint:

```sh
cd extensions/engines
go run ./container/cmd/engine-image-conformance \
  --repo-root ../.. \
  --platform linux/arm64 \
  --daemon-visible-root /tmp \
  --engine-image DIRECTORY=REGISTRY/REPOSITORY@sha256:...
```

Repeat `--engine-image` once for every row in `tool-inventory.json`. The
runner rejects missing, duplicate, or unknown directories instead of accepting
a matching count.

## Tool Inventory Guard

[`tool-inventory.json`](tool-inventory.json) is the closed inventory for all
first-party image lanes. It binds each engine directory to its engine binary,
and exact scanner tool versions. It describes runtime ownership only: it does
not declare a conformance command, is not an Engine Package field, and does
not select a runtime image reference.

Run the source guard with:

```sh
make verify-engine-image-tool-inventory
```

The guard discovers every immediate Engine directory with both `engine.json`
and `Dockerfile`, requires an exact inventory row and an Engine-local
`tests/container/image-conformance.json` profile plus a regular non-symlink
`tests/container/container-conformance.sh`, checks that each tool is pinned
and installed only in its owning image, and rejects those tools from the Agent,
Server, Frontend, and Nginx Dockerfiles. The Worker image and its centralized
tool build are retired. Scanner tools belong only to their owning Engine Runtime
Image; this guard rejects any restored Worker Dockerfile or Worker CI build/test
job.

Each Engine-local profile declares only development/release fixture facts: the
canonical test target, enabled sections, typed result, offline tool stubs,
argv assertions, SIGTERM gate, and optional platform-resource/image-local
probe. Referenced fixture files stay below that Engine's `tests/container/`
directory. The shared runner validates this profile against the manifest and
inventory before invoking Docker. These files are not included in Engine
Package v2 and are never a production catalog or runtime image-selection
source.

Release or image CI can additionally pass built refs with repeated
`--product-image ID=REF` and `--engine-image DIRECTORY=REF`. In that mode the
guard copies the source-owned conformance script into a disposable container
through the Docker API, and verifies product images do not resolve any
inventoried tool. The inventory guard therefore does not require its client and
Docker daemon to share a temporary directory.
`--require-runtime-images` makes the complete
four-product/every-discovered-Engine ref set mandatory. This runtime mode complements the
Task 4.5 lifecycle runner; it does not substitute for Engine API bootstrap,
signal, mount, or exit-code conformance.

## Image-first Release Input

`tools/engine-release -command discover` is the release discovery boundary. It
enumerates every immediate Engine directory, strictly validates its
`engine.v5` `engine.json`, requires the matching Dockerfile, and derives the
`lunafox-engine-runtime-<slug>` repository with the contracts-owned helper. It
does not read a source `package.json`, runtime manifest, Registry address, tag,
or digest.

`scripts/ci/build-engine-release.sh` first publishes and verifies one
multi-platform Runtime Image index per discovered Engine. Only the resulting
ephemeral build receipt may be consumed by package-v2 generation. Development
receipts contain one host-daemon-reachable `localhost:<port>` digest ref;
production receipts contain ordered Docker Hub/GHCR locations for the same
copied index digest. Run `make verify-engine-release-contract` for the static
release boundary and its artifact-aware form in publisher CI.

When BuildKit runs in a separate VM or network, set
`ENGINE_REGISTRY_TRANSPORT_HOST` to the endpoint used only for the BuildKit
push transport and keep `ENGINE_REGISTRY_PUBLIC_HOST` on a host-reachable
`localhost:<port>`. Publisher-side index inspection reads the same pushed tag
through the public endpoint because transport-only DNS is not assumed to be
resolvable on the host. The exporter explicitly enables OCI media types, so
the retained digest identifies an OCI image index rather than a Docker
manifest list. The transport endpoint is never advertised to Agent and is
never copied into `package.json`. Before creating or bootstrapping a builder,
the publisher makes
an explicit plain-HTTP request to the public `/v2/` endpoint and requires a
Distribution Registry `200` or `401` response with the
`Docker-Distribution-Api-Version: registry/2.0` header. This fast preflight does
not replace the post-build pull and platform inspect through Agent's host Docker
daemon. For an HTTP-only development Registry transport, set
`ENGINE_REGISTRY_TRANSPORT_INSECURE=true`; a fresh publisher builder receives a
minimal `buildkitd.toml`; it is also required when the transport and public
endpoint are the same. The publisher never retries HTTPS as HTTP. A custom
`ENGINE_BUILDER_CONFIG` likewise requires a fresh `ENGINE_BUILDER_NAME`, because
Buildx cannot apply new daemon config to an existing builder.

## Nuclei Runtime Pin

The `nuclei_vulnerability` inventory row pins the image-local Nuclei
`v3.4.10` release ZIP by architecture. Its Dockerfile verifies the published
SHA-256 before installing the binary (`linux/amd64`:
`234c12cc5288af071abdcd6f854245b6067345556e1235cf96b76725c1004357`;
`linux/arm64`:
`d1ed1a5c0df49d8fcd64cab4ff5840b793d6bf133b082bb6a7f67d5fc0f9c327`). The
inventory version, image marker, and local conformance script must agree on
`v3.4.10`; a mutable Go-module install is not an accepted release source.
