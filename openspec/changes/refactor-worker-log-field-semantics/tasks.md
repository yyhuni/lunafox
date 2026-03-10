## 1. Spec and docs
- [x] 1.1 Write proposal / design / delta spec
- [x] 1.2 Write `docs/plans/2026-03-06-worker-log-field-semantics-design.md`
- [x] 1.3 Write `docs/plans/2026-03-06-worker-log-field-semantics.md`

## 2. Worker log cleanup (TDD)
- [x] 2.1 Add failing tests for remaining worker legacy log keys
- [x] 2.2 Rename worker runtime identity log keys to semantic dotted fields
- [x] 2.3 Rename queue / retry / workflow processing log keys to semantic dotted fields
- [x] 2.4 Run affected worker tests

## 3. Enforcement
- [x] 3.1 Add failing fixture coverage for worker/server `camelCase` zap keys in `scripts/ci/check-interface-naming-test.sh`
- [x] 3.2 Extend `scripts/ci/check-interface-naming.sh` to reject new worker/server `camelCase` zap keys while allowing documented OTel semantic fields
- [x] 3.3 Run naming check fixture tests and a real repository scan

## 4. Validation
- [x] 4.1 Run strict OpenSpec validation for this change
