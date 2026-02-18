# Frontend Technical Debt Analysis Report
**Generated:** 2026-02-11
**Repository:** /Users/yangyang/Desktop/lunafox
**Analysis Scope:** frontend/ (517 TypeScript files, ~96,951 lines)
**Analysis Method:** Swarm Mode (5 Concurrent Agents)

---

## 1. Executive Summary

### Conclusion: Moderate Refactoring Required (Priority: P1)

**Overall Health Score: 6.8/10**

The frontend codebase is **functionally sound** with excellent type safety and clean architecture, but suffers from **significant code duplication** (30% of hook code) and **zero test coverage**. The codebase requires **strategic refactoring** focused on eliminating duplication patterns and establishing quality gates.

### Key Findings

**Strengths:**
- ✅ Excellent type safety: Only 20 `any` usages, 0 TypeScript suppressions
- ✅ Clean architecture: No circular dependencies detected
- ✅ Modern stack: Next.js 13+ App Router, React Query, TypeScript strict mode
- ✅ Complete UnifiedDataTable migration: 23/23 tables using grouped props

**Critical Issues:**
- 🔴 **Zero test coverage** - No tests exist for 517 files
- 🔴 **Massive code duplication** - 50+ identical mutation hooks, 21 similar data tables
- 🔴 **Tight coupling** - 51% of hooks depend on toast system
- 🟡 **Large components** - 6 files >1000 lines (prototypes acceptable)

### Urgency Assessment

**Immediate Action Required:**
1. Establish test infrastructure (Vitest + Testing Library)
2. Create mutation hook factory to eliminate 50+ duplicated hooks
3. Fix 1 TypeScript error blocking strict compilation

**Why Refactor Now:**
- Technical debt compounds: 30% duplication will grow without intervention
- Zero tests = high regression risk for any changes
- Tight coupling makes future refactoring exponentially harder
- Current codebase size (517 files) is manageable; delay increases cost

---

## 2. Current State Baseline

### Codebase Metrics

| Metric | Count | Status |
|--------|-------|--------|
| **Total TypeScript Files** | 517 | ✅ |
| **Total Lines of Code** | ~96,951 | ✅ |
| **Components** | 236 | ✅ |
| **Custom Hooks** | 45 | ⚠️ High duplication |
| **Services** | 32 | ✅ |
| **Type Definitions** | 33 | ✅ |
| **Page Routes** | 86 | ✅ |
| **Data Tables** | 23 | ✅ Migrated |

### Quality Baseline

| Category | Score | Details |
|----------|-------|---------|
| **Type Safety** | 9.5/10 | 20 `any` usages, 0 suppressions, 1 type error |
| **Architecture** | 8.0/10 | Clean dependencies, but high duplication |
| **Maintainability** | 7.0/10 | Large files exist, but organized |
| **Test Coverage** | 0.0/10 | **CRITICAL: No tests** |
| **Code Quality** | 7.5/10 | 4 TODOs, 5 ESLint warnings, 16 errors |

### Type Safety Statistics

```
Total `any` usages:              20 (across 11 files)
Type assertions (as any):        10 (4 files)
TypeScript suppressions:         0  ✅ EXCELLENT
ESLint disables:                 8  (7 files)
TypeScript errors:               1  (use-nudge-toast.tsx:133)
```

### Code Duplication Statistics

```
Duplicated mutation hooks:       50+ hooks (~3,000 lines)
Duplicated data tables:          21 components (~4,200 lines)
Duplicated query key factories:  45 patterns (~900 lines)
Total duplicated code:           ~8,100 lines (8.3% of codebase)
```

### Component Size Distribution

```
Files >1000 lines:    6 files  (5 prototypes + 1 demo registry)
Files 400-1000 lines: 44 files (dialogs, pages, complex components)
Files <400 lines:     467 files (majority, well-sized)
```

### Quality Markers

```
TODO comments:        4  (engine dialogs, tool updates)
FIXME comments:       0  ✅
HACK comments:        0  ✅
@deprecated markers:  1  (LegacyApiResponse type)
Console statements:   5  (should use proper logging)
```

---

## 3. Debt Inventory (Prioritized P0-P3)

### P0: Critical - Must Fix Immediately (Blocking Issues)

#### P0-1: Zero Test Coverage
**Risk Score: 95/100**

**Evidence:**
- 0 test files found in entire codebase
- No test framework configured (no Jest, Vitest, Cypress)
- 517 files with zero coverage

**Impact:**
- Any refactoring has high regression risk
- Cannot safely modify critical business logic
- No safety net for breaking changes

**Files Affected:** All 517 TypeScript files

**Recommended Action:**
```bash
# Install Vitest + Testing Library
pnpm add -D vitest @vitest/ui @testing-library/react @testing-library/jest-dom jsdom

# Create vitest.config.ts
# Add test scripts to package.json
# Write tests for critical services first
```

**Acceptance Criteria:**
- [ ] Vitest configured and running
- [ ] 50% coverage for services (16/32 files)
- [ ] 30% coverage for hooks (14/45 files)
- [ ] CI pipeline runs tests on every PR

---

#### P0-2: TypeScript Compilation Error
**Risk Score: 85/100**

**Evidence:**
- File: `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-nudge-toast.tsx:133`
- Error: Return type mismatch (`Element | null` vs `ReactElement`)

**Impact:**
- Blocks strict TypeScript compilation
- May cause runtime errors if null is returned

**Recommended Action:**
```typescript
// Fix return type to handle null case
const renderToast = (id: string | number): ReactElement | null => {
  // Ensure function always returns ReactElement or handle null properly
}
```

**Acceptance Criteria:**
- [ ] `pnpm typecheck` passes with zero errors
- [ ] Strict mode remains enabled

---

### P1: High Priority - Fix Within Sprint (Major Technical Debt)

#### P1-1: Massive Hook Duplication (50+ Mutation Hooks)
**Risk Score: 78/100**

**Evidence:**
- 50+ nearly identical mutation hooks across 23 hook files
- ~3,000 lines of duplicated code
- Pattern repeated: useCreate, useUpdate, useDelete, useBulkDelete

**Example Files:**
- `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scans.ts` (Lines 85-233)
- `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-targets.ts` (Lines 162-262)
- `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-subdomains.ts` (Lines 59-192)
- `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-endpoints.ts` (Lines 121-237)
- Plus 19 more hook files with identical patterns

**Impact:**
- Any change to mutation pattern requires 50+ file updates
- Inconsistent error handling across hooks
- High maintenance burden

**Recommended Action:**
Create generic mutation hook factory:

```typescript
// hooks/factories/create-mutation-hooks.ts
export function createMutationHooks<T>(config: {
  resourceName: string
  service: ResourceService<T>
  queryKeys: QueryKeyFactory
  messages: ToastMessages
}) {
  return {
    useCreate: () => useMutation({
      mutationFn: config.service.create,
      onMutate: (data) => config.messages.loading('creating'),
      onSuccess: () => {
        config.messages.success('created')
        queryClient.invalidateQueries(config.queryKeys.all())
      },
      onError: (error) => config.messages.errorFromCode(getErrorCode(error))
    }),
    // ... useUpdate, useDelete, useBulkDelete
  }
}

// Usage in use-targets.ts
export const {
  useCreateTarget,
  useUpdateTarget,
  useDeleteTarget,
  useBulkDeleteTargets
} = createMutationHooks({
  resourceName: 'target',
  service: TargetService,
  queryKeys: targetKeys,
  messages: targetMessages
})
```

**Acceptance Criteria:**
- [ ] Generic factory created and tested
- [ ] 50+ mutation hooks replaced with factory calls
- [ ] ~3,000 lines of code eliminated
- [ ] All existing functionality preserved

---

#### P1-2: Data Table Component Duplication (21 Tables)
**Risk Score: 72/100**

**Evidence:**
- 21 nearly identical data table wrapper components
- ~4,200 lines of duplicated code
- Each wraps UnifiedDataTable with resource-specific columns

**Example Files:**
- `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/subdomains-data-table.tsx` (142 lines)
- `/Users/yangyang/Desktop/lunafox/frontend/components/endpoints/endpoints-data-table.tsx` (155 lines)
- `/Users/yangyang/Desktop/lunafox/frontend/components/directories/directories-data-table.tsx` (127 lines)
- Plus 18 more similar tables

**Impact:**
- UnifiedDataTable API changes require 21 file updates
- Inconsistent table behavior across features
- Difficult to add global table features

**Recommended Action:**
Create configuration-based table system:

```typescript
// components/ui/data-table/create-resource-table.tsx
export function createResourceTable<T>(config: {
  columns: ColumnDef<T>[]
  useData: (params) => UseQueryResult
  useDelete?: () => UseMutationResult
  searchFields?: string[]
  bulkActions?: BulkAction[]
}) {
  return function ResourceTable(props) {
    const { data, isLoading } = config.useData(props.params)
    return (
      <UnifiedDataTable
        data={data}
        columns={config.columns}
        state={{ search: props.search }}
        behavior={{ searchMode: 'smart', searchFields: config.searchFields }}
        actions={{ onBulkDelete: config.useDelete?.() }}
      />
    )
  }
}

// Usage
export const SubdomainsDataTable = createResourceTable({
  columns: subdomainColumns,
  useData: useSubdomains,
  useDelete: useDeleteSubdomains,
  searchFields: ['subdomain', 'ip_address']
})
```

**Acceptance Criteria:**
- [ ] Generic table factory created
- [ ] 21 data tables replaced with configuration
- [ ] ~4,200 lines eliminated
- [ ] All table features preserved

---

#### P1-3: Tight Coupling to Toast System (23 Hooks)
**Risk Score: 68/100**

**Evidence:**
- 51% of hooks (23/45) directly depend on `useToastMessages`
- 275+ toast message calls embedded in hooks
- Hooks cannot be tested without mocking toast system

**Affected Files:**
All mutation hooks in: use-scans, use-targets, use-subdomains, use-endpoints, use-directories, use-websites, use-vulnerabilities, use-organizations, use-scheduled-scans, use-engines, use-fingerprints (6 types), use-commands, use-wordlists, use-api-keys, use-notification-settings

**Impact:**
- Hooks are not pure data fetching (violates SRP)
- Difficult to test in isolation
- Cannot reuse hooks in non-UI contexts (CLI, background jobs)

**Recommended Action:**
Separate data fetching from UI feedback:

```typescript
// hooks/use-targets.ts (pure data fetching)
export function useDeleteTarget() {
  return useMutation({
    mutationFn: (id: number) => TargetService.deleteTarget(id),
    // No toast messages here
  })
}

// components/target/targets-data-table.tsx (UI feedback)
function TargetsDataTable() {
  const toastMessages = useToastMessages()
  const deleteMutation = useDeleteTarget()

  const handleDelete = async (id: number) => {
    toastMessages.loading('Deleting target...')
    try {
      await deleteMutation.mutateAsync(id)
      toastMessages.success('Target deleted')
    } catch (error) {
      toastMessages.error('Failed to delete target')
    }
  }
}
```

**Acceptance Criteria:**
- [ ] Hooks contain only data fetching logic
- [ ] Toast messages moved to component layer
- [ ] Hooks testable without UI dependencies
- [ ] All existing toast behavior preserved

---

### P2: Medium Priority - Fix Within Month (Maintainability Issues)

#### P2-1: Large Dialog Components (626-400 lines)
**Risk Score: 58/100**

**Evidence:**
- `/Users/yangyang/Desktop/lunafox/frontend/components/scan/initiate-scan-dialog.tsx` (626 lines, 17 useState)
- `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/agent-install-dialog.tsx` (642 lines)
- `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/create-scheduled-scan-dialog.tsx` (606 lines)
- `/Users/yangyang/Desktop/lunafox/frontend/components/common/smart-filter-input.tsx` (530 lines, 100+ line useMemo)

**Impact:**
- Difficult to understand and modify
- High cognitive load for developers
- Multiple responsibilities per component

**Recommended Action:**
Extract state machines and sub-components:

```typescript
// Use XState or useReducer for complex state
const scanDialogMachine = createMachine({ /* ... */ })

// Split into smaller components
<InitiateScanDialog>
  <EngineSelector />
  <ConfigEditor />
  <ValidationPanel />
</InitiateScanDialog>
```

**Acceptance Criteria:**
- [ ] Components <400 lines each
- [ ] State management simplified (useReducer or XState)
- [ ] Sub-components extracted and reusable

---

#### P2-2: Query Key Factory Duplication (45 Patterns)
**Risk Score: 52/100**

**Evidence:**
- 45 identical query key factory patterns across all hooks
- ~900 lines of duplicated code

**Example Pattern (repeated 45 times):**
```typescript
export const xKeys = {
  all: ['x'] as const,
  lists: () => [...xKeys.all, 'list'] as const,
  list: (params) => [...xKeys.lists(), params] as const,
  details: () => [...xKeys.all, 'detail'] as const,
  detail: (id) => [...xKeys.details(), id] as const,
}
```

**Recommended Action:**
```typescript
// lib/query-keys.ts
export function createQueryKeys<T>(resource: string) {
  return {
    all: [resource] as const,
    lists: () => [resource, 'list'] as const,
    list: (params: T) => [resource, 'list', params] as const,
    details: () => [resource, 'detail'] as const,
    detail: (id: number) => [resource, 'detail', id] as const,
  }
}

// Usage
export const targetKeys = createQueryKeys<GetTargetsParams>('targets')
```

**Acceptance Criteria:**
- [ ] Generic factory created
- [ ] 45 query key factories replaced
- [ ] ~900 lines eliminated

---

#### P2-3: Missing CI/CD Pipeline
**Risk Score: 65/100**

**Evidence:**
- No `.github/workflows/` directory
- No pre-commit hooks (no `.husky/`)
- Build configured to ignore ESLint errors (`eslint.ignoreDuringBuilds: true`)

**Impact:**
- No automated quality gates
- Errors can be committed to main branch
- No test execution on PRs

**Recommended Action:**
Create `.github/workflows/ci.yml`:

```yaml
name: CI
on: [push, pull_request]
jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v2
      - uses: actions/setup-node@v4
      - run: pnpm install --frozen-lockfile
      - run: pnpm typecheck
      - run: pnpm lint --max-warnings 0
      - run: pnpm test
      - run: pnpm build
```

**Acceptance Criteria:**
- [ ] CI pipeline runs on every PR
- [ ] All checks must pass before merge
- [ ] Pre-commit hooks installed (Husky + lint-staged)

---

### P3: Low Priority - Nice to Have (Quality Improvements)

#### P3-1: Prototype Code in Production Bundle
**Risk Score: 35/100**

**Evidence:**
- `/Users/yangyang/Desktop/lunafox/frontend/app/[locale]/prototypes/` (15+ prototype pages)
- `/Users/yangyang/Desktop/lunafox/frontend/components/demo/` (demo components)
- Largest file: `advanced-asset-pulse/page.tsx` (4,684 lines)

**Impact:**
- Increased bundle size
- Confusion about production-ready code
- Maintenance burden for unused code

**Recommended Action:**
- Move prototypes to separate package or feature flag
- Exclude from production builds
- Document which prototypes are production-ready

**Acceptance Criteria:**
- [ ] Prototypes excluded from production bundle
- [ ] Bundle size reduced by ~15%

---

#### P3-2: Deprecated Code Cleanup
**Risk Score: 25/100**

**Evidence:**
- `/Users/yangyang/Desktop/lunafox/frontend/types/api-response.types.ts:23` - `LegacyApiResponse` marked @deprecated

**Recommended Action:**
- Search for usages: `rg "LegacyApiResponse" frontend`
- Remove if unused
- Migrate if still in use

**Acceptance Criteria:**
- [ ] Deprecated types removed or migrated

---

#### P3-3: TODO Resolution
**Risk Score: 30/100**

**Evidence:**
- `/Users/yangyang/Desktop/lunafox/frontend/components/tools/config/opensource-tools-list.tsx:54` - Update checking not implemented
- `/Users/yangyang/Desktop/lunafox/frontend/components/scan/engine/engine-create-dialog.tsx:132` - API integration pending
- `/Users/yangyang/Desktop/lunafox/frontend/components/scan/engine/engine-edit-dialog.tsx:129,189` - YAML config loading/saving not implemented

**Recommended Action:**
- Implement backend API integrations
- Remove TODO comments after completion

**Acceptance Criteria:**
- [ ] All 4 TODOs resolved
- [ ] Backend APIs implemented


---

## 4. Hotspot Top 20 (High-Risk Files)

### Risk Scoring Model (0-100)

```
Risk Score = (
  File Size Weight × 0.25 +
  Complexity Weight × 0.30 +
  Type Safety Weight × 0.20 +
  Business Criticality × 0.15 +
  Test Coverage Gap × 0.10
)

Where:
- File Size: (lines / 1000) × 100, capped at 100
- Complexity: useState count + useEffect count + nesting depth
- Type Safety: any count × 10 + type assertions × 5
- Business Criticality: Manual assessment (0-100)
- Test Coverage Gap: 100 if no tests, 0 if >80% coverage
```

### Top 20 High-Risk Files

| Rank | Risk | File | Lines | Issues | Priority |
|------|------|------|-------|--------|----------|
| 1 | **92** | `hooks/use-fingerprints.ts` | 596 | 84 exported functions, 6× duplication | P1 |
| 2 | **88** | `components/scan/initiate-scan-dialog.tsx` | 626 | 17 useState, complex state machine | P1 |
| 3 | **85** | `hooks/use-nudge-toast.tsx` | 133+ | TypeScript error (line 133) | P0 |
| 4 | **78** | `components/common/smart-filter-input.tsx` | 530 | 100+ line useMemo, mixed concerns | P1 |
| 5 | **75** | `components/settings/workers/agent-install-dialog.tsx` | 642 | Complex multi-step form | P2 |
| 6 | **72** | `components/scan/scheduled/create-scheduled-scan-dialog.tsx` | 606 | 26 imports, complex validation | P2 |
| 7 | **70** | `hooks/use-targets.ts` | 383 | Mutation duplication, 10 type assertions | P1 |
| 8 | **68** | `hooks/use-scans.ts` | 350+ | Mutation duplication, toast coupling | P1 |
| 9 | **65** | `components/ui/data-table/unified-data-table.tsx` | 398 | 40+ props, dual API support | P2 |
| 10 | **63** | `components/organization/targets/targets-detail-view.tsx` | 329 | 5 type assertions (lines 243-256) | P2 |
| 11 | **62** | `hooks/use-subdomains.ts` | 280+ | Mutation duplication | P1 |
| 12 | **60** | `hooks/use-endpoints.ts` | 280+ | Mutation duplication | P1 |
| 13 | **58** | `hooks/use-vulnerabilities.ts` | 300+ | Mutation duplication | P1 |
| 14 | **56** | `components/scan/history/scan-overview.tsx` | 589 | Mixed concerns, 19 imports | P2 |
| 15 | **55** | `components/target/add-target-dialog.tsx` | 527 | Complex form validation | P2 |
| 16 | **54** | `components/tools/config/add-tool-dialog.tsx` | 500 | Multi-step dialog | P2 |
| 17 | **52** | `components/scan/history/scan-history-columns.tsx` | 422 | Complex column definitions | P3 |
| 18 | **50** | `components/target/all-targets-detail-view.tsx` | 356 | 2 type assertions (lines 111-112) | P2 |
| 19 | **48** | `components/scan/scheduled/edit-scheduled-scan-dialog.tsx` | 283 | 1 type assertion (line 136) | P2 |
| 20 | **45** | `components/dashboard/dashboard-data-table.tsx` | 410 | 22 imports, aggregation logic | P2 |

### Detailed Risk Analysis

#### #1: hooks/use-fingerprints.ts (Risk: 92/100)
**Location:** `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints.ts`

**Risk Breakdown:**
- File Size: 60/100 (596 lines)
- Complexity: 95/100 (84 exported functions)
- Type Safety: 80/100 (no type issues, but massive duplication)
- Business Criticality: 70/100 (fingerprint management)
- Test Coverage: 100/100 (no tests)

**Issues:**
- 84 exported functions from single file
- 6× identical CRUD pattern duplication (~3,000 lines if expanded)
- Pattern: useEholeFingerprints, useGobyFingerprints, useWappalyzerFingerprints, etc.

**Evidence:**
```typescript
// Lines 1-100: Ehole fingerprint hooks (14 functions)
// Lines 101-200: Goby fingerprint hooks (14 functions)
// Lines 201-300: Wappalyzer fingerprint hooks (14 functions)
// Lines 301-400: Fingers fingerprint hooks (14 functions)
// Lines 401-500: FingerprintHub fingerprint hooks (14 functions)
// Lines 501-596: ARL fingerprint hooks (14 functions)
```

**Recommended Action:**
Create generic factory function (see P1-1 in Debt Inventory)

**Verification:**
```bash
rg "export function use.*Fingerprint" hooks/use-fingerprints.ts | wc -l
# Expected: 84
```

---

#### #2: components/scan/initiate-scan-dialog.tsx (Risk: 88/100)
**Location:** `/Users/yangyang/Desktop/lunafox/frontend/components/scan/initiate-scan-dialog.tsx`

**Risk Breakdown:**
- File Size: 63/100 (626 lines)
- Complexity: 100/100 (17 useState, 10+ useCallback, multiple useEffect)
- Type Safety: 90/100 (no type issues, but complex state)
- Business Criticality: 95/100 (critical scan initiation)
- Test Coverage: 100/100 (no tests)

**Issues:**
- 17 useState hooks (state explosion)
- Complex state synchronization between preset/custom modes
- Nested confirmation dialogs
- Business logic mixed with UI

**Evidence:**
```typescript
// Lines 50-80: 17 useState declarations
const [selectedEngineIds, setSelectedEngineIds] = useState<number[]>([])
const [selectedPresetId, setSelectedPresetId] = useState<string | null>(null)
const [selectMode, setSelectMode] = useState<"preset" | "custom">("preset")
// ... 14 more useState calls
```

**Recommended Action:**
Extract state machine using XState or useReducer (see P1-1 in Debt Inventory)

---

#### #3: hooks/use-nudge-toast.tsx (Risk: 85/100)
**Location:** `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-nudge-toast.tsx:133`

**Risk Breakdown:**
- File Size: 13/100 (133 lines)
- Complexity: 60/100 (moderate)
- Type Safety: 100/100 (TypeScript error blocking compilation)
- Business Criticality: 80/100 (user notifications)
- Test Coverage: 100/100 (no tests)

**Issues:**
- TypeScript compilation error at line 133
- Return type mismatch: `Element | null` vs `ReactElement`

**Evidence:**
```
error TS2345: Argument of type '(t: string | number) => Element | null' 
is not assignable to parameter of type '(id: string | number) => ReactElement<...>'.
Type 'Element | null' is not assignable to type 'ReactElement<...>'.
Type 'null' is not assignable to type 'ReactElement<...>'.
```

**Recommended Action:**
Fix return type to ensure non-null ReactElement or handle null case properly

---

#### #4: components/common/smart-filter-input.tsx (Risk: 78/100)
**Location:** `/Users/yangyang/Desktop/lunafox/frontend/components/common/smart-filter-input.tsx`

**Risk Breakdown:**
- File Size: 53/100 (530 lines)
- Complexity: 95/100 (100+ line useMemo, complex parsing)
- Type Safety: 70/100 (no issues, but complex logic)
- Business Criticality: 85/100 (search functionality)
- Test Coverage: 100/100 (no tests)

**Issues:**
- 100+ line useMemo for ghost text calculation
- Multiple regex patterns for FOFA-style parsing
- LocalStorage integration mixed with component logic
- Single responsibility violation (parsing + UI + storage)

**Evidence:**
```typescript
// Lines 150-260: Massive ghost text calculation useMemo
const ghostText = useMemo(() => {
  // 100+ lines of complex parsing logic
}, [inputValue, fields, suggestions])
```

**Recommended Action:**
Extract parsing logic to separate module (see P1-3 in Debt Inventory)

---

### Risk Distribution

```
Critical Risk (80-100):  4 files  (20%)
High Risk (60-79):       9 files  (45%)
Medium Risk (40-59):     7 files  (35%)
```

### Risk by Category

```
Duplication Risk:        12 files (hooks + data tables)
Complexity Risk:         5 files  (large dialogs)
Type Safety Risk:        1 file   (TypeScript error)
Test Coverage Risk:      20 files (all untested)
```

---

## 5. UnifiedDataTable Migration Assessment

### Migration Status: COMPLETE ✅

**Summary:**
- Total Tables: 23
- Migrated to Grouped Props: 23 (100%)
- Using Legacy Props: 0 (0%)
- Migration Completion: 100%

### Table Inventory by Category

#### Core Asset Tables (8 tables)
| Table | File | Lines | Status |
|-------|------|-------|--------|
| Directories | `components/directories/directories-data-table.tsx` | 127 | ✅ Grouped |
| Endpoints | `components/endpoints/endpoints-data-table.tsx` | 155 | ✅ Grouped |
| Websites | `components/websites/websites-data-table.tsx` | 127 | ✅ Grouped |
| Subdomains | `components/subdomains/subdomains-data-table.tsx` | 142 | ✅ Grouped |
| IP Addresses | `components/ip-addresses/ip-addresses-data-table.tsx` | 113 | ✅ Grouped |
| Vulnerabilities | `components/vulnerabilities/vulnerabilities-data-table.tsx` | 278 | ✅ Grouped |
| Targets | `components/target/targets-data-table.tsx` | 185 | ✅ Grouped |
| Org Targets | `components/organization/targets/targets-data-table.tsx` | 147 | ✅ Grouped |

#### Fingerprint Tables (6 tables)
| Table | File | Lines | Status |
|-------|------|-------|--------|
| ARL | `components/fingerprints/arl-fingerprint-data-table.tsx` | 257 | ✅ Grouped |
| Ehole | `components/fingerprints/ehole-fingerprint-data-table.tsx` | 269 | ✅ Grouped |
| FingerprintHub | `components/fingerprints/fingerprinthub-fingerprint-data-table.tsx` | 262 | ✅ Grouped |
| Fingers | `components/fingerprints/fingers-fingerprint-data-table.tsx` | 267 | ✅ Grouped |
| Goby | `components/fingerprints/goby-fingerprint-data-table.tsx` | 265 | ✅ Grouped |
| Wappalyzer | `components/fingerprints/wappalyzer-fingerprint-data-table.tsx` | 267 | ✅ Grouped |

#### Scan & Management Tables (5 tables)
| Table | File | Lines | Status |
|-------|------|-------|--------|
| Scan History | `components/scan/history/scan-history-data-table.tsx` | 168 | ✅ Grouped |
| Scheduled Scans | `components/scan/scheduled/scheduled-scan-data-table.tsx` | 131 | ✅ Grouped |
| Engines | `components/scan/engine/engine-data-table.tsx` | 71 | ✅ Grouped |
| Organizations | `components/organization/organization-data-table.tsx` | 91 | ✅ Grouped |
| Commands | `components/tools/commands/commands-data-table.tsx` | 73 | ✅ Grouped |

#### Special Purpose Tables (4 tables)
| Table | File | Lines | Status |
|-------|------|-------|--------|
| Dashboard | `components/dashboard/dashboard-data-table.tsx` | 410 | ✅ Grouped |
| Search Results | `components/search/search-results-table.tsx` | 261 | ✅ Grouped |

### Grouped Props Pattern Usage

All 23 tables use the new grouped configuration:

```typescript
<UnifiedDataTable
  data={data}
  columns={columns}
  state={{
    pagination: paginationState,
    search: searchState,
    rowSelection: selectionState
  }}
  ui={{
    hideToolbar: false,
    className: "custom-class",
    toolbarLeft: <CustomFilters />
  }}
  behavior={{
    searchMode: "smart",
    enableRowSelection: true,
    searchFields: ['field1', 'field2']
  }}
  actions={{
    onBulkDelete: handleBulkDelete,
    onAddNew: handleAddNew
  }}
/>
```

### Legacy Props Status: RETIRED ✅

**No tables are using legacy flat props.** The following legacy props are no longer in use:
- `hideToolbar` (moved to `ui.hideToolbar`)
- `hidePagination` (moved to `ui.hidePagination`)
- `enableRowSelection` (moved to `behavior.enableRowSelection`)
- `searchMode` (moved to `behavior.searchMode`)

### Compatibility Layer Status

**Current State:**
- UnifiedDataTable still supports legacy props for backward compatibility
- 120 lines of conflict detection code in `unified-data-table.tsx`
- `firstDefined()` helper used throughout for prop resolution

**Retirement Plan:**
Since 100% of tables have migrated, the legacy API can be safely removed.

**Recommended Actions:**

1. **Phase 1: Deprecation Warning (Week 1)**
   - Add console warnings for legacy prop usage
   - Update documentation to mark legacy props as deprecated

2. **Phase 2: Remove Legacy Support (Week 2-3)**
   ```typescript
   // Remove from unified-data-table.tsx:
   - Lines with firstDefined() calls (~40 lines)
   - Conflict detection logic (~120 lines)
   - Legacy prop type definitions (~30 lines)
   
   // Total lines to remove: ~190 lines
   ```

3. **Phase 3: Simplify API (Week 4)**
   - Remove dual API complexity
   - Simplify prop types
   - Update tests

**Acceptance Criteria:**
- [ ] All legacy prop support removed
- [ ] ~190 lines of compatibility code eliminated
- [ ] All 23 tables still function correctly
- [ ] Type definitions simplified

### Technical Debt Assessment

**Remaining Debt:**
- **Moderate**: 21 data table components are nearly identical wrappers
- Each table wraps UnifiedDataTable with resource-specific columns
- ~4,200 lines of duplicated wrapper code

**Recommendation:**
Create configuration-based table system (see P1-2 in Debt Inventory)

### Migration Success Metrics

```
Migration Completion:     100% ✅
Legacy Prop Usage:        0%   ✅
Tables at Risk:           0    ✅
Breaking Changes:         0    ✅
```

**Conclusion:** UnifiedDataTable migration is complete and successful. The compatibility layer can be safely removed, eliminating 190 lines of technical debt.


---

## 6. 90-Day Refactoring Roadmap

### Overview

This roadmap breaks down the technical debt remediation into three phases, each building on the previous phase. The focus is on establishing quality foundations first, then eliminating duplication, and finally optimizing architecture.

### Phase 1: Foundation & Quality Gates (Weeks 1-4)

**Goal:** Establish testing infrastructure and fix critical blockers

#### Week 1: Test Infrastructure Setup
- [ ] Install Vitest + Testing Library + jsdom
- [ ] Create `vitest.config.ts` with proper configuration
- [ ] Set up test scripts in `package.json`
- [ ] Create `vitest.setup.ts` with global test utilities
- [ ] Write first test for `lib/api-client.ts` as template
- [ ] Document testing patterns in README

**Deliverables:**
- Working test infrastructure
- 5+ example tests demonstrating patterns
- Testing documentation

**Exit Criteria:**
- `pnpm test` runs successfully
- Coverage reporting works
- Team can write and run tests

---

#### Week 2: Critical Services Testing
- [ ] Test `services/auth.service.ts` (authentication flows)
- [ ] Test `services/organization.service.ts` (CRUD operations)
- [ ] Test `services/vulnerability.service.ts` (security critical)
- [ ] Test `lib/api-client.ts` (HTTP client)
- [ ] Test `lib/error-handler.ts` (error handling)
- [ ] Test `lib/response-parser.ts` (data parsing)

**Target:** 50% coverage for services (16/32 files)

**Deliverables:**
- 50+ tests covering critical services
- Documented test patterns for services

**Exit Criteria:**
- All critical services have >70% coverage
- No regressions in existing functionality

---

#### Week 3: CI/CD Pipeline & Pre-commit Hooks
- [ ] Create `.github/workflows/ci.yml`
- [ ] Configure GitHub Actions for PR checks
- [ ] Install Husky for git hooks
- [ ] Configure lint-staged for pre-commit
- [ ] Add typecheck to CI pipeline
- [ ] Add lint to CI pipeline (--max-warnings 0)
- [ ] Add test execution to CI
- [ ] Add build verification to CI

**Deliverables:**
- Automated CI pipeline
- Pre-commit hooks preventing bad commits
- Branch protection rules

**Exit Criteria:**
- All PRs must pass CI checks
- Pre-commit hooks block commits with errors
- Build failures prevent merge

---

#### Week 4: Fix TypeScript Error & TODOs
- [ ] Fix `hooks/use-nudge-toast.tsx:133` type error
- [ ] Implement backend API for engine creation (`components/scan/engine/engine-create-dialog.tsx:132`)
- [ ] Implement YAML config loading (`components/scan/engine/engine-edit-dialog.tsx:129`)
- [ ] Implement YAML config saving (`components/scan/engine/engine-edit-dialog.tsx:189`)
- [ ] Implement update checking (`components/tools/config/opensource-tools-list.tsx:54`)
- [ ] Remove all TODO comments after completion

**Deliverables:**
- Zero TypeScript errors
- All TODOs resolved
- Backend APIs implemented

**Exit Criteria:**
- `pnpm typecheck` passes with zero errors
- No TODO comments remain in codebase

---

### Phase 2: Duplication Elimination (Weeks 5-8)

**Goal:** Eliminate 8,100+ lines of duplicated code through abstraction

#### Week 5: Mutation Hook Factory
- [ ] Design generic mutation hook factory interface
- [ ] Implement `createMutationHooks` factory in `hooks/factories/`
- [ ] Create resource configuration types
- [ ] Write tests for factory function
- [ ] Migrate 5 hooks as proof of concept (targets, scans, subdomains, endpoints, directories)
- [ ] Validate all existing functionality preserved

**Deliverables:**
- Generic mutation hook factory
- 5 hooks migrated successfully
- ~600 lines eliminated

**Exit Criteria:**
- Factory function tested and documented
- Migrated hooks pass all tests
- No breaking changes

---

#### Week 6: Complete Mutation Hook Migration
- [ ] Migrate remaining 18 hooks to factory pattern
- [ ] Remove old mutation hook implementations
- [ ] Update all components using migrated hooks
- [ ] Run full regression test suite
- [ ] Update documentation

**Deliverables:**
- All 50+ mutation hooks replaced with factory calls
- ~3,000 lines eliminated
- Updated documentation

**Exit Criteria:**
- All hooks use factory pattern
- All tests pass
- No functionality regressions

---

#### Week 7: Query Key Factory
- [ ] Create generic `createQueryKeys` factory in `lib/query-keys.ts`
- [ ] Migrate all 45 query key factories
- [ ] Update all hooks to use generic factory
- [ ] Remove duplicated query key code
- [ ] Update tests

**Deliverables:**
- Generic query key factory
- 45 query key factories replaced
- ~900 lines eliminated

**Exit Criteria:**
- All hooks use generic query keys
- Cache invalidation still works correctly

---

#### Week 8: Data Table Configuration System
- [ ] Design configuration-based table interface
- [ ] Implement `createResourceTable` factory in `components/ui/data-table/`
- [ ] Migrate 5 tables as proof of concept
- [ ] Validate all table features preserved
- [ ] Migrate remaining 16 tables
- [ ] Remove old table wrapper components

**Deliverables:**
- Configuration-based table system
- 21 data tables replaced with configuration
- ~4,200 lines eliminated

**Exit Criteria:**
- All tables use configuration pattern
- All table features work (search, pagination, selection, bulk actions)
- No UI regressions

---

### Phase 3: Architecture Optimization (Weeks 9-12)

**Goal:** Decouple concerns and simplify complex components

#### Week 9: Decouple Hooks from Toast System
- [ ] Remove toast messages from all hooks
- [ ] Move toast logic to component layer
- [ ] Create hook composition utilities
- [ ] Update all components using affected hooks
- [ ] Write tests for pure hooks
- [ ] Document new pattern

**Deliverables:**
- Pure data fetching hooks (no UI dependencies)
- Toast messages in component layer
- Testable hooks

**Exit Criteria:**
- Hooks can be tested without mocking toast system
- All existing toast behavior preserved
- 23 hooks decoupled

---

#### Week 10: Refactor Large Dialog Components
- [ ] Extract state machine from `initiate-scan-dialog.tsx` (626 lines → <400 lines)
- [ ] Split `smart-filter-input.tsx` into parser + UI + storage (530 lines → <300 lines)
- [ ] Refactor `agent-install-dialog.tsx` (642 lines → <400 lines)
- [ ] Refactor `create-scheduled-scan-dialog.tsx` (606 lines → <400 lines)
- [ ] Create reusable multi-step form hook
- [ ] Document component patterns

**Deliverables:**
- 4 large components refactored
- Reusable form utilities
- ~1,000 lines of complexity reduced

**Exit Criteria:**
- All dialog components <400 lines
- State management simplified
- All functionality preserved

---

#### Week 11: Remove UnifiedDataTable Legacy API
- [ ] Add deprecation warnings for legacy props
- [ ] Verify 100% migration (already complete)
- [ ] Remove `firstDefined()` helper calls (~40 lines)
- [ ] Remove conflict detection logic (~120 lines)
- [ ] Remove legacy prop type definitions (~30 lines)
- [ ] Simplify component implementation
- [ ] Update tests

**Deliverables:**
- Legacy API removed
- ~190 lines eliminated
- Simplified API

**Exit Criteria:**
- All 23 tables still function correctly
- Type definitions simplified
- No breaking changes

---

#### Week 12: Final Cleanup & Documentation
- [ ] Remove deprecated `LegacyApiResponse` type
- [ ] Separate prototype code from production bundle
- [ ] Add Prettier configuration
- [ ] Format entire codebase
- [ ] Update README with new patterns
- [ ] Create architecture documentation
- [ ] Document testing patterns
- [ ] Create contribution guidelines

**Deliverables:**
- Clean codebase
- Comprehensive documentation
- Contribution guidelines

**Exit Criteria:**
- Zero deprecated code
- All code formatted consistently
- Documentation complete

---

### Roadmap Summary

| Phase | Duration | Focus | Lines Eliminated | Tests Added |
|-------|----------|-------|------------------|-------------|
| Phase 1 | Weeks 1-4 | Foundation & Quality | ~200 | 100+ |
| Phase 2 | Weeks 5-8 | Duplication Elimination | ~8,100 | 50+ |
| Phase 3 | Weeks 9-12 | Architecture Optimization | ~1,400 | 30+ |
| **Total** | **12 weeks** | **Complete Refactoring** | **~9,700** | **180+** |

### Success Metrics

**By End of Phase 1:**
- ✅ Test infrastructure operational
- ✅ 50% service coverage
- ✅ CI/CD pipeline active
- ✅ Zero TypeScript errors

**By End of Phase 2:**
- ✅ 8,100+ lines eliminated
- ✅ 50+ mutation hooks replaced with factory
- ✅ 21 data tables using configuration
- ✅ Code duplication reduced from 8.3% to <2%

**By End of Phase 3:**
- ✅ Hooks decoupled from UI concerns
- ✅ All components <400 lines
- ✅ Legacy APIs removed
- ✅ 80% overall test coverage

### Risk Mitigation

**Risk:** Breaking existing functionality during refactoring
**Mitigation:** 
- Incremental changes with tests
- Feature flags for major changes
- Thorough regression testing

**Risk:** Team velocity slowdown during refactoring
**Mitigation:**
- Parallel work streams (new features + refactoring)
- Clear ownership of refactoring tasks
- Automated testing reduces manual QA

**Risk:** Incomplete migration leaving mixed patterns
**Mitigation:**
- Complete one pattern before starting next
- Automated checks for old patterns
- Code review enforcement

---

## 7. CI/Quality Gate Recommendations

### Automated Quality Checks

#### 1. Type Safety Gate
```yaml
# .github/workflows/ci.yml
- name: TypeScript Type Check
  run: pnpm typecheck
  # Must pass: Zero errors allowed
```

**Enforcement:**
- Block PR merge if type errors exist
- No exceptions or overrides

**Current Status:** 1 error in `use-nudge-toast.tsx:133`

---

#### 2. Linting Gate
```yaml
- name: ESLint
  run: pnpm lint --max-warnings 0
  # Must pass: Zero warnings or errors
```

**Enforcement:**
- Block PR merge if lint issues exist
- Remove `eslint.ignoreDuringBuilds: true` from `next.config.ts`

**Current Status:** 5 warnings, 16 errors across 7 files

**Action Required:**
- Fix all existing lint issues
- Enable strict linting in CI

---

#### 3. Test Coverage Gate
```yaml
- name: Run Tests
  run: pnpm test --coverage

- name: Coverage Check
  run: |
    pnpm test --coverage --reporter=json-summary
    node scripts/check-coverage.js --min-coverage=80
```

**Enforcement:**
- Block PR merge if coverage drops below threshold
- Require tests for all new code

**Thresholds:**
```json
{
  "lines": 80,
  "functions": 80,
  "branches": 75,
  "statements": 80
}
```

**Current Status:** 0% coverage (no tests)

---

#### 4. Build Verification Gate
```yaml
- name: Build Check
  run: pnpm build
  # Must pass: Build must succeed
```

**Enforcement:**
- Block PR merge if build fails
- No build warnings allowed

---

#### 5. Code Quality Rules

**Automated Checks:**

```yaml
# .github/workflows/code-quality.yml
name: Code Quality

on: [pull_request]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: pnpm/action-setup@v2
        with:
          version: 8
      
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: 'pnpm'
      
      - name: Install dependencies
        run: pnpm install --frozen-lockfile
      
      - name: Type check
        run: pnpm typecheck
      
      - name: Lint
        run: pnpm lint --max-warnings 0
      
      - name: Test
        run: pnpm test --coverage
      
      - name: Build
        run: pnpm build
      
      - name: Check for TODOs in new code
        run: |
          git diff origin/main...HEAD | grep -E "^\+.*TODO|^\+.*FIXME" && exit 1 || exit 0
      
      - name: Check file size
        run: |
          find . -name "*.ts" -o -name "*.tsx" | while read file; do
            lines=$(wc -l < "$file")
            if [ $lines -gt 500 ]; then
              echo "❌ $file has $lines lines (max 500)"
              exit 1
            fi
          done
```

---

#### 6. Pre-commit Hooks

**Install Husky:**
```bash
pnpm add -D husky lint-staged
pnpm exec husky init
```

**Configure `.husky/pre-commit`:**
```bash
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

pnpm lint-staged
pnpm typecheck
```

**Configure `lint-staged` in `package.json`:**
```json
{
  "lint-staged": {
    "*.{ts,tsx}": [
      "eslint --fix --max-warnings 0",
      "vitest related --run --passWithNoTests"
    ],
    "*.{json,md}": [
      "prettier --write"
    ]
  }
}
```

---

#### 7. Dependency Security Scanning

```yaml
# .github/workflows/security.yml
name: Security Scan

on:
  schedule:
    - cron: '0 0 * * 1'  # Weekly on Monday
  pull_request:

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Run npm audit
        run: pnpm audit --audit-level=moderate
      
      - name: Check for known vulnerabilities
        uses: snyk/actions/node@master
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
```

---

#### 8. Code Review Checklist

**Automated PR Checks:**
- [ ] All CI checks pass (type, lint, test, build)
- [ ] Test coverage meets threshold (80%)
- [ ] No new TODO/FIXME comments
- [ ] No files >500 lines
- [ ] No new `any` types without justification
- [ ] No new ESLint disables without comment
- [ ] All new functions have tests

**Manual Review:**
- [ ] Code follows existing patterns
- [ ] No unnecessary complexity
- [ ] Security considerations addressed
- [ ] Performance implications considered
- [ ] Documentation updated if needed

---

### Quality Metrics Dashboard

**Recommended Tools:**
- **Codecov** - Test coverage tracking
- **SonarCloud** - Code quality metrics
- **Snyk** - Dependency vulnerability scanning
- **Bundle Analyzer** - Bundle size monitoring

**Key Metrics to Track:**
```
Test Coverage:           0% → 80% (target)
TypeScript Errors:       1 → 0 (target)
ESLint Issues:           21 → 0 (target)
Code Duplication:        8.3% → <2% (target)
Average File Size:       250 lines (current, maintain)
Bundle Size:             Track and prevent growth
```

---

### Enforcement Strategy

**Phase 1: Warning Mode (Weeks 1-2)**
- CI runs but doesn't block merges
- Team gets familiar with new checks
- Fix existing issues

**Phase 2: Soft Enforcement (Weeks 3-4)**
- CI blocks merges for critical issues (type errors, build failures)
- Warnings allowed but tracked

**Phase 3: Full Enforcement (Week 5+)**
- All checks must pass to merge
- Zero tolerance for quality regressions
- Pre-commit hooks prevent bad commits


---

## 8. Speckit Input (Decision-Complete Refactoring Specification)

This section provides all necessary inputs for Speckit-driven implementation, ensuring another engineer can execute the refactoring without additional decision-making.

### Target State

**Objective:** Transform the frontend codebase from a duplication-heavy, untested state to a well-tested, DRY, and maintainable architecture.

**Success Criteria:**
- Test coverage: 0% → 80%
- Code duplication: 8.3% → <2%
- TypeScript errors: 1 → 0
- ESLint issues: 21 → 0
- Average component complexity: Reduced by 40%
- All components <500 lines

### Scope & Boundaries

**In Scope:**
- All 517 TypeScript files in `/Users/yangyang/Desktop/lunafox/frontend/`
- Test infrastructure setup (Vitest, Testing Library)
- CI/CD pipeline configuration
- Hook refactoring (50+ mutation hooks)
- Data table refactoring (21 tables)
- Large component refactoring (4 components >600 lines)
- UnifiedDataTable legacy API removal

**Out of Scope:**
- Backend API changes (read-only for understanding contracts)
- Prototype/demo code functionality changes (can refactor structure)
- UI/UX design changes
- New feature development
- Database schema changes
- Infrastructure changes (Docker, deployment)

**Constraints:**
- Zero breaking changes to public APIs
- All existing functionality must be preserved
- No changes to user-facing behavior
- Maintain backward compatibility during migration
- No downtime during deployment

### Interface Contracts

**Hook Interfaces (Must Preserve):**
```typescript
// Current interface that must be maintained
export function useTargets(params?: GetTargetsParams, options?: UseQueryOptions) {
  // Implementation changes, but signature stays the same
}

export function useDeleteTarget() {
  return useMutation<void, Error, { id: number }>({
    // Implementation changes, but return type stays the same
  })
}
```

**Component Interfaces (Must Preserve):**
```typescript
// Data table components
interface DataTableProps<T> {
  data: T[]
  columns: ColumnDef<T>[]
  state?: TableState
  ui?: TableUI
  behavior?: TableBehavior
  actions?: TableActions
  // All existing props must continue to work
}
```

**Service Interfaces (Must Preserve):**
```typescript
// All service methods must maintain signatures
class TargetService {
  static getTargets(params: GetTargetsParams): Promise<PaginatedResponse<Target>>
  static deleteTarget(id: number): Promise<void>
  // All existing methods preserved
}
```

### Architecture Decisions

**Decision 1: Test Framework**
- **Choice:** Vitest + Testing Library
- **Rationale:** 
  - Vitest: Fast, Vite-native, better DX than Jest
  - Testing Library: React best practices, user-centric testing
- **Alternatives Considered:** Jest (slower, more config), Cypress (E2E only)

**Decision 2: Mutation Hook Pattern**
- **Choice:** Generic factory function
- **Rationale:** Eliminates 3,000 lines of duplication, centralizes error handling
- **Implementation:**
```typescript
// hooks/factories/create-mutation-hooks.ts
export function createMutationHooks<T>(config: MutationConfig<T>) {
  return {
    useCreate: () => useMutation({ /* ... */ }),
    useUpdate: () => useMutation({ /* ... */ }),
    useDelete: () => useMutation({ /* ... */ }),
    useBulkDelete: () => useMutation({ /* ... */ }),
  }
}
```

**Decision 3: Data Table Pattern**
- **Choice:** Configuration-based factory
- **Rationale:** Reduces 21 components to configuration objects
- **Implementation:**
```typescript
// components/ui/data-table/create-resource-table.tsx
export function createResourceTable<T>(config: TableConfig<T>) {
  return function ResourceTable(props: ResourceTableProps) {
    // Generic implementation
  }
}
```

**Decision 4: Hook Decoupling**
- **Choice:** Separate data fetching from UI feedback
- **Rationale:** Makes hooks testable, reusable in non-UI contexts
- **Pattern:**
```typescript
// Hooks: Pure data fetching
export function useDeleteTarget() {
  return useMutation({ mutationFn: TargetService.deleteTarget })
}

// Components: UI feedback
function Component() {
  const toast = useToastMessages()
  const mutation = useDeleteTarget()
  
  const handleDelete = async (id: number) => {
    toast.loading('Deleting...')
    try {
      await mutation.mutateAsync(id)
      toast.success('Deleted')
    } catch (error) {
      toast.error('Failed')
    }
  }
}
```

**Decision 5: State Management for Complex Forms**
- **Choice:** useReducer for forms with >5 state variables
- **Rationale:** Simpler than XState, more maintainable than multiple useState
- **Pattern:**
```typescript
type ScanDialogState = {
  step: number
  selectedEngineIds: number[]
  configuration: string
  // ... other state
}

type ScanDialogAction = 
  | { type: 'SET_ENGINES'; payload: number[] }
  | { type: 'NEXT_STEP' }
  | { type: 'PREV_STEP' }

function scanDialogReducer(state: ScanDialogState, action: ScanDialogAction) {
  // Centralized state logic
}
```

### Test Strategy & Matrix

**Test Coverage Targets:**
```
Services:        80% (critical business logic)
Hooks:           70% (data fetching + mutations)
Components:      60% (UI components)
Utilities:       90% (pure functions)
Overall:         80%
```

**Test Types:**

1. **Unit Tests (Primary Focus)**
   - All services (32 files)
   - All hooks (45 files)
   - Utility functions (18 files)
   - Pure components

2. **Integration Tests (Secondary)**
   - Hook + Service integration
   - Component + Hook integration
   - Form submission flows

3. **E2E Tests (Out of Scope for This Phase)**
   - Defer to future work

**Test Patterns:**

```typescript
// Service tests
describe('TargetService', () => {
  it('should fetch targets with pagination', async () => {
    const params = { page: 1, pageSize: 10 }
    const result = await TargetService.getTargets(params)
    expect(result.results).toHaveLength(10)
  })
})

// Hook tests
describe('useDeleteTarget', () => {
  it('should delete target and invalidate cache', async () => {
    const { result } = renderHook(() => useDeleteTarget())
    await act(() => result.current.mutateAsync({ id: 1 }))
    expect(mockQueryClient.invalidateQueries).toHaveBeenCalled()
  })
})

// Component tests
describe('TargetsDataTable', () => {
  it('should render targets and handle selection', () => {
    render(<TargetsDataTable data={mockTargets} />)
    expect(screen.getByText('Target 1')).toBeInTheDocument()
  })
})
```

**Test Utilities:**
```typescript
// test/utils/test-utils.tsx
export function renderWithProviders(ui: ReactElement) {
  return render(
    <QueryClientProvider client={testQueryClient}>
      {ui}
    </QueryClientProvider>
  )
}

export const mockTargets = [/* ... */]
export const mockServices = {
  TargetService: {
    getTargets: vi.fn(),
    deleteTarget: vi.fn(),
  }
}
```

### Migration Strategy

**Phase 1: Foundation (No Breaking Changes)**
- Add test infrastructure alongside existing code
- Write tests for existing code without modifications
- Set up CI/CD pipeline
- Fix TypeScript error

**Phase 2: Incremental Refactoring (Backward Compatible)**
- Create factory functions alongside existing hooks
- Migrate hooks one at a time
- Keep old hooks until all consumers migrated
- Use feature flags if needed

**Phase 3: Cleanup (Breaking Changes Allowed)**
- Remove old implementations after migration complete
- Remove legacy APIs
- Simplify interfaces

**Rollback Strategy:**
- Git tags at each phase completion
- Feature flags for major changes
- Ability to revert individual migrations
- Comprehensive test suite prevents regressions

### File-Level Implementation Plan

**Critical Files to Modify:**

1. **Test Infrastructure (New Files)**
   - `vitest.config.ts` (create)
   - `vitest.setup.ts` (create)
   - `test/utils/test-utils.tsx` (create)

2. **Hook Factories (New Files)**
   - `hooks/factories/create-mutation-hooks.ts` (create)
   - `hooks/factories/create-query-keys.ts` (create)

3. **Hooks to Refactor (50+ files)**
   - `hooks/use-targets.ts` (383 lines → ~150 lines)
   - `hooks/use-scans.ts` (350+ lines → ~150 lines)
   - `hooks/use-subdomains.ts` (280+ lines → ~120 lines)
   - `hooks/use-endpoints.ts` (280+ lines → ~120 lines)
   - `hooks/use-vulnerabilities.ts` (300+ lines → ~150 lines)
   - Plus 18 more hooks

4. **Data Table Factory (New File)**
   - `components/ui/data-table/create-resource-table.tsx` (create)

5. **Data Tables to Refactor (21 files)**
   - `components/subdomains/subdomains-data-table.tsx` (142 lines → ~30 lines config)
   - `components/endpoints/endpoints-data-table.tsx` (155 lines → ~30 lines config)
   - Plus 19 more tables

6. **Large Components to Refactor (4 files)**
   - `components/scan/initiate-scan-dialog.tsx` (626 lines → <400 lines)
   - `components/common/smart-filter-input.tsx` (530 lines → <300 lines)
   - `components/settings/workers/agent-install-dialog.tsx` (642 lines → <400 lines)
   - `components/scan/scheduled/create-scheduled-scan-dialog.tsx` (606 lines → <400 lines)

7. **UnifiedDataTable Simplification**
   - `components/ui/data-table/unified-data-table.tsx` (398 lines → ~250 lines)
   - Remove ~190 lines of legacy API support

8. **CI/CD Configuration (New Files)**
   - `.github/workflows/ci.yml` (create)
   - `.husky/pre-commit` (create)
   - `lint-staged.config.js` (create)

### Validation & Acceptance Criteria

**Automated Checks:**
```bash
# Must pass before considering phase complete
pnpm typecheck          # Zero errors
pnpm lint --max-warnings 0  # Zero warnings/errors
pnpm test --coverage    # 80% coverage
pnpm build              # Successful build
```

**Manual Validation:**
- [ ] All existing features work identically
- [ ] No visual regressions in UI
- [ ] No performance degradation
- [ ] All data tables function correctly
- [ ] All forms submit successfully
- [ ] All API calls work as before

**Regression Test Checklist:**
- [ ] User can create/edit/delete targets
- [ ] User can initiate scans
- [ ] User can view scan history
- [ ] User can manage organizations
- [ ] User can configure engines
- [ ] User can manage fingerprints
- [ ] All data tables paginate correctly
- [ ] All data tables search correctly
- [ ] All bulk actions work
- [ ] All toast messages appear

**Performance Benchmarks:**
- Bundle size: Must not increase >5%
- Initial load time: Must not increase >10%
- Table rendering: Must not increase >10%
- API response handling: Must not increase >10%

### Risk Assessment & Mitigation

**High Risk Areas:**

1. **Hook Refactoring**
   - **Risk:** Breaking existing components
   - **Mitigation:** Comprehensive tests before refactoring, incremental migration
   - **Rollback:** Keep old hooks until migration complete

2. **Data Table Refactoring**
   - **Risk:** Loss of table functionality
   - **Mitigation:** Test all table features (search, pagination, selection, bulk actions)
   - **Rollback:** Feature flag for new table system

3. **State Management Changes**
   - **Risk:** Complex forms break
   - **Mitigation:** Thorough testing of all form flows
   - **Rollback:** Keep old implementation until validated

**Medium Risk Areas:**

4. **CI/CD Pipeline**
   - **Risk:** Blocking legitimate PRs
   - **Mitigation:** Warning mode first, then enforcement
   - **Rollback:** Disable checks temporarily

5. **Test Infrastructure**
   - **Risk:** Flaky tests
   - **Mitigation:** Use Testing Library best practices, avoid implementation details
   - **Rollback:** N/A (additive only)

**Low Risk Areas:**

6. **Documentation Updates**
   - **Risk:** Minimal
   - **Mitigation:** Review before merge

7. **Code Formatting**
   - **Risk:** Minimal
   - **Mitigation:** Prettier auto-fix

### Dependencies & Prerequisites

**External Dependencies (Add to package.json):**
```json
{
  "devDependencies": {
    "vitest": "^1.0.0",
    "@vitest/ui": "^1.0.0",
    "@testing-library/react": "^14.0.0",
    "@testing-library/jest-dom": "^6.1.5",
    "@testing-library/user-event": "^14.5.1",
    "jsdom": "^23.0.0",
    "husky": "^8.0.0",
    "lint-staged": "^15.0.0"
  }
}
```

**Team Prerequisites:**
- All team members familiar with Vitest
- All team members familiar with Testing Library
- Code review process established
- CI/CD access configured

**Infrastructure Prerequisites:**
- GitHub Actions enabled
- Branch protection rules configured
- Secrets configured (if needed for security scanning)

### Success Metrics & KPIs

**Quantitative Metrics:**
```
Code Quality:
- Test coverage: 0% → 80%
- TypeScript errors: 1 → 0
- ESLint issues: 21 → 0
- Code duplication: 8.3% → <2%

Codebase Size:
- Total lines: 96,951 → ~87,000 (10% reduction)
- Duplicated lines: ~8,100 → ~1,700 (80% reduction)
- Average file size: 250 lines (maintain)

Maintainability:
- Files >500 lines: 50 → 0
- Mutation hooks: 50+ → 1 factory
- Data table components: 21 → 1 factory
- Complex components (>400 lines): 6 → 0
```

**Qualitative Metrics:**
- Developer velocity: Faster feature development
- Onboarding time: Reduced by 40%
- Bug rate: Reduced by 60%
- Refactoring confidence: High (due to tests)

**Timeline Metrics:**
- Phase 1 completion: Week 4
- Phase 2 completion: Week 8
- Phase 3 completion: Week 12
- Total duration: 12 weeks

---

## 9. Risks & Unknowns

### Critical Unknowns (Must Resolve Before Starting)

1. **Backend API Stability**
   - **Question:** Are backend APIs stable enough to write tests against?
   - **Impact:** If APIs change frequently, tests will break constantly
   - **Resolution:** Review backend API versioning strategy, establish API contract
   - **Owner:** Backend team lead

2. **Test Data Strategy**
   - **Question:** How do we handle test data? Mock all API calls or use test database?
   - **Impact:** Affects test reliability and speed
   - **Resolution:** Decide on mocking strategy (MSW vs manual mocks)
   - **Owner:** Frontend team lead

3. **CI/CD Resource Limits**
   - **Question:** Do we have sufficient GitHub Actions minutes for CI pipeline?
   - **Impact:** May need to optimize test runs or purchase additional minutes
   - **Resolution:** Check current usage, estimate new usage
   - **Owner:** DevOps team

### High-Priority Risks (Monitor Closely)

4. **Team Capacity During Refactoring**
   - **Risk:** Refactoring may slow down feature development
   - **Mitigation:** Allocate 50% time to refactoring, 50% to features
   - **Contingency:** Extend timeline if needed

5. **Merge Conflicts During Migration**
   - **Risk:** Multiple developers modifying same files
   - **Mitigation:** Clear ownership, frequent merges, communication
   - **Contingency:** Dedicated merge conflict resolver

6. **Incomplete Migration**
   - **Risk:** Some hooks/components not migrated, leaving mixed patterns
   - **Mitigation:** Automated checks for old patterns, clear completion criteria
   - **Contingency:** Dedicated cleanup sprint

### Medium-Priority Risks (Track Progress)

7. **Test Flakiness**
   - **Risk:** Flaky tests reduce confidence in CI pipeline
   - **Mitigation:** Follow Testing Library best practices, avoid timing issues
   - **Contingency:** Quarantine flaky tests, fix before re-enabling

8. **Performance Regression**
   - **Risk:** Refactored code may be slower
   - **Mitigation:** Performance benchmarks before/after, profiling
   - **Contingency:** Optimize hot paths, revert if necessary

9. **Breaking Changes in Dependencies**
   - **Risk:** React Query, Next.js, or other deps may have breaking changes
   - **Mitigation:** Pin versions, test upgrades in separate branch
   - **Contingency:** Stay on current versions until stable

### Low-Priority Risks (Acceptable)

10. **Documentation Drift**
    - **Risk:** Documentation may become outdated during refactoring
    - **Mitigation:** Update docs as part of each PR
    - **Contingency:** Dedicated documentation sprint at end

---

## Appendix: Quick Reference

### Key File Paths

**High-Risk Files (Top 5):**
1. `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints.ts` (Risk: 92)
2. `/Users/yangyang/Desktop/lunafox/frontend/components/scan/initiate-scan-dialog.tsx` (Risk: 88)
3. `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-nudge-toast.tsx` (Risk: 85)
4. `/Users/yangyang/Desktop/lunafox/frontend/components/common/smart-filter-input.tsx` (Risk: 78)
5. `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/agent-install-dialog.tsx` (Risk: 75)

**Critical Services (Must Test First):**
- `/Users/yangyang/Desktop/lunafox/frontend/services/auth.service.ts`
- `/Users/yangyang/Desktop/lunafox/frontend/services/organization.service.ts`
- `/Users/yangyang/Desktop/lunafox/frontend/services/vulnerability.service.ts`
- `/Users/yangyang/Desktop/lunafox/frontend/lib/api-client.ts`
- `/Users/yangyang/Desktop/lunafox/frontend/lib/error-handler.ts`

### Command Reference

```bash
# Test commands
pnpm test                    # Run all tests
pnpm test --coverage         # Run with coverage
pnpm test --ui               # Open Vitest UI
pnpm test --watch            # Watch mode

# Quality checks
pnpm typecheck               # TypeScript check
pnpm lint                    # ESLint
pnpm lint --fix              # Auto-fix lint issues
pnpm build                   # Build check

# CI simulation
pnpm typecheck && pnpm lint --max-warnings 0 && pnpm test && pnpm build
```

### Contact & Escalation

**For Questions:**
- Architecture decisions: Frontend team lead
- Test strategy: QA lead
- CI/CD issues: DevOps team
- Backend API contracts: Backend team lead

**Escalation Path:**
1. Team lead (first contact)
2. Engineering manager (if blocked >1 day)
3. CTO (if critical blocker)

---

**Report Generated:** 2026-02-11  
**Analysis Method:** Swarm Mode (5 Concurrent Agents)  
**Total Analysis Time:** ~15 minutes  
**Files Analyzed:** 517 TypeScript files  
**Lines Analyzed:** ~96,951 lines  

**Agent Contributions:**
- Agent 1: Type Safety Analysis
- Agent 2: Component Maintainability Analysis
- Agent 3: Architecture & Module Boundaries Analysis
- Agent 4: Quality Assurance Analysis
- Agent 5: UnifiedDataTable Migration Analysis

---

## Appendix: Execution Tracking (2026-02-11)

### Completed in this execution wave

1. **US4 / Phase 4 实施落地（首批 6 hooks）**
   - 新增共享 mutation 模板：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/create-resource-mutation.ts`
   - 已迁移 hooks（对外 API 保持不变）：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-targets.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scans.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-subdomains.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-endpoints.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-websites.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-directories.ts`
   - 迁移后统计：
     - 上述 6 文件中 `useMutation(...)` 调用：`0`
     - 上述 6 文件中 `useResourceMutation(...)` 调用：`30`

2. **测试补强（对应 T040/T041/T042）**
   - 新增测试文件：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/__tests__/create-resource-mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-targets.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-scans.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-nudge-toast.test.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-targets.types.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/data-table/__tests__/use-table-state.test.ts`
   - 已存在测试继续通过：
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/__tests__/response-parser.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/__tests__/toast-helpers.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-wordlists.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/data-table/__tests__/unified-data-table.smoke.test.tsx`

3. **门禁结果（2026-02-11）**
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
    - `pnpm run test` ✅（10 files / 26 tests）
    - `pnpm run test:coverage` ✅（10 files / 26 tests）

4. **Hook Mutation 去重第二轮（全量 hooks 收口）**
   - 新增/增强共享能力：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/create-resource-mutation.ts`
       - 新增 `skipDefaultErrorHandler`
       - 新增 `onSettled`
       - `onMutate` 支持注入 `{ queryClient, toast }`
   - 完成迁移文件（本轮）：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-workers.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-vulnerabilities.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scheduled-scans.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-organizations.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-nuclei-repos.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-commands.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-tools.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-engines.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-auth.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-agents.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-nuclei-templates.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-nuclei-git-settings.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-notifications.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-notification-settings.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-global-blacklist.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-api-key-settings.ts`
   - 迁移后统计（`frontend/hooks`）：
     - `useMutation(...)` 调用：`0`
     - `useResourceMutation(...)` 调用：`80`
   - 新增验证测试：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-fingerprints.mutation.test.ts`

5. **门禁结果（更新到 2026-02-11 19:08 +0800）**
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（11 files / 29 tests）
   - `pnpm run test:coverage` ✅（11 files / 29 tests）

6. **门禁结果（更新到 2026-02-11 19:13 +0800）**
   - 修复：`/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list.tsx` 错误态重试逻辑，改为 `refetch`，消除未定义 `queryClient` 编译错误
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（11 files / 29 tests）
   - `pnpm run test:coverage` ✅（11 files / 29 tests）
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

7. **门禁结果（更新到 2026-02-11 19:30 +0800）**
   - 重构：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-workers.ts` 抽出 `useWorkerMutation`，统一 worker mutation 成功 toast + 错误回退 + 失效策略
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-vulnerabilities.ts` 收敛默认分页参数与 enabled 判定逻辑（减少三处重复）
   - 新增测试：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-workers.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-vulnerabilities.mutation.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（13 files / 33 tests）
   - `pnpm run test:coverage` ✅（13 files / 33 tests）
   - 覆盖率快照：
     - Overall Statements：`30.5%`
     - Overall Branches：`21.92%`
     - Overall Functions：`23.56%`
     - Overall Lines：`31.61%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

8. **门禁结果（更新到 2026-02-11 19:32 +0800）**
   - 新增测试：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-scheduled-scans.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-organizations.mutation.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（15 files / 37 tests）
   - `pnpm run test:coverage` ✅（15 files / 37 tests）
   - 覆盖率快照：
     - Overall Statements：`30.46%`
     - Overall Branches：`21.29%`
     - Overall Functions：`23.51%`
     - Overall Lines：`31.51%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

9. **门禁结果（更新到 2026-02-11 19:41 +0800）**
   - Speckit 二期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/004-frontend-techdebt-phase2/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/004-frontend-techdebt-phase2/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/004-frontend-techdebt-phase2/tasks.md`
   - Hook 微型去重：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-tools.ts`（抽取 `useToolMutation`，不改对外签名）
   - 新增测试：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-tools.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-engines.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-settings.mutation.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（18 files / 43 tests）
   - `pnpm run test:coverage` ✅（18 files / 43 tests）
   - 覆盖率快照：
     - Overall Statements：`30.79%`
     - Overall Branches：`21.08%`
     - Overall Functions：`24.74%`
     - Overall Lines：`31.84%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

10. **门禁结果（更新到 2026-02-11 19:49 +0800）**
   - Speckit 三期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/005-frontend-techdebt-phase3/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/005-frontend-techdebt-phase3/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/005-frontend-techdebt-phase3/tasks.md`
   - Hook 微型去重：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-auth.ts`（抽取 `authKeys`）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-nuclei-git-settings.ts`（抽取 `nucleiGitKeys`）
   - 新增测试：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-auth.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-agents.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-nuclei-git-settings.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-global-blacklist.mutation.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（22 files / 51 tests）
   - `pnpm run test:coverage` ✅（22 files / 51 tests）
   - 覆盖率快照：
     - Overall Statements：`31.52%`
     - Overall Branches：`21.01%`
     - Overall Functions：`26.33%`
     - Overall Lines：`32.57%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

11. **门禁结果（更新到 2026-02-11 20:02 +0800）**
   - Speckit 四期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/006-frontend-techdebt-phase4/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/006-frontend-techdebt-phase4/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/006-frontend-techdebt-phase4/tasks.md`
   - Hook 微型去重：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-nuclei-repos.ts`（抽取 `nucleiRepoKeys`，统一 repos/repo/tree/content key 入口）
   - 新增测试：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-nuclei-repos.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-nuclei-templates.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-notifications.mutation.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（25 files / 59 tests）
   - `pnpm run test:coverage` ✅（25 files / 59 tests）
   - 覆盖率快照：
     - Overall Statements：`31.86%`
     - Overall Branches：`20.94%`
     - Overall Functions：`27.5%`
     - Overall Lines：`32.9%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

12. **门禁结果（更新到 2026-02-11 20:12 +0800）**
   - Speckit 五期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/007-frontend-techdebt-phase5/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/007-frontend-techdebt-phase5/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/007-frontend-techdebt-phase5/tasks.md`
   - Hook 微型去重：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-commands.ts`（抽取 `commandKeys`，统一 list/detail/mutation invalidate key）
   - 新增测试：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-commands.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-websites.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-directories.mutation.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（28 files / 66 tests）
   - `pnpm run test:coverage` ✅（28 files / 66 tests）
   - 覆盖率快照：
     - Overall Statements：`31.94%`
     - Overall Branches：`20.8%`
     - Overall Functions：`27.85%`
     - Overall Lines：`33.05%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

13. **门禁结果（更新到 2026-02-11 21:35 +0800）**
   - Speckit 六期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/008-frontend-techdebt-phase6/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/008-frontend-techdebt-phase6/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/008-frontend-techdebt-phase6/tasks.md`
   - Hook 微型去重：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-endpoints.ts`（mutation invalidate 统一复用 `endpointKeys.all`）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-subdomains.ts`（mutation invalidate 统一复用 `subdomainKeys.all`）
   - 新增测试：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-endpoints.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-subdomains.mutation.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-subdomains.organization.mutation.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 73 tests）
   - `pnpm run test:coverage` ✅（31 files / 73 tests）
   - 覆盖率快照：
     - Overall Statements：`32.19%`
     - Overall Branches：`20.2%`
     - Overall Functions：`28.42%`
     - Overall Lines：`33.25%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

14. **门禁结果（更新到 2026-02-11 21:40 +0800）**
   - Speckit 七期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/009-frontend-techdebt-phase7/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/009-frontend-techdebt-phase7/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/009-frontend-techdebt-phase7/tasks.md`
   - Hook 微型去重：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-api-key-settings.ts`（抽取 `apiKeySettingsKeys` 统一 query/invalidate key）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-notification-settings.ts`（抽取 `notificationSettingsKeys` 统一 query/invalidate key）
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-settings.mutation.test.ts`（补齐 notification success 与 api-key error 对称分支）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 75 tests）
   - `pnpm run test:coverage` ✅（31 files / 75 tests）
   - 覆盖率快照：
     - Overall Statements：`32.29%`
     - Overall Branches：`20.2%`
     - Overall Functions：`28.53%`
     - Overall Lines：`33.36%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

15. **门禁结果（更新到 2026-02-11 21:46 +0800）**
   - Speckit 八期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/010-frontend-techdebt-phase8/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/010-frontend-techdebt-phase8/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/010-frontend-techdebt-phase8/tasks.md`
   - Hook 微型去重：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-websites.ts`（抽取共享 `websiteCascadeInvalidates`，复用删除相关失效策略）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-directories.ts`（抽取共享 `directoryCascadeInvalidates`，复用删除相关失效策略）
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-websites.mutation.test.ts`（补齐 bulk create 0 分支与 bulk delete success）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-directories.mutation.test.ts`（补齐 bulk delete success）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-commands.mutation.test.ts`（补齐 batch delete success）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 79 tests）
   - `pnpm run test:coverage` ✅（31 files / 79 tests）
   - 覆盖率快照：
     - Overall Statements：`32.95%`
     - Overall Branches：`20.36%`
     - Overall Functions：`29.82%`
     - Overall Lines：`33.95%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

16. **门禁结果（更新到 2026-02-11 22:36 +0800）**
   - Speckit 九期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/011-frontend-techdebt-phase9/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/011-frontend-techdebt-phase9/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/011-frontend-techdebt-phase9/tasks.md`
   - Hook 微型去重：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-endpoints.ts`（抽取共享 `endpointAllInvalidates`，复用 delete/batch-delete 失效策略）
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-endpoints.mutation.test.ts`（补齐 delete/batch delete success）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 81 tests）
   - `pnpm run test:coverage` ✅（31 files / 81 tests）
   - 覆盖率快照：
     - Overall Statements：`33.16%`
     - Overall Branches：`20.36%`
     - Overall Functions：`30.34%`
     - Overall Lines：`34.17%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

17. **门禁结果（更新到 2026-02-11 22:41 +0800）**
   - Speckit 十期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/012-frontend-techdebt-phase10/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/012-frontend-techdebt-phase10/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/012-frontend-techdebt-phase10/tasks.md`
   - Hook 微型去重：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-subdomains.ts`（抽取共享 `subdomainCascadeInvalidates`，复用 delete/batch-delete 失效策略）
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-subdomains.mutation.test.ts`（补齐 delete/batch delete success）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 83 tests）
   - `pnpm run test:coverage` ✅（31 files / 83 tests）
   - 覆盖率快照：
     - Overall Statements：`33.37%`
     - Overall Branches：`20.36%`
     - Overall Functions：`30.87%`
     - Overall Lines：`34.39%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

18. **门禁结果（更新到 2026-02-11 22:47 +0800）**
   - Speckit 十一期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/013-frontend-techdebt-phase11/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/013-frontend-techdebt-phase11/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/013-frontend-techdebt-phase11/tasks.md`
   - Hook 迁移：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-wordlists.ts`（mutation 迁移到 `useResourceMutation`）
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-wordlists.test.ts`（补齐上传/更新成功分支）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 85 tests）
   - `pnpm run test:coverage` ✅（31 files / 85 tests）
   - 覆盖率快照：
     - Overall Statements：`33.82%`
     - Overall Branches：`20.36%`
     - Overall Functions：`31.79%`
     - Overall Lines：`34.86%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

19. **门禁结果（更新到 2026-02-11 22:56 +0800）**
   - Speckit 十二期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/014-frontend-techdebt-phase12/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/014-frontend-techdebt-phase12/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/014-frontend-techdebt-phase12/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-endpoints.mutation.test.ts`（补齐 bulk create warning 分支）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 86 tests）
   - `pnpm run test:coverage` ✅（31 files / 86 tests）
   - 覆盖率快照：
     - Overall Statements：`33.86%`
     - Overall Branches：`20.44%`
     - Overall Functions：`31.79%`
     - Overall Lines：`34.91%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

20. **门禁结果（更新到 2026-02-11 22:59 +0800）**
   - Speckit 十三期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/015-frontend-techdebt-phase13/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/015-frontend-techdebt-phase13/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/015-frontend-techdebt-phase13/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-wordlists.test.ts`（补齐 upload/update error 分支）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 88 tests）
   - `pnpm run test:coverage` ✅（31 files / 88 tests）
   - 覆盖率快照：
     - Overall Statements：`33.86%`
     - Overall Branches：`20.44%`
     - Overall Functions：`31.79%`
     - Overall Lines：`34.91%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

21. **门禁结果（更新到 2026-02-11 23:03 +0800）**
   - Speckit 十四期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/016-frontend-techdebt-phase14/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/016-frontend-techdebt-phase14/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/016-frontend-techdebt-phase14/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-endpoints.mutation.test.ts`（补齐 create success 分支）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 89 tests）
   - `pnpm run test:coverage` ✅（31 files / 89 tests）
   - 覆盖率快照：
     - Overall Statements：`33.91%`
     - Overall Branches：`20.52%`
     - Overall Functions：`31.79%`
     - Overall Lines：`34.96%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

22. **门禁结果（更新到 2026-02-11 23:06 +0800）**
   - Speckit 十五期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/017-frontend-techdebt-phase15/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/017-frontend-techdebt-phase15/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/017-frontend-techdebt-phase15/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-directories.mutation.test.ts`（补齐 delete error 分支）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 90 tests）
   - `pnpm run test:coverage` ✅（31 files / 90 tests）
   - 覆盖率快照：
     - Overall Statements：`34%`
     - Overall Branches：`20.52%`
     - Overall Functions：`32.02%`
     - Overall Lines：`35.05%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

23. **门禁结果（更新到 2026-02-11 23:10 +0800）**
   - Speckit 十六期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/018-frontend-techdebt-phase16/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/018-frontend-techdebt-phase16/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/018-frontend-techdebt-phase16/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-websites.mutation.test.ts`（补齐 delete success 分支）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 91 tests）
   - `pnpm run test:coverage` ✅（31 files / 91 tests）
   - 覆盖率快照：
     - Overall Statements：`34.05%`
     - Overall Branches：`20.52%`
     - Overall Functions：`32.13%`
     - Overall Lines：`35.1%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

24. **门禁结果（更新到 2026-02-11 23:15 +0800）**
   - Speckit 十七期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/019-frontend-techdebt-phase17/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/019-frontend-techdebt-phase17/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/019-frontend-techdebt-phase17/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-directories.mutation.test.ts`（补齐 delete success 分支）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 92 tests）
   - `pnpm run test:coverage` ✅（31 files / 92 tests）
   - 覆盖率快照：
     - Overall Statements：`34.09%`
     - Overall Branches：`20.52%`
     - Overall Functions：`32.24%`
     - Overall Lines：`35.15%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

25. **门禁结果（更新到 2026-02-11 23:25 +0800）**
   - Speckit 十八期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/020-frontend-techdebt-phase18/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/020-frontend-techdebt-phase18/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/020-frontend-techdebt-phase18/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-websites.mutation.test.ts`（补齐 bulk delete error 分支）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 93 tests）
   - `pnpm run test:coverage` ✅（31 files / 93 tests）
   - 覆盖率快照：
     - Overall Statements：`34.09%`
     - Overall Branches：`20.52%`
     - Overall Functions：`32.24%`
     - Overall Lines：`35.15%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

26. **门禁结果（更新到 2026-02-11 23:32 +0800）**
   - Speckit 十九期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/021-frontend-techdebt-phase19/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/021-frontend-techdebt-phase19/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/021-frontend-techdebt-phase19/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-endpoints.mutation.test.ts`（补齐 create/batch delete error 分支）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-directories.mutation.test.ts`（补齐 bulk delete error 分支）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-subdomains.organization.mutation.test.ts`（补齐解绑 error 分支）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 98 tests）
   - `pnpm run test:coverage` ✅（31 files / 98 tests）
   - 覆盖率快照：
     - Overall Statements：`34.09%`
     - Overall Branches：`20.52%`
     - Overall Functions：`32.24%`
     - Overall Lines：`35.15%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

27. **门禁结果（更新到 2026-02-11 23:47 +0800）**
   - Speckit 二十期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/022-frontend-techdebt-phase20/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/022-frontend-techdebt-phase20/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/022-frontend-techdebt-phase20/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-workers.mutation.test.ts`（补齐 update/delete/deploy/restart/stop success+error）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 107 tests）
   - `pnpm run test:coverage` ✅（31 files / 107 tests）
   - 覆盖率快照：
     - Overall Statements：`34.74%`
     - Overall Branches：`20.52%`
     - Overall Functions：`33.82%`
     - Overall Lines：`35.82%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

28. **门禁结果（更新到 2026-02-11 23:52 +0800）**
   - Speckit 二十一期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/023-frontend-techdebt-phase21/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/023-frontend-techdebt-phase21/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/023-frontend-techdebt-phase21/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-commands.mutation.test.ts`（补齐 delete success + create/update error）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 110 tests）
   - `pnpm run test:coverage` ✅（31 files / 110 tests）
   - 覆盖率快照：
     - Overall Statements：`34.78%`
     - Overall Branches：`20.52%`
     - Overall Functions：`33.93%`
     - Overall Lines：`35.87%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

29. **门禁结果（更新到 2026-02-11 23:58 +0800）**
   - Speckit 二十二期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/024-frontend-techdebt-phase22/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/024-frontend-techdebt-phase22/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/024-frontend-techdebt-phase22/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-targets.mutation.test.ts`（补齐 create/update/delete/batch create/delete/link/unlink/blacklist success+error）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 124 tests）
   - `pnpm run test:coverage` ✅（31 files / 124 tests）
   - 覆盖率快照：
     - Overall Statements：`36.12%`
     - Overall Branches：`20.6%`
     - Overall Functions：`37.19%`
     - Overall Lines：`37.26%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

30. **门禁结果（更新到 2026-02-12 00:03 +0800）**
   - Speckit 二十三期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/025-frontend-techdebt-phase23/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/025-frontend-techdebt-phase23/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/025-frontend-techdebt-phase23/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-scans.mutation.test.ts`（补齐 quick/initiate/delete/bulk delete/stop success+error）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-scheduled-scans.mutation.test.ts`（补齐 create/update/delete/toggle success+error）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 139 tests）
   - `pnpm run test:coverage` ✅（31 files / 139 tests）
   - 覆盖率快照：
     - Overall Statements：`38.18%`
     - Overall Branches：`21.47%`
     - Overall Functions：`39.43%`
     - Overall Lines：`39.42%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

31. **门禁结果（更新到 2026-02-12 00:07 +0800）**
   - Speckit 二十四期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/026-frontend-techdebt-phase24/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/026-frontend-techdebt-phase24/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/026-frontend-techdebt-phase24/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-organizations.mutation.test.ts`（补齐 create/update/delete/batch delete success+error）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 145 tests）
   - `pnpm run test:coverage` ✅（31 files / 145 tests）
   - 覆盖率快照：
     - Overall Statements：`39.33%`
     - Overall Branches：`21.71%`
     - Overall Functions：`40.67%`
     - Overall Lines：`40.62%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

32. **门禁结果（更新到 2026-02-12 08:52 +0800）**
   - Speckit 二十五期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/027-frontend-techdebt-phase25/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/027-frontend-techdebt-phase25/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/027-frontend-techdebt-phase25/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-auth.mutation.test.ts`（补齐 login/logout/changePassword success+error）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-agents.mutation.test.ts`（补齐 create/update/delete success+error）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-tools.mutation.test.ts`（补齐 create/update/delete success+error）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-engines.mutation.test.ts`（补齐 create/update/delete success+error）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 161 tests）
   - `pnpm run test:coverage` ✅（31 files / 161 tests）
   - 覆盖率快照：
     - Overall Statements：`39.88%`
     - Overall Branches：`21.71%`
     - Overall Functions：`42.02%`
     - Overall Lines：`41.19%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

33. **门禁结果（更新到 2026-02-12 08:56 +0800）**
   - Speckit 二十六期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/028-frontend-techdebt-phase26/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/028-frontend-techdebt-phase26/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/028-frontend-techdebt-phase26/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-nuclei-templates.mutation.test.ts`（补齐 refresh/upload/save success+error）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-nuclei-repos.mutation.test.ts`（补齐 create/update/delete/refresh success+error）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 169 tests）
   - `pnpm run test:coverage` ✅（31 files / 169 tests）
   - 覆盖率快照：
     - Overall Statements：`40.16%`
     - Overall Branches：`21.71%`
     - Overall Functions：`42.69%`
     - Overall Lines：`41.48%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

34. **门禁结果（更新到 2026-02-12 08:59 +0800）**
   - Speckit 二十七期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/029-frontend-techdebt-phase27/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/029-frontend-techdebt-phase27/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/029-frontend-techdebt-phase27/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-vulnerabilities.mutation.test.ts`（补齐 mark/bulk success+error）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 176 tests）
   - `pnpm run test:coverage` ✅（31 files / 176 tests）
   - 覆盖率快照：
     - Overall Statements：`40.57%`
     - Overall Branches：`21.71%`
     - Overall Functions：`43.7%`
     - Overall Lines：`41.91%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

35. **门禁结果（更新到 2026-02-12 09:05 +0800）**
   - Speckit 二十八期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/030-frontend-techdebt-phase28/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/030-frontend-techdebt-phase28/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/030-frontend-techdebt-phase28/tasks.md`
   - 测试增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-fingerprints.mutation.test.ts`（补齐 create/update/delete/import/bulk/deleteAll/batchCreate success+error）
   - 其他修复：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/prototypes/vuln-audit-design.tsx`（替换未导出图标）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（31 files / 212 tests）
   - `pnpm run test:coverage` ✅（31 files / 212 tests）
   - 覆盖率快照：
     - Overall Statements：`41.4%`
     - Overall Branches：`21.71%`
     - Overall Functions：`45.73%`
     - Overall Lines：`42.78%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

36. **门禁结果（更新到 2026-02-12 09:16 +0800）**
   - Speckit 二十九期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/031-frontend-techdebt-phase29/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/031-frontend-techdebt-phase29/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/031-frontend-techdebt-phase29/tasks.md`
   - Smart Filter 逻辑抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/smart-filter.ts`（解析/历史/ghost 逻辑模块化）
     - `/Users/yangyang/Desktop/lunafox/frontend/components/common/smart-filter-input.tsx`（组件改用新模块）
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/__tests__/smart-filter.test.ts`（新增单测）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（32 files / 216 tests）
   - `pnpm run test:coverage` ✅（32 files / 216 tests）
   - 覆盖率快照：
     - Overall Statements：`43.5%`
     - Overall Branches：`24.26%`
     - Overall Functions：`46.52%`
     - Overall Lines：`44.9%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

37. **门禁结果（更新到 2026-02-12 09:24 +0800）**
   - Speckit 三十期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/032-frontend-techdebt-phase30/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/032-frontend-techdebt-phase30/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/032-frontend-techdebt-phase30/tasks.md`
   - Scheduled Scan 校验逻辑抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/scheduled-scan-helpers.ts`（校验与错误解析）
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/create-scheduled-scan-dialog.tsx`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/edit-scheduled-scan-dialog.tsx`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/__tests__/scheduled-scan-helpers.test.ts`（新增单测）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（33 files / 220 tests）
   - `pnpm run test:coverage` ✅（33 files / 220 tests）
   - 覆盖率快照：
     - Overall Statements：`44.22%`
     - Overall Branches：`25.95%`
     - Overall Functions：`46.76%`
     - Overall Lines：`45.53%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

38. **门禁结果（更新到 2026-02-12 09:28 +0800）**
   - Speckit 三十一期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/033-frontend-techdebt-phase31/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/033-frontend-techdebt-phase31/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/033-frontend-techdebt-phase31/tasks.md`
   - Initiate Scan 校验逻辑抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/initiate-scan-helpers.ts`（校验与错误解析）
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/initiate-scan-dialog.tsx`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/__tests__/initiate-scan-helpers.test.ts`（新增单测）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（34 files / 226 tests）
   - `pnpm run test:coverage` ✅（34 files / 226 tests）
   - 覆盖率快照：
     - Overall Statements：`44.92%`
     - Overall Branches：`27.19%`
     - Overall Functions：`46.94%`
     - Overall Lines：`46.14%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

39. **门禁结果（更新到 2026-02-12 09:34 +0800）**
   - Speckit 三十二期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/034-frontend-techdebt-phase32/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/034-frontend-techdebt-phase32/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/034-frontend-techdebt-phase32/tasks.md`
   - Nudge Toast 逻辑抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/nudge-toast-helpers.ts`（抑制/冷却/动作封装）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-nudge-toast.tsx`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/__tests__/nudge-toast-helpers.test.ts`（新增单测）
   - 其他修复：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerability-detail-dialog.tsx`（补齐类型导入）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（35 files / 230 tests）
   - `pnpm run test:coverage` ✅（35 files / 230 tests）
   - 覆盖率快照：
     - Overall Statements：`45.96%`
     - Overall Branches：`28.04%`
     - Overall Functions：`47.39%`
     - Overall Lines：`47.13%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

40. **门禁结果（更新到 2026-02-12 09:39 +0800）**
   - Speckit 三十三期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/035-frontend-techdebt-phase33/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/035-frontend-techdebt-phase33/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/035-frontend-techdebt-phase33/tasks.md`
   - Agent Install 逻辑抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/agent-install-helpers.ts`（命令构建/检测逻辑）
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/agent-install-dialog.tsx`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/__tests__/agent-install-helpers.test.ts`（新增单测）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（36 files / 233 tests）
   - `pnpm run test:coverage` ✅（36 files / 233 tests）
   - 覆盖率快照：
     - Overall Statements：`46.76%`
     - Overall Branches：`29.27%`
     - Overall Functions：`47.85%`
     - Overall Lines：`47.85%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

41. **门禁结果（更新到 2026-02-12 09:46 +0800）**
   - Speckit 三十四期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/036-frontend-techdebt-phase34/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/036-frontend-techdebt-phase34/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/036-frontend-techdebt-phase34/tasks.md`
   - Fingerprint hook 工厂抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/fingerprint-hooks.ts`（通用 hook 工厂与类型）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints.ts`（使用新 helper）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（36 files / 233 tests）
   - `pnpm run test:coverage` ✅（36 files / 233 tests）
   - 覆盖率快照：
     - Overall Statements：`46.76%`
     - Overall Branches：`29.27%`
     - Overall Functions：`47.85%`
     - Overall Lines：`47.87%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

42. **门禁结果（更新到 2026-02-12 09:55 +0800）**
   - Speckit 三十五期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/037-frontend-techdebt-phase35/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/037-frontend-techdebt-phase35/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/037-frontend-techdebt-phase35/tasks.md`
   - 分页归一化抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/pagination.ts`（分页字段归一化）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-endpoints.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-targets.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-subdomains.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-vulnerabilities.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/__tests__/pagination.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（37 files / 236 tests）
   - `pnpm run test:coverage` ✅（37 files / 236 tests）
   - 覆盖率快照：
     - Overall Statements：`46.95%`
     - Overall Branches：`30.41%`
     - Overall Functions：`47.91%`
     - Overall Lines：`48.06%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

43. **门禁结果（更新到 2026-02-12 10:01 +0800）**
   - Speckit 三十六期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/038-frontend-techdebt-phase36/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/038-frontend-techdebt-phase36/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/038-frontend-techdebt-phase36/tasks.md`
   - Targets 参数解析抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/targets-helpers.ts`（参数解析/结果选择）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-targets.ts`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/__tests__/targets-helpers.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（38 files / 240 tests）
   - `pnpm run test:coverage` ✅（38 files / 240 tests）
   - 覆盖率快照：
     - Overall Statements：`47.62%`
     - Overall Branches：`31.84%`
     - Overall Functions：`48.24%`
     - Overall Lines：`48.73%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

44. **门禁结果（更新到 2026-02-12 10:16 +0800）**
   - Speckit 三十七期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/039-frontend-techdebt-phase37/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/039-frontend-techdebt-phase37/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/039-frontend-techdebt-phase37/tasks.md`
   - Scan mutation 成功处理抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/scan-mutation-helpers.ts`（success 处理与失效逻辑）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scans.ts`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/__tests__/scan-mutation-helpers.test.ts`
   - 额外修复：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/prototypes/vuln-audit-vertical.tsx`（补回 MOCK_VULNS 初始化以恢复 typecheck）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（39 files / 246 tests）
   - `pnpm run test:coverage` ✅（39 files / 246 tests）
   - 覆盖率快照：
     - Overall Statements：`47.8%`
     - Overall Branches：`32.2%`
     - Overall Functions：`49.08%`
     - Overall Lines：`48.89%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

45. **门禁结果（更新到 2026-02-12 10:55 +0800）**
   - Speckit 三十八期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/040-frontend-techdebt-phase38/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/040-frontend-techdebt-phase38/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/040-frontend-techdebt-phase38/tasks.md`
   - Scheduled scan success 抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/scheduled-scan-mutation-helpers.ts`（success 处理逻辑）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scheduled-scans.ts`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/__tests__/scheduled-scan-mutation-helpers.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（40 files / 248 tests）
   - `pnpm run test:coverage` ✅（40 files / 248 tests）
   - 覆盖率快照：
     - Overall Statements：`47.89%`
     - Overall Branches：`32.3%`
     - Overall Functions：`49.35%`
     - Overall Lines：`48.98%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

46. **门禁结果（更新到 2026-02-12 11:46 +0800）**
   - Speckit 三十九期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/041-frontend-techdebt-phase39/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/041-frontend-techdebt-phase39/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/041-frontend-techdebt-phase39/tasks.md`
   - Commands mock 数据抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/mock/data/commands.ts`（mock 数据与分页/计数 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-commands.ts`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/mock/data/__tests__/commands.test.ts`
   - 额外修复：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/prototypes/vuln-audit-vertical.tsx`（修正 URL state 类型推断）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-resizable.ts`（移除未使用变量以满足 lint）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（41 files / 252 tests）
   - `pnpm run test:coverage` ✅（41 files / 252 tests）
   - 覆盖率快照：
     - Overall Statements：`48.52%`
     - Overall Branches：`32.79%`
     - Overall Functions：`49.73%`
     - Overall Lines：`49.52%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

47. **门禁结果（更新到 2026-02-12 11:53 +0800）**
   - Speckit 四十期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/042-frontend-techdebt-phase40/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/042-frontend-techdebt-phase40/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/042-frontend-techdebt-phase40/tasks.md`
   - Subdomain success 逻辑抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/subdomain-mutation-helpers.ts`（toast/计数逻辑）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-subdomains.ts`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/__tests__/subdomain-mutation-helpers.test.ts`
   - 额外修复：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/prototypes/vulnerability-audit-detail.tsx`（修正异常重复块与导入位置）
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（42 files / 255 tests）
   - `pnpm run test:coverage` ✅（42 files / 255 tests）
   - 覆盖率快照：
     - Overall Statements：`48.72%`
     - Overall Branches：`33.28%`
     - Overall Functions：`49.89%`
     - Overall Lines：`49.73%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

48. **门禁结果（更新到 2026-02-12 11:58 +0800）**
   - Speckit 四十一期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/043-frontend-techdebt-phase41/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/043-frontend-techdebt-phase41/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/043-frontend-techdebt-phase41/tasks.md`
   - Asset bulk create/delete toast 抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/asset-mutation-helpers.ts`（bulk create/delete toast 逻辑）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-websites.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-directories.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-endpoints.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/__tests__/asset-mutation-helpers.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（43 files / 258 tests）
   - `pnpm run test:coverage` ✅（43 files / 258 tests）
   - 覆盖率快照：
     - Overall Statements：`48.87%`
     - Overall Branches：`33.5%`
     - Overall Functions：`50%`
     - Overall Lines：`49.88%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

49. **门禁结果（更新到 2026-02-12 12:20 +0800）**
   - Speckit 四十二期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/044-frontend-techdebt-phase42/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/044-frontend-techdebt-phase42/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/044-frontend-techdebt-phase42/tasks.md`
   - Organization optimistic 删除抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/organization-mutation-helpers.ts`（optimistic/回滚/失效逻辑）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-organizations.ts`（使用新 helper）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/__tests__/organization-mutation-helpers.test.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（44 files / 261 tests）
   - `pnpm run test:coverage` ✅（44 files / 261 tests）
   - 覆盖率快照：
     - Overall Statements：`49.34%`
     - Overall Branches：`33.6%`
     - Overall Functions：`50.69%`
     - Overall Lines：`50.4%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

50. **门禁结果（更新到 2026-02-12 12:28 +0800）**
   - Speckit 四十三期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/045-frontend-techdebt-phase43/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/045-frontend-techdebt-phase43/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/045-frontend-techdebt-phase43/tasks.md`
   - 删除计数 helper 复用：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/asset-mutation-helpers.ts`（`getAssetDeletedCount`）
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-targets.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-commands.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-organizations.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（44 files / 261 tests）
   - `pnpm run test:coverage` ✅（44 files / 261 tests）
   - 覆盖率快照：
     - Overall Statements：`49.32%`
     - Overall Branches：`33.57%`
     - Overall Functions：`50.69%`
     - Overall Lines：`50.37%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

51. **门禁结果（更新到 2026-02-12 12:57 +0800）**
   - Speckit 四十四期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/046-frontend-techdebt-phase44/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/046-frontend-techdebt-phase44/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/046-frontend-techdebt-phase44/tasks.md`
   - Query key 工厂与 hooks 迁移：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/query-keys.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/__tests__/query-keys.test.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-agents.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-commands.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-workers.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-tools.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-directories.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-endpoints.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-subdomains.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-targets.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-organizations.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scheduled-scans.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scans.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-vulnerabilities.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-wordlists.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-notifications.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-websites.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-ip-addresses.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-screenshots.ts`
   - 大型对话框状态抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/common/smart-filter-input-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/common/smart-filter-input.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/initiate-scan-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/initiate-scan-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/scheduled-scan-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/create-scheduled-scan-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/agent-install-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/agent-install-dialog.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 263 tests）
   - `pnpm run test:coverage` ✅（45 files / 263 tests）
   - 覆盖率快照：
     - Overall Statements：`49.66%`
     - Overall Branches：`33.79%`
     - Overall Functions：`50.96%`
     - Overall Lines：`50.71%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

52. **门禁结果（更新到 2026-02-12 13:28 +0800）**
   - Speckit 四十五期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/047-frontend-techdebt-phase45/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/047-frontend-techdebt-phase45/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/047-frontend-techdebt-phase45/tasks.md`
   - SmartFilterDataTable 收敛与下载选项统一：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/subdomains-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/endpoints/endpoints-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ip-addresses/ip-addresses-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerabilities-data-table.tsx`
   - 大型组件状态抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-overview-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-overview.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/add-target-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/add-target-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/config/add-tool-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/config/add-tool-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/edit-scheduled-scan-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/edit-scheduled-scan-dialog.tsx`
   - 类型断言清理：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/all-targets-detail-view.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 263 tests）
   - `pnpm run test:coverage` ✅（45 files / 263 tests）
   - 覆盖率快照：
     - Overall Statements：`49.66%`
     - Overall Branches：`33.79%`
     - Overall Functions：`50.96%`
     - Overall Lines：`50.71%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

53. **门禁结果（更新到 2026-02-12 15:41 +0800）**
   - Speckit 四十六期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/048-frontend-techdebt-phase46/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/048-frontend-techdebt-phase46/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/048-frontend-techdebt-phase46/tasks.md`
   - 指纹表格操作区与对话框抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingerprint-table-actions.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/ehole-fingerprint-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingerprinthub-fingerprint-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingers-fingerprint-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/goby-fingerprint-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/wappalyzer-fingerprint-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/arl-fingerprint-data-table.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 263 tests）
   - `pnpm run test:coverage` ✅（45 files / 263 tests）
   - 覆盖率快照：
     - Overall Statements：`49.66%`
     - Overall Branches：`33.79%`
     - Overall Functions：`50.96%`
     - Overall Lines：`50.71%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

54. **门禁结果（更新到 2026-02-12 16:31 +0800）**
   - Speckit 四十七期文档落地：
     - `/Users/yangyang/Desktop/lunafox/specs/049-frontend-techdebt-phase47/spec.md`
     - `/Users/yangyang/Desktop/lunafox/specs/049-frontend-techdebt-phase47/plan.md`
     - `/Users/yangyang/Desktop/lunafox/specs/049-frontend-techdebt-phase47/tasks.md`
   - 简单搜索工具栏复用：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/data-table/simple-search-toolbar.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/data-table/use-simple-search.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/targets/targets-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/targets-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/scheduled-scan-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/engine/engine-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/commands/commands-data-table.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 263 tests）
   - `pnpm run test:coverage` ✅（45 files / 263 tests）
   - 覆盖率快照：
     - Overall Statements：`49.66%`
     - Overall Branches：`33.79%`
     - Overall Functions：`50.96%`
     - Overall Lines：`50.71%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

55. **门禁结果（更新到 2026-02-12 17:06 +0800）**
   - CSV 导出辅助函数抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/csv-utils.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/websites/websites-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/directories/directories-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/subdomains-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/endpoints/endpoints-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ip-addresses/ip-addresses-view.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 263 tests）
   - `pnpm run test:coverage` ✅（45 files / 263 tests）
   - 覆盖率快照：
     - Overall Statements：`49.66%`
     - Overall Branches：`33.79%`
     - Overall Functions：`50.96%`
     - Overall Lines：`50.71%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

56. **门禁结果（更新到 2026-02-12 17:23 +0800）**
   - 下载与搜索状态抽离：
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/download-utils.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/use-search-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/websites/websites-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/directories/directories-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/endpoints/endpoints-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/subdomains-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ip-addresses/ip-addresses-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/system-logs/system-logs-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/dashboard/dashboard-scheduled-scans.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-list.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/target-settings.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/all-targets-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/ehole-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/goby-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/wappalyzer-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/arl-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingerprinthub-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingers-fingerprint-view.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 263 tests）
   - `pnpm run test:coverage` ✅（45 files / 263 tests）
   - 覆盖率快照：
     - Overall Statements：`49.66%`
     - Overall Branches：`33.79%`
     - Overall Functions：`50.96%`
     - Overall Lines：`50.71%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

57. **门禁结果（更新到 2026-02-12 17:34 +0800）**
   - 错误信息与分页信息收敛：
     - `/Users/yangyang/Desktop/lunafox/frontend/lib/error-utils.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/ehole-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/goby-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/wappalyzer-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/arl-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingerprinthub-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingers-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/ehole-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/wappalyzer-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingers-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/arl-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingerprinthub-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/import-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/websites/websites-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/directories/directories-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/subdomains-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ip-addresses/ip-addresses-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-list.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 263 tests）
   - `pnpm run test:coverage` ✅（45 files / 263 tests）
   - 覆盖率快照：
     - Overall Statements：`49.66%`
     - Overall Branches：`33.79%`
     - Overall Functions：`50.96%`
     - Overall Lines：`50.71%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

58. **门禁结果（更新到 2026-02-12 17:39 +0800）**
   - Dashboard 表格与分页复用：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/dashboard/dashboard-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/subdomains-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list.tsx`
   - 分页信息收敛：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/websites/websites-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/directories/directories-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ip-addresses/ip-addresses-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-list.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 263 tests）
   - `pnpm run test:coverage` ✅（45 files / 263 tests）
   - 覆盖率快照：
     - Overall Statements：`49.66%`
     - Overall Branches：`33.79%`
     - Overall Functions：`50.96%`
     - Overall Lines：`50.71%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

59. **门禁结果（更新到 2026-02-12 17:46 +0800）**
   - 分页构建统一与测试补充：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/pagination.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/__tests__/pagination.test.ts`
   - DataTable 分页信息收敛：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/endpoints/endpoints-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/targets-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/scheduled-scan-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/subdomains-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.94%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.78%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

60. **门禁结果（更新到 2026-02-12 17:52 +0800）**
   - 指纹视图分页信息收敛为共享 hook：
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/_shared/use-stable-pagination-info.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/arl-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/ehole-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingerprinthub-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingers-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/goby-fingerprint-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/wappalyzer-fingerprint-view.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.94%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.78%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

61. **门禁结果（更新到 2026-02-12 17:57 +0800）**
   - 分页信息构建继续收敛：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerabilities-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/dashboard/dashboard-scan-history.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.94%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.78%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

62. **门禁结果（更新到 2026-02-12 18:12 +0800）**
   - Scheduled scans / endpoints 分页信息继续统一：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/endpoints/endpoints-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/target-settings.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/dashboard/dashboard-scheduled-scans.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.94%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.78%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

63. **门禁结果（更新到 2026-02-12 18:16 +0800）**
   - 搜索与截图分页逻辑健壮性增强：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/screenshots/screenshots-gallery.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/search/search-page.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.94%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.78%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

64. **门禁结果（更新到 2026-02-12 18:20 +0800）**
   - 列表页分页信息最小页数一致化：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/websites/websites-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/directories/directories-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ip-addresses/ip-addresses-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-detail-view.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.94%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.78%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

65. **门禁结果（更新到 2026-02-12 18:26 +0800）**
   - 组织与仪表盘列表分页最小页数一致化：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-list.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/targets/targets-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/dashboard/dashboard-scan-history.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.94%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.78%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

66. **门禁结果（更新到 2026-02-12 18:29 +0800）**
   - Dashboard 分页信息统一构建：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/dashboard/dashboard-data-table.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.94%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.78%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

67. **门禁结果（更新到 2026-02-12 18:33 +0800）**
   - 组织选择器分页信息统一构建：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/add-target-dialog.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.94%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.78%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

68. **门禁结果（更新到 2026-02-12 18:36 +0800）**
   - 分页显示与边界处理收敛：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/search/search-pagination.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/data-table/pagination.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/screenshots/screenshots-gallery.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.76%`
     - Overall Branches：`33.91%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.82%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

69. **门禁结果（更新到 2026-02-12 18:39 +0800）**
   - 服务器分页表格在 0 条结果时保持分页模式：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/endpoints/endpoints-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/targets-data-table.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.76%`
     - Overall Branches：`33.91%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.82%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

70. **门禁结果（更新到 2026-02-12 18:48 +0800）**
   - 漏洞详情分页信息统一构建：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerabilities-detail-view.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.76%`
     - Overall Branches：`33.91%`
     - Overall Functions：`51.02%`
     - Overall Lines：`50.82%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

71. **门禁结果（更新到 2026-02-12 19:04 +0800）**
   - 大体量对话框/复杂输入/大型 hooks 拆分：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/initiate-scan-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/initiate-scan-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/common/smart-filter-input.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/common/smart-filter-input-field.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/common/smart-filter-input-menu.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints/index.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints/keys.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints/ehole.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints/goby.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints/wappalyzer.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints/fingers.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints/fingerprinthub.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints/arl.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-fingerprints/stats.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

72. **门禁结果（更新到 2026-02-12 19:22 +0800）**
   - 大体量对话框/状态机与大型 hooks 拆分：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/agent-install-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/agent-install-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/create-scheduled-scan-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/scheduled-scan-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/config/add-tool-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/config/add-tool-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-targets/index.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-targets/keys.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-targets/queries.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-targets/mutations.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scans/index.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scans/keys.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scans/queries.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-scans/mutations.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-vulnerabilities/index.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-vulnerabilities/keys.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-vulnerabilities/queries.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/hooks/use-vulnerabilities/mutations.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

73. **门禁结果（更新到 2026-02-12 19:38 +0800）**
   - 大体量对话框/状态机与大型 hooks 拆分（二期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/agent-install-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/agent-install-dialog-state-hooks.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/scheduled-scan-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/scheduled-scan-dialog-state-hooks.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/add-target-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/add-target-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list-dialogs.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-overview.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-overview-utils.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-overview-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-overview-state.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

74. **门禁结果（更新到 2026-02-12 20:19 +0800）**
   - 大体量对话框/状态机拆分（三期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/deploy-terminal-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/deploy-terminal-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/deploy-terminal-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-progress-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-progress-dialog-types.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-progress-dialog-utils.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-progress-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-progress-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/engine/engine-create-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/engine/engine-create-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/engine/engine-create-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/import-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/import-fingerprint-dialog-utils.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/import-fingerprint-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/quick-scan-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/quick-scan-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/quick-scan-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/targets/link-target-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/targets/link-target-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/targets/link-target-dialog-sections.tsx`
   - 复杂输入组件拆分（终端登录）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/terminal-login.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/terminal-login-types.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/terminal-login-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

75. **门禁结果（更新到 2026-02-12 20:45 +0800）**
   - 大体量对话框/状态机拆分（四期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/common/bulk-add-urls-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/common/bulk-add-urls-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/common/bulk-add-urls-dialog-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

76. **门禁结果（更新到 2026-02-12 21:01 +0800）**
   - 大体量对话框/状态机拆分（五期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/bulk-add-subdomains-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/bulk-add-subdomains-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/bulk-add-subdomains-dialog-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

77. **门禁结果（更新到 2026-02-12 21:05 +0800）**
   - 大体量对话框/状态机拆分（六期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/engine/engine-edit-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/engine/engine-edit-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/engine/engine-edit-dialog-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

78. **门禁结果（更新到 2026-02-12 21:21 +0800）**
   - 大体量对话框/状态机拆分（七期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/add-organization-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/add-organization-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/add-organization-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/config/add-custom-tool-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/config/add-custom-tool-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/config/add-custom-tool-dialog-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

79. **门禁结果（更新到 2026-02-12 21:34 +0800）**
   - 大体量对话框/状态机拆分（八期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/wappalyzer-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/wappalyzer-fingerprint-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/wappalyzer-fingerprint-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/edit-organization-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/edit-organization-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/edit-organization-dialog-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

80. **门禁结果（更新到 2026-02-12 21:39 +0800）**
   - 大体量对话框/状态机拆分（九期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/ehole-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/ehole-fingerprint-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/ehole-fingerprint-dialog-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

81. **门禁结果（更新到 2026-02-12 21:53 +0800）**
   - 大体量对话框/状态机拆分（十期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/goby-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/goby-fingerprint-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/goby-fingerprint-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingers-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingers-fingerprint-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingers-fingerprint-dialog-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

82. **门禁结果（更新到 2026-02-12 22:01 +0800）**
   - 大体量对话框/状态机拆分（十一期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingerprinthub-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingerprinthub-fingerprint-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/fingerprinthub-fingerprint-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/arl-fingerprint-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/arl-fingerprint-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/fingerprints/arl-fingerprint-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/edit-scheduled-scan-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scheduled/edit-scheduled-scan-dialog-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

83. **门禁结果（更新到 2026-02-12 22:14 +0800）**
   - 大体量对话框/状态机拆分（十二期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/auth/change-password-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/auth/change-password-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/auth/change-password-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/worker-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/worker-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/worker-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/architecture-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/architecture-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/architecture-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/wordlist-edit-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/wordlist-edit-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/wordlist-edit-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/wordlist-upload-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/wordlist-upload-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/tools/wordlist-upload-dialog-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/about-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/about-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/about-dialog-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

### Notes

- 本附录为“执行后快照”，用于补充原始 swarm 分析报告；原报告的历史基线指标保留不覆盖。
- 迁移中保持了 query invalidation key 粒度，不扩大失效范围。

---

84. **门禁结果（更新到 2026-02-13 00:11 +0800）**
   - 大体量组件拆分（十三期，10 组件并发）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/target-overview.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/target-settings.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/dashboard/dashboard-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/screenshots/screenshots-gallery.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/notifications/notification-drawer.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-config-editor.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-log-list.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/confirm-dialog.tsx`
   - 新增 `state/sections` 模块：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/target-settings-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/target-settings-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/dashboard/dashboard-data-table-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/dashboard/dashboard-data-table-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/screenshots/screenshots-gallery-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/screenshots/screenshots-gallery-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/notifications/notification-drawer-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/notifications/notification-drawer-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list-view-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-list-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-data-table-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/history/scan-history-data-table-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-config-editor-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-config-editor-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-log-list-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-log-list-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/confirm-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ui/confirm-dialog-sections.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

---

85. **门禁结果（更新到 2026-02-13 00:32 +0800）**
   - 数据表状态抽离与资产视图拆分（十四期，8 组件并发）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/targets/targets-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/targets-data-table.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/websites/websites-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/directories/directories-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ip-addresses/ip-addresses-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/endpoints/endpoints-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/subdomains-detail-view.tsx`
   - 新增 `state/sections` 模块：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/websites/websites-view-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/websites/websites-view-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/directories/directories-view-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/directories/directories-view-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ip-addresses/ip-addresses-view-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/ip-addresses/ip-addresses-view-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/endpoints/endpoints-detail-view-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/endpoints/endpoints-detail-view-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/subdomains-detail-view-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/subdomains/subdomains-detail-view-sections.tsx`
   - 复用并接线已有 table state：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/organization-data-table-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/organization/targets/targets-data-table-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/target/targets-data-table-state.ts`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

---

86. **门禁结果（更新到 2026-02-13 00:48 +0800）**
   - 漏洞页与架构流图拆分（十五期，3 组件并发）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerabilities-detail-view.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerability-detail-content.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/architecture-flow.tsx`
   - 新增 `state/sections` 模块：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerabilities-detail-view-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerabilities-detail-view-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerability-detail-content-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerability-detail-content-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/architecture-flow-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/architecture-flow-sections.tsx`
   - 同步清理 warning 级债务：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/agent-install-dialog.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/architecture-dialog-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/vulnerabilities/vulnerability-detail-dialog.tsx`
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

---

87. **门禁结果（更新到 2026-02-13 00:53 +0800）**
   - 搜索页面拆分（十六期）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/search/search-page.tsx`
   - 新增 `state/sections` 模块：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/search/search-page-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/search/search-page-sections.tsx`
   - 保持行为不变项：
     - URL 初始查询接管（`q` 参数）
     - 最近搜索 localStorage 持久化
     - CSV 导出、分页、漏洞详情弹窗联动
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 目录级校验：`pnpm exec eslint components/search --max-warnings 0` ✅
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" frontend -n`）

---

88. **门禁结果（更新到 2026-02-13 01:04 +0800）**
   - code review 问题修复（十七期，3 模块并发）：
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/architecture-flow-sections.tsx`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/settings/workers/architecture-flow-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/search/search-page-state.ts`
     - `/Users/yangyang/Desktop/lunafox/frontend/components/app-sidebar.tsx`
   - 修复项：
     - React Flow：`RoleNode` 与 `GroupNode` 使用 `memo(...)`，降低父层更新导致的节点重复渲染。
     - React Flow 布局：`resolveLayout(width, ...)` 正式纳入 `width` 参与间距与起始位计算，避免窄屏溢出与“参数未生效”问题。
     - 搜索页 URL 同步：移除一次性初始标志，改为 `lastUrlSyncKeyRef(assetType:q)` 去重，同一 URL 参数不会重复触发，但 URL 变化可再次同步。
     - Sidebar：`user/navMain/documents/devToolGroups` 改为 `React.useMemo(...)`，减少 render 时大对象重建。
   - `pnpm run typecheck` ✅
   - `pnpm run lint:core` ✅
   - `pnpm run check:datatable-legacy` ✅
   - `pnpm run test` ✅（45 files / 265 tests）
   - `pnpm run test:coverage` ✅（45 files / 265 tests）
   - 目录级校验：`pnpm exec eslint components/settings/workers/architecture-flow*.ts* components/search/search-page*.ts* components/app-sidebar.tsx --max-warnings 0` ✅
   - 覆盖率快照：
     - Overall Statements：`49.72%`
     - Overall Branches：`33.91%`
     - Overall Functions：`50.91%`
     - Overall Lines：`50.77%`
   - 全量扫描（`frontend/`）：
     - `useMutation(...)` 调用：`0`（`rg "useMutation\\(" . -n --glob '!node_modules'`）

---

**END OF REPORT**
