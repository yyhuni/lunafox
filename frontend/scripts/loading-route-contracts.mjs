export const DEFAULT_PROTECTED_ROUTE_OWNER = {
  owner: "auth-layout-route-content",
  layer: "route",
  intent: "route",
}

export const DEFAULT_LOADING_GEOMETRY_TOLERANCE = Object.freeze({
  top: 1,
  left: 1,
  width: 1,
  height: 2,
})

const LOADING_GEOMETRY_RECT_FIELDS = Object.freeze(["top", "left", "width", "height"])

function isRecord(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value)
}

function isNonEmptyString(value) {
  return typeof value === "string" && Boolean(value.trim())
}

export function hasUsableLoadingGeometrySlots(slots) {
  return Array.isArray(slots) && slots.length > 0 &&
    slots.every((slot) => isNonEmptyString(slot)) &&
    new Set(slots).size === slots.length
}

export function hasUsableLoadingGeometryTolerance(tolerance) {
  return LOADING_GEOMETRY_RECT_FIELDS.every((key) => (
    Number.isFinite(tolerance?.[key]) && tolerance[key] >= 0
  ))
}

/**
 * Static coverage backstop for geometry contracts. Runtime smoke still proves
 * that a slot is paired in both branches and that its rectangle is stable;
 * this only prevents a contract from naming a slot that production source
 * never declares at all.
 */
export function getMissingLoadingGeometrySlotSourceDeclarations(requiredSlots, sourceTexts) {
  const sources = Array.isArray(sourceTexts) ? sourceTexts : []

  return (Array.isArray(requiredSlots) ? requiredSlots : [])
    .filter((slot) => isNonEmptyString(slot))
    .filter((slot) => !sources.some((source) => (
      typeof source === "string" && (source.includes(`"${slot}"`) || source.includes(`'${slot}'`))
    )))
}

function getStableFirstFrameValidationErrors(firstFrame, expectedMeasurement = null) {
  const errors = []

  if (!isRecord(firstFrame)) {
    return ["first-frame-missing"]
  }

  if (!isNonEmptyString(firstFrame.measurement)) {
    errors.push("first-frame-measurement-missing")
  } else if (expectedMeasurement && firstFrame.measurement !== expectedMeasurement) {
    errors.push("first-frame-measurement-invalid")
  }

  if (!Number.isInteger(firstFrame.stableFrames) || firstFrame.stableFrames < 2) {
    errors.push("first-frame-stable-frames-invalid")
  }

  if (firstFrame.contentOnlyStable !== true) {
    errors.push("first-frame-content-only-stability-missing")
  }

  return errors
}

export function getBoundedGeometryExceptionValidationErrors(exception) {
  if (!isRecord(exception)) {
    return ["bounded-exception-invalid"]
  }

  const errors = []
  if (!isNonEmptyString(exception.owner)) {
    errors.push("bounded-exception-owner-missing")
  }
  if (!hasUsableLoadingGeometrySlots(exception.affectedSlots)) {
    errors.push("bounded-exception-affected-slots-invalid")
  }
  if (!isNonEmptyString(exception.reason)) {
    errors.push("bounded-exception-reason-missing")
  }
  if (!isNonEmptyString(exception.reviewTrigger)) {
    errors.push("bounded-exception-review-trigger-missing")
  }
  if (!isNonEmptyString(exception.recoveryPath)) {
    errors.push("bounded-exception-recovery-path-missing")
  }
  errors.push(...getStableFirstFrameValidationErrors(exception.firstFrame).map(
    (error) => `bounded-exception-${error}`
  ))

  return errors
}

export function getResolvedIntrinsicTableSettlementValidationErrors(settlement, requiredSlots) {
  if (!isRecord(settlement)) {
    return ["resolved-intrinsic-table-settlement-invalid"]
  }

  const errors = []
  const slots = Array.isArray(requiredSlots) ? requiredSlots : []

  if (!isNonEmptyString(settlement.bodySlot)) {
    errors.push("resolved-intrinsic-table-settlement-body-slot-missing")
  } else if (settlement.bodySlot === "surface" || !slots.includes(settlement.bodySlot)) {
    errors.push("resolved-intrinsic-table-settlement-body-slot-invalid")
  }

  if (settlement.paginationSlot !== undefined) {
    if (!isNonEmptyString(settlement.paginationSlot) ||
      settlement.paginationSlot === "surface" ||
      settlement.paginationSlot === settlement.bodySlot ||
      !slots.includes(settlement.paginationSlot)) {
      errors.push("resolved-intrinsic-table-settlement-pagination-slot-invalid")
    }
  }

  if (settlement.allowContentGrowth !== undefined && settlement.allowContentGrowth !== true) {
    errors.push("resolved-intrinsic-table-settlement-content-growth-invalid")
  }

  if (!isNonEmptyString(settlement.reason)) {
    errors.push("resolved-intrinsic-table-settlement-reason-missing")
  }
  if (!isNonEmptyString(settlement.reviewTrigger)) {
    errors.push("resolved-intrinsic-table-settlement-review-trigger-missing")
  }

  return errors
}

function createResolvedIntrinsicTableSettlement(bodySlot, paginationSlot, options = {}) {
  return {
    bodySlot,
    ...(paginationSlot ? { paginationSlot } : {}),
    ...(options.allowContentGrowth === undefined ? {} : { allowContentGrowth: options.allowContentGrowth }),
    reason: options.reason ?? "The shared paginated table releases its loading-only row reservation when resolved data is sparse.",
    reviewTrigger: options.reviewTrigger ?? "Review when the shared table surface or natural-flow pagination placement changes.",
  }
}

const OVERVIEW_SECTION_GEOMETRY_SLOTS = Object.freeze([
  "surface",
  "overview-runtime-region",
  "overview-asset-region",
  "overview-operational-region",
])

const TARGET_DETAIL_SHELL_GEOMETRY_SLOTS = Object.freeze([
  "surface",
  "target-detail-shell-header",
  "target-detail-shell-primary-tabs",
  "target-detail-shell-content",
])

const SCAN_HISTORY_DETAIL_SHELL_GEOMETRY_SLOTS = Object.freeze([
  "surface",
  "scan-history-detail-shell-header",
  "scan-history-detail-shell-primary-tabs",
  "scan-history-detail-shell-content",
])

const targetDetailRouteIds = [
  "route_targets_id_directories",
  "route_targets_id_endpoints",
  "route_targets_id_ip_addresses",
  "route_targets_id_overview",
  "route_targets_id_screenshots",
  "route_targets_id_settings",
  "route_targets_id_settings_scheduled_scans",
  "route_targets_id_subdomains",
  "route_targets_id_vulnerabilities",
  "route_targets_id_websites",
  "route_targets_id_websites_websiteid_section",
]

const scanHistoryDetailRouteIds = [
  "route_scan_history_id_directories",
  "route_scan_history_id_endpoints",
  "route_scan_history_id_ip_addresses",
  "route_scan_history_id_overview",
  "route_scan_history_id_screenshots",
  "route_scan_history_id_subdomains",
  "route_scan_history_id_vulnerabilities",
  "route_scan_history_id_websites",
]

const nonPairedLoadingGeometryOwnerReasons = Object.freeze({
  [DEFAULT_PROTECTED_ROUTE_OWNER.owner]:
    "ContentReveal only represents the resolved route shell and never mounts paired skeleton/content structure states.",
})

const routeNonPairedLoadingGeometryOwnerReasons = Object.freeze({
  route_login: Object.freeze({
    "login-page-content":
      "The public login route exposes a resolved-only ContentReveal while the boot layer owns its initial visual wait; it never mounts paired skeleton/content structure states.",
  }),
  route_targets_id_websites: Object.freeze({
    "target-website-evidence-view-content":
      "The target detail shell owns cold first entry; the deferred evidence child suppresses its skeleton and appears only as resolved content after the shell handoff.",
  }),
  route_targets_id_websites_websiteid_section: Object.freeze({
    "website-relation-detail-view-content":
      "The target detail shell owns cold first entry; the deferred website-detail child suppresses its skeleton and appears only as resolved content after the shell handoff.",
  }),
})

// Geometry coverage is intentionally route-and-owner specific. The generated
// `surface` wrapper is not a fallback: an entry containing only `surface` is an
// explicit inventory decision for a region that has no independently moving
// first-frame subregion.
const routeLoadingGeometryInventory = {
  route_overview: {
    "overview-sections-loader": {
      requiredSlots: OVERVIEW_SECTION_GEOMETRY_SLOTS,
    },
  },
  route_organizations: {
    "organization-list-content": {
      requiredSlots: [
        "surface",
        "organization-list-toolbar",
        "organization-list-body",
        "organization-list-pagination",
      ],
      resolvedIntrinsicTableSettlement: createResolvedIntrinsicTableSettlement(
        "organization-list-body",
        "organization-list-pagination"
      ),
    },
  },
  route_organizations_id: {
    "organization-detail-view-content": {
      requiredSlots: [
        "surface",
        "organization-detail-summary",
        "organization-detail-primary-table",
      ],
      resolvedIntrinsicTableSettlement: createResolvedIntrinsicTableSettlement(
        "organization-detail-primary-table"
      ),
    },
  },
  route_scan_history: {
    "scan-history-list-view-content": {
      requiredSlots: [
        "surface",
        "scan-history-list-toolbar",
        "scan-history-list-body",
        "scan-history-list-pagination",
      ],
      resolvedIntrinsicTableSettlement: createResolvedIntrinsicTableSettlement(
        "scan-history-list-body",
        "scan-history-list-pagination",
        {
          allowContentGrowth: true,
          reason: "Complete scan-summary and executed-engine Badge groups may wrap after the dense loading rows resolve.",
          reviewTrigger: "Review when scan-history Badge wrapping or natural-flow pagination placement changes.",
        }
      ),
    },
  },
  route_scan_scheduled: {
    "scheduled-scan-page-route": {
      requiredSlots: [
        "surface",
        "scheduled-scan-header",
        "scheduled-scan-insight-grid",
        "scheduled-scan-table",
      ],
      resolvedIntrinsicTableSettlement: createResolvedIntrinsicTableSettlement(
        "scheduled-scan-table"
      ),
    },
  },
  route_scan_config_workflows: {
    "scan-workflow-page-content": {
      requiredSlots: [
        "surface",
        "scan-workflow-toolbar",
        "scan-workflow-primary-region",
      ],
    },
  },
  route_settings_api_keys: {
    "api-keys-page-route": {
      requiredSlots: [
        "surface",
        "api-keys-header",
        "api-keys-provider-list",
        "api-keys-provider-detail",
        "api-keys-notice",
      ],
    },
  },
  route_settings_agents: {
    "agent-list-overview": {
      requiredSlots: ["surface", "agent-list-overview-region"],
    },
    "agent-list-toolbar": {
      requiredSlots: ["surface", "agent-list-toolbar-region"],
    },
    "agent-list-results": {
      requiredSlots: ["surface", "agent-list-results-primary-region"],
    },
  },
  route_settings_blacklist: {
    "blacklist-page-content": {
      requiredSlots: [
        "surface",
        "blacklist-header",
        "blacklist-controls",
        "blacklist-list",
      ],
    },
  },
  route_settings_database_health: {
    "database-health-page-route": {
      requiredSlots: [
        "surface",
        "database-health-header",
        "database-health-snapshot",
        "database-health-metrics",
        "database-health-findings",
      ],
    },
  },
  route_settings_login_visual: {
    "login-visual-settings-page-route": {
      requiredSlots: [
        "surface",
        "login-visual-header",
        "login-visual-preview",
        "login-visual-publication-controls",
      ],
    },
  },
  route_settings_notifications: {
    "notification-settings-page-route": {
      // The two external destination workbench is the complete visible
      // cold-entry surface; it has no inbox or preferences geometry.
      requiredSlots: [
        "surface",
        "notification-settings-header",
        "notification-settings-channel-workbench",
      ],
    },
  },
  route_settings_system_logs: {
    "system-logs-page-route": {
      requiredSlots: [
        "surface",
        "system-logs-header",
        "system-logs-terminal-toolbar",
        "system-logs-log-surface",
      ],
    },
  },
  route_targets: {
    "all-targets-detail-view-content": {
      requiredSlots: [
        "surface",
        "all-targets-toolbar",
        "all-targets-table-body",
        "all-targets-pagination",
      ],
      resolvedIntrinsicTableSettlement: createResolvedIntrinsicTableSettlement(
        "all-targets-table-body",
        "all-targets-pagination"
      ),
    },
  },
  route_vulnerabilities: {
    "vulnerabilities-vertical-view-content": {
      requiredSlots: [
        "surface",
        "vulnerabilities-severity-summary",
        "vulnerabilities-table-toolbar",
        "vulnerabilities-rows",
        "vulnerabilities-pagination",
      ],
      resolvedIntrinsicTableSettlement: createResolvedIntrinsicTableSettlement(
        "vulnerabilities-rows",
        "vulnerabilities-pagination"
      ),
    },
  },
  route_scan_config_engines: {
    "engine-catalog-content": {
      requiredSlots: [
        "surface",
        "engine-catalog-controls",
        "engine-catalog-grid",
        "engine-catalog-card-rhythm",
      ],
    },
  },
  route_tools_fingerprints_fingerprinthub: {
    "fingerprinthub-fingerprint-view-content": {
      requiredSlots: [
        "surface",
        "fingerprinthub-fingerprint-toolbar",
        "fingerprinthub-fingerprint-rows",
        "fingerprinthub-fingerprint-pagination",
      ],
      resolvedIntrinsicTableSettlement: createResolvedIntrinsicTableSettlement(
        "fingerprinthub-fingerprint-rows",
        "fingerprinthub-fingerprint-pagination"
      ),
    },
  },
  route_tools_wordlists: {
    "wordlists-page-content": {
      requiredSlots: [
        "surface",
        "wordlists-controls",
        "wordlists-list",
      ],
    },
  },
  "search-query-results": {
    "search-results-content": {
      requiredSlots: [
        "surface",
        "search-results-toolbar",
        "search-results-body",
        "search-results-pagination",
      ],
    },
  },
  ...Object.fromEntries(targetDetailRouteIds.map((routeId) => [routeId, {
    "target-detail-shell": {
      requiredSlots: TARGET_DETAIL_SHELL_GEOMETRY_SLOTS,
    },
  }])),
  ...Object.fromEntries(scanHistoryDetailRouteIds.map((routeId) => [routeId, {
    "scan-history-detail-shell": {
      requiredSlots: SCAN_HISTORY_DETAIL_SHELL_GEOMETRY_SLOTS,
    },
  }])),
  route_settings_support: {
    "support-page-route": {
      requiredSlots: [
        "surface",
        "support-page-header",
        "support-page-value-band",
        "support-page-actions",
      ],
    },
  },
  route_tools_nuclei: {
    "nuclei-poc-catalog-content": {
      requiredSlots: [
        "surface",
        "nuclei-poc-catalog-source",
        "nuclei-poc-catalog-table",
      ],
      resolvedIntrinsicTableSettlement: createResolvedIntrinsicTableSettlement(
        "nuclei-poc-catalog-table",
        undefined,
        {
          reason: "The Nuclei catalog releases its ten-row loading reservation when a sparse committed source resolves.",
          reviewTrigger: "Review when the Nuclei catalog table or its natural-flow pagination placement changes.",
        }
      ),
    },
  },
}

export function getLoadingGeometryInventoryEntry(routeId, owner) {
  return routeLoadingGeometryInventory[routeId]?.[owner] ?? null
}

export function getLoadingGeometryInventoryValidationErrors(routeId, owner, requiredSlots) {
  const inventoryEntry = getLoadingGeometryInventoryEntry(routeId, owner)
  if (!inventoryEntry) {
    return ["geometry-inventory-entry-missing"]
  }

  if (!hasUsableLoadingGeometrySlots(inventoryEntry.requiredSlots)) {
    return ["geometry-inventory-definition-invalid"]
  }

  if (!hasUsableLoadingGeometrySlots(requiredSlots)) {
    return ["geometry-inventory-required-slots-invalid"]
  }

  const hasExactSlots = inventoryEntry.requiredSlots.length === requiredSlots.length &&
    inventoryEntry.requiredSlots.every((slot, index) => slot === requiredSlots[index])
  return hasExactSlots ? [] : ["geometry-inventory-slots-mismatch"]
}

function createLoadingGeometryOwner(routeId, owner) {
  const inventoryEntry = getLoadingGeometryInventoryEntry(routeId, owner)
  const requiredSlots = inventoryEntry?.requiredSlots
  const settlement = inventoryEntry?.resolvedIntrinsicTableSettlement
  const errors = [
    ...getLoadingGeometryInventoryValidationErrors(routeId, owner, requiredSlots),
    ...(settlement === undefined
      ? []
      : getResolvedIntrinsicTableSettlementValidationErrors(settlement, requiredSlots)),
  ]
  if (errors.length > 0) {
    throw new Error(`Loading geometry inventory invalid for ${routeId}/${owner}: ${errors.join(", ")}.`)
  }

  return {
    owner,
    requiredSlots,
    tolerance: DEFAULT_LOADING_GEOMETRY_TOLERANCE,
    ...(settlement ? { resolvedIntrinsicTableSettlement: settlement } : {}),
  }
}

function getNonPairedLoadingGeometryOwnerReason(routeId, owner) {
  return routeNonPairedLoadingGeometryOwnerReasons[routeId]?.[owner]
    ?? nonPairedLoadingGeometryOwnerReasons[owner]
}

function createNonPairedGeometryOwner(routeId, owner) {
  const reason = getNonPairedLoadingGeometryOwnerReason(routeId, owner)
  if (!reason) {
    throw new Error(`Loading geometry exclusion requires a documented reason for ${routeId}/${owner}.`)
  }

  return { owner, reason }
}

function withLoadingGeometryContract(contract) {
  if (contract.geometry) {
    return contract
  }

  const owners = []
  const excludedOwners = []

  for (const { owner } of contract.expectedOwners) {
    if (getNonPairedLoadingGeometryOwnerReason(contract.id, owner)) {
      excludedOwners.push(createNonPairedGeometryOwner(contract.id, owner))
      continue
    }

    owners.push(createLoadingGeometryOwner(contract.id, owner))
  }

  return {
    ...contract,
    geometry: owners.length > 0
      ? {
          disposition: "verify",
          owners,
          excludedOwners,
        }
      : {
          disposition: "not-applicable",
          reason: "The first-screen contract contains only resolved-only owners with no paired skeleton/content structure state.",
          owners: [],
          excludedOwners,
        },
  }
}

const protectedRouteIds = [
  "route_organizations_id",
  "route_scan_history_id_directories",
  "route_scan_history_id_endpoints",
  "route_scan_history_id_ip_addresses",
  "route_scan_history_id_overview",
  "route_scan_history_id_screenshots",
  "route_scan_history_id_subdomains",
  "route_scan_history_id_vulnerabilities",
  "route_scan_history_id_websites",
  "route_scan_scheduled",
  "route_settings_api_keys",
  "route_settings_agents",
  "route_settings_blacklist",
  "route_settings_database_health",
  "route_settings_login_visual",
  "route_settings_notifications",
  "route_settings_support",
  "route_settings_system_logs",
  "route_targets_id_directories",
  "route_targets_id_endpoints",
  "route_targets_id_ip_addresses",
  "route_targets_id_overview",
  "route_targets_id_screenshots",
  "route_targets_id_settings",
  "route_targets_id_settings_scheduled_scans",
  "route_targets_id_subdomains",
  "route_targets_id_vulnerabilities",
  "route_targets_id_websites",
  "route_targets_id_websites_websiteid_section",
  "route_tools_space_mapping",
  "route_tools_space_mapping_provider",
  "route_tools",
  "route_tools_fingerprints_fingerprinthub",
]

function createFirstScreenOwner(owner, layer = "workspace") {
  return { owner, layer, intent: "data" }
}

function createFirstScreenOwnerConfig(owners, expectedFirstScreenLayer = "workspace") {
  return {
    classification: expectedFirstScreenLayer === "section" ? "sectioned-dashboard" : "direct-client-route",
    expectedFirstScreenLayer,
    owners,
  }
}

// One route-level source of truth for every paired first-screen owner. Deferred
// children remain outside this map because their parent shell owns cold entry.
const routeFirstScreenOwnerConfigs = {
  route_organizations_id: createFirstScreenOwnerConfig([
    createFirstScreenOwner("organization-detail-view-content"),
  ]),
  route_scan_history_id_directories: createFirstScreenOwnerConfig([
    createFirstScreenOwner("scan-history-detail-shell"),
  ]),
  route_scan_history_id_endpoints: createFirstScreenOwnerConfig([
    createFirstScreenOwner("scan-history-detail-shell"),
  ]),
  route_scan_history_id_ip_addresses: createFirstScreenOwnerConfig([
    createFirstScreenOwner("scan-history-detail-shell"),
  ]),
  route_scan_history_id_overview: createFirstScreenOwnerConfig([
    createFirstScreenOwner("scan-history-detail-shell"),
  ]),
  route_scan_history_id_screenshots: createFirstScreenOwnerConfig([
    createFirstScreenOwner("scan-history-detail-shell"),
  ]),
  route_scan_history_id_subdomains: createFirstScreenOwnerConfig([
    createFirstScreenOwner("scan-history-detail-shell"),
  ]),
  route_scan_history_id_vulnerabilities: createFirstScreenOwnerConfig([
    createFirstScreenOwner("scan-history-detail-shell"),
  ]),
  route_scan_history_id_websites: createFirstScreenOwnerConfig([
    createFirstScreenOwner("scan-history-detail-shell"),
  ]),
  route_scan_scheduled: createFirstScreenOwnerConfig([
    createFirstScreenOwner("scheduled-scan-page-route"),
  ]),
  route_scan_config_workflows: createFirstScreenOwnerConfig([
    createFirstScreenOwner("scan-workflow-page-content"),
  ]),
  route_scan_config_engines: createFirstScreenOwnerConfig([
    createFirstScreenOwner("engine-catalog-content"),
  ]),
  route_settings_agents: createFirstScreenOwnerConfig([
    createFirstScreenOwner("agent-list-overview", "section"),
    createFirstScreenOwner("agent-list-toolbar", "section"),
    createFirstScreenOwner("agent-list-results", "section"),
  ], "section"),
  route_settings_api_keys: createFirstScreenOwnerConfig([
    createFirstScreenOwner("api-keys-page-route"),
  ]),
  route_settings_blacklist: createFirstScreenOwnerConfig([
    createFirstScreenOwner("blacklist-page-content"),
  ]),
  route_settings_database_health: createFirstScreenOwnerConfig([
    createFirstScreenOwner("database-health-page-route"),
  ]),
  route_settings_login_visual: createFirstScreenOwnerConfig([
    createFirstScreenOwner("login-visual-settings-page-route"),
  ]),
  route_settings_notifications: createFirstScreenOwnerConfig([
    createFirstScreenOwner("notification-settings-page-route"),
  ]),
  route_settings_support: createFirstScreenOwnerConfig([
    createFirstScreenOwner("support-page-route"),
  ]),
  route_settings_system_logs: createFirstScreenOwnerConfig([
    createFirstScreenOwner("system-logs-page-route"),
  ]),
  route_targets: createFirstScreenOwnerConfig([
    createFirstScreenOwner("all-targets-detail-view-content"),
  ]),
  route_targets_id_directories: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_targets_id_endpoints: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_targets_id_ip_addresses: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_targets_id_overview: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_targets_id_screenshots: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_targets_id_settings: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_targets_id_settings_scheduled_scans: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_targets_id_subdomains: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_targets_id_vulnerabilities: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_targets_id_websites: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_targets_id_websites_websiteid_section: createFirstScreenOwnerConfig([
    createFirstScreenOwner("target-detail-shell"),
  ]),
  route_tools_fingerprints_fingerprinthub: createFirstScreenOwnerConfig([
    createFirstScreenOwner("fingerprinthub-fingerprint-view-content"),
  ]),
}

const deferredDetailChildOwners = {
  route_targets_id_websites: {
    owner: "target-website-evidence-view-content",
    layer: "section",
    intent: "data",
  },
  route_targets_id_websites_websiteid_section: {
    owner: "website-relation-detail-view-content",
    layer: "section",
    intent: "data",
  },
}

function protectedRouteContract(id) {
  const firstScreenOwnerConfig = routeFirstScreenOwnerConfigs[id]
  const deferredDetailChildOwner = deferredDetailChildOwners[id]

  return {
    id,
    classification: firstScreenOwnerConfig?.classification ?? "protected-route-content",
    expectedFirstScreenLayer: firstScreenOwnerConfig?.expectedFirstScreenLayer ?? "route",
    allowedLayers: ["boot", "app-shell", "route", "workspace", "section", "interaction"],
    expectedOwners: [
      DEFAULT_PROTECTED_ROUTE_OWNER,
      ...(firstScreenOwnerConfig?.owners ?? []),
      ...(deferredDetailChildOwner ? [deferredDetailChildOwner] : []),
    ],
  }
}

function firstScreenRouteContract(id) {
  const firstScreenOwnerConfig = routeFirstScreenOwnerConfigs[id]
  if (!firstScreenOwnerConfig) {
    throw new Error(`Missing first-screen owner config for ${id}.`)
  }

  return {
    id,
    classification: firstScreenOwnerConfig.classification,
    expectedFirstScreenLayer: firstScreenOwnerConfig.expectedFirstScreenLayer,
    allowedLayers: ["boot", "app-shell", "route", "workspace", "section", "interaction"],
    expectedOwners: [
      DEFAULT_PROTECTED_ROUTE_OWNER,
      ...firstScreenOwnerConfig.owners,
    ],
  }
}

const routeLoadingContractsWithoutGeometry = [
  ...protectedRouteIds.map(protectedRouteContract),
  {
    id: "route_root",
    classification: "server-redirect",
    expectedFirstScreenLayer: null,
    allowedLayers: [],
    expectedOwners: [],
    redirectTarget: "/overview/",
    geometry: {
      disposition: "not-applicable",
      reason: "The root page performs a server redirect before any client loading owner can mount; the canonical overview route verifies the destination handoff.",
      owners: [],
      excludedOwners: [],
    },
  },
  {
    id: "route_login",
    classification: "public-auth-entry",
    expectedFirstScreenLayer: "route",
    allowedLayers: ["boot", "route", "interaction"],
    expectedOwners: [
      {
        owner: "login-page-content",
        layer: "route",
        intent: "route",
      },
    ],
  },
  {
    id: "route_organizations",
    classification: "direct-client-route",
    expectedFirstScreenLayer: "workspace",
    allowedLayers: ["boot", "app-shell", "route", "workspace", "interaction"],
    expectedOwners: [
      DEFAULT_PROTECTED_ROUTE_OWNER,
      {
        owner: "organization-list-content",
        layer: "workspace",
        intent: "data",
      },
    ],
  },
  {
    id: "route_overview",
    classification: "sectioned-dashboard",
    expectedFirstScreenLayer: "section",
    allowedLayers: ["boot", "app-shell", "route", "section", "interaction"],
    expectedOwners: [
      DEFAULT_PROTECTED_ROUTE_OWNER,
      {
        owner: "overview-sections-loader",
        layer: "section",
        intent: "data",
      },
    ],
  },
  {
    id: "route_scan_history",
    classification: "direct-client-route",
    expectedFirstScreenLayer: "workspace",
    allowedLayers: ["boot", "app-shell", "route", "workspace", "interaction"],
    expectedOwners: [
      DEFAULT_PROTECTED_ROUTE_OWNER,
      {
        owner: "scan-history-list-view-content",
        layer: "workspace",
        intent: "data",
      },
    ],
  },
  firstScreenRouteContract("route_scan_config_workflows"),
  firstScreenRouteContract("route_scan_config_engines"),
  {
    id: "route_search",
    classification: "query-driven-workspace",
    expectedFirstScreenLayer: "route",
    allowedLayers: ["boot", "app-shell", "route", "section", "interaction"],
    expectedOwners: [DEFAULT_PROTECTED_ROUTE_OWNER],
  },
  firstScreenRouteContract("route_targets"),
  {
    id: "route_vulnerabilities",
    classification: "direct-client-route",
    expectedFirstScreenLayer: "workspace",
    allowedLayers: ["boot", "app-shell", "route", "workspace", "interaction"],
    expectedOwners: [
      DEFAULT_PROTECTED_ROUTE_OWNER,
      {
        owner: "vulnerabilities-vertical-view-content",
        layer: "workspace",
        intent: "data",
      },
    ],
  },
  {
    id: "route_tools_nuclei",
    classification: "direct-client-route",
    expectedFirstScreenLayer: "workspace",
    allowedLayers: ["boot", "app-shell", "route", "workspace", "interaction"],
    expectedOwners: [
      DEFAULT_PROTECTED_ROUTE_OWNER,
      {
        owner: "nuclei-poc-catalog-content",
        layer: "workspace",
        intent: "data",
      },
    ],
  },
  {
    id: "route_tools_wordlists",
    classification: "direct-client-route",
    expectedFirstScreenLayer: "workspace",
    allowedLayers: ["boot", "app-shell", "route", "workspace", "interaction"],
    expectedOwners: [
      DEFAULT_PROTECTED_ROUTE_OWNER,
      {
        owner: "wordlists-page-content",
        layer: "workspace",
        intent: "data",
      },
    ],
  },
  {
    id: "search-query-results",
    classification: "query-triggered-section",
    expectedFirstScreenLayer: "section",
    allowedLayers: ["boot", "app-shell", "route", "section", "interaction"],
    expectedOwners: [
      DEFAULT_PROTECTED_ROUTE_OWNER,
      {
        owner: "search-results-content",
        layer: "section",
        intent: "data",
      },
    ],
  },
]

export const routeLoadingContracts = routeLoadingContractsWithoutGeometry.map(withLoadingGeometryContract)

const routeLoadingContractById = new Map(routeLoadingContracts.map((contract) => [contract.id, contract]))

export function getRouteLoadingContract(id) {
  return routeLoadingContractById.get(id) || null
}
