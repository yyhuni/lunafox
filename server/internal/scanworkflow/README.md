# scanworkflow

- Dynamic configuration shape parsing (`steps -> stepId -> engineConfig`) is owned by `contracts/scanworkflow/configuration`. Callers must use it instead of reimplementing top-level `steps`, unknown step, entry shape, or `engineConfig` object checks.
- The contract parser stays pure: it must not load workflow manifests, installed Engine Definitions, repositories, or application services. Callers provide the known step set and perform catalog/Definition-backed validation at their own boundary.
- `manifest` owns release definition loading and structural validation only. Bootstrap reads the configured `WORKFLOW_DEFINITIONS_ROOT`, requires `scanWorkflows/default`, and transactionally synchronizes built-ins before Server readiness. Catalog, Profile, Scan creation, and scheduled-scan execution must not read release files at runtime.
- Workflow definitions are pure orchestration: ordered Stage/Step component IDs, stable `engineId`, and the required `profileDefaultEnabled` Profile metadata. They must not contain `engineConfig`, Engine defaults, package identity, or operation selectors.
- The Profile singleton is derived by catalog application code from the persisted workflow plus current Engine Definitions. It owns default materialization; Scan creation owns strict complete-configuration validation and never deep-merges defaults.
- The built-in `default` Workflow keeps `fingerprint_detection` and
  `url_collection` as independent Steps in the same `url_collection` Stage.
  Both consume the same finalized `websiteURLs` snapshot when explicitly
  enabled; neither consumes the other's result. Its Profile default is declared
  by the Workflow Step's `profileDefaultEnabled` field.
- The built-in `default` Workflow places `directory_scan` in its own final
  Stage after `screenshot`. Its Profile default is declared by the Workflow
  Step's `profileDefaultEnabled` field and it consumes the finalized
  `websiteURLs` product, never Screenshot output.
- The built-in `default` Workflow places `nuclei_vulnerability` in one
  independent final Stage after `directory_scan`. Its Profile outer Step is
  explicitly `enabled:false`; the inner `nuclei` config section remains
  `defaultEnabled:true` and is not an outer Workflow switch. Existing saved
  plans, custom Workflows, scheduled Workflows, and execution budgets are not
  rewritten when this new Step is introduced. Nuclei templates are selected
  at task start, so an empty enabled catalog fails as `no_enabled_templates`
  without rolling back earlier Stage evidence.
- `profileDefaultEnabled` is a hard-cut contract in development: persisted
  Workflow JSON that omits it, uses `null`, or uses another JSON type is
  rejected during repository decoding. There is no migration or read-time
  fallback; reset the development database or recreate the affected Workflow.
  Saved Scan and frozen plan records are not rewritten.
- Nuclei rollout/rollback may revert only the Engine binary, digest-qualified
  Runtime Image, Engine Package, and Workflow/Profile configuration. It must
  not delete, hide, or compensate acknowledged vulnerability findings; a
  rollback is unsuccessful if an unacknowledged batch is reported as a
  successful task.
- Installing an Engine Package only creates or replaces its current catalog
  registration. It must not write `WORKFLOW_DEFINITIONS_ROOT`, infer stages or
  steps from package metadata, or synthesize a workflow. An installed Engine is
  executable only when a persisted workflow explicitly references its stable
  Engine ID and passes exact-package validation.
