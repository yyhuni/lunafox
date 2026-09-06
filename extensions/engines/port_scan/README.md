# Port Scan Engine

This Engine scans a workspace candidate file built from `Execution.Target` and
finalized `Subdomains` facts with `naabu` and reports
validated `asset.host_port.v1` results through its generated Engine API v2
Facade.

## Engine Runtime Image

`Dockerfile` defines an independent multi-stage `port_scan` Runtime Image. It
cross-compiles the pure-Go engine executable and installs the official Naabu
`v2.4.0` Linux release archive selected by `TARGETARCH`. The amd64 and arm64
archives are pinned by their upstream SHA-256 values before extraction. Naabu's
official Linux build enables CGO for packet capture, so the final Ubuntu Noble
image includes the matching `libpcap0.8t64` runtime library instead of trying to
cross-compile Naabu with `CGO_ENABLED=0`. A target-independent Alpine downloader
uses only its built-in `wget`, `sha256sum`, and `unzip`; it installs no build
packages, and only the checksum-verified archive selection varies by
`TARGETARCH`. The `verify` build stage runs
`tests/container/container-conformance.sh` through a transient read-only
BuildKit bind mount to verify the engine, tool path, dynamic runtime
dependencies, and pinned version. The marker-gated final `runtime` stage does
not retain that script or any other container-test payload. Build it from the repository root
with the canonical `extensions/engines` context and the contracts module as a
named BuildKit context:

```sh
docker buildx build \
  --build-arg ENGINE_IMAGE_VERSION=0.0.0-dev \
  --build-context contracts=./contracts \
  --build-context engine-go=./engine-go \
  -f extensions/engines/port_scan/Dockerfile \
  extensions/engines
```

The Runtime Image starts the generated Engine API v2 Facade. No legacy server
entrypoint is built or started.

The final image explicitly keeps the image default user at `0:0` and requests
no `Privileged`, `CapAdd`, `CapDrop`, `SecurityOpt`, or networking override.
It follows the shared [`container` contract](../container/README.md) for the
exact root identity, OCI labels and startup metadata, bootstrap names, and
canonical mount targets.

The existing command tests lock Naabu's current CONNECT-style active argv and
passive argv. This task neither runs a real SYN scan nor claims that SYN/raw
network behavior works; the later generic image runner owns real container
lifecycle conformance, while special network capabilities remain out of scope.

The Engine root keeps production source and `engine.json` together; its root
`Dockerfile` is the Runtime Image build definition, and `tests/container/`
contains development/release-only image conformance scripts, profiles, and
offline stubs. Ordinary Go package tests remain beside the packages they test.

## Workflow Boundary

The builtin `default` scan workflow runs `subdomain_discovery` first and
`port_scan` second. The second stage stays blocked until the discovery stage
reaches terminal success, so Server can produce a Subdomains from current-scan
evidence before Agent materializes it and starts the Engine Container.

`port_scan` uses strict `engine.v5` and declares applicability independently
with `supportedTargetTypes: [domain, ip, cidr]`. It does not declare an input
allow-set. Server projects finalized Subdomains facts, while the complete
generated Registry surface exposes `Execution.Input.Subdomains.Path(ctx)` when
this runtime actually needs those facts.

## Runtime Input

Runtime business logic calls `Execution.Input.Subdomains.Path(ctx)` at its input
stage; each non-empty line is one finalized DNS fact. The Engine owns Domain/IP/CIDR Target baseline
expansion and appends those facts in snapshot order. Agent and Engine code do
not query Server persistence. A valid zero-byte Subdomains product is still
combined with the Target baseline; Naabu never receives the zero-byte facts path
directly.

Candidate preparation is streaming and returns a checked `uint64` host count.
CIDR addresses are expanded incrementally without an Engine-local record,
byte, host, or address cap. Line framing, write/close, cancellation, and
counter-overflow failures stop before Naabu starts and remove the incomplete
candidate file; complete input is never silently truncated. Each Subdomains
line is copied as the candidate identity after the Target baseline; this stage
does not build an input-sized deduplication map or Engine retry/replay state.
The valid zero-record product still leaves the baseline candidate set intact.

## Config

The engine uses the standard LunaFox section-level config shape:

- `naabu_active`: `enabled`, `timeout`, `threads`, `port-mode`, `ports`, `top-ports`, `rate`
- `naabu_passive`: `enabled`, `timeout`

Both tools default to enabled. `timeout` is an integer process timeout in seconds; active defaults to 3600 seconds and passive defaults to 600 seconds. Defaults come from `engine.json` and generated config artifacts, not from a second hand-maintained runtime default table.

Active port selection is controlled by `port-mode`: `top` emits only `-top-ports`, `custom` emits only `-p`, and `top-and-custom` emits `-top-ports` plus `-p` when `ports` is non-empty. `top-and-custom` is the default because it preserves naabu's additive behavior; only `custom` requires non-empty `ports`. `top-ports` is a string enum matching naabu's accepted presets: `100`, `1000`, and `full`. The previous `top-plus-custom` value is no longer accepted.

## Execution And Results

The Engine owns Naabu process execution inside its Runtime Image and propagates
the handler context. `naabu` writes JSON lines to Engine-owned workspace output
files via `-o`; there is no platform process capability. Active and passive
Naabu stages each produce an independently meaningful final result artifact,
so this Engine intentionally aggregates both artifacts before result reporting.

Result reporting parses those JSON lines, skips malformed or invalid rows, and
translates Naabu's empty or whitespace-only `host` plus valid IPv4 form into a
complete canonical `host=ip` fact before result encoding. This is a Naabu
source-adapter rule, not a Server or downstream consumer fallback. It then
globally deduplicates canonical `(host, ip, port)` records through a
workspace-local, byte-bounded external merge before streaming them through
`Execution.Results.HostPorts`. Agent projects typed batches to Server
resultingest, which remains the final semantic validation, authorization,
snapshot persistence, target-scope filtering, retry idempotency, and current
asset projection boundary.

Final-artifact parsing bounds each physical scanner record at 4 MiB. Malformed,
invalid, empty, and oversized physical records are counted and skipped so later
records can still be reported. Invalid host, IPv4, or port identity fields are
not guessed.

The result-reporting merge implementation is owned by this Engine and does not
depend on a cross-Engine deduplication helper.

When duplicate canonical host/IP/port identities cross parser artifacts, the
bounded merge retains the last valid record in deterministic source order.

## Runtime Logging Boundary

Engine code uses the generated message-only progress port for operator-facing
port-scan state changes. Agent projects those explicit events into task progress
logs without accepting scope or summary fields from the Engine.

Enabled stages that fail during command execution emit `fail stage ... reason=command: ...` before returning the runtime error. Successful enabled stages keep `complete stage ... success=1 failed=0 outputs=1`; disabled stages keep `skip stage ... reason=disabled`.

External tool stdout, stderr, output artifacts, and detailed result reporting diagnostics remain runtime logs or process diagnostics. After both active and passive result artifacts are parsed and submitted, the engine emits one `complete result reporting` progress event with source, parsed, submitted, and aggregated skip-reason counts, including oversized records; it does not include individual hosts or ports.

## Tooling

The Engine Runtime Image carries `naabu` on its own `PATH`; release tooling
publishes and executes this immutable image directly.
