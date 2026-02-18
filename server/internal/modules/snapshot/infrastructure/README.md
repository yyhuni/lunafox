# Snapshot Infrastructure

This directory contains concrete infrastructure implementations for the `snapshot` module's application ports.

## Scope
- Implementations of interfaces defined in `server/internal/modules/snapshot/application`.
- Technical adapters (e.g. codec/serialization helpers) used by application services.

## Non-goals
- Do not place dependency wiring/composition logic here.
- Do not place HTTP handler, router, or request/response mapping code here.
- Do not place domain business rules here.

## Wiring Rule
All objects in this directory should be instantiated and injected from bootstrap wiring (for example: `server/internal/bootstrap/wiring/snapshot`).

## Current Implementations
- `vulnerability_raw_output_codec.go`: encodes vulnerability raw output payloads to `datatypes.JSON`.
