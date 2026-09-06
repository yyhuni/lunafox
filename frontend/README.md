# Frontend Runtime Boundary

`frontend/` follows one runtime ownership chain for production data access:

`app / components -> hooks -> services -> lib/api-client -> backend`

If a change breaks this chain, it needs a very explicit reason and a local README or contract test that explains why.

## Directory Ownership

| Directory | Owns | Must not own |
| --- | --- | --- |
| `app/` | route composition, route params, layout wiring, route-level redirects, page shells | production HTTP calls, React Query orchestration details |
| `components/` | rendering, local UI state, view-state composition, event wiring into hooks | production service imports for runtime data access |
| `hooks/` | React Query queries, mutations, invalidation, polling, prefetch, export actions, imperative loaders | JSX-heavy page rendering, raw transport setup |
| `services/` | HTTP transport, blob exports, request/response shaping, backend URL ownership | toasts, invalidation, polling, prefetch timing, component state |
| `lib/api-client.ts` | shared HTTP client, interceptors, auth headers, low-level transport behavior | business-domain query logic |
| `types/` | UI-safe shared DTOs and transport-adjacent types reused outside one service file | hook state machines or backend call ownership |
| `mock/` | network-layer mock handlers and scenarios | service-local mock branches in production services |

## Placement Decision Tree

When adding code, decide in this order:

1. If it renders UI or only coordinates local component state, put it in `app/` or `components/`.
2. If it fetches, mutates, polls, preloads, exports, or invalidates server data, put it in `hooks/`.
3. If it only knows how to talk to one backend endpoint or build one backend-owned export URL, put it in `services/`.
4. If a type is needed by UI and service code, move it to `types/` instead of importing a service file just for types.
5. If the need is mock support, add or extend network-layer handlers under `mock/`, not `services/`.

## Import Rules

- `app/*` and `components/*` may import `hooks/*`, `types/*`, UI helpers, and shared presentation primitives.
- `app/*` and `components/*` must not import production `services/*` for runtime data access.
- `hooks/*` may import `services/*`, `types/*`, React Query helpers, and shared mutation/query utilities.
- `services/*` may import `lib/api-client`, transport helpers, and `types/*`.
- `services/*` must not import `hooks/*`, `components/*`, or production `mock/*`.

## Structural Rules

- A component that needs an imperative action still goes through a hook. Examples: `useLoadScanDetail`, `usePrefetchOverviewData`, `useExportAssetSearch`.
- Route or view-state files such as `*-state.ts` are still UI layer. They may coordinate hooks, but they must not become hidden service adapters.
- Query keys stay with the hook layer. Complex domains may split into `hooks/use-<domain>/{keys,queries,mutations}.ts`.
- Service methods should expose the final API shape once. Do not keep rename-only aliases such as `batchDeleteX` and `bulkDeleteX` in parallel.

## No Compatibility Layer Rule

When a boundary migration changes an API name or ownership:

- move the caller in the same change
- update the nearby tests in the same change
- delete the legacy alias in the same change

Do not leave temporary pass-through methods only to protect old imports. The boundary should describe the current architecture, not a transition state.

## Recommended Verification

- `frontend/__tests__/service-boundary.contract.test.ts`
- focused hook mutation/query tests near the changed domain
- route or component contract tests when a page-local boundary changes

See also:

- `frontend/hooks/README.md`
- `frontend/services/README.md`
- `frontend/components/ui/README.md`
