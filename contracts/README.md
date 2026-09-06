# Contracts Module

`contracts` contains shared protocol shapes, canonical value helpers, and engine
runtime helper surfaces used across the monorepo. It is not a single domain-model
module. Before renaming, moving, or expanding a package here, classify the
package role and keep executable runtime behavior out of passive contract
packages.

## Package Role Inventory

| Package | Role | Notes |
|---|---|---|
| `contracts/enginemanifest` | protocol contract | Strict `engine.v5` definition, Package v2 layout contracts, locale validation, and generic Runtime Image identity validation. Legacy root/runtime manifest DTOs are retired. |
| `contracts/enginemanifest/repositoryname` | release-tool helper | LunaFox publisher-local stable-ID and OCI repository naming helpers. Generic package validation, installation and planning must not call these helpers. |
| `contracts/enginemanifest/runtimeimage` | contract helper | Ordered generic Runtime Image candidate parsing: canonical digest refs in publisher-chosen repositories, unique locations, and one shared image digest. Remote descriptor/platform verification remains caller-owned. |
| `contracts/enginemanifest/packagecatalog` | runtime helper | Closed Engine Package v2 four-file layout loader and normalized definition registry. It does not discover legacy package roots or runtime artifacts. |
| `contracts/enginemanifest/packagemanifest` | contract helper | Strict Engine Package v2 `package.json`, archive digest and safe package-path helpers. No files list, checksums projection, or runtime bundle declaration is supported. |
| `contracts/enginemanifest/enginepackagebuild` | build/release helper | Deterministic Engine Package v2 archive writer and digest helpers. Runtime bundle assembly is retired. |
| `contracts/agentcontrol` | protocol contract helper | Strict guards for removed Agent control-plane Worker identity fields while preserving unrelated protobuf unknown fields for forward compatibility. |
| `contracts/ociartifact` | protocol contract | OCI digest-reference, artifact media-type, manifest, and package-layer descriptor validation for LunaFox engine-package distribution. It never validates package contents. |
| `contracts/ocidistribution` | release transport helper | First-party OCI download transport mappings for Cloudflare acceleration. It preserves the GHCR signature identity and permits fallback only for transient transport failures. |
| `contracts/ocisignature` | security contract helper | Shared Sigstore/Cosign keyless OCI manifest signature verifier for Server bootstrap and production installer. It validates trusted-root cryptography, transparency proof and fixed GitHub release identity, but never decodes engine package contents. |
| `contracts/enginecontract/engineexecution` | engine protocol contract | Strict engine.v5 execution-definition DTOs plus definition-backed config defaulting, presence, type, enum, range, length, and pattern validation used by Server, Engine authoring generation, and tooling. `requiredEnabled` is limited to one unconditional config-section invariant and requires `defaultEnabled:true`; it does not encode groups, conditions, or Engine runtime orchestration. Agent consumes only the saved plan and closed shared constants needed to materialize it. |
| `contracts/agentexecution` | protocol contract helper | Passive validation and deterministic binding-order helpers for the saved Agent-neutral `ResolvedEngineExecutionPlan`; it does not compile packages, config, workflow semantics, node paths, or container policy. |
| `contracts/executionartifact` | protocol contract | Closed phase-one registry mapping each typed execution input/resource role to its exact file kind, versioned content type, and record-count presence rule. |
| `contracts/engineapi/version` | canonical value helper | One-to-one Engine API major to versioned `EngineExecutionContext` package/message mapping; phase one contains only major `2` and has no `contextVersion` or `supportedContextVersions` dimension. |
| `contracts/engineapi/conformance` | conformance support | Test-only Engine API v2 protocol-direct harness. Its `testdata/protocol_direct_engine` binary imports only raw generated bindings and proves Context/UDS/reporting interoperability without creating an author SDK, Facade, or supported fourth Engine. |
| `contracts/engineapi/conformance/testdata/protocol_direct_engine` | conformance support | Compiled test fixture for the raw Engine API v2 Context/reporting path. It is invoked only by the conformance harness and is not a production package, release Engine, SDK, or scaffold. |
| `contracts/gen/lunafox/...` | generated binding | Generated protobuf bindings. Do not hand-edit generated `.pb.go` or `_grpc.pb.go` files. |
| `contracts/resourcenames` | canonical value helper | Canonical resource name format/parse helpers. |
| `contracts/results` | Server result-registry helper | Ordered, versioned result descriptors plus Server-side closed decoding, Schema-basics admission, and complex validation. It is not an Engine authoring or normalization surface. |
| `contracts/versioning` | canonical value helper | Shared semantic-version normalization and validation. |
| `contracts/agentinstall` | canonical value helper | Shared agent installation defaults used by Server-generated install scripts and installer tooling. |
| `contracts/scanworkflow` | protocol contract | Formal scan workflow definition JSON DTOs, strict decoder, and semantic validators; every Step requires the boolean `profileDefaultEnabled` orchestration metadata. |
| `contracts/scanworkflow/configuration` | canonical value helper | Scan workflow dynamic configuration object normalization and strict `configuration.steps.<stepId>` envelope validation; requests require exact Step coverage, `enabled: true` requires `engineConfig`, `enabled: false` forbids it, and callers provide known steps and own catalog/Engine Definition validation. Server Profiles are a separate all-disabled draft shape with complete Engine defaults and do not pass through request fallback. |
| `contracts/sharedstorage` | canonical value helper | Deterministic shared storage path constants and builders. |

## Boundary Rules

- Protocol contract packages own DTOs, constants, decoders, and validators. They
  must fail closed for missing required fields and unsupported versions.
- Engine API reporting errors use canonical gRPC status/details semantics and
  never carry task terminal state or a second failure RPC.
- `contracts/enginemanifest` owns strict `engine.v5` validation. A canonical
	`Execution.Target` is always supplied at execution time. Execution inputs are
	not an Engine-manifest declaration: the closed global Execution Input Registry
	contains `subdomains`, `hostPorts`, `websiteURLs`, and `endpointURLs`, and every generated
	Engine receives all four typed handles. A handle's `Path(ctx)` lazily requests
	and materializes its role; config and platform resources keep their eager,
	plan-bound contracts.
- `contracts/ociartifact` owns only OCI transport identity and descriptor
  validation. OCI manifest digests and package-layer digests are distinct;
  `contracts/enginemanifest` remains the sole owner of package-content,
  platform, and engine-version validation, including strict rejection of
  legacy payload/runtime-bundle fields and layouts.
- The active OCI APIs use `EnginePackageArtifactType`,
  `EnginePackageLayerMediaType`, `DecodeEnginePackageManifest`, and
  `ValidateEnginePackageManifest`; v1 media types and validators are absent
  and cannot be selected through compatibility fallback.
  The v2 decoder is strict about closed OCI JSON fields, trailing content,
  `schemaVersion: 2`, an explicit OCI image-manifest root media type, the exact
  OCI empty `{}` config descriptor, the exact v2 artifact/layer media types,
  and the one-positive-layer shape. Old envelopes must be republished; no v1,
  tag-based, or old-envelope compatibility fallback exists.
  Installers must use
  `ParseEnginePackageArtifactReference` and `ParseArtifactCandidates` for
  canonical artifact-manifest identity and ordered same-digest locations.
  Candidate adapters may wrap only positively classified unavailable outcomes
  in `CandidateFailure`; `IsCandidateUnavailable` fails closed for plain,
  unknown, cancellation, integrity, and local failures.
- The package v2 contract pairs strict `package.json` decoding with strict
  `engine.v5` decoding through `packagecatalog.DecodeEnginePackageDefinition`.
	Its detached `EngineDefinition` preserves Engine API major, target support,
	Registry-independent config/resource metadata, and platform-resource IDs for Server,
  catalog, and generator consumers. Agent and Engine Containers do not consume
  this authoring definition. `packagecatalog.DecodeEnginePackageLayout`
  and `LoadEnginePackageLayoutFromRoot` enforce the exact `package.json`,
  `engine.json`, `locales/en.json`, and `locales/zh.json` layout, regular
  non-executable entries, canonical safe paths, and locale keys derived from
  that same normalized Definition.
- `enginemanifest.ValidateEngineID` owns generic stable Engine ID validation.
  Package validators use `enginemanifest/runtimeimage.ParseCandidates` for
  ordered, unique same-digest Runtime Image refs in arbitrary repositories.
  Registry descriptor, media-type, digest availability and platform checks
  remain installer responsibilities; publisher, namespace and repository are
  not package-install authorization facts.
- Existing `engine.lunafox.*` identities remain stable for workflow continuity,
  but they have no special trust semantics. Catalog/UI projections show the
  package-declared `publisher` as metadata only; a duplicate display name never
  merges identities.
- `engine.json.execution.configSections` is the only engine configuration
  authoring source. Config defaults and constraints are evaluated directly from
  the normalized definition; standalone config-schema files, schema refs/paths,
  package payloads, catalog fields, and HTTP projections are not supported.
- `contracts/ocisignature` owns OCI signature referrer, Sigstore bundle and
  first-party GitHub Actions identity verification. It must not become a second
  engine package manifest or payload validator.
- Engine root manifests require a canonical `publisher` declaration and reject
  all trust, signature, permission and other unknown authorization fields.
- `contracts/results` owns the passive, ordered first-party result descriptor
  registry and the Server-side closed decode/admission boundary. Website,
  Endpoint, Directory, Screenshot, WebsiteTechnology, and Vulnerability URL
  values use the shared non-mutating observed-URL rule: accepted bytes are
  neither trimmed, parsed/rebuilt, nor percent-decoded. Website and Endpoint
  `host` values are separate assertions checked against a derived URL authority.
  The descriptor list is the single generation input for each Engine-local
  generated contract; each generated port fixes its wire type, local Go item
  type, and Schema-basics encoder before calling the `engine-go/sdk` ResultSink.
  Engines do not maintain a `contract/results.json` sidecar, and unused
  generated ports connect to no ResultSink until their first `Submit`. New
  result kinds must be registered here before Server materialization accepts
  them, but Engine business code must not import this package.
  `asset.website_technology.v1` is a closed current-only descriptor with exact
  `{url,tech}` items: its URL preserves accepted observed bytes, `tech` is an
  explicit array that may be empty, and values are preserved without trim or
  replacement after strict nonblank, NUL, UTF-8, and 100-Unicode-code-point
  validation. It does not route through `asset.website.v1` transformation.
  Descriptors and generated typed Results do not authorize an Engine or
  package; every wire request must still carry the explicit canonical
  `result_type` and pass Server scope/schema validation. No manifest result
  allow-set may substitute for those runtime boundaries.
- The Engine API v2 protocol is generated from the Context, reporting, and
	independent execution-input service protos into the independent
	`engine-go/protocol` Go package. The Context message is passive and exact for
	major 2; its config and resource partitions use closed typed entries rather
	than `Struct` or generic binding maps. Reporting remains unary and
	message-only, while `EngineExecutionInputService` is the sole lazy input
	materialization RPC. SDKs and package-local aliases must not redefine these
	protocol identities.
- Engine API compatibility is selected by `contracts/engineapi/version`:
  major `2` maps exactly to
  `lunafox.engine.execution.v2.EngineExecutionContext` (logical label
  `engine.execution-context.v2`). Runtime manifests, runtime capability lists,
  and `runtimeRef`/`runtimeSource` are not accepted by the package, catalog,
  Server plan, Agent validator, or Engine Container paths. No runtime-manifest
  decoder or runtime-bundle path exists in the active tree.
- The Engine API v2 protocol-direct conformance fixture lives only under
  `contracts/engineapi/conformance/testdata`. It strictly consumes the binary
  Context protobuf, local file bindings, the task credential file, and the
  filesystem UDS through raw generated bindings. It is test support, not a
  production client, non-Go SDK/scaffold, release Engine, or alternate protocol
  owner.
- `contracts/results` also owns the v2 batch admission helpers.
  `ValidateEncodedBatch` performs schema-neutral result-type syntax,
  JSON-object, exact-byte, item-count, and byte-limit checks without consulting
  the closed registry; `ValidateCanonicalBatch` adds Server-owned type and item
  schema validation, including phase-one IPv4-only result authorities. The
  returned `EncodedBatch` is detached and preserves item order/bytes for an
  allowed same-session replay.

### Nuclei vulnerability result

`asset.vulnerability.v1` is the closed typed result used by the first-party
`engine.lunafox.nuclei_vulnerability` Engine. The descriptor requires the
canonical `source=nuclei` identity, normalized severity (`unknown`, `info`,
`low`, `medium`, `high`, or `critical`), an optional 0--10 CVSS score, and the
complete parsed Nuclei object in `rawOutput`. Invalid JSONL source lines never
enter this contract; the Engine reports only bounded invalid-line counters.
Server result ingestion validates the complete batch before materialization and
returns the existing empty acknowledgement shape.
- `contracts/executionartifact` is the single owner of the four phase-one
  execution-artifact representation identities. Callers derive exact file kind,
  versioned content type, and required-or-absent record-count presence from the
  typed input/config-resource/platform-resource role; they must reject aliases,
  unknown versions, and values copied from another role. Authorization, source,
  path, cache, sensitivity, mount, retry, chunk, and aggregate limits remain
  outside this passive registry.
- `contracts/scanworkflow/configuration` is the source of truth for dynamic scan
  workflow configuration shape parsing. Callers must supply the selected
  workflow's known step set and keep manifest loading, effective config merge,
  normalized Engine Definition validation, and application error wrapping at their own
  boundary.
- A Profile draft may retain complete `engineConfig` beside `enabled: false` for
  editing, but a persisted or execution request must canonicalize that branch to
  `{enabled:false}`. No caller may infer a missing Step flag from Workflow,
  Profile, Engine identity, Engine section defaults, or language zero values.
- Canonical value helpers own shared parse/build/normalize invariants. Active
  boundary-facing code should use them instead of duplicating equivalent string
  construction.
- Runtime helper packages may serve sockets, read injected environment
  metadata, or manage execution lifecycle, but implementation details such as
  registries, token guards, callback maps, and stream sessions should stay
  package-private unless an OpenSpec change approves a public extension point.
- First-party runtime command/business production code under
  `extensions/engines/**` must not import `contracts/results`, Server result
  materializers, or shared result normalizers. Each Engine owns its typed item
  definitions and Schema-basics encoder in `contract/`; the exact generated
  Engine API v2 Run adapter may additionally import `engine-go/sdk` and
  `engine-go/protocol`.
  Handwritten Engine-owned contract source may directly import only
  `contracts/enginecontract/engineexecution` for execution metadata, config
  defaulting, and structural presence validation. The exact generated execution
  contract may import only its Engine-local result ports and canonical item
  definitions. Engine result-parser tests may use only their Engine-local
  contract; exact generated Run-adapter tests may use the canonical major-2
  Context binding. Other lower-level
  transport, launch-environment, generated-proto, or resource-name packages must
  stay out of Engine tests and release packages.
- Runtime launch variables, socket paths, engine identity, API major, install
  digest, workspace/materialization roots, package digests, and protocol
  credentials are injected by Agent/runtimekit. Engine manifests, config
  declarations, and runtime install environment overrides must not make those
  platform-owned inputs engine-controlled.
- Examples, scaffold output, and conformance tests verify contracts. Long-lived
  semantics belong in OpenSpec or package documentation.
- Generated bindings under `contracts/gen` are regenerated from source inputs;
  manual edits belong in the proto/generator source, not in generated Go files.

## Verification

Use `make verify-boundary-contracts` after changing package roles, generated
binding boundaries, runtime/engine boundary semantics, proto/JSON payloads, or
resource-name semantics. The guard checks this README covers every
Go package under `contracts` and that generated Go files under `contracts/gen`
carry a generated-code marker.
