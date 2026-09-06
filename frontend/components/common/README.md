# Common Components

Shared components in this directory own reusable production UI patterns that sit above raw `components/ui` primitives.

## Page Header

`PageHeader` owns the shared title, code, action, and description rhythm for route-level pages. Its optional `middle` slot creates the wide-screen three-column command layout used by `/overview/`: title/code, secondary middle context, and commands share the header's bottom edge, while the middle context remains horizontally centered within the command row. Do not offset a page-local heading or action with custom margins to compensate for this layout.

## Page Refresh Status

`PageRefreshStatusButton` owns the compact page-level manual refresh
presentation: ghost button treatment, semantic refresh icon, shared minimum
pending feedback, accessible busy state, unavailable state, and the fixed local
`YYYY/MM/DD HH:mm:ss` timestamp. The page owner records the first client mount
time and replaces it after each declared manual refresh attempt settles. Use
`display="icon-only"` only for a responsive duplicate of the same page command,
such as the narrow Overview header entry.

The component does not own query keys or decide what a page refreshes. Route or
feature owners pass `isRefreshing`, `lastRefreshedAt`, and `onRefresh`. The
timestamp means the current client page lifecycle time, or the completion time
of the page's latest declared manual refresh attempt; it must not be sourced
from one panel's backend update time, an Agent heartbeat, or an automatic
polling cycle.

## Bulk line validation input

Use `BulkLineValidationInput` for dialogs that accept one item per line and need line-numbered validation feedback. Feature modules own parsing and validation rules; the shared component owns the editor shell, helper text rhythm, empty/success/error summaries, line highlights, badges, and collapsible issue list. `showSuccessSummary` defaults to `true`; a workbench that already owns one compact persistent validation summary may set it to `false`, while retaining the shared error panel and line-level feedback. `showEmptySummary` also defaults to `true`; a loading owner may set it to `false` when its skeleton overlays the editor content and must preserve the resolved editor geometry without showing an empty-state message.

`LineNumberedTextarea` keeps the native textarea as the scroll and editing owner. Its gutter and validation overlay virtualize presentation rows for large logical line counts, so callers must continue to pass the logical line count and line-level highlights without creating route-local row lists. Use `countLineNumberedTextareaLines` when a caller needs a display count from raw text; validators remain responsible for parsing records and enforcing their backend-aligned limits.

Bulk validation editors use the shared responsive line-numbered viewport by default so drawer and dialog forms keep the primary input usable across mobile, desktop, and tall screens. Pass `viewportClassName` only when a surrounding owner already controls the editor height, such as a full-page flex editor.

Do not create page-local copies of this pattern for target, URL, subdomain, endpoint, website, directory, blacklist, or import forms. If a new issue type or density is needed, extend the shared component contract and add a focused contract test for the migrated dialog.

### Batch limits

Each validator that feeds `BulkLineValidationInput` declares a `MAX_*_BATCH_LINES` constant aligned with the backend contract (5,000 per batch). The shared component does not enforce the limit itself; the owning validator/dialog checks the count and surfaces `too_many_lines` errors.

| Validator | Constant | Value |
|---|---|---|
| `frontend/lib/target-validator.ts` | `MAX_TARGET_BATCH_LINES` | 5,000 |
| `frontend/lib/subdomain-validator.ts` | `MAX_SUBDOMAIN_BATCH_LINES` | 5,000 |
| `frontend/lib/url-validator.ts` | `MAX_URL_BATCH_LINES` | 5,000 |
| `frontend/lib/domain-validator.ts` | `MAX_DOMAIN_BATCH_LINES` | 5,000 |
| `frontend/lib/endpoint-validator.ts` | `MAX_ENDPOINT_BATCH_LINES` | 5,000 |

Backend DTO struct tags enforce the same limit (`max=5000`) on all batch create/delete/update/link endpoints. See `server/internal/modules/*/dto/README.md` for per-module details.
