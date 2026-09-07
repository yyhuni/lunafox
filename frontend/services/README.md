# Frontend Service Contract

`frontend/services/*` is the transport layer in the frontend runtime chain:

`app / components -> hooks -> services -> lib/api-client -> backend`

For overall placement rules, read `frontend/README.md` first. This file defines what the service layer may and may not own.

## Services Own

- HTTP calls
- backend route and payload ownership
- blob export requests
- backend-owned download URL builders
- response shaping that is still transport-adjacent
- strict parsing of submitted Workflow configuration objects; malformed or
  non-object configuration must fail instead of being replaced with `{}`.
- strict Scan and Scheduled Scan `inputSource` parsing: the only supported
  values are `scanSnapshot` and `targetInventory`; missing, blank, or unknown
  request and response values fail at the service boundary and never receive a
  client fallback.
- Scan and Scheduled Scan request producers must preserve the canonical
  `configuration.steps` envelope: every Workflow Step is explicit, disabled
  branches contain only `enabled:false`, and enabled branches carry the
  complete Engine object supplied by the Profile/form. Services never fetch a
  Profile or merge catalog defaults while constructing a request.

## Services Must Not Own

- React Query `useQuery` or `useMutation`
- invalidation scopes
- polling
- warmup or prefetch timing
- toast lifecycle
- page or component state
- local compatibility aliases that only mirror another method name

If UI needs one of those behaviors, add or extend a hook under `frontend/hooks/*`.

## Runtime Boundary

- `frontend/app/*` and `frontend/components/*` must not import production `services/*` for runtime data access.
- `frontend/hooks/*` owns runtime orchestration: React Query state, mutations, invalidation, imperative loaders, polling, and export actions.
- UI modules that only need shared types must import them from `frontend/types/*`, not from service files.
- Services may import `@/lib/api-client`, transport helpers, and `frontend/types/*`, but not `hooks/*` or production `mock/*`.

## Scan Execution Input Source

Quick and Batch Scan payloads carry one required top-level `inputSource`; Batch
`requests[]` items never receive per-target or per-organization source fields.
Callers that need mixed sources issue separate requests. Scheduled Scan Create
always sends its persisted source; Update sends it and adds `inputSource` to
the AIP update mask only when the caller explicitly changes it. Scan response
adaptation keeps the persisted source on list/detail and creation-result
projections without inferring `scanSnapshot`.

## Scan History Batch Stop

`batchStopScans(ids)` owns the synchronous `POST /scans:batchStop` transport
boundary. It accepts 1–100 unique positive safe-integer Scan IDs, converts them
to canonical `scans/{id}` resource names, and rejects malformed input before
issuing a request. The response is strict: `stoppedCount`, `skippedCount`,
and `revokedTaskCount` must all be non-negative safe integers with no unknown
fields. React Query invalidation and user feedback belong to
`useBatchStopScans`; components must not call this service directly.

## Engine Catalog Contract

Engine Catalog v5 projections expose Engine API compatibility, target support,
platform resources, configuration sections, and localized metadata. They MUST
NOT contain `execution.inputs`: the complete non-sensitive execution-input
Registry is generated uniformly for every Engine and is not Catalog membership
metadata. The adapter rejects a stale `inputs` field rather than silently
dropping or defaulting it.

## AIP Boundary Adaptation

Backend HTTP/JSON boundaries follow the repository's Google AIP precedence rules. Frontend pages and components must not adapt directly to backend AIP DTOs.

- Services own AIP request payloads, route names, resource names, custom methods, and response DTO adaptation.
- Hooks and UI consume frontend-facing types from `frontend/types/*`, not raw backend DTOs.
- AIP resource names such as `targets/7`, `organizations/3`, `scanWorkflows/subdomain_discovery`, and `scheduledScans/4` should be constructed and parsed with shared helpers such as `frontend/lib/resource-name.ts`.
- Response fields that are AIP transport concerns should be normalized before leaving the service layer. For example, backend `name: "targets/7"` plus `displayName: "example.com"` should become a UI `Target` with `name: "example.com"` and optional `resourceName: "targets/7"`.
- When a backend DTO uses AIP `name` for a resource name and exposes a separate display field such as `displayName`, `dnsName`, `url`, or `host`, services MUST keep the resource name in `resourceName` or another explicit transport field and expose the human-readable value through the UI-facing field that components already render. Do not let components decide whether `name` means resource identity or display text.
- New or migrated AIP list/detail services that return user-visible rows MUST add contract coverage for this mapping when the backend DTO contains both a resource `name` and a display field. The test should assert both the AIP path/payload and the UI-facing response shape, including `resourceName` when retained.
- If the backend does not expose an AIP endpoint for an action, services must fail fast with an explicit error instead of silently falling back to a legacy route or pretending success.
- New or migrated services must add focused service contract tests that assert the AIP path, payload shape, resource-name conversion, and UI-facing response shape.

## Mock Mode

- A service that calls `@/lib/api-client`, `axios`, or `fetch` must stay in its real HTTP form.
- `dev:mock` and `dev:mock:noauth` must not accidentally hit the real backend for supported page data; the network-layer mock contract owns interception.
- Production services must not import `@/mock` or branch on `USE_MOCK`.
- New mockable services require network-layer handlers and scenario coverage instead of service-local mock branches.
- Unsupported backend-dependent actions must fail explicitly or be recorded in `frontend/mock/service-mock-exceptions.json`; they must not silently fall through to the real backend.

The notification locale service accepts only the canonical page locale (`zh`
or `en`) through the current-user locale resource. It has no browser-locale
bootstrap endpoint; locale inference belongs to the page i18n layer. Within one
browser session, locale writes are serialized and an explicit page switch blocks
an older shell sync until the newly selected page locale is active.

## Cleanup Rule

When a service API name changes, migrate callers and tests in the same change and delete the old alias immediately. Do not keep parallel names such as `batchDelete*` and `bulkDelete*` without a product-level reason.

`scan-workflow.service.ts` owns only Create/List/Get/Update and
`GET /scanWorkflows/{scanWorkflow}/profile`. Workflow List/Get are pure
orchestration resources; no service may restore a `scanWorkflowProfiles`
collection, workflow `configuration`, or a compatibility alias for either.

## Suggested Verification

- `frontend/__tests__/service-boundary.contract.test.ts`
- `services/__tests__/<name>.service.contract.test.ts`
- focused hook tests for lifecycle and invalidation behavior
- browser smoke or interaction coverage that exercises the network-layer mock contract

## Nuclei POC Source Sync

`nuclei-poc.service.ts` is the only frontend transport boundary for the
Nuclei POC source, sync-task, catalog, detail, and `isEnabled` update APIs. It
uses canonical `nucleiPocSources/*`, `nucleiPocSyncTasks/*`, and
`nucleiPocs/{templateId}` resource names and rejects malformed names before an
HTTP request. The service must not fall back to the retired `/nuclei/repos` or
local preview transport. List responses intentionally omit YAML; detail
requests are the only YAML payload.

The same service owns `GET /nucleiPocs/filterOptions` for the tag facet. It
sends exactly one `field=tags` query parameter and strictly parses
`results[{value,label,count}]`; unsupported fields or malformed option data
fail at the transport boundary. The endpoint is a complete committed-catalog
projection, so it is independent of list pagination and active list filters.

For sync creation conflicts, the service parses only the canonical
`google.rpc.ErrorInfo` detail with reason `SYNC_ALREADY_RUNNING` and validates
`metadata.task` through the existing task-name parser. Invalid or missing
metadata returns no task name; callers must show localized recovery copy rather
than rendering the raw server message.

Collection activation uses `POST /nucleiPocs:setActivation` with exactly one
explicit boolean field, `enabled`. The service rejects extra request fields and
strictly validates the response target and non-negative safe-integer
`affectedCount`; pagination, filters, and source metadata are not part of this
transport contract.
