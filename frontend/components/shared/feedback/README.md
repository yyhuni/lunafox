# Shared Feedback Rules

`frontend/components/shared/feedback` owns reusable operator-facing feedback patterns beyond raw UI primitives.

## Confirm Dialog

- `ConfirmDialog` owns standard confirmation layout, cancellation, pending state, and destructive confirmation styling.
- Destructive confirmations MUST pass `variant="destructive"` and localized `confirmText`; use `processingText` when the operation has domain-specific pending copy.
- Route components own the title, description, open state, and deletion semantics. A mock mutation and a remote mutation may share the dialog without implying the same persistence behavior.

## Business Error State

- `AppErrorState` is the shared owner for production business failures such as `resource-not-found`, `permission-denied`, `rate-limited`, `service-unavailable`, `network-error`, and `unexpected-error`.
- Pages must pass normalized `AppError` objects into `AppErrorState`; they must not pass raw `AxiosError`, transport status strings, or direct `error.message` output.
- Use `variant="page"` when a query failure replaces the primary page/content region. Use `variant="section"` when the surrounding page remains usable and one query-owned section fails; both variants keep the same semantic copy and recovery rules.
- Detail-resource queries keep the default `normalizeError(error)` 404 mapping. Collection and workspace queries must use `normalizeError(error, { notFoundKind: "unexpected-error" })` when a transport 404 does not prove that a specific business resource is missing.
- Use `title` and `description` overrides only when the page must name a specific missing business resource with approved copy.
- Recoverable failures should pass `onRetry`.
- Non-recoverable detail failures should pass a safe navigation `actionHref`, typically the owning list route or overview route.
- `AppErrorState` is for shell-scoped business failures only. Framework 404s and route render crashes belong to `app/not-found.tsx`, `app/error.tsx`, or `app/global-error.tsx`.
- `business-error-state-adoption.contract.test.ts` owns the bounded production query-owner inventory. Add a new page/query owner there when it adopts `AppErrorState`; do not broaden the contract to form fields, mutation toasts, stream diagnostics, or framework error files.
