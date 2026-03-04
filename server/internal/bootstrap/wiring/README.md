# Wiring Conventions (Module Adapter Layer)

This document standardizes naming and organization for module wiring under `server/internal/bootstrap/wiring/*` to prevent style drift over time.

## Goals
- Standardize exported factory naming for easier global search and refactoring.
- Standardize adapter type naming to reduce cognitive load.
- Lock interface boundaries with compile-time assertions so breakages fail fast during refactors.

## Directory Template
Each module subdirectory (for example `scan/` or `catalog/`) should include:

- `exports.go`: public wiring entry points for the module.
- `wiring_<module>_*_adapter.go`: concrete adapter implementations.
- `wiring_<module>_adapter_assertions.go`: interface assertions (strongly recommended).
- App-level service assembly should be defined in `exports.go` by default.
- `wiring_<module>_*_service.go`: optional, only when service assembly is complex and needs decomposition.

## Naming Rules

### 1) Exported factories (`exports.go`)
- Adapter factories: `New<Module><Role>Adapter`
  - Example: `NewScanTaskStoreAdapter`
  - Example: `NewScanTargetLookupAdapter`
- Application service factories: `New<Module>ApplicationService`
  - Example: `NewScanLogApplicationService`
  - Example: `NewWorkerApplicationService`

> Default placement: keep `New<Module>ApplicationService` in `exports.go`; extract to `wiring_<module>_*_service.go` only when assembly logic is non-trivial.

> Constraint: exported adapter factories should return application/domain interfaces instead of concrete structs whenever possible.

### 2) Private constructors
- Pattern: `new<Module><Role>Adapter`
  - Example: `newScanTaskRuntimeStoreAdapter`
  - Example: `newScanLogScanLookupAdapter`

### 3) Adapter types
- Pattern: `<module><role>Adapter`
  - Example: `scanTaskStoreAdapter`
  - Example: `catalogTargetStoreAdapter`

## Interface Assertion Rules
Each module should define `wiring_<module>_adapter_assertions.go` with compile-time assertions in the standard form:

```go
var _ someapp.SomePort = (*someAdapter)(nil)
```

Recommended minimum coverage:
- All interfaces returned by `exports.go`.
- Composite Query/Command interfaces implemented by a shared adapter.

## Minimum Checklist for New Wiring Modules
- [ ] `exports.go` exists.
- [ ] Exported factory names follow `New<Module><Role>Adapter` / `New<Module>ApplicationService`.
- [ ] Adapter naming follows `<module><role>Adapter`.
- [ ] `wiring_<module>_adapter_assertions.go` exists.
- [ ] `internal/bootstrap/wiring.go` injects only through exported factory functions.

## Anti-patterns (Avoid)
- `NewApplicationService` (missing module prefix)
- `newCommandStore` (missing module prefix)
- `taskRuntimeScanStoreAdapter` (module/role word order inconsistency)

## Notes
This convention has lower priority than system-level or repo-level instructions. If you must deviate, add a local README in the target module directory explaining why.
