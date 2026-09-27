# Mock Data Contract

`dev:mock` / `dev:mock:noauth` must be hermetic: pages should not depend on a local 8080 backend unless the service is explicitly recorded as unsupported.

## Production Boundary

Production frontend builds (`NODE_ENV=production`) always use the real API and
authentication boundary. `NEXT_PUBLIC_USE_MOCK` and `NEXT_PUBLIC_SKIP_AUTH` are
development/verification switches only; the Next.js build fails closed if either
is enabled in production. Vercel and Docker/Compose release builds must not set
them.

A Vercel deployment also requires a project-level `BACKEND_URL` containing the
root HTTPS origin of the real backend. Next.js rewrites same-origin `/v1/*`
requests through that server-side value, so it must not be replaced with a
browser-facing cross-origin API URL. Vercel must not define
`NEXT_PUBLIC_BACKEND_URL`; missing, malformed, or browser-public backend
configuration fails the production build before it can deploy a frontend with
no real API.

When a real-mode page starts, it removes only registrations whose worker script
path is `/mockServiceWorker.js`. This retires browsers that previously visited a
mock deployment without unregistering unrelated application workers. The API
client waits for that cleanup before sending the first real request.

## Ownership

- Mock ownership now lives in the network-layer platform.
- The runtime platform owns `USE_MOCK`, `MSW`, scenario selection, and request interception.
- Mock bootstrap must not change the server-rendered or first-client-rendered React tree. UI providers may start the browser worker eagerly, but client request dispatch must wait in `lib/api-client` for the shared MSW startup promise before supported `/v1/*` calls leave the browser.
- Production `services/*` stay in their real HTTP form and must not branch on mock state.
- Mock state lives under `frontend/mock/data/*`, `frontend/mock/builders/*`, and `frontend/mock/scenarios/*`.
- UI components and hooks call the same service in real and mock modes; they must not branch on mock state.
- Mock update helpers should return cloned data or stable snapshots so pages can edit without mutating unrelated tests.
- Notification locale mock parity is limited to `PATCH /users/current/notificationLocale` with canonical `zh` or `en`; mock mode must not restore browser-locale bootstrap behavior.
- Runtime mock payload values use English in every locale branch. Chinese copy belongs to `frontend/messages/zh.json`, not mock data.
- Scan and Scheduled Scan handlers enforce the same explicit Workflow Step
  envelope as the Server. Mock fixtures use canonical branches (at least one
  enabled for executable requests) and never infer enablement from the mock
  Workflow or Engine catalog.
- Website Mock List/Get mirrors the real `GET /v1/targets/{target}/websites` and `GET /v1/websites/{website}` contracts, including canonical `name` and optional Screenshot summaries without image bytes. The old `websiteRelations` pseudo resource is intentionally absent.
- Website detail asset requests use the same mandatory leading scope filter as real mode: `host==` is an exact normalized observed-host filter, while `websiteUrl==` is a structured HTTP(S) origin and path-boundary filter. Mock handlers reject an `OR` or non-leading scope clause rather than widening to Target-wide data.
- Nuclei POC collection activation is intercepted at the network layer with
  the same `{enabled}`-only payload, full-catalog count semantics, empty-catalog
  no-op, and `SYNC_ALREADY_RUNNING` conflict shape as the Server; it never
  becomes a page-local mock branch.
- Nuclei POC tag options are intercepted at
  `GET /nucleiPocs/filterOptions?field=tags`. The mock validates the required
  single field parameter and returns the complete mock catalog projection with
  lowercase canonical values, per-POC deduplication, catalog-wide counts, and
  deterministic ordering. The empty scenario returns `{ results: [] }`.
- Nuclei POC sync cancellation is intercepted at
  `POST /nucleiPocSyncTasks/{uuid}:cancel`. The mock returns `CANCELLING`, the
  next task poll converges to terminal `CANCELLED`, releases the mock active
  slot, and leaves the committed source/catalog unchanged. Repeated cancel
  requests are idempotent observations of the same task.

## Coverage Rules

Mock data is a UI verification surface, not just a placeholder. It MUST cover realistic stress cases for the screens it drives so design, truncation, density, filtering, empty states, and destructive actions do not look correct only because the fixtures are unusually short or clean.

- Business-list mocks SHOULD include a mix of short and long names on the first page, not only tidy short labels.
- Table mocks SHOULD expose relationship variance on the first page when the UI renders relationship cells, such as one-to-many badges, expandable lists, or repeated ownership labels.
- Table mocks SHOULD include mixed target/status/type categories and at least one missing optional field when the production UI supports that case.
- For `targets` specifically, the first page MUST include multi-organization rows, longer organization labels, at least one longer target name, `domain` / `ip` / `cidr` variation, and at least one item without `lastScannedAt`.
- When a mock fixture is intentionally curated to exercise a UI contract, preserve that intent with a contract test under `frontend/mock/data/__tests__/*` instead of relying on comments alone.

## Scenario Standard

Scenario coverage follows a fixed baseline instead of ad hoc fake rows:

- `happy`: default business path with realistic density.
- `empty`: true zero-state and empty-table behavior.
- `stress`: long labels, large counts, and pagination pressure.
- `edge`: missing optional fields, blank descriptions, null-ish values, and awkward but valid payloads.
- `error`: handler-owned network failure path.

For list-heavy pages, the standard is not “add more rows until it feels enough”. The standard is:

1. one network-contract guard;
2. one scenario-difference guard (`empty`, `stress`, or `edge`);
3. one visual surface in Storybook when the state changes are meaningfully visible.

The current project coverage matrix lives in [docs/plans/2026-05-27-mock-coverage-matrix.md](/Users/yangyang/Desktop/lunafox-frontend-refactor/docs/plans/2026-05-27-mock-coverage-matrix.md).

Upgrade lifecycle fixtures are also scenario-backed: `happy` advances a
server Operation through stopping, updating, restarting, verification, and
success; `stress` ends in a failed verification; `edge` ends in
`needs_attention`; and `error` makes the status resource reconnectable. The
browser UI must consume these responses through the normal version service,
not branch on the selected mock scenario. The mock Operation and its polling
cursor are persisted in a versioned browser-storage fixture so a full-page
refresh exercises the same recovery path as a live server. An obsolete fixture
version is discarded instead of resuming an incompatible simulated upgrade;
`resetMockUpgradeOperation()` clears fixture state for tests.

Mock FULL responses preserve the same compatibility boundary as Server
responses: `hostActivity` appears only when the request is exactly
`view=FULL&includeHostActivity=true`. An enhanced active fixture supplies one
closed action with start and heartbeat timestamps; an enhanced terminal fixture
returns `hostActivity: null`. The mock must not add heartbeat rows to `logs`,
and it must never simulate raw Compose output, command text, paths, images,
environment values, or credentials.

## Exceptions

Temporary files that still retain inline service mock ownership must be listed in `frontend/mock/service-mock-exceptions.json` with:

- `owner`
- `reason`
- `status`
- `reviewTrigger`
- `recoveryPath`

Run `pnpm run check:service-mock-mode` before claiming mock-mode coverage.
