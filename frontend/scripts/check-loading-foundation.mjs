#!/usr/bin/env node

import { readFileSync } from "node:fs"
import {
  applyLedger,
  lineForIndex,
  loadLedger,
  parseMode,
  printFindings,
  printSummary,
  resolveProductionFiles,
  toRelativePath,
} from "./ui-foundation-guardrail-lib.mjs"
import {
  getBoundedGeometryExceptionValidationErrors,
  getLoadingGeometryInventoryValidationErrors,
  getMissingLoadingGeometrySlotSourceDeclarations,
  getResolvedIntrinsicTableSettlementValidationErrors,
  hasUsableLoadingGeometrySlots,
  hasUsableLoadingGeometryTolerance,
  routeLoadingContracts,
} from "./loading-route-contracts.mjs"
import { collectRouteInventory } from "./route-inventory.mjs"

const mode = parseMode()
const ledger = loadLedger()
const routeInventory = collectRouteInventory()
const loadingRouteContractsSource = readFileSync(
  new URL("./loading-route-contracts.mjs", import.meta.url),
  "utf8"
)

const guardrailContract = {
  modes: ["inventory", "verify"],
  ledger: "foundation-exceptions.json",
  roots: ["components", "app"],
}

const approvedLoadingOwners = new Set([
  "components/ui/skeleton.tsx",
  "components/ui/button.tsx",
  "components/ui/sonner.tsx",
  "components/shared/loading/action-skeleton.tsx",
  "components/shared/loading/spinner.tsx",
  "components/shared/loading/content-handoff.tsx",
  "components/shared/loading/app-warmup-loader.tsx",
  "components/shared/loading/app-shell-warmup.tsx",
])

// These files deliberately render the same named regions for skeleton and
// resolved states while their ContentHandoff lives in a route/workspace parent.
// Keep this exact so a new source file cannot introduce a pairing contract by
// declaring a slot without an explicit owner or shared-template review.
const approvedLoadingStructureSlotTemplates = new Set([
  "app/scan/history/[id]/scan-history-detail-shell-layout.tsx",
  "app/settings/api-keys/content.tsx",
  "app/targets/[id]/target-detail-shell-layout.tsx",
  "components/overview/overview-lazy-sections.tsx",
  "components/overview/overview-sections-skeleton.tsx",
  "components/organization/organization-detail-view-sections.tsx",
  "components/scan/scheduled/scheduled-scan-page-sections.tsx",
  "components/scan/workflow/scan-workflow-page.tsx",
  "components/search/search-page-sections.tsx",
  "components/settings/api-keys/api-keys-settings-loading-state.tsx",
  "components/settings/agents/agent-list-loading-state.tsx",
  "components/settings/agents/agent-overview-loading-state.tsx",
  "components/settings/agents/agent-overview-section.tsx",
  "components/settings/agents/agent-results-region.tsx",
  "components/settings/blacklist/blacklist-settings-loading-state.tsx",
  "components/settings/database-health/database-health-loading-state.tsx",
  "components/settings/database-health/database-health-view.tsx",
  "components/settings/login-visual/login-visual-settings-workspace.tsx",
  "components/settings/notifications/notification-settings-loading-state.tsx",
  "components/settings/notifications/notification-settings-page-content.tsx",
  "components/settings/support/support-page-layout.tsx",
  "components/settings/system-logs/system-logs-loading-state.tsx",
  "components/settings/system-logs/system-logs-view.tsx",
  "components/shared/data-table/unified-data-table.tsx",
  "components/shared/loading/content-handoff.tsx",
  "components/shared/loading/loading-owner.ts",
  "components/tools/nuclei-poc-catalog-page.tsx",
  "components/tools/engines/engine-installation-page.tsx",
  "components/tools/wordlists-page.tsx",
  "components/vulnerabilities/vulnerabilities-vertical-view.tsx",
  "components/websites/website-relation-detail-view.tsx",
  "components/websites/website-relation-evidence-view.tsx",
])

const routeProgressAdapter = "components/route-progress.tsx"
const requiredLoadingContracts = [
  {
    component: "ContentHandoff",
    prop: "owner",
    ownerFile: "components/shared/loading/content-handoff.tsx",
  },
  {
    component: "ContentHandoff",
    prop: "skeleton",
    ownerFile: "components/shared/loading/content-handoff.tsx",
  },
  {
    component: "ContentHandoff",
    prop: "isLoading",
    ownerFile: "components/shared/loading/content-handoff.tsx",
  },
  {
    component: "ContentReveal",
    prop: "owner",
    ownerFile: "components/shared/loading/content-reveal.tsx",
  },
  {
    component: "DataTableSkeleton",
    prop: "owner",
    ownerFile: "components/shared/loading/data-table-skeleton.tsx",
    allowOwnerlessProp: "nested",
  },
  {
    component: "CardGridSkeleton",
    prop: "owner",
    ownerFile: "components/shared/loading/card-grid-skeleton.tsx",
  },
  {
    component: "MasterDetailSkeleton",
    prop: "owner",
    ownerFile: "components/shared/loading/master-detail-skeleton.tsx",
  },
  {
    component: "PageSectionSkeleton",
    prop: "owner",
    ownerFile: "components/shared/loading/page-section-skeleton.tsx",
  },
  {
    component: "SettingsPageSkeleton",
    prop: "owner",
    ownerFile: "components/shared/loading/settings-page-skeleton.tsx",
  },
  {
    component: "AppShellWarmup",
    prop: "owner",
    ownerFile: "components/shared/loading/app-shell-warmup.tsx",
  },
  {
    component: "AppWarmupLoader",
    prop: "intent",
    ownerFile: "components/shared/loading/app-warmup-loader.tsx",
  },
]

const retainedDomainStatusAnimations = [
  {
    scope: "components/nudges/nudge-minimal.tsx",
    pattern: "animate-pulse bg-green-500",
    semanticOwner: "nudge terminal cursor",
    reason: "Decorative terminal-style cursor for an already-rendered nudge message, not a data-loading placeholder.",
    approvedHelperOrException: "loading-status-nudge-terminal-cursor",
    reviewTrigger: "When nudge terminal visuals or motion policy are reworked.",
    recoveryPath: "Move the cursor animation into a shared terminal/status affordance or remove the blink.",
  },
  {
    scope: "components/auth/terminal-login-sections.tsx",
    pattern: "animate-pulse bg-foreground/80",
    semanticOwner: "auth terminal cursor",
    reason: "Terminal boot cursor communicates an auth shell affordance after text is present, not data loading.",
    approvedHelperOrException: "loading-status-auth-terminal-cursor",
    reviewTrigger: "When auth terminal boot visuals change.",
    recoveryPath: "Move the cursor animation into the auth terminal primitive or remove the blink.",
  },
  {
    scope: "components/settings/support/support-page-content.tsx",
    pattern: "animate-pulse",
    semanticOwner: "support reveal affordance",
    reason: "Decorative reveal prompt in the support page visual surface, not a loading placeholder.",
    approvedHelperOrException: "loading-status-support-reveal-affordance",
    reviewTrigger: "When the support hero interaction is redesigned.",
    recoveryPath: "Replace with a shared affordance or static text if the motion is no longer needed.",
  },
  {
    scope: "components/settings/system-logs/system-logs-view.tsx",
    pattern: "animate-pulse bg-green-500",
    semanticOwner: "system log auto-refresh status",
    reason: "Live auto-refresh heartbeat for an active log viewer, not an initial data load.",
    approvedHelperOrException: "loading-status-system-logs-auto-refresh",
    reviewTrigger: "When system log polling or toolbar status changes.",
    recoveryPath: "Move the heartbeat into a shared live-status indicator helper.",
  },
  {
    scope: "components/settings/agents/agent-install-connection-status.tsx",
    pattern: "animate-spin",
    semanticOwner: "agent installation connection status",
    reason: "The spinner marks an active registration-token attribution observation and stops when the dialog closes or the bounded observation ends.",
    approvedHelperOrException: "loading-status-agent-install-connection",
    reviewTrigger: "When Agent installation connection feedback or polling semantics change.",
    recoveryPath: "Move the wait indicator into a shared live-status helper or replace it with a static state icon.",
  },
  {
    scope: "components/scan/scan-status-badge.tsx",
    pattern: "animate-",
    semanticOwner: "scan runtime status",
    reason: "Running scan icon/progress animation describes domain execution state.",
    approvedHelperOrException: "loading-status-scan-badge-running",
    reviewTrigger: "When scan status badges or progress visualization change.",
    recoveryPath: "Move running-state motion into scan status helpers or make the badge static.",
  },
  {
    scope: "components/scan/history/scan-overview-sections.tsx",
    pattern: "animate-",
    semanticOwner: "scan history runtime status",
    reason: "Running scan and auto-refresh pulses describe active scan telemetry.",
    approvedHelperOrException: "loading-status-scan-history-runtime",
    reviewTrigger: "When scan history runtime telemetry changes.",
    recoveryPath: "Move scan runtime motion into shared scan status helpers.",
  },
  {
    scope: "components/scan/history/scan-runtime-detail-drawer.tsx",
    pattern: "animate-",
    semanticOwner: "scan runtime task status",
    reason: "Task-level running indicators describe active scan execution.",
    approvedHelperOrException: "loading-status-scan-runtime-detail",
    reviewTrigger: "When scan runtime drawer task state changes.",
    recoveryPath: "Move running task motion into shared scan status helpers.",
  },
]

const sharedSkeletonOwners = new Set([
  "DataTableSkeleton",
  "CardGridSkeleton",
  "MasterDetailSkeleton",
  "PageSectionSkeleton",
  "SettingsPageSkeleton",
  "AppShellWarmup",
  "AppWarmupLoader",
])

const approvedLegacySkeletonDeclarationFiles = new Map([
  ["components/nav-user.tsx", new Set(["NavUserSkeleton"])],
  ["components/shared/dropdown-menu-owners.tsx", new Set(["SidebarUserMenuSkeleton"])],
  ["components/shared/metrics/stat-metric-row.tsx", new Set(["MetricValueSkeleton"])],
])

function isApprovedSkeletonDeclaration(file, name) {
  if (file.startsWith("components/shared/loading/")) return true
  if (file === "components/ui/skeleton.tsx") return true
  if (file === "components/ui/sidebar.tsx") return true
  if (file === "components/overview/overview-sections-skeleton.tsx") return true
  return approvedLegacySkeletonDeclarationFiles.get(file)?.has(name) ?? false
}

const controlledTableSkeletonVariantMarker = "data-loading-controlled-table-skeleton-variant"

const reviewedVisibleDynamicFallbacks = [
  {
    scope: "components/unified-header.tsx",
    fallback: "HeaderIconPlaceholder",
    semanticOwner: "header icon action slots",
    reason: "Keeps the top-bar icon slots stable while client-only dialogs load.",
  },
  {
    scope: "components/settings/agents/architecture-dialog-sections.tsx",
    fallback: "ArchitectureFlowCanvasLoadingState",
    semanticOwner: "agent architecture canvas",
    reason: "Local canvas loading state inside an opened dialog, not a route or page skeleton.",
  },
  {
    scope: "components/auth/terminal-login-sections.tsx",
    fallback: "TerminalLoginBrandStatic",
    semanticOwner: "auth login brand text",
    reason: "Static auth brand fallback for the animated login heading.",
  },
  {
    scope: "components/scan/workflow/scan-workflow-page.tsx",
    fallback: "WorkflowConfigPreviewLoadingState",
    semanticOwner: "workflow config preview editor",
    reason: "Local preview-pane fallback for a client-only YAML viewer.",
  },
]

function isTestOrGenerated(file) {
  return file.includes("__tests__") || file.includes(".test.") || file.includes(".contract.")
}

function classifyRetainedDomainStatus(file, snippet) {
  return retainedDomainStatusAnimations.find((entry) => {
    if (entry.scope !== file) return false
    return snippet.includes("animate-")
  })
}

function finding(checkId, file, source, match, ruleId, label) {
  return {
    checkId,
    file,
    line: lineForIndex(source, match.index ?? 0),
    ruleId,
    label,
    snippet: match[0].replace(/\s+/g, " ").slice(0, 140),
  }
}

function findJsxOpeningTagEnd(source, startIndex) {
  let braceDepth = 0
  let quote = null
  let escaped = false

  for (let index = startIndex; index < source.length; index += 1) {
    const char = source[index]

    if (quote) {
      if (escaped) {
        escaped = false
        continue
      }

      if (char === "\\") {
        escaped = true
        continue
      }

      if (char === quote) {
        quote = null
      }

      continue
    }

    if (char === "\"" || char === "'" || char === "`") {
      quote = char
      continue
    }

    if (char === "{") {
      braceDepth += 1
      continue
    }

    if (char === "}") {
      braceDepth = Math.max(0, braceDepth - 1)
      continue
    }

    if (char === ">" && braceDepth === 0) {
      return index
    }
  }

  return -1
}

function extractContentHandoffSourceRanges(source) {
  const ranges = []

  for (const match of source.matchAll(/<ContentHandoff\b/g)) {
    const start = match.index ?? 0
    const openingTagEnd = findJsxOpeningTagEnd(source, start)
    if (openingTagEnd === -1) continue

    const openingTag = source.slice(start, openingTagEnd + 1)
    if (/\/\s*>$/.test(openingTag)) {
      ranges.push({ start, end: openingTagEnd })
      continue
    }

    const nestedTagPattern = /<\/?ContentHandoff\b[^>]*>/g
    nestedTagPattern.lastIndex = openingTagEnd + 1
    let depth = 1
    let end = openingTagEnd
    let nestedTag

    while ((nestedTag = nestedTagPattern.exec(source))) {
      if (nestedTag[0].startsWith("</")) {
        depth -= 1
        if (depth === 0) {
          end = (nestedTag.index ?? openingTagEnd) + nestedTag[0].length - 1
          break
        }
        continue
      }

      if (!/\/\s*>$/.test(nestedTag[0])) {
        depth += 1
      }
    }

    ranges.push({ start, end })
  }

  return ranges
}

function isApprovedLoadingStructureSlotContext(file, handoffRanges, index) {
  if (approvedLoadingStructureSlotTemplates.has(file)) return true
  return handoffRanges.some((range) => index >= range.start && index <= range.end)
}

function findBalancedGroupEnd(source, startIndex) {
  const openingPairs = new Map([
    ["(", ")"],
    ["{", "}"],
    ["[", "]"],
  ])
  const closingPairs = new Map(Array.from(openingPairs, ([open, close]) => [close, open]))
  const stack = []
  let quote = null
  let escaped = false

  for (let index = startIndex; index < source.length; index += 1) {
    const char = source[index]

    if (quote) {
      if (escaped) {
        escaped = false
        continue
      }

      if (char === "\\") {
        escaped = true
        continue
      }

      if (char === quote) {
        quote = null
      }

      continue
    }

    if (char === "\"" || char === "'" || char === "`") {
      quote = char
      continue
    }

    if (openingPairs.has(char)) {
      stack.push(char)
      continue
    }

    if (!closingPairs.has(char)) continue

    const expectedOpen = closingPairs.get(char)
    if (stack.at(-1) !== expectedOpen) {
      return -1
    }

    stack.pop()
    if (stack.length === 0) {
      return index
    }
  }

  return -1
}

function splitTopLevelArguments(source) {
  const args = []
  const openingPairs = new Map([
    ["(", ")"],
    ["{", "}"],
    ["[", "]"],
  ])
  const closingPairs = new Map(Array.from(openingPairs, ([open, close]) => [close, open]))
  const stack = []
  let quote = null
  let escaped = false
  let startIndex = 0

  for (let index = 0; index < source.length; index += 1) {
    const char = source[index]

    if (quote) {
      if (escaped) {
        escaped = false
        continue
      }

      if (char === "\\") {
        escaped = true
        continue
      }

      if (char === quote) {
        quote = null
      }

      continue
    }

    if (char === "\"" || char === "'" || char === "`") {
      quote = char
      continue
    }

    if (openingPairs.has(char)) {
      stack.push(char)
      continue
    }

    if (closingPairs.has(char)) {
      if (stack.at(-1) === closingPairs.get(char)) {
        stack.pop()
      }
      continue
    }

    if (char !== "," || stack.length > 0) continue

    args.push(source.slice(startIndex, index).trim())
    startIndex = index + 1
  }

  const tail = source.slice(startIndex).trim()
  if (tail || source.trim()) {
    args.push(tail)
  }

  return args
}

function extractJsxOpeningTagStaticProps(openTag) {
  const props = new Map()
  const body = openTag.replace(/^<[A-Za-z0-9_.:-]+\s*/, "").replace(/\/?>$/, "")
  let index = 0

  while (index < body.length) {
    const char = body[index]
    if (/\s/.test(char)) {
      index += 1
      continue
    }

    if (char === "{") {
      let braceDepth = 1
      index += 1
      while (index < body.length && braceDepth > 0) {
        if (body[index] === "{") braceDepth += 1
        if (body[index] === "}") braceDepth -= 1
        index += 1
      }
      continue
    }

    const nameMatch = /^[A-Za-z_$][A-Za-z0-9_$:-]*/.exec(body.slice(index))
    if (!nameMatch) {
      index += 1
      continue
    }

    const name = nameMatch[0]
    index += name.length
    while (index < body.length && /\s/.test(body[index])) index += 1

    if (body[index] !== "=") {
      props.set(name, true)
      continue
    }

    index += 1
    while (index < body.length && /\s/.test(body[index])) index += 1

    const valueStart = body[index]
    if (valueStart === "\"" || valueStart === "'") {
      const quote = valueStart
      index += 1
      let value = ""
      while (index < body.length) {
        const valueChar = body[index]
        if (valueChar === quote) {
          index += 1
          break
        }
        value += valueChar
        index += 1
      }
      props.set(name, value)
      continue
    }

    if (valueStart === "{") {
      let braceDepth = 1
      let value = "{"
      index += 1
      while (index < body.length && braceDepth > 0) {
        const valueChar = body[index]
        value += valueChar
        if (valueChar === "{") braceDepth += 1
        if (valueChar === "}") braceDepth -= 1
        index += 1
      }
      props.set(name, value)
      continue
    }

    let value = ""
    while (index < body.length && !/\s/.test(body[index])) {
      value += body[index]
      index += 1
    }
    props.set(name, value)
  }

  return props
}

function getJsxOpeningTagStaticProp(openTag, propName) {
  return extractJsxOpeningTagStaticProps(openTag).get(propName)
}

function classifyReviewedVisibleDynamicFallback(file, fallbackName) {
  return reviewedVisibleDynamicFallbacks.find((entry) => (
    entry.scope === file && entry.fallback === fallbackName
  ))
}

function extractLoadingOwnerSourceRanges(source) {
  const ranges = []
  const declarationPattern = /\b(?:export\s+)?(?:function\s+[A-Za-z0-9]*(?:Skeleton|LoadingState)\s*\(|const\s+[A-Za-z0-9]*(?:Skeleton|LoadingState)\s*=)/g

  for (const match of source.matchAll(declarationPattern)) {
    const start = match.index ?? 0
    let end = -1

    if (match[0].includes("function")) {
      const bodyStart = source.indexOf("{", start)
      if (bodyStart !== -1) {
        end = findBalancedGroupEnd(source, bodyStart)
      }
    } else {
      const arrowIndex = source.indexOf("=>", start)
      if (arrowIndex !== -1) {
        const bodyStart = source.indexOf("{", arrowIndex)
        const parenStart = source.indexOf("(", arrowIndex)
        const jsxStart = source.indexOf("<", arrowIndex)
        const candidates = [bodyStart, parenStart, jsxStart]
          .filter((index) => index !== -1)
          .sort((a, b) => a - b)
        const expressionStart = candidates[0]

        if (expressionStart !== undefined) {
          end = findBalancedGroupEnd(source, expressionStart)
        }
      }
    }

    ranges.push({ start, end: end === -1 ? source.length - 1 : end })
  }

  return ranges
}

function collectMatchesInRanges(source, ranges, pattern) {
  const matches = []

  for (const range of ranges) {
    const fragment = source.slice(range.start, range.end + 1)
    for (const match of fragment.matchAll(pattern)) {
      matches.push({ ...match, index: range.start + (match.index ?? 0) })
    }
  }

  return matches
}

function isContentHandoffSkeletonChild(source, componentStartIndex) {
  const handoffStart = source.lastIndexOf("<ContentHandoff", componentStartIndex)
  if (handoffStart === -1) return false

  const openTagEnd = findJsxOpeningTagEnd(source, handoffStart)
  if (openTagEnd === -1 || componentStartIndex > openTagEnd) return false

  const openTag = source.slice(handoffStart, openTagEnd + 1)
  const skeletonPropIndex = openTag.lastIndexOf("skeleton=")
  if (skeletonPropIndex === -1) return false

  return handoffStart + skeletonPropIndex < componentStartIndex
}

const loadingFindings = []
const routeEngineFindings = []
const duplicateRouteStartFindings = []
const legacyWrapperFindings = []
const missingContractFindings = []
const hardReplacementHandoffFindings = []
const featureLocalDynamicFallbacks = []
const featureLocalDirectSkeletonBranches = []
const retained = []
const routeLocalButtonSkeletonHelperFindings = []
const routeLocalControlShellLookalikeFindings = []
const routeLocalTableSkeletonReplicaFindings = []
const legacyComponentSkeletonBreadcrumbFindings = []
const lazyPageDefaultFallbackFindings = []
const routeLazyPageVisibleFallbackFindings = []
const routeDynamicVisibleFallbackFindings = []
const unreviewedVisibleDynamicFallbackFindings = []
const contentHandoffWrapperGeometryFindings = []
const routeLocalSkeletonGeometryLiteralFindings = []
const routeLocalSkeletonContainerGeometryLiteralFindings = []
const businessSkeletonDeclarationFindings = []
const legacySkeletonRowCountNameFindings = []
const routeLoadingGeometryContractFindings = []
const loadingGeometrySlotSourceFindings = []
const orphanLoadingStructureSlotFindings = []
const productionLoadingStructureSources = []
const reportedMissingLoadingGeometrySlotSources = new Set()

for (const filePath of resolveProductionFiles(["components", "app"])) {
  const file = toRelativePath(filePath)
  if (isTestOrGenerated(file)) continue
  const source = readFileSync(filePath, "utf8")
  productionLoadingStructureSources.push(source)
  const contentHandoffSourceRanges = extractContentHandoffSourceRanges(source)

  for (const match of source.matchAll(/\bdata-loading-slot\s*=/g)) {
    if (isApprovedLoadingStructureSlotContext(file, contentHandoffSourceRanges, match.index ?? 0)) continue
    orphanLoadingStructureSlotFindings.push(
      finding(
        "loading",
        file,
        source,
        match,
        "orphan-loading-structure-slot",
        "data-loading-slot declared outside ContentHandoff or an approved shared template"
      )
    )
  }

  for (const match of source.matchAll(/\bgetLoadingStructureSlotAttributes\s*\(/g)) {
    if (isApprovedLoadingStructureSlotContext(file, contentHandoffSourceRanges, match.index ?? 0)) continue
    orphanLoadingStructureSlotFindings.push(
      finding(
        "loading",
        file,
        source,
        match,
        "orphan-loading-structure-slot",
        "loading structure slot helper called outside ContentHandoff or an approved shared template"
      )
    )
  }

  for (const match of source.matchAll(/\banimate-(?:spin|pulse)\b/g)) {
    const item = finding("loading", file, source, match, "loading-animation-snippet", "loading or status animation")
    if (approvedLoadingOwners.has(file)) {
      continue
    }

    const retainedStatus = classifyRetainedDomainStatus(file, item.snippet)
    if (retainedStatus) {
      retained.push({ ...item, retainedStatus })
      continue
    }

    loadingFindings.push(item)
  }

  for (const match of source.matchAll(/from\s+["'][^"']*loading-spinner["']/g)) {
    legacyWrapperFindings.push(
      finding("loading", file, source, match, "legacy-loading-wrapper-import", "legacy loading wrapper import")
    )
  }

  for (const match of source.matchAll(/from\s+["'][^"']*shield-loader["']/g)) {
    legacyWrapperFindings.push(
      finding("loading", file, source, match, "legacy-shield-loader-import", "legacy shield loader import")
    )
  }

  for (const match of source.matchAll(/\b(?:export\s+)?(?:function|const)\s+([A-Za-z][A-Za-z0-9]*Skeleton)\b/g)) {
    const name = match[1]
    if (isApprovedSkeletonDeclaration(file, name)) continue
    businessSkeletonDeclarationFindings.push(
      finding(
        "loading",
        file,
        source,
        match,
        "business-skeleton-declaration",
        `business loading state declares legacy Skeleton component ${name}`
      )
    )
  }

  for (const match of source.matchAll(/\bskeletonRowCount\b/g)) {
    legacySkeletonRowCountNameFindings.push(
      finding(
        "loading",
        file,
        source,
        match,
        "legacy-skeleton-row-count-name",
        "business loading row-count variable uses legacy skeleton naming"
      )
    )
  }

  for (const contract of requiredLoadingContracts) {
    const pattern = new RegExp(`<${contract.component}(?![^>]*\\b${contract.prop}=)[^>]*\\/?>`, "g")
    for (const match of source.matchAll(pattern)) {
      if (file === contract.ownerFile) continue
      if (contract.prop === "owner" && isContentHandoffSkeletonChild(source, match.index ?? 0)) continue
      if (contract.allowOwnerlessProp && match[0].includes(contract.allowOwnerlessProp)) continue
      missingContractFindings.push(
        finding(
          "loading",
          file,
          source,
          match,
          "missing-loading-contract-prop",
          `${contract.component} missing required ${contract.prop}`
        )
      )
    }
  }

  for (const match of source.matchAll(/<ContentHandoff\b/g)) {
    const openTagEnd = findJsxOpeningTagEnd(source, match.index ?? 0)
    if (openTagEnd === -1) continue

    const openTag = source.slice(match.index ?? 0, openTagEnd + 1)
    const className = getJsxOpeningTagStaticProp(openTag, "className")
    if (typeof className !== "string" || !/(?:h-full|min-h-0|flex-1)/.test(className)) continue
    if (
      getJsxOpeningTagStaticProp(openTag, "skeletonClassName") !== undefined &&
      getJsxOpeningTagStaticProp(openTag, "contentClassName") !== undefined
    ) continue

    contentHandoffWrapperGeometryFindings.push(
      finding(
        "loading",
        file,
        source,
        match,
        "content-handoff-missing-wrapper-geometry",
        "viewport-fill ContentHandoff missing matching skeleton/content wrapper geometry"
      )
    )
  }

  for (const match of source.matchAll(/from\s+["'](?:nextjs-toploader(?:\/app)?|nprogress)["']/g)) {
    if (file === routeProgressAdapter) continue
    routeEngineFindings.push(finding("loading", file, source, match, "direct-route-progress-engine-import", "direct route-progress engine import"))
  }

  for (const match of source.matchAll(/lunafox:route-progress-start/g)) {
    if (file === routeProgressAdapter) continue
    duplicateRouteStartFindings.push(finding("loading", file, source, match, "manual-route-progress-start", "manual route progress start outside adapter"))
  }

  for (const match of source.matchAll(/\b(?:function|const)\s+[A-Za-z0-9]*ButtonSkeleton\b/g)) {
    if (approvedLoadingOwners.has(file)) continue
    routeLocalButtonSkeletonHelperFindings.push(
      finding("loading", file, source, match, "route-local-button-skeleton-helper", "route-local button skeleton helper")
    )
  }

  if (file.endsWith("skeleton.tsx")) {
    for (const match of source.matchAll(/className="[^"]*border border-input bg-background[^"]*"[\s\S]{0,220}?<Skeleton\b/g)) {
      if (approvedLoadingOwners.has(file)) continue
      routeLocalControlShellLookalikeFindings.push(
        finding("loading", file, source, match, "route-local-control-shell-lookalike", "route-local control-shell lookalike")
      )
    }
  }

  const ownsSkeletonGeometry =
    /\bfunction\s+[A-Za-z0-9]*(?:Skeleton|LoadingState)\s*\(/.test(source) ||
    /data-slot="[^"]*skeleton"/.test(source)

  if (ownsSkeletonGeometry && file !== "components/shared/loading/detail-page-shell-skeleton.tsx") {
    const isSharedLoadingOwner = file.startsWith("components/shared/loading/")
    const skeletonGeometrySourceRanges = file.endsWith("skeleton.tsx")
      ? [{ start: 0, end: source.length - 1 }]
      : extractLoadingOwnerSourceRanges(source)

    if (!isSharedLoadingOwner) {
      for (const match of collectMatchesInRanges(source, skeletonGeometrySourceRanges, /(?:min-h-\[[^\]]+\]|rows\s*=\s*\d+)/g)) {
        routeLocalSkeletonGeometryLiteralFindings.push(
          finding(
            "loading",
            file,
            source,
            match,
            "route-local-skeleton-geometry-literal",
            "route-local skeleton owns fixed geometry instead of deriving resolved component dimensions"
          )
        )
      }

      if (file.endsWith("skeleton.tsx")) {
        for (const match of collectMatchesInRanges(source, skeletonGeometrySourceRanges, /<(?!(?:Skeleton|[A-Za-z][A-Za-z0-9.]*Skeleton)\b)[A-Za-z][A-Za-z0-9.]*\b[^>\n]*className="[^"]*(?:\bh-\[[^\]]+\]|\bw-\[[^\]]+\])[^"]*"[^>\n]*>/g)) {
          routeLocalSkeletonContainerGeometryLiteralFindings.push(
            finding(
              "loading",
              file,
              source,
              match,
              "route-local-skeleton-container-geometry-literal",
              "route-local skeleton container owns fixed h/w geometry instead of sharing a resolved layout owner"
            )
          )
        }
      }
    }

    const hasTableReplicaShell =
      source.includes('data-slot="data-table"') ||
      source.includes("<TableHeader") ||
      source.includes("<TableBody") ||
      source.includes("<colgroup>") ||
      source.includes("TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX")
    const hasTableReplicaControls =
      source.includes("SearchToolbarSkeleton") ||
      source.includes("CompactPaginationSkeleton") ||
      source.includes("ActionSkeleton")

    const hasControlledTableSkeletonVariant = source.includes(controlledTableSkeletonVariantMarker)

    if (!isSharedLoadingOwner && !hasControlledTableSkeletonVariant && hasTableReplicaShell && hasTableReplicaControls) {
      const match = /data-slot="data-table"|<TableHeader|<TableBody|<colgroup>|TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX/.exec(source)
      if (match) {
        routeLocalTableSkeletonReplicaFindings.push(
          finding(
            "loading",
            file,
            source,
            match,
            "route-local-table-skeleton-replica",
            "route-local table skeleton replica instead of shared table loading owner"
          )
        )
      }
    }

    for (const match of source.matchAll(/<span[^>]*>\s*\/\s*<\/span>/g)) {
      legacyComponentSkeletonBreadcrumbFindings.push(
        finding(
          "loading",
          file,
          source,
          match,
          "component-skeleton-legacy-breadcrumb",
          "component-owned skeleton contains legacy breadcrumb separator"
        )
      )
    }

    if (
      (source.includes("border border-input bg-background") || source.includes("dark:bg-input/30")) &&
      !source.includes("SearchToolbarSkeleton") &&
      !source.includes("SelectShellSkeleton") &&
      !source.includes("CompactPaginationSkeleton")
    ) {
      const match = /border border-input bg-background|dark:bg-input\/30/.exec(source)
      if (match) {
        routeLocalControlShellLookalikeFindings.push(
          finding("loading", file, source, match, "route-local-control-shell-lookalike", "route-local control-shell lookalike")
        )
      }
    }
  }

  if (/lazyPage\(/.test(source)) {
    for (const match of source.matchAll(/lazyPage\(/g)) {
      const callStart = match.index ?? 0
      const argsStart = callStart + "lazyPage".length
      const callEnd = findBalancedGroupEnd(source, argsStart)
      if (callEnd === -1) continue

      const args = splitTopLevelArguments(source.slice(argsStart + 1, callEnd))
      const fallbackArg = args[2]?.trim()

      if (fallbackArg === undefined) {
        lazyPageDefaultFallbackFindings.push(
          finding(
            "loading",
            file,
            source,
            match,
            "lazy-page-visible-default-fallback",
            "lazyPage route uses visible default PageSectionSkeleton fallback"
          )
        )
        continue
      }

      if (file.startsWith("app/") && fallbackArg !== "null") {
        routeLazyPageVisibleFallbackFindings.push(
          finding(
            "loading",
            file,
            source,
            match,
            "route-lazy-page-visible-fallback",
            "app route lazyPage uses visible chunk fallback instead of null"
          )
        )
      }
    }
  }

  if (file.startsWith("app/")) {
    for (const match of source.matchAll(/dynamic(?:<[^>]+>)?\([\s\S]{0,1600}?loading:\s*\(\)\s*=>\s*(?!null)(?:<|[A-Za-z])/g)) {
      routeDynamicVisibleFallbackFindings.push(
        finding(
          "loading",
          file,
          source,
          match,
          "route-dynamic-visible-fallback",
          "route dynamic chunk uses visible loading fallback"
        )
      )
    }
  }

  for (const match of source.matchAll(/loading:\s*\(\)\s*=>\s*<([A-Za-z][A-Za-z0-9]+)\b/g)) {
    const fallbackName = match[1]
    if (file.startsWith("app/")) continue
    if (classifyReviewedVisibleDynamicFallback(file, fallbackName)) continue

    unreviewedVisibleDynamicFallbackFindings.push(
      finding(
        "loading",
        file,
        source,
        match,
        "unreviewed-visible-dynamic-fallback",
        `component dynamic chunk uses unreviewed visible fallback ${fallbackName}`
      )
    )
  }

  for (const match of source.matchAll(/loading:\s*\(\)\s*=>\s*<([A-Za-z][A-Za-z0-9]+Skeleton)\b/g)) {
    const skeletonName = match[1]
    if (sharedSkeletonOwners.has(skeletonName)) continue
    featureLocalDynamicFallbacks.push({
      skeletonName,
      finding: finding("loading", file, source, match, "duplicate-loading-owner", `duplicate loading owner via ${skeletonName}`),
    })
  }

  for (const match of source.matchAll(/if\s*\(\s*isLoading\s*\)\s*{[\s\S]{0,200}?return\s*<([A-Za-z][A-Za-z0-9]+Skeleton)\b[^>]*\/?>/g)) {
    const skeletonName = match[1]

    if (source.includes("<ContentHandoff")) {
      hardReplacementHandoffFindings.push(
        finding(
          "loading",
          file,
          source,
          match,
          "hard-replacement-loading-handoff",
          `hard replacement loading handoff via ${skeletonName}`
        )
      )
    }

    if (sharedSkeletonOwners.has(skeletonName)) continue
    featureLocalDirectSkeletonBranches.push({
      skeletonName,
      finding: finding("loading", file, source, match, "duplicate-loading-owner", `duplicate loading owner via ${skeletonName}`),
    })
  }
}

for (const contract of routeLoadingContracts) {
  for (const geometryOwner of contract.geometry.owners ?? []) {
    const missingSlots = getMissingLoadingGeometrySlotSourceDeclarations(
      geometryOwner.requiredSlots,
      productionLoadingStructureSources
    )

    for (const slot of missingSlots) {
      const key = `${geometryOwner.owner}:${slot}`
      if (reportedMissingLoadingGeometrySlotSources.has(key)) continue
      reportedMissingLoadingGeometrySlotSources.add(key)

      loadingGeometrySlotSourceFindings.push({
        checkId: "loading",
        file: "scripts/loading-route-contracts.mjs",
        line: lineForIndex(
          loadingRouteContractsSource,
          Math.max(0, loadingRouteContractsSource.indexOf(`"${slot}"`))
        ),
        ruleId: "loading-geometry-slot-source-missing",
        label: `loading geometry slot ${slot} is not declared by production source for ${geometryOwner.owner}`,
        snippet: `${contract.id}/${geometryOwner.owner}/${slot}`,
      })
    }
  }
}

const duplicateLoadingOwnerFindings = []
for (const fallback of featureLocalDynamicFallbacks) {
  for (const branch of featureLocalDirectSkeletonBranches) {
    if (fallback.skeletonName !== branch.skeletonName) continue
    duplicateLoadingOwnerFindings.push(fallback.finding, branch.finding)
  }
}

const routeLoadingContractById = new Map(routeLoadingContracts.map((contract) => [contract.id, contract]))
for (const route of routeInventory.routes ?? []) {
  const contract = routeLoadingContractById.get(route.id)
  const routeFinding = (ruleId, label) => ({
    checkId: "loading",
    file: route.sourcePath ?? "app",
    line: 1,
    ruleId,
    label,
    snippet: route.id,
  })

  if (!contract) {
    routeLoadingGeometryContractFindings.push(
      routeFinding("loading-route-contract-missing", "production route missing loading contract")
    )
    continue
  }

  const geometry = contract.geometry
  if (!geometry || typeof geometry !== "object" || !["verify", "bounded", "not-applicable"].includes(geometry.disposition)) {
    routeLoadingGeometryContractFindings.push(
      routeFinding("loading-geometry-contract-missing", "production route missing loading geometry contract")
    )
    continue
  }

  const geometryOwners = Array.isArray(geometry.owners) ? geometry.owners : []
  const excludedOwners = Array.isArray(geometry.excludedOwners) ? geometry.excludedOwners : []
  const boundedExceptions = Array.isArray(geometry.boundedExceptions) ? geometry.boundedExceptions : []
  const expectedOwnerNames = new Set(contract.expectedOwners.map((item) => item.owner))
  const geometryOwnerNames = new Set(geometryOwners.map((item) => item?.owner))
  const excludedOwnerNames = new Set(excludedOwners.map((item) => item?.owner))
  const boundedExceptionOwnerNames = new Set(boundedExceptions.map((item) => item?.owner))

  if (["verify", "bounded"].includes(geometry.disposition) && geometryOwners.length === 0) {
    routeLoadingGeometryContractFindings.push(
      routeFinding("loading-geometry-contract-invalid", `${geometry.disposition} geometry contract has no paired owners`)
    )
  }

  if (geometry.disposition === "bounded" && boundedExceptions.length === 0) {
    routeLoadingGeometryContractFindings.push(
      routeFinding("loading-geometry-bounded-exception-missing", "bounded geometry contract requires an exception record")
    )
  }

  if (geometry.disposition !== "bounded" && boundedExceptions.length > 0) {
    routeLoadingGeometryContractFindings.push(
      routeFinding("loading-geometry-bounded-exception-unexpected", "only bounded geometry contracts may declare bounded exceptions")
    )
  }

  if (geometry.disposition === "not-applicable") {
    if (geometryOwners.length > 0) {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-contract-invalid", "not-applicable geometry contract must not declare paired owners")
      )
    }

    if (typeof geometry.reason !== "string" || !geometry.reason.trim()) {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-not-applicable-reason-missing", "not-applicable geometry contract requires a reason")
      )
    }
  }

  for (const expectedOwner of contract.expectedOwners) {
    const geometryOwnerCount = geometryOwners.filter((item) => item?.owner === expectedOwner.owner).length
    const excludedOwnerCount = excludedOwners.filter((item) => item?.owner === expectedOwner.owner).length
    const boundedExceptionCount = boundedExceptions.filter((item) => item?.owner === expectedOwner.owner).length

    if (geometryOwnerCount === 0 && excludedOwnerCount === 0) {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-owner-missing", `loading geometry disposition missing for ${expectedOwner.owner}`)
      )
      continue
    }

    const hasConflictingDisposition = geometryOwnerCount > 1 || excludedOwnerCount > 1 || boundedExceptionCount > 1 ||
      (geometryOwnerCount > 0 && excludedOwnerCount > 0) ||
      (excludedOwnerCount > 0 && boundedExceptionCount > 0) ||
      (geometryOwnerCount > 0 && boundedExceptionCount > 0 && geometry.disposition !== "bounded")
    if (hasConflictingDisposition) {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-owner-disposition-conflict", `loading geometry owner disposition conflicts for ${expectedOwner.owner}`)
      )
      continue
    }

    const geometryOwner = geometryOwners.find((item) => item?.owner === expectedOwner.owner)
    if (geometryOwner) {
      if (!["verify", "bounded"].includes(geometry.disposition)) {
        routeLoadingGeometryContractFindings.push(
          routeFinding("loading-geometry-contract-invalid", `not-applicable geometry contract cannot verify ${expectedOwner.owner}`)
        )
      }
    }

    const boundedException = boundedExceptions.find((item) => item?.owner === expectedOwner.owner)
    if (boundedException && geometry.disposition !== "bounded") {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-bounded-exception-unexpected", `only bounded geometry may exempt ${expectedOwner.owner}`)
      )
    }
    if (boundedException && !geometryOwner) {
      routeLoadingGeometryContractFindings.push(
        routeFinding(
          "loading-geometry-bounded-exception-owner-unpaired",
          `bounded exception for ${expectedOwner.owner} cannot replace strict paired geometry`
        )
      )
    }

    const excludedOwner = excludedOwners.find((item) => item?.owner === expectedOwner.owner)
    if (!geometryOwner && !boundedException &&
      (typeof excludedOwner?.reason !== "string" || !excludedOwner.reason.trim())) {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-owner-exclusion-reason-missing", `loading geometry exclusion requires a reason for ${expectedOwner.owner}`)
      )
    }
  }

  for (const owner of geometryOwnerNames) {
    if (!expectedOwnerNames.has(owner)) {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-owner-unexpected", `loading geometry declares unexpected owner ${String(owner)}`)
      )
    }
  }

  for (const owner of excludedOwnerNames) {
    if (!expectedOwnerNames.has(owner)) {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-owner-exclusion-unexpected", `loading geometry excludes unexpected owner ${String(owner)}`)
      )
    }
  }

  for (const owner of boundedExceptionOwnerNames) {
    if (!expectedOwnerNames.has(owner)) {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-bounded-exception-owner-unexpected", `bounded geometry exempts unexpected owner ${String(owner)}`)
      )
    }
  }

  for (const geometryOwner of geometryOwners) {
    if (!geometryOwner || typeof geometryOwner !== "object") {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-contract-invalid", "loading geometry owner must be an object")
      )
      continue
    }

    const slots = geometryOwner.requiredSlots
    const tolerance = geometryOwner.tolerance
    const hasUsableSlots = hasUsableLoadingGeometrySlots(slots)
    const hasUsableTolerance = hasUsableLoadingGeometryTolerance(tolerance)

    if (!hasUsableSlots || !hasUsableTolerance) {
      routeLoadingGeometryContractFindings.push(
        routeFinding("loading-geometry-contract-invalid", `loading geometry contract invalid for ${String(geometryOwner?.owner)}`)
      )
    }

    if (geometryOwner.preparedExternalSurface !== undefined) {
      routeLoadingGeometryContractFindings.push(
        routeFinding(
          "loading-geometry-prepared-surface-removed",
          `prepared external geometry is removed for ${String(geometryOwner.owner)}`
        )
      )
    }

    if (hasUsableSlots) {
      for (const error of getLoadingGeometryInventoryValidationErrors(
        route.id,
        geometryOwner.owner,
        slots
      )) {
        routeLoadingGeometryContractFindings.push(
          routeFinding(
            `loading-geometry-${error}`,
            `loading geometry slots must match the explicit inventory for ${String(geometryOwner.owner)}`
          )
        )
      }
    }

    if (geometryOwner.resolvedIntrinsicTableSettlement !== undefined) {
      for (const error of getResolvedIntrinsicTableSettlementValidationErrors(
        geometryOwner.resolvedIntrinsicTableSettlement,
        slots
      )) {
        routeLoadingGeometryContractFindings.push(
          routeFinding(
            `loading-geometry-${error}`,
            `resolved intrinsic table settlement is invalid for ${String(geometryOwner.owner)}`
          )
        )
      }
    }
  }

  for (const exception of boundedExceptions) {
    for (const error of getBoundedGeometryExceptionValidationErrors(exception)) {
      routeLoadingGeometryContractFindings.push(
        routeFinding(`loading-geometry-${error}`, "bounded geometry exception metadata is invalid")
      )
    }

    const pairedOwner = geometryOwners.find((item) => item?.owner === exception?.owner)
    if (!pairedOwner) {
      routeLoadingGeometryContractFindings.push(
        routeFinding(
          "loading-geometry-bounded-exception-owner-unpaired",
          `bounded exception for ${String(exception?.owner)} requires a strict paired owner`
        )
      )
    } else if (Array.isArray(exception?.affectedSlots) &&
      exception.affectedSlots.some((slot) => pairedOwner.requiredSlots?.includes(slot))) {
      routeLoadingGeometryContractFindings.push(
        routeFinding(
          "loading-geometry-bounded-exception-slot-overlap",
          `bounded slots for ${exception.owner} must remain outside the owner’s compared slots`
        )
      )
    }
  }
}

const checked = applyLedger(
  [
    ...loadingFindings,
    ...routeEngineFindings,
    ...duplicateRouteStartFindings,
    ...legacyWrapperFindings,
    ...missingContractFindings,
    ...hardReplacementHandoffFindings,
    ...routeLocalButtonSkeletonHelperFindings,
    ...routeLocalControlShellLookalikeFindings,
    ...routeLocalTableSkeletonReplicaFindings,
    ...routeLocalSkeletonGeometryLiteralFindings,
    ...routeLocalSkeletonContainerGeometryLiteralFindings,
    ...businessSkeletonDeclarationFindings,
    ...legacySkeletonRowCountNameFindings,
    ...legacyComponentSkeletonBreadcrumbFindings,
    ...lazyPageDefaultFallbackFindings,
    ...routeLazyPageVisibleFallbackFindings,
    ...routeDynamicVisibleFallbackFindings,
    ...unreviewedVisibleDynamicFallbackFindings,
    ...contentHandoffWrapperGeometryFindings,
    ...duplicateLoadingOwnerFindings,
    ...routeLoadingGeometryContractFindings,
    ...loadingGeometrySlotSourceFindings,
  ],
  ledger
)
// An orphaned structure slot is never a ledger exception: it must be attached
// to a real handoff owner or an explicitly reviewed shared template first.
checked.push(...orphanLoadingStructureSlotFindings)
const unapproved = checked.filter((item) => !item.exception)

console.log(`Loading foundation check mode: ${mode}`)
console.log(`Ledger: ${guardrailContract.ledger}`)
console.log("Production roots:")
for (const root of guardrailContract.roots) {
  console.log(`  ${root}`)
}
console.log(`Retained domain status animations: ${retained.length}`)
for (const item of retained) {
  const status = item.retainedStatus
  console.log(
    `  ${item.file}:${item.line} owner=${status.semanticOwner} exception=${status.approvedHelperOrException} reason=${status.reason} reviewTrigger=${status.reviewTrigger} recoveryPath=${status.recoveryPath}`
  )
}

printFindings("Unclassified loading findings:", checked)
printSummary(checked)
console.log(`unclassified loading snippets = ${unapproved.length}`)

if (mode === "verify" && unapproved.length > 0) {
  process.exit(1)
}
