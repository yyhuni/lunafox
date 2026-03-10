# Design: Worker log field semantics

## Context
The current repository standard is already clear:
- API / JSON contracts use `camelCase`
- Gin context values are accessed through middleware helpers
- application structured logs use semantic field names, preferably dotted namespaces
- Loki / Prometheus labels keep `snake_case` only when they are external provider-facing label contracts

After the previous naming cleanup, the remaining inconsistency is not in API contracts or labels. It is in worker application logs, where a small group of files still emits `camelCase` zap keys. Existing server-side semantic log fields stay unchanged in this focused cleanup.

## Decision
Adopt a strict semantic-field cleanup for the remaining worker domain logs:

1. Replace remaining worker `camelCase` zap keys with semantic dotted names
2. Keep OpenTelemetry-aligned HTTP fields unchanged, even when they contain underscores such as `http.response.status_code`
3. Keep provider-facing Loki / Prometheus labels unchanged and out of this migration
4. Extend CI to reject newly introduced `camelCase` zap keys in non-test Go source, with an explicit allowlist only for documented external/generated exceptions

## Mapping strategy
### Runtime identity fields
- `taskId` -> `task.id`
- `scanId` -> `scan.id`
- `targetId` -> `target.id`
- `targetName` -> `target.name`
- `targetType` -> `target.type`

### Queue / retry / batch fields
- `queueLength` -> `queue.length`
- `maxQueuedItems` -> `queue.max_items`
- `retryCount` -> `retry.count`
- `droppedCount` -> `drop.count`
- `droppedTotal` -> `drop.total`
- `statusCode` -> `http.response.status_code`
- `totalSent` -> `send.item_total`
- `totalBatches` -> `send.batch_total`

### Workflow processing fields
- `expansionRatio` -> `expansion.ratio`
- `originalCount` -> `original.count`
- `sampleCount` -> `sample.count`
- `sampleSize` -> `sample.size`
- `inputFiles` -> `input.file_count`
- `maxBytes` -> `buffer.max_bytes`
- `exitCode` -> `process.exit_code`

## Why not snake_case log bodies
This change intentionally does not move application log fields to `snake_case`. Loki collects stdout, but those collected log bodies are still application-level structured logs, not Loki label names. The standard path remains:
- application log body: semantic dotted fields
- collection-time labels: provider-compatible `snake_case` only where needed

## Scope
### In scope
- Remaining worker/runtime `camelCase` zap keys
- Targeted tests proving the new log keys
- CI guardrail for future worker/server `camelCase` zap regressions

### Out of scope
- Rewriting HTTP semantic convention fields
- Renaming Loki / Prometheus labels
- Changing JSON, route params, SQL, or generated code in this change

## Verification strategy
- Add or update targeted tests around affected log emitters / helper output
- Run worker tests covering touched files
- Run the interface naming script and its test fixture suite
- Run strict OpenSpec validation for this focused change
