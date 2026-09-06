# Hook Layer Contract

`frontend/hooks` owns runtime data orchestration between UI modules and transport services.

If UI needs server-backed state, the default answer is "add or extend a hook", not "import a service from the component".

## Hooks Own

- React Query `useQuery` and `useMutation` wiring
- query keys
- invalidation scopes
- polling and refetch cadence
- prefetch and warmup flows
- export actions
- imperative loaders such as `useLoadScanDetail`
- toast timing that belongs to mutation lifecycle

## Hooks Do Not Own

- page layout or JSX-heavy rendering
- raw `api-client` configuration
- mock branching inside production runtime code
- shared DTO definitions that are reused outside the hook layer

Notification locale synchronization accepts the already-resolved page locale
from `useLocale()` and sends it through the notification locale service. Hooks
must not independently read `navigator.language`, cookies, or mock state.

`useNotificationSSE` treats every received SSE byte as connection activity but
only `refresh` events invalidate notification queries. A 45-second no-data
watchdog aborts a half-open read and reuses the bounded reconnect backoff; every
successful connection refetches the durable inbox to recover missed hints.
Each connection establishes an inbox-name baseline without Toasts. Later
refreshes compare the authoritative inbox and show one shared error Toast for
each newly observed high-priority `scan-failed`, using the persisted title and
message. Reconnects reset the baseline, so historical notifications are never
replayed.

## File Shapes

- Simple domains may stay in one file such as `use-search.ts`.
- Larger domains should split keys, queries, and mutations, for example `use-scans/{keys,queries,mutations}.ts`.
- Reusable cross-domain helpers belong in `hooks/_shared/` only when the pattern is repeated and naming stays domain-neutral. `use-page-refresh-timestamp.ts` owns the client page mount timestamp and manual refresh completion timestamp shared by page-level refresh coordinators; it does not own query keys or requests.
- `use-session-renewal.ts` owns the authenticated protected-shell timer and visibility/online wake-ups. JWT parsing, token persistence, single-flight renewal, request-time freshness, and terminal login redirect remain in `lib/api-client`; the hook must not duplicate those transport rules.

## Authoring Rules

- Expose hook APIs in the vocabulary the UI needs: `useRecentVulnerabilities`, `useBulkDeleteScans`, `usePrefetchOverviewData`.
- Keep transport details inside `services/*`. Hooks should call service functions, not rebuild URLs or duplicate request shaping.
- If a hook needs a shared type that UI also imports, source it from `types/*`, not from a service file.
- If a UI flow needs preloading or imperative fetches, add a hook entrypoint instead of hiding a direct service import inside a component state file.

## Remote Mutation Feedback

Remote command mutations own their toast lifecycle in the hook layer. If a production mutation sends a server-backed create, update, delete, toggle, link, unlink, sync, stop, or similar command and reports success or error with toast feedback, it MUST also expose immediate pending feedback.

- Prefer `useResourceMutation` with `loadingToast` for pending feedback.
- `loadingToast.id` is required. Use a domain-qualified stable id per operation, such as `delete-scan-${id}` or `update-target-${id}`, so repeated transitions update the same operation without colliding with another resource domain.
- `useResourceMutation` automatically scopes `onSuccess`, `onError`, and `onSettled` toast methods to the active loading id. Omit the terminal toast id for the normal same-operation path; pass a different explicit id only when the feedback is genuinely independent.
- Keep loading active until shared invalidation and callback-owned recovery work completes, then replace it in place with success, warning, or error. The shared owner dismisses it only when the operation intentionally emits no terminal feedback, is cancelled, or fails before terminal ownership is established.
- Feature hooks MUST NOT call `toast.loading` or `toast.dismiss` manually. Extend the shared mutation owner if a repeated lifecycle cannot be represented by `loadingToast`.
- Components MUST NOT create terminal feedback for a remote command after `mutateAsync` when the hook owns its pending state. The hook owns pending and terminal feedback; the component owns local validation, dialog closure, selection reset, and other view state.
- Distinct concurrent operations use distinct ids and may appear in the global collapsed stack. Never preserve loading as a second historical toast underneath its own terminal state.
- Local validation, copy-to-clipboard, route query, and export/download feedback do not need mutation loading toast unless they are implemented as remote commands with user-visible success/error toast semantics.

## Testing Guidance

- Query hooks should keep query-key and select-shape coverage close to the hook.
- Mutation hooks should keep success toast, error fallback, and invalidation coverage close to the hook.
- Mutation hooks with user-visible success/error toast feedback should also satisfy `hooks/__tests__/remote-mutation-loading-feedback.contract.test.ts`, which guards the loading-toast lifecycle against drift.
- When a component stops importing a service directly, update the nearest contract test so it asserts the hook boundary instead of old service markers.

For scan workflows, list queries are shaped by backend `pageSize`, `pageToken`,
and `filter`; details and the parent-scoped Profile have separate cache keys.
Create invalidates the workflow collection, while Update refreshes both detail
and list state. Do not reintroduce a Profile collection hook or a compatibility
detail/profile alias.

## Nuclei POC Source Sync

`use-nuclei-pocs.ts` owns Nuclei source/catalog query keys, the dedicated
complete-catalog tag filter-options query, the sync mutation, one-second task
polling, and cache invalidation. Task polling is enabled while
the catalog page is mounted and owns a canonical non-terminal task; dialog
visibility controls presentation only. Closing the dialog never sends
cancellation, and reopening resumes the same task query. A successful task
invalidates the current source, catalog lists, tag options, and detail
projections exactly once. The hook exposes typed conflict parsing helpers so the page can adopt the
canonical task from `SYNC_ALREADY_RUNNING` without matching raw messages. The
persisted `isEnabled` mutation owns optimistic rollback, pending protection,
invalidation, and mutation feedback.

`useSetNucleiPocActivation` owns the full-catalog `setActivation` mutation. A
confirmed success invalidates every list and detail projection while retaining
the source query and reports the server's `affectedCount` (including a zero
count no-op). Definitive API failures preserve the current catalog cache;
transport-uncertain failures refresh list/detail projections once so the UI can
observe the eventual server state, without automatic retry.
