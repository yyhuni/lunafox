# Runtime gRPC Foundation

## Metadata Authentication (Phase 1)

Phase-1 runtime authentication keeps existing token semantics and maps them into gRPC metadata:

- `x-agent-key`: agent identity token on `agent <-> server` runtime control stream.
- `x-worker-token`: service-to-service worker authorization token used by agent while proxying to server.
- `x-task-token`: per-task worker token on local `worker <-> agent` UDS requests.

No mTLS/SPIFFE is required in phase-1.

## Error Mapping Policy

Server runtime auth layer maps domain/auth failures to gRPC status codes:

- invalid agent/worker/task token -> `codes.Unauthenticated`
- task scope mismatch -> `codes.PermissionDenied`
- retired legacy runtime path access -> `codes.Unimplemented`
- unspecified internal failures -> `codes.Internal`

This mapping is implemented in:

- `server/internal/grpc/runtime/auth/errors.go`

## Release Runbook

发布窗口、回滚、演练、冻结门禁、以及 `443` 统一入口下的 HTTP/gRPC 转发验证，见：

- `docs/grpc-runtime-cutover-runbook.md`
