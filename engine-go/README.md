# Engine Go SDK

`engine-go` owns the production Engine API v2 protocol bindings and the single
SDK execution lifecycle. It is a platform boundary, not an Engine business
library, package loader, or control-plane client.

## Generated Diagnostics

The generated Facade obtains the SDK-owned diagnostic collector and records
only common result-delivery boundaries: typed items received, successful JSON
encodes, logical batch submissions, and confirmed gRPC acknowledgements. State
is fixed per execution and bounded by the canonical result-type limit. It never
keeps per-item lists, result payload copies, raw errors, or high-cardinality
metric labels.

`Run` first establishes the mandatory matching-revision diagnostics session on
the existing authenticated UDS. A missing or mismatched confirmation is a
bootstrap failure before handler code can use reporting or input APIs. The
single terminal snapshot upload is best effort after handler completion; a
diagnostic transport failure never replaces the handler result, rolls back an
acknowledged batch, or changes task success.

Only generated code uses `Reporter.GeneratedDiagnostics`. Handwritten Engines
remain limited to the existing typed Context, Progress, and Results surfaces and
must not import diagnostic requests, clients, stages, or error classifications.
