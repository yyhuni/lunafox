# Website Discovery Engine

This Engine probes a workspace candidate file built from Target baselines and
finalized HostPort facts with `httpx` and reports
validated `asset.website.v1` results through its generated Engine API v2 Facade.

## Engine Runtime Image

`Dockerfile` defines an independent multi-stage `website_discovery` Runtime
Image. It cross-compiles the engine executable and image-local httpx `v1.8.1`.
The `verify` build stage runs `tests/container/container-conformance.sh`
through a transient read-only BuildKit bind mount. The conformance checks the
engine/tool executable and version, performs a bounded empty-list httpx smoke
without network probing, and rejects a Docker CLI fallback. The marker-gated
final `runtime` stage does not retain the script or other container-test
payload.

Build it from the repository root with the canonical `extensions/engines`
context and named contracts context:

```sh
docker buildx build \
  --build-arg ENGINE_IMAGE_VERSION=0.0.0-dev \
  --build-context contracts=./contracts \
  --build-context engine-go=./engine-go \
  -f extensions/engines/website_discovery/Dockerfile \
  extensions/engines
```

The Runtime Image starts the generated Engine API v2 Facade. No legacy server
entrypoint is built or started.

The final image explicitly keeps the image default user at `0:0` and requests
no `Privileged`, `CapAdd`, `CapDrop`, `SecurityOpt`, or networking override. It
follows the shared [`container` contract](../container/README.md) for the
exact root identity, OCI labels and startup metadata, bootstrap names, and
canonical mount targets.

Existing Engine-owned httpx argv tests remain the source of truth for command mapping;
this task's minimal smoke is not a real website discovery run.

The Engine root keeps production source and `engine.json` together; its root
`Dockerfile` is the Runtime Image build definition, and `tests/container/`
contains development/release-only image conformance scripts, profiles, and
offline stubs. Ordinary Go package tests remain beside the packages they test.

## Workflow Boundary

The builtin `default` scan workflow runs `subdomain_discovery`, then `port_scan`,
then `website_discovery`. Workflow owns that Task dependency and keeps the
website stage blocked until the port stage reaches terminal success. Once
Workflow authorizes the Website Task, Server produces its input; the Engine
does not query or interpret predecessor Task or Engine state.

`website_discovery` uses strict `engine.v5` and declares applicability
independently with `supportedTargetTypes: [domain, ip, cidr]`. It does not
declare an input allow-set. Server projects finalized HostPorts facts, while the
complete generated Registry surface exposes
`Execution.Input.HostPorts.Path(ctx)` when this runtime actually needs them.

## Runtime Input

Runtime business logic calls `Execution.Input.HostPorts.Path(ctx)` at its input
stage; each non-empty line is one complete `host`/`ip`/`port` JSONL fact. The Engine owns Target
baseline expansion, HostPort conversion, deterministic ordering, and overlap
handling. Agent and Engine code do not query Server persistence, regenerate
snapshot facts, or infer predecessor Task or Engine state. A zero-record
successful artifact is valid; the Engine still writes its non-empty Target
baseline workspace file before starting httpx.

Every HostPort fact must already be canonical and complete. An empty or invalid
host, IP, or port fails candidate preparation before HTTPX starts; Website
Discovery never substitutes the IP for a missing host.

HostPort preparation reads records incrementally into fixed-size sorted chunks,
then uses a bounded fan-in merge to deduplicate host+port keys and stream
deterministic host/port order after the Target baseline. Chunk and framing
limits are protocol/workspace boundaries, not aggregate business caps; complete
large inputs are never truncated. Merge, write, close, cancellation, or
malformed-record failures remove task-local state before HTTPX launch. The
Engine does not retain a retry/replay collection; any configured HTTPX retry is
owned by the external tool and reuses the completed candidate artifact.

## Config

The engine uses the standard LunaFox section-level config shape:

- `httpx`: `enabled`, `timeout`, `threads`, `rate-limit`, `request-timeout`, `retries`

The `httpx` stage defaults to enabled. `timeout` is the process timeout in seconds; `request-timeout` is mapped to httpx's per-request timeout.

## Execution And Results

The Engine owns httpx process execution inside its Runtime Image and propagates
the handler context. `httpx` writes JSON lines to one Engine-owned final
workspace output artifact; there is no platform process capability.

Result reporting parses those JSON lines, skips malformed or invalid rows,
validates Website URLs without changing their bytes, and globally deduplicates
them through a workspace-local, byte-bounded external merge before streaming
findings through `Execution.Results.Websites`. Only exactly equal accepted URL
strings collide, and the last valid scanner observation in deterministic parser
order is retained. Agent projects the typed batches
to Server resultingest, which owns snapshot persistence, target-scope filtering,
retry idempotency, and current asset projection.

Final-artifact parsing bounds each physical scanner record at 4 MiB. An
oversized record is counted and skipped so later records can still be reported.
The raw input URL remains the only Website identity source: invalid URL
identity is skipped; an invalid explicit host, status code, content length, or
optional value rejects the complete record. Accepted v1 records are complete
observations, so omitted optional evidence is submitted as its zero value.

The result-reporting merge implementation is owned by this Engine and does not
depend on a cross-Engine deduplication helper.

After the final result artifact is parsed and submitted, the engine emits one
`complete result reporting` progress event with source, parsed, submitted, and
aggregated skip-reason counts, including oversized records and recovered
metadata. It does not include individual URLs or response evidence.

## Tooling

The Engine Runtime Image carries `httpx` on its own `PATH`; release tooling
publishes and executes this immutable image directly.
