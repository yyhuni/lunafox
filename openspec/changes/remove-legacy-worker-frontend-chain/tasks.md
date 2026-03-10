## 1. Spec and docs
- [x] 1.1 Write proposal / design / delta spec
- [x] 1.2 Write `docs/plans/2026-03-06-worker-legacy-chain-cleanup-design.md`
- [x] 1.3 Write `docs/plans/2026-03-06-worker-legacy-chain-cleanup.md`

## 2. Cleanup implementation
- [x] 2.1 Remove legacy `/workers` frontend service and hook chain
- [x] 2.2 Remove related worker-only tests and mocks that only validate the removed chain
- [x] 2.3 Remove or migrate unsupported worker-only placeholder UI artifacts
- [x] 2.4 Update imports / exports / demo registry references affected by the cleanup

## 3. Verification
- [x] 3.1 Prove `/settings/workers/` still runs on the agent contract
- [x] 3.2 Search the repository to confirm no runtime `/workers` frontend chain remains
- [x] 3.3 Run targeted frontend tests for the workers settings surface
- [x] 3.4 Run `openspec validate remove-legacy-worker-frontend-chain --strict --no-interactive`
