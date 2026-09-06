# Shared Component Reuse Guide

This directory owns reusable production UI patterns above raw `components/ui` primitives. Route code should check this guide before composing a new local overlay, table, feedback, loading, editor, status, upload, or visualization surface.

## Decision Matrix

| Need | Use | Do not use |
| --- | --- | --- |
| Create or edit a record in a right-side panel | `FormDrawer` from `@/components/shared/form-drawer` | Page-local `SheetContent` with hand-rolled header/footer |
| Inspect a record, finding, or runtime object in a right-side panel | `DetailDrawer` from `@/components/shared/detail-drawer` | A form drawer, centered dialog, or route-local sheet shell |
| Compose a visible right-side panel header | `EdgePanelHeader` through the owning shared panel/workbench owner | Route-local copies that mix title, description, icon, close, and progress geometry |
| Show a dense diagnostic workspace with tabs/logs/tables | `DetailDrawer` first; extend it only when a repeated `WorkbenchDrawer` need is proven | One-off wide `SheetContent` shells |
| Confirm destructive action | shared feedback confirm dialog | Ad hoc `AlertDialog` copies |
| Render a production business error state | `AppErrorState` from `@/components/shared/feedback/app-error-state` | Page-local alert icon + `error.message` blocks |
| Copy text with feedback | shared copy helpers | Direct `navigator.clipboard` snippets in route UI |
| Bulk line input with validation | `BulkLineValidationInput` from `@/components/common` | Local line-numbered textarea shells |
| Business list table | shared data-table components | Page-local table toolbar/pagination/action systems |
| Search input with an inline search icon | `SearchInput` from `@/components/shared/search-input` | Page-local absolute icon + `Input` bundles |
| Captured HTTP response evidence | `ResponseEvidencePanel` from `@/components/shared/response-evidence` | Duplicated response Tabs and scroll regions in route consumers |
| Fixed-height metric strip with loading numeric values | shared metrics primitives such as `StatMetricRow` | Wrapping the whole strip in a page-wide handoff when only the value slot is pending |

## Overlay Ownership

Use shared overlay owners before raw primitives:

- `FormDrawer` owns create/edit drawer geometry: header, close affordance, scroll body, fixed footer, and the shared form drawer width tier.
- `DetailDrawer` owns read/inspect drawer geometry: right-side shell, lighter backdrop, header, close affordance, content region, and shared drawer tabs.
- Both drawer owners use shared drawer motion tokens; differences in backdrop strength must be named in `overlay-styles.ts`.
- Raw `Sheet` is allowed inside shared owners or for new owners. Route modules should not compose raw `SheetContent` unless the nearest README explains why no shared owner fits.

## When To Add A New Shared Owner

Add a new component here only when at least one of these is true:

- The same pattern is needed by more than one route.
- A page needs a new overlay width, footer model, or scroll contract.
- Foundation checks cannot distinguish a valid local exception from drift without a named owner.

Keep component-specific rules in the component folder README. Keep high-level cross-module facts in `frontend/components/ui/README.md` or OpenSpec when they become long-lived product rules.

## Business Error Ownership

- Shell-scoped business failures must render through `AppErrorState` after the page normalizes transport/runtime failures into `AppError`.
- Protected-route `401` is not a business error state. Auth/runtime must redirect to `/login/?returnTo=...` instead of rendering `AppErrorState`.
- Valid detail routes with missing resources should keep the authenticated shell and render `AppErrorState` with `resource-not-found`; only invalid routes belong to App Router `not-found.tsx`.
- Successful empty results remain empty states and must not reuse `AppErrorState` just for visual convenience.
