# Engine Author API

`contracts/engineapi` contains explicitly test-only protocol conformance support.
The production Engine Protocol, fixed container ABI, and SDK live in the
independent `engine-go` module.

The `conformance` package and its `testdata` are not author APIs or release payloads.
Do not place host runtime internals, shared host/engine contract DTOs,
filesystem package loaders, transport sessions, build helpers, or compatibility
aliases here.

Current packages:

- `engine-go/protocol`: language-neutral generated Engine API v2 bindings and
  fixed ABI values. It is intentionally outside this module.
- `engine-go/sdk`: the single production `Run(adapter) error` lifecycle and
  narrow progress/result reporting ports. It owns no package, manifest, or
  business-operation contract.
- `conformance`: test-only Engine API v2 harness. Its protocol-direct Engine
  fixture is compiled from `testdata`, imports the canonical raw generated
  bindings without runtimekit or a generated author Facade, and is not a fourth
  Engine, SDK, scaffold, or production client.

## Engine API v2 Protocol

The canonical major-2 wire/file contracts are generated under
`engine-go/protocol` from the three versioned proto sources.
`EngineExecutionContext` is a passive exact-schema binary protobuf message;
`EngineExecutionReportingService` contains only unary `ReportProgress` and
`SubmitResultBatch`, and `EngineExecutionInputService` is an independent
`MaterializeExecutionInput` service. Successful responses remain empty for
reporting and carry only the canonical Registry path for input materialization.
These raw bindings are protocol facts, not a second ordinary author SDK or a
runtimekit alias.

## Terminal Diagnostics

`EngineExecutionDiagnosticsService` is a separate, task-private UDS side
channel. `EstablishExecutionDiagnostics` is a mandatory matching-revision
confirmation and carries no task, execution, Agent, or credential identity.
`ReportTerminalDiagnostics` then accepts at most one bounded terminal snapshot
for that authenticated session. It does not alter the status-only
`ReportProgress` or `SubmitResultBatch` RPCs.

The generated Facade and `engine-go/sdk` own typed-item receive/encode and
logical-batch submit/acknowledgement watermarks. Handwritten Engine handlers
continue to use only typed Context, Progress, and Results APIs: they must not
import diagnostic protobufs, acquire a diagnostic client, choose failure
stages/error types, or upload terminal observations. Snapshots carry only
closed classifications and fixed per-result-type counters. They never carry
result payloads, raw errors, tool output, paths, or credentials.

The terminal report remains best effort. A lost, rejected, or timed-out report
is evidence loss only; Agent remains responsible for cancellation, timeout,
OOM, container, and exit-code facts and must still finalize the task.

This repository is still in disposable pre-release development, so this input
contract is rebuilt in place while retaining Engine API major 2. Server, Agent,
generators, conformance fixtures, and first-party Engines must come from the
same post-cut revision; mixed old/new v2 components are unsupported. Old saved
plans, installed bundles, caches, and other pre-cut state are invalidated and
cleared rather than migrated. A development rollback restores the old revision
and rebuilds empty state. After the first stable release, any breaking
Engine-visible change requires a new major such as v3.

The test-only protocol-direct fixture reads exactly the Context-file, filesystem
UDS endpoint, and credential-file bootstrap entries. It validates the complete
Target and complete typed Registry handle surface, consumes eager config/platform
resource paths locally, waits for UDS readiness, and exercises message-only
progress plus explicit status-only result submission. Ordinary Engine code never
sees the input RPC, role strings, credential, or transport client.

The `engine-go/sdk` surface is intentionally small:

Generated adapters call `Run(adapter) error`; the SDK reads fixed binary
Context/credential files, connects to the filesystem UDS, provides message-only
progress and a bound encoded-item `ResultSink`, enforces limits, performs only
the strict indeterminate retry allowed by Protocol, and closes transport and
clears credentials on every exit. Raw gRPC status and control-plane scope never
enter an ordinary Engine handler.

`tools/engine-manifest-artifact-gen` owns the Go Engine-local contract lane.
For one strict normalized `engine.v5` definition it emits exactly one
`contract/execution_generated.go` type surface and one `cmd/.../execution_run_generated.go`
entry adapter. The contract contains complete typed Target/Input/Config/resource/
workspace/progress/results values; the entry adapter alone sees Protocol, SDK,
bootstrap, and process lifecycle. Result item types, Schema-basics encoders,
fields, and submitters are generated for every canonical descriptor in
`contracts/results`; Engines do not maintain a `contract/results.json` sidecar.
The adapter registers and creates each ResultSink lazily on first `Submit`, so
unused result ports do not affect startup. Synthetic compile fixtures cover the
complete registry and reject cross-port result item types.

Context file bindings use the contracts-owned `executionartifact` registry and
the fixed `engine-go/protocol` ABI: the one read-only `/run/lunafox/inputs`
directory contains Subdomains at `/run/lunafox/inputs/subdomains.txt`, HostPorts
at `/run/lunafox/inputs/host-ports.jsonl`, and WebsiteURLs at
`/run/lunafox/inputs/website-urls.txt`; config resources below
`/run/lunafox/resources/config/<section>/<param>/`, platform resources below
`/run/lunafox/resources/platform/<id>/`, and writable workspace exactly at
`/workspace`. Bootstrap targets are `/run/lunafox/context/execution.pb`,
`/run/lunafox/socket/engine.sock`, and `/run/lunafox/credential/token`; Agent
mounts them at those fixed paths without environment-variable overrides. Runtime
validation rejects wrong-role content contracts, path
drift, symlink components, writable input/resource mounts, read-only workspace,
and result limits above the platform maximum. Permission bits alone are not
treated as proof of mount access.

The `engine-go` SDK, Engine-local typed contracts, and generated adapters are
the only active Engine Container authoring path. Ordinary Engine business code
must not receive SDK clients, bootstrap values, credentials, or raw v2 bindings
as its long-term authoring API.

## Engine API v2 Authoring Shape

`engine.json` uses strict `engine.v5`. `execution.engineApiMajor` is `2`,
`supportedTargetTypes` independently declares applicability, and
`execution.inputs` (and aliases) is rejected. Input membership is never saved in
a plan; every Engine receives the complete closed Registry handle set.
Target is never an input: every executable handler receives the complete
canonical `Execution.Target`.

The generated handler receives only typed author values:

- `Execution.Target`: canonical type and value.
- `Execution.Input`: the complete `Subdomains`, `HostPorts`, and `WebsiteURLs`
  typed handles. Each handle exposes only `Path(ctx context.Context) (string,
  error)`; the first call lazily materializes its role and later calls reuse the
  successful execution-local path.
- `Execution.Config`: section values and direct config-resource paths.
- `Execution.PlatformResources`: direct declared platform-resource paths;
  the author manifest declares these through `execution.executionResources`.
- `Execution.Workspace`: the writable Engine directory.
- `Execution.Progress`: one message-only method.
- `Execution.Results`: typed result streams with error/nil acknowledgement.

The generated Run adapter is the sole Engine API v2 Context-to-`Execution`
validation and projection boundary. Once it invokes an ordinary handler, that
handler and its runtime call typed input `Path(ctx)` at the point of use; they do
not read a path field, call `Ensure`, handle role strings, or access RPC/UDS
transports. Config and platform resource paths remain eager typed values.
Engine code still validates its own cross-field rules, workspace-local paths,
and untrusted scanner output.

Server owns applicability, plan compilation, resource selection, and finalized
Scan-fact projection. Agent owns transfer, integrity validation, read-only materialization,
workspace mounting, and container lifecycle. Engine code consumes those local
values, owns image-local tools and workspace files, parses tool output, and
submits typed results. It must not query persistence or reconstruct Scan facts;
each first-party consumer owns its confirmed Target baseline composition.

Declared config resources are conditional on their owning configSection. An
enabled section requires each declared resource path exactly once; a disabled
section requires those paths to be absent and leaves the generated typed
`FilePath` fields at their zero value. Business code may read such fields only
inside the matching enabled branch. Declared execution-wide platform resources
remain eager and are required regardless of section state. Missing required or
unexpected disabled-section paths fail; business stages must not request
materialization or fall back to user/default tool paths. Input/resource files
are immutable. A transformation reads or copies them into an Engine-owned child
path under `Execution.Workspace` before writing.

A file-backed handler calls the relevant typed `Path(ctx)` before starting its
scanner and distinguishes a valid zero-byte fact product from an absent or
failed input. Independent
Target/Context/config validation still runs; first-party Engines combine valid
empty facts with their own Target baseline and never pass the zero-byte facts
path directly to an external tool. A missing, unreadable, or malformed file is
an error.

Scanner processes run inside the Runtime Image through narrow Engine-owned,
cancellation-aware functions. They write complete workspace files, which are
then parsed incrementally. Progress carries only a message. Result submission
returns only status (`error`/`nil`); accepted counts, summaries, terminal state,
and failure RPCs are not author APIs.

The complete generated typed Results surface assumes the closed, equally
trusted LunaFox first-party Engine set. It fixes types and encoders but grants
no result authorization; a manifest allow-set also cannot grant authority.
Before a third-party publisher or any other unequal-trust Engine class is
accepted, an independent security change must define a Server-enforced result
capability policy.

Ordinary Engine code must not import SDK bootstrap, raw Context/protobuf/
gRPC, or Server/Agent internals. It must not use materialization clients,
generic binding maps, raw result type strings, or platform process/file RPCs.
The exact generated `execution_run_generated.go` adapter alone may use
`engine-go/sdk` and the raw major-2 Protocol binding; the generated typed contract and
ordinary handler remain transport-free.
