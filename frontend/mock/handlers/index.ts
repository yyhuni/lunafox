import { delay, http, HttpResponse } from "msw"
import type { GetIPAddressesParams } from "@/types/ip-address.types"
import type { DirectoryFilterOptionField } from "@/types/directory.types"
import type { EndpointFilterOptionField } from "@/types/endpoint.types"
import type { Organization, OrganizationsResponse } from "@/types/organization.types"
import type { ScheduledScan } from "@/types/scheduled-scan.types"
import { isScanInputSource, type ScanRecord } from "@/types/scan.types"
import type { ScanWorkflow, ScanWorkflowStageView } from "@/types/scan-workflow.types"
import type { Target, TargetsResponse } from "@/types/target.types"
import type { VulnerabilitySeverity } from "@/types/vulnerability.types"
import { NOTIFICATION_DESTINATION_PROVIDERS, type NotificationDestinationProvider } from "@/types/notification-settings.types"
import { serializeCanonicalWorkflowConfiguration } from "@/lib/workflow-config"
import { buildWorkflowWithEngineCatalog } from "@/lib/engine-catalog"
import {
  getNextCronExecutions,
  isCronExpressionValid,
} from "@/lib/scheduled-scan-helpers"
import type { WebSite, WebsiteFilterOptionField } from "@/types/website.types"
import { extractExactWebsiteHostFilter, extractWebsiteURLScopeFilter, matchesExactWebsiteHost, matchesWebsiteURLScope, removeLeadingWebsiteScopeFilter } from "@/lib/website-scope"
import { getMockEngineCatalog, getMockEngineCatalogDetail, installMockEngine } from "@/mock/data/engine-catalog"
import {
  MOCK_DELAY,
  getMockCommandById,
  getMockCommands,
  getMockAgents,
  getMockAgentClusterSummary,
  getMockAgentFilterOptions,
  getMockAgentById,
  getMockAgentLocationMap,
  getMockEndpointById,
  getMockFingerprintFilterOptions,
  getMockFingerprintByName,
  getMockGlobalBlacklistPolicy,
  getMockNotification,
  getMockNotificationDestination,
  getMockNotificationDestinations,
  getMockNotificationTestDeliveryUnavailableResult,
  getMockNotificationLocale,
  getMockNotifications,
  createMockNucleiPocSync,
  getMockNucleiPoc,
  getMockNucleiPocFilterOptions,
  getMockNucleiPocs,
  getMockNucleiPocSource,
  getMockNucleiPocSyncTask,
  setMockNucleiPocActivation,
  getMockRegistrationToken,
  getMockRegistrationTokenById,
  getMockScheduledScanById,
  getMockScheduledScanOverviewSummary,
  getMockScheduledScans,
  getMockScanLogs,
  getMockScanWorkflowProfile,
  getMockScanWorkflowByName,
  getMockScanWorkflows,
  getMockScreenshotImageSvg,
  getMockSearchResults,
  getMockSubdomainById,
  getMockSystemLogs,
  getMockTargetBlacklistPolicy,
  getMockToolById,
  getMockWordlistContent,
  getMockWordlistTags,
  getMockWordlists,
  isMockWordlistEditable,
  getMockUpdateCheckResult,
  getMockVersionInfo,
  getMockWebsites,
  getMockLoginVisualSettings,
  getMockLoginVisualDiscoverability,
  getMockPublicLoginVisual,
  getMockWebsiteById,
  getMockWebsiteFilterOptions,
  getMockDirectoryFilterOptions,
  createMockCommand,
  createMockTool,
  createMockWordlist,
  batchDeleteMockEndpoints,
  bulkDeleteMockDirectories,
  bulkDeleteMockIPAddresses,
  mockIPAddressMatchesFilter,
  getMockPortOptions,
  bulkDeleteMockOrganizations,
  bulkDeleteMockScans,
  bulkDeleteMockSubdomains,
  deleteMockCommand,
  deleteMockDirectory,
  deleteMockAgent,
  deleteMockEndpoint,
  deleteMockOrganization,
  deleteMockScan,
  deleteMockScheduledScan,
  deleteMockSubdomain,
  deleteMockTool,
  deleteMockWebsite,
  exportMockFingerprints,
  importMockFingerprints,
  parseMockFingerprintImport,
  mockDirectories,
  mockScanWorkflows,
  mockIPAddresses,
  mockOrganizations,
  mockLoginResponse,
  mockMeResponse,
  mockScheduledScans,
  mockTargets,
  bulkDeleteMockTargets,
  deleteMockTarget,
  updateMockApiKeySettings,
  updateMockCommand,
  patchMockGlobalBlacklistPolicy,
  markAllMockNotificationsRead,
  markMockNotificationRead,
  updateMockNotificationDestination,
  updateMockNotificationLocale,
  publishMockLoginVisual,
  restoreMockLoginVisual,
  unlockMockLoginVisualDiscoverability,
  uploadMockLoginVisual,
  updateMockNucleiPocEnabled,
  patchMockTargetBlacklistPolicy,
  updateMockTool,
  updateMockWordlistMetadata,
  updateMockVulnerabilityReviewState,
  batchUpdateMockVulnerabilityReviewStates,
  bulkDeleteMockVulnerabilities,
  bulkDeleteMockWebsites,
  batchDeleteMockFingerprints,
  clearMockFingerprints,
  batchDeleteMockCommands,
  bulkRemoveMockOrganizationSubdomains,
  bulkDeleteMockScreenshots,
  buildEndpointFilterOptions,
  buildEndpointResults,
  buildEndpoints,
  buildFingerprintList,
  getMockFingerprintStats,
  buildScanScreenshotFilterOptions,
  buildScreenshotsByScan,
  buildScreenshotsByTarget,
  buildTargetScreenshotFilterOptions,
  buildTargetEndpoints,
  getMockApiKeySettings,
  generateMockMcpKey,
  getMockMcpKeyStatus,
  getMockDiskStats,
  getMockScanById,
  getMockTargetById,
  getMockUnreadCount,
  stopMockScan,
  batchStopMockScans,
} from "../index"
import {
  buildAssetStatistics,
  buildOrganizations,
  buildOverviewStats,
  buildScanById,
  buildScans,
  buildScanStatistics,
  buildStatisticsHistory,
  buildTargetDetail,
  buildTargets,
  buildVulnerabilities,
  buildVulnerabilityById,
  buildVulnerabilityStats,
  scenarioReturnsError,
} from "../builders/core"
import { mockServerRuntimeMetrics } from "../data/overview"
import { buildTools } from "../builders/tools"
import {
  buildAllSubdomains,
  buildOrganizationSubdomains,
  buildSubdomainResults,
} from "../builders/subdomains"
import type { MockFingerprintSource } from "../data/fingerprints"
import { getMockScenario } from "../scenarios"

function normalizeApiPath(pathname: string) {
  const stripped = pathname.replace(/^\/v1/, "") || "/"
  if (stripped === "/") {
    return stripped
  }
  return stripped.replace(/\/+$/, "")
}

function json(payload: unknown, init?: ResponseInit) {
  return HttpResponse.json(payload as never, init)
}

function toAipWebsite(website: WebSite) {
  const { resourceName, screenshot, ...fields } = website
  if (!resourceName) {
    throw new TypeError("Mock Website response requires a canonical resource name")
  }

  return {
    ...fields,
    name: resourceName,
    ...(screenshot ? {
      screenshot: {
        id: screenshot.id,
        name: screenshot.resourceName,
        url: screenshot.url,
        statusCode: screenshot.statusCode,
        createdAt: screenshot.createdAt,
        updatedAt: screenshot.updatedAt,
      },
    } : {}),
  }
}

function toAipWebsiteList(response: ReturnType<typeof getMockWebsites>) {
  return {
    results: response.results.map(toAipWebsite),
    totalSize: response.total,
    page: response.page,
    pageSize: response.pageSize,
    totalPages: response.totalPages,
    ...(response.nextPageToken ? { nextPageToken: response.nextPageToken } : {}),
  }
}

function empty(status = 204) {
  return new HttpResponse(null, { status })
}

function text(body: string, contentType = "text/plain; charset=utf-8", init?: ResponseInit) {
  return new HttpResponse(body, {
    ...init,
    headers: {
      "Content-Type": contentType,
      ...(init?.headers ?? {}),
    },
  })
}

function notificationRefreshStream() {
  const encoder = new TextEncoder()
  return new HttpResponse(
    new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(encoder.encode("event: refresh\ndata: {}\n\n"))
      },
    }),
    {
      headers: {
        "Content-Type": "text/event-stream; charset=utf-8",
        "Cache-Control": "no-cache",
      },
    }
  )
}

function serializeMockWorkflowConfiguration(input: unknown, workflow: ScanWorkflow) {
  const details = Array.from(new Set(workflow.steps.map((step) => step.engineId)))
    .map((engineId) => getMockEngineCatalogDetail(engineId))
    .filter((detail): detail is NonNullable<typeof detail> => detail !== undefined)
  const enrichedWorkflow = buildWorkflowWithEngineCatalog(workflow, details, "en")
  return serializeCanonicalWorkflowConfiguration(input, workflow, enrichedWorkflow)
}

function parseNumeric(value: string | null | undefined) {
  const parsed = Number.parseInt(String(value ?? ""), 10)
  return Number.isFinite(parsed) ? parsed : undefined
}

function hasOnlyKeys(value: Record<string, unknown>, allowed: string[]) {
  const accepted = new Set(allowed)
  return Object.keys(value).every((key) => accepted.has(key))
}

function parseIdFromResourceName(value: string | undefined, collection: string) {
  const match = value?.match(new RegExp(`^${collection}/(\\d+)$`))
  return match ? Number.parseInt(match[1], 10) : undefined
}

function parseNestedIdsFromResourceNames(values: string[] | undefined, parent: string, collection: string) {
  return (values ?? [])
    .map((value) => value.match(new RegExp(`^${parent}/\\d+/${collection}/(\\d+)$`))?.[1])
    .filter((value): value is string => Boolean(value))
    .map((value) => Number.parseInt(value, 10))
}

const WEBSITE_FILTER_OPTION_FIELDS = new Set<string>([
  "statusCode",
  "tech",
  "webserver",
  "contentType",
  "vhost",
])

function parseWebsiteFilterOptionField(value: string | null): WebsiteFilterOptionField | undefined {
  return value && WEBSITE_FILTER_OPTION_FIELDS.has(value) ? value as WebsiteFilterOptionField : undefined
}

function parseEndpointFilterOptionField(value: string | null): EndpointFilterOptionField | undefined {
  return value && WEBSITE_FILTER_OPTION_FIELDS.has(value) ? value as EndpointFilterOptionField : undefined
}

function parseDirectoryFilterOptionField(value: string | null): DirectoryFilterOptionField | undefined {
  return value === "status" || value === "contentType" ? value : undefined
}

function toAipTarget(target: Target) {
  return {
    ...target,
    name: `targets/${target.id}`,
    displayName: target.name,
    organizations: target.organizations?.map((organization) => ({
      ...organization,
      name: `organizations/${organization.id}`,
      displayName: organization.name,
    })),
  }
}

function toAipOrganization(organization: Organization) {
  return {
    ...organization,
    name: `organizations/${organization.id}`,
    displayName: organization.name,
  }
}

function toAipOrganizationsResponse(response: OrganizationsResponse) {
  const page = response.page ?? 1
  const pageSize = response.pageSize ?? 10
  const totalPages = response.totalPages ?? Math.ceil(response.total / pageSize)
  const nextPage = page < totalPages ? page + 1 : undefined

  return {
    results: response.results.map(toAipOrganization),
    totalSize: response.total,
    page,
    pageSize,
    totalPages,
    ...(nextPage && { nextPageToken: `mock-organization-page-${nextPage}` }),
  }
}

function toLegacyTargetsResponse(response: TargetsResponse) {
  return {
    ...response,
    results: response.results.map(toAipTarget),
  }
}

const VULNERABILITY_FILTER_OPTION_FIELDS = new Set(["severity", "source", "vulnType"])

function parseMockPageToken(value: string | null | undefined, prefix: string) {
  const match = value?.match(new RegExp(`^${prefix}-(\\d+)$`))
  return match ? Number.parseInt(match[1], 10) : 1
}

function parseMockFilterValues(filter: string | undefined, field: string) {
  if (!filter) return []
  const expression = new RegExp(`${field}(?:==|=)"((?:\\\\.|[^"\\\\])*)"`, "g")
  return Array.from(filter.matchAll(expression), (match) => (
    (match[1] ?? "").replace(/\\"/g, '"').replace(/\\\\/g, "\\")
  ))
}

function parseMockVulnerabilityFilter(filter: string | null | undefined) {
  const result: {
    url?: string
    severity?: VulnerabilitySeverity
    source?: string
    vulnType?: string
    isReviewed?: boolean
  } = {}
  for (const match of String(filter ?? "").matchAll(/(url|severity|source|vulnType|isReviewed)(?:==|=)"((?:\\.|[^"\\])*)"/g)) {
    const [, field, rawValue] = match
    const value = rawValue.replace(/\\"/g, '"').replace(/\\\\/g, "\\")
    if (field === "url") result.url = value
    if (field === "severity") result.severity = value as VulnerabilitySeverity
    if (field === "source") result.source = value
    if (field === "vulnType") result.vulnType = value
    if (field === "isReviewed") result.isReviewed = value === "true"
  }
  return result
}

function buildCanonicalVulnerabilityResponse(url: URL, scope?: { targetId?: number; scanId?: number }) {
  const pageSize = parseNumeric(url.searchParams.get("pageSize")) || 10
  const page = parseMockPageToken(url.searchParams.get("pageToken"), "mock-vulnerability-page")
  const parsedFilter = parseMockVulnerabilityFilter(url.searchParams.get("filter"))
  const scan = scope?.scanId ? getMockScanById(scope.scanId) : undefined
  const targetId = scope?.targetId ?? scan?.targetId
  const response = buildVulnerabilities({
    page: 1,
    pageSize: 1000,
    targetId,
    url: parsedFilter.url,
    severity: parsedFilter.severity,
    isReviewed: scope?.scanId ? undefined : parsedFilter.isReviewed,
  })

  let results = response.results
  if (parsedFilter.source) {
    results = results.filter((item) => item.source === parsedFilter.source)
  }
  if (parsedFilter.vulnType) {
    results = results.filter((item) => item.vulnType === parsedFilter.vulnType)
  }
  const websiteScope = extractWebsiteURLScopeFilter(url.searchParams.get("filter") || undefined)
  if (websiteScope) {
    results = results.filter((item) => matchesWebsiteURLScope(websiteScope, item.url))
  }

  const orderBy = url.searchParams.get("orderBy") || "createdAt desc"
  results = [...results].sort((left, right) => {
    const diff = new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime()
    return orderBy === "createdAt" || orderBy === "createdAt asc" ? diff : -diff
  })

  const total = results.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const nextPage = page < totalPages ? page + 1 : undefined

  return {
    results: results.slice(start, start + pageSize),
    totalSize: total,
    ...(nextPage && { nextPageToken: `mock-vulnerability-page-${nextPage}` }),
  }
}

function buildVulnerabilityFilterOptions(url: URL, scope?: { targetId?: number; scanId?: number }) {
  const field = url.searchParams.get("field")
  if (!field || !VULNERABILITY_FILTER_OPTION_FIELDS.has(field)) {
    return { error: "Unsupported vulnerability filter option field" }
  }
  const scan = scope?.scanId ? getMockScanById(scope.scanId) : undefined
  const targetId = scope?.targetId ?? scan?.targetId
  const rows = buildVulnerabilities({ page: 1, pageSize: 1000, targetId }).results
  const counts = new Map<string, number>()
  for (const row of rows) {
    const value = String(row[field as "severity" | "source" | "vulnType"] ?? "")
    if (!value) continue
    counts.set(value, (counts.get(value) ?? 0) + 1)
  }
  return {
    results: Array.from(counts.entries())
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([value, count]) => ({ value, label: value, count })),
  }
}

function shouldError(path: string) {
  if (!scenarioReturnsError(getMockScenario())) {
    return false
  }

  return (
    path.startsWith("/dashboard/stats") ||
    path.startsWith("/assetStatistics") ||
    path.startsWith("/organizations") ||
    path.startsWith("/targets") ||
    path.startsWith("/scans") ||
    path.startsWith("/vulnerabilities")
  )
}

function buildOrganizationTargets(organizationId: number, url: URL) {
  const page = parseMockPageToken(
    url.searchParams.get("pageToken"),
    "mock-organization-target-page"
  )
  const pageSize = parseNumeric(url.searchParams.get("pageSize")) || 10
  const search = (url.searchParams.get("filter") || "").toLowerCase()
  const type = url.searchParams.get("type") || ""

  let filtered = mockTargets.filter((target) =>
    target.organizations?.some((organization) => organization.id === organizationId)
  )

  if (search) {
    filtered = filtered.filter(
      (target) =>
        target.name.toLowerCase().includes(search) ||
        target.description?.toLowerCase().includes(search)
    )
  }

  if (type) {
    filtered = filtered.filter((target) => target.type === type)
  }

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const nextPage = page < totalPages ? page + 1 : undefined

  return {
    results: filtered.slice(start, start + pageSize).map(toAipTarget),
    totalSize: total,
    ...(nextPage && { nextPageToken: `mock-organization-target-page-${nextPage}` }),
  }
}

function buildDirectoryResponse(url: URL, targetId?: number, scanId?: number) {
  const page = parseMockPageToken(url.searchParams.get("pageToken"), "mock-directory-page")
  const pageSize = parseNumeric(url.searchParams.get("pageSize")) || 10
  const rawFilter = url.searchParams.get("filter") || undefined
  const websiteScope = extractWebsiteURLScopeFilter(rawFilter)
  const filter = removeLeadingWebsiteScopeFilter(rawFilter, "websiteUrl")
  const target = targetId ? getMockTargetById(targetId) : undefined
  const scan = scanId ? getMockScanById(scanId) : undefined
  const domain = (target?.name || scan?.target?.name || "").toLowerCase()

  let filtered = mockDirectories

  if (domain) {
    filtered = filtered.filter((directory) => directory.url.toLowerCase().includes(domain))
  }

  if (websiteScope) {
    filtered = filtered.filter((directory) => matchesWebsiteURLScope(websiteScope, directory.url))
  }

  const urlValues = parseMockFilterValues(filter, "url")
  if (urlValues.length > 0) {
    filtered = filtered.filter((directory) => urlValues.every((value) => directory.url === value))
  }

  const statusValues = parseMockFilterValues(filter, "status")
  if (statusValues.length > 0) {
    filtered = filtered.filter((directory) => statusValues.includes(String(directory.status)))
  }

  const contentTypeValues = parseMockFilterValues(filter, "contentType")
  if (contentTypeValues.length > 0) {
    filtered = filtered.filter((directory) => contentTypeValues.includes(directory.contentType))
  }

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const nextPage = page < totalPages ? page + 1 : undefined

  return {
    results: filtered.slice(start, start + pageSize),
    total,
    totalSize: total,
    page,
    pageSize,
    totalPages,
    ...(nextPage && { nextPageToken: `mock-directory-page-${nextPage}` }),
  }
}

function buildIpResponse(url: URL, targetId?: number, scanId?: number) {
  const params: GetIPAddressesParams = {
    pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
    pageToken: url.searchParams.get("pageToken") || undefined,
    filter: url.searchParams.get("filter") || undefined,
    orderBy: url.searchParams.get("orderBy") || undefined,
  }
  const pageTokenMatch = params.pageToken?.match(/^mock-ip-page-(\d+)$/)
  const page = pageTokenMatch ? Number.parseInt(pageTokenMatch[1], 10) : 1
  const websiteHostScope = extractExactWebsiteHostFilter(params.filter)
  const filter = removeLeadingWebsiteScopeFilter(params.filter, "host")
  const target = targetId ? getMockTargetById(targetId) : undefined
  const scan = scanId ? getMockScanById(scanId) : undefined
  const domain = (target?.name || scan?.target?.name || "").toLowerCase()

  let filtered = mockIPAddresses

  if (domain) {
    filtered = filtered.filter((item) =>
      item.hosts.some((host) => host.toLowerCase().includes(domain))
    )
  }

  if (websiteHostScope) {
    filtered = filtered.filter((item) =>
      item.hosts.some((host) => matchesExactWebsiteHost(websiteHostScope, host))
    )
  }

  if (filter) {
    filtered = filtered.filter((item) => mockIPAddressMatchesFilter(item, filter))
  }

  const total = filtered.length
  const totalPages = Math.ceil(total / params.pageSize!)
  const start = (page - 1) * params.pageSize!
  const nextPage = page < totalPages ? page + 1 : undefined

  return {
    results: filtered.slice(start, start + params.pageSize!),
    total,
    page,
    pageSize: params.pageSize!,
    totalPages,
    totalSize: total,
    ...(nextPage && { nextPageToken: `mock-ip-page-${nextPage}` }),
  }
}

function buildSubdomainResponse(url: URL, source: "all" | "target" | "scan", sourceId?: number) {
  const page = parseNumeric(url.searchParams.get("page")) || 1
  const pageSize = parseNumeric(url.searchParams.get("pageSize")) || 10

  if (source === "all") {
    return buildAllSubdomains({
      page,
      pageSize,
      search: url.searchParams.get("search") || undefined,
    })
  }

  return buildSubdomainResults({
    source,
    sourceId: sourceId!,
    page,
    pageSize,
    filter: url.searchParams.get("filter") || undefined,
    orderBy: url.searchParams.get("orderBy") || undefined,
  })
}

function getFingerprintSource(path: string): MockFingerprintSource | undefined {
  const source = path.match(/^\/fingerprintLibraries\/([^/:]+)/)?.[1]
	if (source === "fingerprinthub") {
    return source
  }
  return undefined
}

function fingerprintImportError(
  library: MockFingerprintSource,
  status: number,
  kind: "TRANSPORT" | "ENCODING" | "SYNTAX" | "FORMAT" | "RECORD",
  errorInfoReason: string,
  diagnosticReason: string,
  extra: Record<string, string | number> = {}
) {
  return json(
    {
      error: {
        code: status,
        message: "Fingerprint import validation failed.",
        status: status === 413 ? "RESOURCE_EXHAUSTED" : "INVALID_ARGUMENT",
        details: [
          {
            "@type": "type.googleapis.com/google.rpc.ErrorInfo",
            reason: errorInfoReason,
            domain: "lunafox",
            metadata: {
              library,
              ...(status === 413 && errorInfoReason === "FINGERPRINT_IMPORT_FILE_TOO_LARGE"
                ? { maxFileSizeBytes: "31457280" }
                : {}),
            },
          },
          {
            "@type": "type.googleapis.com/lunafox.v1.FingerprintImportDiagnostic",
            kind,
            library,
            reason: diagnosticReason,
            ...extra,
          },
        ],
      },
    },
    { status }
  )
}

function isCanonicalFingerprintName(library: MockFingerprintSource, name: string) {
  const match = /^fingerprintLibraries\/([^/]+)\/fingerprints\/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/i.exec(name)
  return match?.[1] === library
}

async function hasEmptyClearRequestBody(request: Request) {
  const body = (await request.text()).trim()
  return body === "" || body === "{}"
}

function isFingerprintUploadFile(value: FormDataEntryValue | null): value is File {
  return typeof value === "object" && value !== null
    && typeof (value as File).name === "string"
    && typeof (value as File).size === "number"
    && typeof (value as File).type === "string"
}

async function resolveMockApi(request: Request) {
  await delay(MOCK_DELAY)

  const url = new URL(request.url)
  const path = normalizeApiPath(url.pathname)
  const method = request.method.toUpperCase()

  if (shouldError(path)) {
    return json({ error: { code: "mock_scenario_error", message: "Mock scenario forced an error response." } }, { status: 503 })
  }

  if (method === "POST" && path === "/sessions") {
    return json(mockLoginResponse)
  }
  if (method === "POST" && path === "/sessions:renew") {
    return json({
      accessToken: mockLoginResponse.accessToken,
      expiresIn: mockLoginResponse.expiresIn,
    })
  }
  if (method === "GET" && path === "/users/current") {
    return json(mockMeResponse)
  }
  if (method === "POST" && path === "/users/me:changePassword") {
    return json({ message: "Password changed successfully" })
  }

  if (method === "GET" && path === "/dashboard/stats") {
    return json(buildOverviewStats())
  }
  if (method === "GET" && path === "/assetStatistics") {
    return json(buildAssetStatistics())
  }
  if (method === "GET" && path === "/assetStatistics/history") {
    const days = parseNumeric(url.searchParams.get("days")) || 7
    return json(buildStatisticsHistory(days))
  }
  if (method === "GET" && path === "/admin/system/runtimeMetrics/current") {
    return json(mockServerRuntimeMetrics)
  }
  if (method === "GET" && path === "/settings/apiKeys") {
    return json(getMockApiKeySettings())
  }
  if (method === "PATCH" && path === "/settings/apiKeys") {
    return json(updateMockApiKeySettings(await request.json()))
  }
  if (method === "GET" && path === "/settings/loginVisual") {
    return json(getMockLoginVisualSettings())
  }
  if (method === "GET" && path === "/settings/loginVisual:checkDiscoverability") {
    return json(getMockLoginVisualDiscoverability())
  }
  if (method === "POST" && path === "/settings/loginVisual:unlockDiscoverability") {
    return json(unlockMockLoginVisualDiscoverability())
  }
  if (method === "POST" && path === "/settings/loginVisual:upload") {
    const form = await request.formData()
    const file = form.get("file")
    if (!(file instanceof File) || file.size === 0) {
      return json({ error: { code: "INVALID_ARGUMENT", message: "Upload one image or video file." } }, { status: 400 })
    }
    return json(uploadMockLoginVisual(file.type))
  }
  if (method === "POST" && path === "/settings/loginVisual:publish") {
    return json(publishMockLoginVisual())
  }
  if (method === "POST" && path === "/settings/loginVisual:restoreDefault") {
    return json(restoreMockLoginVisual())
  }
  if (method === "GET" && path === "/loginVisual/current") {
    return json(getMockPublicLoginVisual())
  }
  if (method === "GET" && path === "/users/me/mcpKey") {
    return json(getMockMcpKeyStatus())
  }
  if (method === "POST" && path === "/users/me:generateMcpKey") {
    return json(generateMockMcpKey())
  }
  if (method === "GET" && path === "/blacklistPolicy") {
    return json(getMockGlobalBlacklistPolicy())
  }
  if (method === "PATCH" && path === "/blacklistPolicy") {
    if (url.searchParams.get("updateMask") !== "patterns" || url.searchParams.size !== 1) {
      return json({ error: { code: "INVALID_ARGUMENT", message: "updateMask must be exactly patterns" } }, { status: 400 })
    }
    const result = patchMockGlobalBlacklistPolicy(await request.json())
    return json(result.body, { status: result.status })
  }
  if (method === "GET" && path === "/databaseHealthReports/current") {
    return json((await import("../data/database-health")).mockDatabaseHealth)
  }
  if (method === "GET" && path === "/system/disk-stats") {
    return json(getMockDiskStats())
  }
  if (method === "GET" && path === "/system/version") {
    return json(getMockVersionInfo())
  }
  if (method === "GET" && path === "/system/check-update") {
    return json(getMockUpdateCheckResult())
  }
  if (method === "GET" && path === "/users/current/notifications") {
    return json(getMockNotifications({
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || undefined,
      pageToken: url.searchParams.get("pageToken") || undefined,
    }))
  }
  if (method === "GET" && path === "/users/current/notifications:unreadCount") {
    return json(getMockUnreadCount())
  }
  if (method === "GET" && path === "/users/current/notifications:stream") {
    const authorization = request.headers.get("authorization")
    if (!authorization?.startsWith("Bearer ")) {
      return json(
        {
          error: {
            code: 401,
            status: "UNAUTHENTICATED",
            message: "Notification stream token is required.",
            details: [{ reason: "TOKEN_EXPIRED" }],
          },
        },
        { status: 401 }
      )
    }
    return notificationRefreshStream()
  }
  if (method === "POST" && path === "/users/current/notifications:markAllRead") {
    markAllMockNotificationsRead()
    return empty()
  }
  {
    const notificationReadMatch = path.match(/^\/users\/current\/notifications\/(\d+):markRead$/)
    if (method === "POST" && notificationReadMatch) {
      const notification = markMockNotificationRead(`users/current/notifications/${notificationReadMatch[1]}`)
      return notification
        ? json(notification)
        : json({ error: { code: 404, status: "NOT_FOUND", message: "Notification not found." } }, { status: 404 })
    }
  }
  {
    const notificationMatch = path.match(/^\/users\/current\/notifications\/(\d+)$/)
    if (method === "GET" && notificationMatch) {
      const notification = getMockNotification(`users/current/notifications/${notificationMatch[1]}`)
      return notification
        ? json(notification)
        : json({ error: { code: 404, status: "NOT_FOUND", message: "Notification not found." } }, { status: 404 })
    }
  }
  if (method === "GET" && path === "/users/current/notificationLocale") {
    return json(getMockNotificationLocale())
  }
  if (method === "PATCH" && path === "/users/current/notificationLocale") {
    const body = await request.json() as { locale?: unknown }
    if (body.locale !== "zh" && body.locale !== "en") {
      throw new TypeError("Notification locale is unsupported.")
    }
    return json(updateMockNotificationLocale(body.locale))
  }
  if (method === "GET" && path === "/settings/notificationDestinations") {
    return json(getMockNotificationDestinations())
  }
  {
    const destinationTestMatch = path.match(/^\/settings\/notificationDestinations\/([^/:]+):testDelivery$/)
    const provider = destinationTestMatch?.[1] as NotificationDestinationProvider | undefined
    if (method === "POST" && provider && NOTIFICATION_DESTINATION_PROVIDERS.includes(provider)) {
      return json(getMockNotificationTestDeliveryUnavailableResult())
    }
  }
  {
    const destinationMatch = path.match(/^\/settings\/notificationDestinations\/([^/:]+)$/)
    const provider = destinationMatch?.[1] as NotificationDestinationProvider | undefined
    if (provider && NOTIFICATION_DESTINATION_PROVIDERS.includes(provider)) {
      if (method === "GET") {
        const destination = getMockNotificationDestination(provider)
        return destination
          ? json(destination)
          : json({ error: { code: 404, status: "NOT_FOUND", message: "Notification destination not found." } }, { status: 404 })
      }
      if (method === "PATCH") {
        const destination = updateMockNotificationDestination(provider, await request.json() as never)
        return destination
          ? json(destination)
          : json({ error: { code: 404, status: "NOT_FOUND", message: "Notification destination not found." } }, { status: 404 })
      }
    }
  }
  if (method === "GET" && path === "/admin/system/logEntries") {
    const direction = url.searchParams.get("direction")
    return json(getMockSystemLogs({
      lines: parseNumeric(url.searchParams.get("pageSize")),
      cursor: url.searchParams.get("pageToken") ?? undefined,
      direction: direction === "newer" || direction === "older" ? direction : undefined,
    }))
  }
  {
    const organizationSubdomainsMatch = path.match(/^\/organizations\/(\d+)\/domains$/)
    if (method === "GET" && organizationSubdomainsMatch) {
      return json(buildOrganizationSubdomains(Number.parseInt(organizationSubdomainsMatch[1], 10), {
        page: parseNumeric(url.searchParams.get("page")) || 1,
        pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
      }))
    }
  }

  if (method === "GET" && path === "/organizations") {
    return json(toAipOrganizationsResponse(buildOrganizations({
      page: parseMockPageToken(url.searchParams.get("pageToken"), "mock-organization-page"),
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
      filter: url.searchParams.get("filter") || undefined,
    })))
  }
  if (method === "POST" && path === "/organizations:batchDelete") {
    const body = (await request.json()) as { names?: string[] }
    return json(bulkDeleteMockOrganizations((body.names ?? []).map((name) => parseIdFromResourceName(name, "organizations")).filter(Number.isFinite) as number[]))
  }
  {
    const organizationMatch = path.match(/^\/organizations\/(\d+)$/)
    if (method === "GET" && organizationMatch) {
      const organizationId = Number.parseInt(organizationMatch[1], 10)
      const organization = buildOrganizations({ page: 1, pageSize: 500 }).results.find(
        (item) => item.id === organizationId
      )
      return organization ? json(toAipOrganization(organization)) : json({ error: "Organization not found" }, { status: 404 })
    }
    if (method === "PATCH" && organizationMatch) {
      const organizationId = Number.parseInt(organizationMatch[1], 10)
      const body = (await request.json()) as { displayName?: string; description?: string }
      const organization = mockOrganizations.find((item) => item.id === organizationId)
      if (!organization) return json({ error: "Organization not found" }, { status: 404 })
      if (body.displayName !== undefined) organization.name = body.displayName
      if (body.description !== undefined) organization.description = body.description
      organization.updatedAt = new Date().toISOString()
      return json(toAipOrganization(organization))
    }
    if (method === "DELETE" && organizationMatch) {
      const organizationId = Number.parseInt(organizationMatch[1], 10)
      const organization = deleteMockOrganization(organizationId)
      return organization
        ? empty()
        : json({ error: "Organization not found" }, { status: 404 })
    }
  }
  {
    const organizationTargetsMatch = path.match(/^\/organizations\/(\d+)\/targets$/)
    if (method === "GET" && organizationTargetsMatch) {
      return json(buildOrganizationTargets(Number.parseInt(organizationTargetsMatch[1], 10), url))
    }
  }
  {
    const organizationDomainRemoveMatch = path.match(/^\/organizations\/(\d+)\/domains\/batch-remove$/)
    if (method === "POST" && organizationDomainRemoveMatch) {
      const body = (await request.json()) as { domainIds?: number[] }
      return json(bulkRemoveMockOrganizationSubdomains(body.domainIds ?? []))
    }
  }

  if (method === "GET" && path === "/targets") {
    return json(toLegacyTargetsResponse(buildTargets({
      page: parseMockPageToken(url.searchParams.get("pageToken"), "mock-target-page"),
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
      search: url.searchParams.get("filter") || undefined,
      orderBy: url.searchParams.get("orderBy"),
    })))
  }
  if (method === "POST" && path === "/targets") {
    const body = (await request.json()) as { name?: string }
    const now = new Date().toISOString()
    const target = {
      id: Math.max(0, ...mockTargets.map((item) => item.id)) + 1,
      name: body.name || "example.com",
      type: "domain" as const,
      createdAt: now,
      organizations: [],
    }
    mockTargets.unshift(target)
    return json(toAipTarget(target), { status: 201 })
  }
  if (method === "POST" && path === "/targets:batchCreate") {
    const body = (await request.json()) as { targets?: Array<{ name?: string }> }
    const now = new Date().toISOString()
    let createdCount = 0
    for (const item of body.targets ?? []) {
      if (!item.name) continue
      const target = {
        id: Math.max(0, ...mockTargets.map((existing) => existing.id)) + 1,
        name: item.name,
        type: "domain" as const,
        createdAt: now,
        organizations: [],
      }
      mockTargets.unshift(target)
      createdCount += 1
    }
    return json({
      createdCount,
      failedCount: 0,
      failedTargets: [],
      message: "ok",
    }, { status: 201 })
  }
  if (method === "POST" && path === "/targets:batchDelete") {
    const body = (await request.json()) as { names?: string[] }
    return json(bulkDeleteMockTargets((body.names ?? []).map((name) => parseIdFromResourceName(name, "targets")).filter(Number.isFinite) as number[]))
  }
  {
    const targetMatch = path.match(/^\/targets\/(\d+)$/)
    if (method === "GET" && targetMatch) {
      const targetId = Number.parseInt(targetMatch[1], 10)
      const target = buildTargetDetail(targetId)
      return target ? json(toAipTarget(target)) : json({ error: "Target not found" }, { status: 404 })
    }
    if (method === "PATCH" && targetMatch) {
      const targetId = Number.parseInt(targetMatch[1], 10)
      const body = (await request.json()) as { displayName?: string }
      const target = mockTargets.find((item) => item.id === targetId)
      if (!target) return json({ error: "Target not found" }, { status: 404 })
      target.name = body.displayName || target.name
      return json(toAipTarget(target))
    }
    if (method === "DELETE" && targetMatch) {
      const targetId = Number.parseInt(targetMatch[1], 10)
      return deleteMockTarget(targetId)
        ? empty()
        : json({ error: "Target not found" }, { status: 404 })
    }
  }
  {
    const targetEndpointsMatch = path.match(/^\/targets\/(\d+)\/endpoints$/)
    if (method === "GET" && targetEndpointsMatch) {
      return json(
        buildTargetEndpoints(
          Number.parseInt(targetEndpointsMatch[1], 10),
          {
            pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
            pageToken: url.searchParams.get("pageToken") || undefined,
            filter: url.searchParams.get("filter") || undefined,
            orderBy: url.searchParams.get("orderBy") || undefined,
          },
        )
      )
    }
  }
  {
    const targetEndpointFilterOptionsMatch = path.match(/^\/targets\/(\d+)\/endpoints\/filterOptions$/)
    if (method === "GET" && targetEndpointFilterOptionsMatch) {
      const field = parseEndpointFilterOptionField(url.searchParams.get("field"))
      if (!field) return json({ error: "Unsupported field" }, { status: 400 })
      return json(buildEndpointFilterOptions({
        source: "target",
        sourceId: Number.parseInt(targetEndpointFilterOptionsMatch[1], 10),
        field,
      }))
    }
  }
  {
    const targetEndpointExportMatch = path.match(/^\/targets\/(\d+)\/endpoints\/exportFiles\/current$/)
    if (method === "GET" && targetEndpointExportMatch) {
      return text("url\nhttps://example.com/api\n", "text/csv; charset=utf-8")
    }
  }
  {
    const targetEndpointBatchCreateMatch = path.match(/^\/targets\/(\d+)\/endpoints:batchCreate$/)
    if (method === "POST" && targetEndpointBatchCreateMatch) {
      const body = (await request.json()) as { urls?: string[] }
      return json({ createdCount: body.urls?.length ?? 0 })
    }
  }
  {
    const targetBlacklistMatch = path.match(/^\/targets\/(\d+)\/blacklistPolicy$/)
    if (targetBlacklistMatch) {
      const targetId = Number.parseInt(targetBlacklistMatch[1], 10)
      if (method === "GET") {
        return json(getMockTargetBlacklistPolicy(targetId))
      }
      if (method === "PATCH") {
        if (url.searchParams.get("updateMask") !== "patterns" || url.searchParams.size !== 1) {
          return json({ error: { code: "INVALID_ARGUMENT", message: "updateMask must be exactly patterns" } }, { status: 400 })
        }
        const result = patchMockTargetBlacklistPolicy(targetId, await request.json())
        return json(result.body, { status: result.status })
      }
    }
  }
  {
    const targetDirectoryFilterOptionsMatch = path.match(/^\/targets\/(\d+)\/directories\/filterOptions$/)
    if (method === "GET" && targetDirectoryFilterOptionsMatch) {
      const field = parseDirectoryFilterOptionField(url.searchParams.get("field"))
      if (!field) {
        return json({ error: "Unsupported directory filter option field" }, { status: 400 })
      }
      return json(getMockDirectoryFilterOptions(field))
    }
  }
  {
    const targetDirectoriesMatch = path.match(/^\/targets\/(\d+)\/directories$/)
    if (method === "GET" && targetDirectoriesMatch) {
      return json(buildDirectoryResponse(url, Number.parseInt(targetDirectoriesMatch[1], 10)))
    }
  }
  {
    const targetDirectoryExportMatch = path.match(/^\/targets\/(\d+)\/directories\/exportFiles\/current$/)
    if (method === "GET" && targetDirectoryExportMatch) {
      return text("url\nhttps://example.com/admin\n", "text/csv; charset=utf-8")
    }
  }
  {
    const targetDirectoryBatchCreateMatch = path.match(/^\/targets\/(\d+)\/directories:batchCreate$/)
    if (method === "POST" && targetDirectoryBatchCreateMatch) {
      const body = (await request.json()) as { urls?: string[] }
      return json({ createdCount: body.urls?.length ?? 0 })
    }
  }
  {
    const targetWebsiteFilterOptionsMatch = path.match(/^\/targets\/(\d+)\/websites\/filterOptions$/)
    if (method === "GET" && targetWebsiteFilterOptionsMatch) {
      const field = parseWebsiteFilterOptionField(url.searchParams.get("field"))
      if (!field) {
        return json({ error: "Unsupported website filter option field" }, { status: 400 })
      }
      return json(getMockWebsiteFilterOptions({
        field,
        targetId: Number.parseInt(targetWebsiteFilterOptionsMatch[1], 10),
      }))
    }
  }
  {
    const targetWebsitesMatch = path.match(/^\/targets\/(\d+)\/websites$/)
    if (method === "GET" && targetWebsitesMatch) {
      return json(toAipWebsiteList(getMockWebsites({
        pageToken: url.searchParams.get("pageToken") || undefined,
        pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
        filter: url.searchParams.get("filter") || undefined,
        targetId: Number.parseInt(targetWebsitesMatch[1], 10),
      })))
    }
  }
  {
    const targetWebsiteExportMatch = path.match(/^\/targets\/(\d+)\/websites\/exportFiles\/current$/)
    if (method === "GET" && targetWebsiteExportMatch) {
      return text("url\nhttps://example.com\n", "text/csv; charset=utf-8")
    }
  }
  {
    const targetWebsiteBatchCreateMatch = path.match(/^\/targets\/(\d+)\/websites:batchCreate$/)
    if (method === "POST" && targetWebsiteBatchCreateMatch) {
      const body = (await request.json()) as { urls?: string[] }
      return json({ createdCount: body.urls?.length ?? 0 })
    }
  }
  {
    const targetSubdomainsMatch = path.match(/^\/targets\/(\d+)\/subdomains$/)
    if (method === "GET" && targetSubdomainsMatch) {
      return json(buildSubdomainResponse(url, "target", Number.parseInt(targetSubdomainsMatch[1], 10)))
    }
  }
  {
    const targetSubdomainExportMatch = path.match(/^\/targets\/(\d+)\/subdomains\/exportFiles\/current$/)
    if (method === "GET" && targetSubdomainExportMatch) {
      return text("dnsName\napi.example.com\n", "text/csv; charset=utf-8")
    }
  }
  {
    const targetSubdomainBatchCreateMatch = path.match(/^\/targets\/(\d+)\/subdomains:batchCreate$/)
    if (method === "POST" && targetSubdomainBatchCreateMatch) {
      const body = (await request.json()) as { dnsNames?: string[] }
      return json({ createdCount: body.dnsNames?.length ?? 0, skippedCount: 0, invalidCount: 0, mismatchedCount: 0, totalReceived: body.dnsNames?.length ?? 0 })
    }
  }
  {
    const targetHostPortFilterOptionsMatch = path.match(/^\/targets\/(\d+)\/hostPorts\/filterOptions$/)
    if (method === "GET" && targetHostPortFilterOptionsMatch) {
      if (url.searchParams.get("field") !== "port") {
        return json({ error: "Unsupported hostPort filter option field" }, { status: 400 })
      }
      const target = getMockTargetById(Number.parseInt(targetHostPortFilterOptionsMatch[1], 10))
      return json(getMockPortOptions({ domain: target?.name }))
    }
  }
  {
    const targetHostPortsMatch = path.match(/^\/targets\/(\d+)\/hostPorts$/)
    if (method === "GET" && targetHostPortsMatch) {
      return json(buildIpResponse(url, Number.parseInt(targetHostPortsMatch[1], 10)))
    }
  }
  {
    const targetHostPortsExportMatch = path.match(/^\/targets\/(\d+)\/hostPorts\/exportFiles\/current$/)
    if (method === "GET" && targetHostPortsExportMatch) {
      return text("ip,host,port\n192.0.2.1,api.example.com,443\n", "text/csv; charset=utf-8")
    }
  }
  {
    const targetScreenshotFilterOptionsMatch = path.match(/^\/targets\/(\d+)\/screenshots\/filterOptions$/)
    if (method === "GET" && targetScreenshotFilterOptionsMatch) {
      const field = url.searchParams.get("field")
      if (field !== "statusCode") {
        return json({ error: "Unsupported screenshot filter option field" }, { status: 400 })
      }
      return json(buildTargetScreenshotFilterOptions(Number.parseInt(targetScreenshotFilterOptionsMatch[1], 10), field))
    }
  }
  {
    const targetScreenshotsMatch = path.match(/^\/targets\/(\d+)\/screenshots$/)
    if (method === "GET" && targetScreenshotsMatch) {
      return json(buildScreenshotsByTarget(Number.parseInt(targetScreenshotsMatch[1], 10), {
        pageSize: parseNumeric(url.searchParams.get("pageSize")) || undefined,
        pageToken: url.searchParams.get("pageToken") || undefined,
        filter: url.searchParams.get("filter") || undefined,
        orderBy: url.searchParams.get("orderBy") || undefined,
      }))
    }
  }

  if (method === "POST" && path === "/websites:batchDelete") {
    const body = (await request.json()) as { names?: string[] }
    return json(bulkDeleteMockWebsites(parseNestedIdsFromResourceNames(body.names, "targets", "websites")))
  }
  {
    const websiteMatch = path.match(/^\/websites\/(\d+)$/)
    if (method === "GET" && websiteMatch) {
      const website = getMockWebsiteById(Number.parseInt(websiteMatch[1], 10))
      return website
        ? json(toAipWebsite(website))
        : json({ error: "Website not found" }, { status: 404 })
    }
    if (method === "DELETE" && websiteMatch) {
      const websiteId = Number.parseInt(websiteMatch[1], 10)
      const website = deleteMockWebsite(websiteId)
      return website
        ? json({
            message: "Deleted website",
            websiteId,
            websiteUrl: website.url,
            deletedCount: 1,
            deletedWebSites: [website.url],
            detail: {
              phase1: "mock unlink complete",
              phase2: "mock delete complete",
            },
          })
        : json({ error: "Website not found" }, { status: 404 })
    }
  }

  if (method === "POST" && path === "/directories:batchDelete") {
    const body = (await request.json()) as { names?: string[] }
    return json(bulkDeleteMockDirectories(parseNestedIdsFromResourceNames(body.names, "targets", "directories")))
  }
  {
    const directoryMatch = path.match(/^\/directories\/(\d+)$/)
    if (method === "DELETE" && directoryMatch) {
      const directoryId = Number.parseInt(directoryMatch[1], 10)
      const directory = deleteMockDirectory(directoryId)
      return directory
        ? json({
            message: "Deleted directory",
            directoryId,
            directoryUrl: directory.url,
            deletedCount: 1,
            deletedDirectories: [directory.url],
            detail: {
              phase1: "mock unlink complete",
              phase2: "mock delete complete",
            },
          })
        : json({ error: "Directory not found" }, { status: 404 })
    }
  }

  if (method === "POST" && path === "/subdomains:batchDelete") {
    const body = (await request.json()) as { names?: string[] }
    return json(bulkDeleteMockSubdomains(parseNestedIdsFromResourceNames(body.names, "targets", "subdomains")))
  }
  {
    const subdomainDeleteMatch = path.match(/^\/subdomains\/(\d+)$/)
    if (method === "DELETE" && subdomainDeleteMatch) {
      const subdomainId = Number.parseInt(subdomainDeleteMatch[1], 10)
      const subdomain = deleteMockSubdomain(subdomainId)
      return subdomain
        ? json({
            message: "Deleted subdomain",
            subdomainId,
            subdomainName: subdomain.name,
            deletedCount: 1,
            deletedSubdomains: [subdomain.name],
            detail: {
              phase1: "mock unlink complete",
              phase2: "mock delete complete",
            },
          })
        : json({ error: "Subdomain not found" }, { status: 404 })
    }
  }

  if (method === "POST" && path === "/hostPorts:batchDelete") {
    const body = (await request.json()) as { ips?: string[] }
    return json(bulkDeleteMockIPAddresses(body.ips ?? []))
  }

  if (method === "GET" && path === "/scans") {
    return json(buildScans({
      page: parseNumeric(url.searchParams.get("page")) || 1,
      pageToken: url.searchParams.get("pageToken") || undefined,
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
      target: parseNumeric(url.searchParams.get("target")) || undefined,
      filter: url.searchParams.get("filter") || undefined,
      orderBy: url.searchParams.get("orderBy") || undefined,
      search: url.searchParams.get("search") || undefined,
      status: (url.searchParams.get("status") as ScanRecord["status"] | null) || undefined,
    }))
  }
  if (method === "GET" && path === "/scanStatistics") {
    return json(buildScanStatistics())
  }
  if (method === "POST" && path === "/scans") {
    const body = (await request.json()) as {
      target?: string
      scanWorkflow?: string
      configuration?: unknown
      inputSource?: unknown
    }
    const workflow = body.scanWorkflow ? getMockScanWorkflowByName(body.scanWorkflow) : undefined
    if (!workflow || body.configuration === undefined || !isScanInputSource(body.inputSource)) {
      return json({ error: "scanWorkflow, complete configuration, and inputSource are required" }, { status: 400 })
    }
    let configuration: ReturnType<typeof serializeCanonicalWorkflowConfiguration>
    try {
      configuration = serializeMockWorkflowConfiguration(body.configuration, workflow)
    } catch (error) {
      return json({ error: error instanceof Error ? error.message : "invalid workflow configuration" }, { status: 400 })
    }
    const targetId = parseIdFromResourceName(body.target, "targets") ?? 0
    const now = new Date().toISOString()
    return json({
      id: Date.now(),
      name: `scans/${Date.now()}`,
      targetId,
      scanWorkflow: body.scanWorkflow || "scanWorkflows/default",
      configuration,
      plannedEngineIds: [],
      triggerType: "manual",
      inputSource: body.inputSource,
      status: "pending",
      progress: 0,
      currentStage: "",
      createdAt: now,
    }, { status: 201 })
  }
  if (method === "POST" && path === "/scans:batchCreate") {
    const body = (await request.json()) as {
      requests?: Array<{ target?: string; organization?: string }>
      scanWorkflow?: string
      configuration?: unknown
      inputSource?: unknown
    }
    const workflow = body.scanWorkflow ? getMockScanWorkflowByName(body.scanWorkflow) : undefined
    if (!workflow || body.configuration === undefined || !isScanInputSource(body.inputSource)) {
      return json({ error: "scanWorkflow, complete configuration, and inputSource are required" }, { status: 400 })
    }
    let configuration: ReturnType<typeof serializeCanonicalWorkflowConfiguration>
    try {
      configuration = serializeMockWorkflowConfiguration(body.configuration, workflow)
    } catch (error) {
      return json({ error: error instanceof Error ? error.message : "invalid workflow configuration" }, { status: 400 })
    }
    const now = new Date().toISOString()
    const scans = (body.requests ?? []).map((item, index) => {
      const targetId = parseIdFromResourceName(item.target, "targets") ?? parseIdFromResourceName(item.organization, "organizations") ?? index + 1
      const id = Date.now() + index
      return {
        id,
        name: `scans/${id}`,
        targetId,
        scanWorkflow: body.scanWorkflow || "scanWorkflows/default",
        configuration,
        plannedEngineIds: [],
        triggerType: "manual",
        inputSource: body.inputSource,
        status: "pending",
        progress: 0,
        currentStage: "",
        createdAt: now,
      }
    })
    return json({
      count: scans.length,
      createdCount: scans.length,
      skipped: [],
      failed: [],
      scans,
    }, { status: 201 })
  }
  if (method === "POST" && path === "/scans:quickCreate") {
    const body = (await request.json()) as {
      scanWorkflow?: string
      configuration?: unknown
      inputSource?: unknown
    }
    const workflow = body.scanWorkflow ? getMockScanWorkflowByName(body.scanWorkflow) : undefined
    if (!workflow || body.configuration === undefined || !isScanInputSource(body.inputSource)) {
      return json({ error: "scanWorkflow, complete configuration, and inputSource are required" }, { status: 400 })
    }
    try {
      serializeMockWorkflowConfiguration(body.configuration, workflow)
    } catch (error) {
      return json({ error: error instanceof Error ? error.message : "invalid workflow configuration" }, { status: 400 })
    }
    return json({
      count: 0,
      targetStats: { created: 0, skipped: 0, failed: 0 },
      assetStats: { websites: 0, endpoints: 0 },
      errors: [],
      scans: [],
    }, { status: 501 })
  }
  {
    const scanMatch = path.match(/^\/scans\/(\d+)$/)
    if (method === "GET" && scanMatch) {
      const scan = buildScanById(Number.parseInt(scanMatch[1], 10))
      return scan ? json(scan) : json({ error: "Scan not found" }, { status: 404 })
    }
    if (method === "DELETE" && scanMatch) {
      const scanId = Number.parseInt(scanMatch[1], 10)
      return deleteMockScan(scanId) ? empty() : json({ error: "Scan not found" }, { status: 404 })
    }
  }
  if (method === "POST" && path === "/scans:batchDelete") {
    const body = (await request.json()) as { names?: string[] }
    return json(bulkDeleteMockScans((body.names ?? []).map((name) => parseIdFromResourceName(name, "scans")).filter(Number.isFinite) as number[]))
  }
  if (method === "POST" && path === "/scans:batchStop") {
    const body = (await request.json()) as { names?: unknown }
    const names = Array.isArray(body.names) && body.names.every((name): name is string => typeof name === "string")
      ? body.names
      : []
    const matches = names.map((name) => /^scans\/([1-9]\d*)$/.exec(name))
    const ids = matches.map((match) => match ? Number(match[1]) : NaN)
    if (
      names.length < 1 ||
      names.length > 100 ||
      names.some((name, index) => !matches[index] || !Number.isSafeInteger(ids[index]) || ids[index] <= 0 || name !== `scans/${ids[index]}`) ||
      new Set(names).size !== names.length
    ) {
      return json({ error: { code: "INVALID_ARGUMENT", message: "Invalid scan names." } }, { status: 400 })
    }
    if (ids.some((id) => !getMockScanById(id))) {
      return json({ error: { code: "NOT_FOUND", message: "Scan not found." } }, { status: 404 })
    }
    return json(batchStopMockScans(ids))
  }
  if (method === "POST" && path === "/scans/deletions") {
    const body = (await request.json()) as { ids?: number[] }
    return json(bulkDeleteMockScans(body.ids ?? []))
  }
  {
    const scanStopMatch = path.match(/^\/scans\/(\d+):stop$/)
    if (method === "POST" && scanStopMatch) {
      return json(stopMockScan(Number.parseInt(scanStopMatch[1], 10)))
    }
  }
  {
    const scanStoppageMatch = path.match(/^\/scans\/(\d+)\/stoppages$/)
    if (method === "POST" && scanStoppageMatch) {
      return json(stopMockScan(Number.parseInt(scanStoppageMatch[1], 10)))
    }
  }
  {
    const scanLogsMatch = path.match(/^\/scans\/(\d+)\/(?:logs|taskProgressLogs)$/)
    if (method === "GET" && scanLogsMatch) {
      const scanId = Number.parseInt(scanLogsMatch[1], 10)
      return json(getMockScanLogs(scanId, {
        pageSize: parseNumeric(url.searchParams.get("pageSize")),
        pageToken: url.searchParams.get("pageToken") || undefined,
      }))
    }
  }
  {
    const scanEndpointsMatch = path.match(/^\/scans\/(\d+)\/endpoints$/)
    if (method === "GET" && scanEndpointsMatch) {
      return json(
        buildEndpointResults({
          source: "scan",
          sourceId: Number.parseInt(scanEndpointsMatch[1], 10),
          pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
          pageToken: url.searchParams.get("pageToken") || undefined,
          filter: url.searchParams.get("filter") || undefined,
          orderBy: url.searchParams.get("orderBy") || undefined,
        })
      )
    }
  }
  {
    const scanEndpointFilterOptionsMatch = path.match(/^\/scans\/(\d+)\/endpoints\/filterOptions$/)
    if (method === "GET" && scanEndpointFilterOptionsMatch) {
      const field = parseEndpointFilterOptionField(url.searchParams.get("field"))
      if (!field) return json({ error: "Unsupported field" }, { status: 400 })
      return json(buildEndpointFilterOptions({
        source: "scan",
        sourceId: Number.parseInt(scanEndpointFilterOptionsMatch[1], 10),
        field,
      }))
    }
  }
  {
    const scanEndpointExportMatch = path.match(/^\/scans\/(\d+)\/endpoints\/exportFiles\/current$/)
    if (method === "GET" && scanEndpointExportMatch) {
      return text("url\nhttps://example.com/api\n", "text/csv; charset=utf-8")
    }
  }
  {
    const scanDirectoryFilterOptionsMatch = path.match(/^\/scans\/(\d+)\/directories\/filterOptions$/)
    if (method === "GET" && scanDirectoryFilterOptionsMatch) {
      const field = parseDirectoryFilterOptionField(url.searchParams.get("field"))
      if (!field) {
        return json({ error: "Unsupported directory filter option field" }, { status: 400 })
      }
      return json(getMockDirectoryFilterOptions(field))
    }
  }
  {
    const scanDirectoriesMatch = path.match(/^\/scans\/(\d+)\/directories$/)
    if (method === "GET" && scanDirectoriesMatch) {
      return json(buildDirectoryResponse(url, undefined, Number.parseInt(scanDirectoriesMatch[1], 10)))
    }
  }
  {
    const scanDirectoryExportMatch = path.match(/^\/scans\/(\d+)\/directories\/exportFiles\/current$/)
    if (method === "GET" && scanDirectoryExportMatch) {
      return text("url\nhttps://example.com/admin\n", "text/csv; charset=utf-8")
    }
  }
  {
    const scanWebsiteFilterOptionsMatch = path.match(/^\/scans\/(\d+)\/websites\/filterOptions$/)
    if (method === "GET" && scanWebsiteFilterOptionsMatch) {
      const field = parseWebsiteFilterOptionField(url.searchParams.get("field"))
      if (!field) {
        return json({ error: "Unsupported website filter option field" }, { status: 400 })
      }
      const scan = getMockScanById(Number.parseInt(scanWebsiteFilterOptionsMatch[1], 10))
      return json(getMockWebsiteFilterOptions({
        field,
        targetId: scan?.targetId,
      }))
    }
  }
  {
    const scanWebsitesMatch = path.match(/^\/scans\/(\d+)\/websites$/)
    if (method === "GET" && scanWebsitesMatch) {
      const scanId = Number.parseInt(scanWebsitesMatch[1], 10)
      const scan = getMockScanById(scanId)
      return json(toAipWebsiteList(getMockWebsites({
        pageToken: url.searchParams.get("pageToken") || undefined,
        pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
        filter: url.searchParams.get("filter") || undefined,
        targetId: scan?.targetId,
      })))
    }
  }
  {
    const scanWebsiteExportMatch = path.match(/^\/scans\/(\d+)\/websites\/exportFiles\/current$/)
    if (method === "GET" && scanWebsiteExportMatch) {
      return text("url\nhttps://example.com\n", "text/csv; charset=utf-8")
    }
  }
  {
    const scanSubdomainsMatch = path.match(/^\/scans\/(\d+)\/subdomains$/)
    if (method === "GET" && scanSubdomainsMatch) {
      return json(buildSubdomainResponse(url, "scan", Number.parseInt(scanSubdomainsMatch[1], 10)))
    }
  }
  {
    const scanSubdomainExportMatch = path.match(/^\/scans\/(\d+)\/subdomains\/exportFiles\/current$/)
    if (method === "GET" && scanSubdomainExportMatch) {
      return text("dnsName\napi.example.com\n", "text/csv; charset=utf-8")
    }
  }
  {
    const scanHostPortFilterOptionsMatch = path.match(/^\/scans\/(\d+)\/hostPorts\/filterOptions$/)
    if (method === "GET" && scanHostPortFilterOptionsMatch) {
      if (url.searchParams.get("field") !== "port") {
        return json({ error: "Unsupported hostPort filter option field" }, { status: 400 })
      }
      const scan = getMockScanById(Number.parseInt(scanHostPortFilterOptionsMatch[1], 10))
      return json(getMockPortOptions({ domain: scan?.target?.name }))
    }
  }
  {
    const scanHostPortsMatch = path.match(/^\/scans\/(\d+)\/hostPorts$/)
    if (method === "GET" && scanHostPortsMatch) {
      return json(buildIpResponse(url, undefined, Number.parseInt(scanHostPortsMatch[1], 10)))
    }
  }
  {
    const scanHostPortsExportMatch = path.match(/^\/scans\/(\d+)\/hostPorts\/exportFiles\/current$/)
    if (method === "GET" && scanHostPortsExportMatch) {
      return text("ip,host,port\n192.0.2.1,api.example.com,443\n", "text/csv; charset=utf-8")
    }
  }
  {
    const scanScreenshotFilterOptionsMatch = path.match(/^\/scans\/(\d+)\/screenshots\/filterOptions$/)
    if (method === "GET" && scanScreenshotFilterOptionsMatch) {
      const field = url.searchParams.get("field")
      if (field !== "statusCode") {
        return json({ error: "Unsupported screenshot filter option field" }, { status: 400 })
      }
      return json(buildScanScreenshotFilterOptions(Number.parseInt(scanScreenshotFilterOptionsMatch[1], 10), field))
    }
  }
  {
    const scanScreenshotsMatch = path.match(/^\/scans\/(\d+)\/screenshots$/)
    if (method === "GET" && scanScreenshotsMatch) {
      return json(buildScreenshotsByScan(Number.parseInt(scanScreenshotsMatch[1], 10), {
        pageSize: parseNumeric(url.searchParams.get("pageSize")) || undefined,
        pageToken: url.searchParams.get("pageToken") || undefined,
        filter: url.searchParams.get("filter") || undefined,
        orderBy: url.searchParams.get("orderBy") || undefined,
      }))
    }
  }

  if (method === "GET" && path === "/vulnerabilities") {
    return json(buildCanonicalVulnerabilityResponse(url))
  }
  if (method === "GET" && path === "/vulnerabilities/filterOptions") {
    const response = buildVulnerabilityFilterOptions(url)
    return "error" in response ? json(response, { status: 400 }) : json(response)
  }
  if (method === "GET" && path === "/vulnerabilityStatistics") {
    return json(buildVulnerabilityStats())
  }
  if (method === "POST" && path === "/vulnerabilities:batchDelete") {
    const body = (await request.json()) as { names?: string[] }
    const ids = (body.names ?? [])
      .map((name) => parseIdFromResourceName(name, "vulnerabilities"))
      .filter(Number.isFinite) as number[]
    return json(bulkDeleteMockVulnerabilities(ids))
  }
  if (method === "POST" && path === "/vulnerabilities:batchUpdate") {
    const body = (await request.json()) as { requests?: Array<{ name?: string; isReviewed?: boolean }> }
    if (!Array.isArray(body.requests) || body.requests.some((item) => !item.name || typeof item.isReviewed !== "boolean")) {
      return json({ error: "Invalid vulnerability batch update request" }, { status: 400 })
    }
    return json(batchUpdateMockVulnerabilityReviewStates(body.requests ?? []))
  }
  {
    const vulnerabilityMatch = path.match(/^\/vulnerabilities\/(\d+)$/)
    if (method === "GET" && vulnerabilityMatch) {
      const vulnerability = buildVulnerabilityById(Number.parseInt(vulnerabilityMatch[1], 10))
      return vulnerability ? json(vulnerability) : json({ error: "Vulnerability not found" }, { status: 404 })
    }
    if (method === "PATCH" && vulnerabilityMatch) {
      const body = (await request.json()) as { isReviewed?: boolean }
      if (typeof body.isReviewed !== "boolean") {
        return json({ error: "Invalid vulnerability update request" }, { status: 400 })
      }
      const vulnerability = updateMockVulnerabilityReviewState(Number.parseInt(vulnerabilityMatch[1], 10), body.isReviewed)
      return vulnerability ? json(vulnerability) : json({ error: "Vulnerability not found" }, { status: 404 })
    }
  }
  {
    const targetVulnerabilityStatsMatch = path.match(/^\/targets\/(\d+)\/vulnerabilityStatistics$/)
    if (method === "GET" && targetVulnerabilityStatsMatch) {
      return json(buildVulnerabilityStats(Number.parseInt(targetVulnerabilityStatsMatch[1], 10)))
    }
  }
  {
    const scanVulnerabilitiesMatch = path.match(/^\/scans\/(\d+)\/vulnerabilities$/)
    if (method === "GET" && scanVulnerabilitiesMatch) {
      return json(buildCanonicalVulnerabilityResponse(url, { scanId: Number.parseInt(scanVulnerabilitiesMatch[1], 10) }))
    }
  }
  {
    const scanVulnerabilityFilterOptionsMatch = path.match(/^\/scans\/(\d+)\/vulnerabilities\/filterOptions$/)
    if (method === "GET" && scanVulnerabilityFilterOptionsMatch) {
      const response = buildVulnerabilityFilterOptions(url, { scanId: Number.parseInt(scanVulnerabilityFilterOptionsMatch[1], 10) })
      return "error" in response ? json(response, { status: 400 }) : json(response)
    }
  }
  {
    const targetVulnerabilitiesMatch = path.match(/^\/targets\/(\d+)\/vulnerabilities$/)
    if (method === "GET" && targetVulnerabilitiesMatch) {
      return json(buildCanonicalVulnerabilityResponse(url, { targetId: Number.parseInt(targetVulnerabilitiesMatch[1], 10) }))
    }
  }
  {
    const targetVulnerabilityFilterOptionsMatch = path.match(/^\/targets\/(\d+)\/vulnerabilities\/filterOptions$/)
    if (method === "GET" && targetVulnerabilityFilterOptionsMatch) {
      const response = buildVulnerabilityFilterOptions(url, { targetId: Number.parseInt(targetVulnerabilityFilterOptionsMatch[1], 10) })
      return "error" in response ? json(response, { status: 400 }) : json(response)
    }
  }

  if (method === "GET" && path === "/endpoints") {
    return json(buildEndpoints({
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
      pageToken: url.searchParams.get("pageToken") || undefined,
      filter: url.searchParams.get("filter") || undefined,
      orderBy: url.searchParams.get("orderBy") || undefined,
    }))
  }
  if (method === "POST" && path === "/endpoints:batchDelete") {
    const body = (await request.json()) as { names?: string[] }
    return json(batchDeleteMockEndpoints(parseNestedIdsFromResourceNames(body.names, "targets", "endpoints")))
  }
  {
    const endpointMatch = path.match(/^\/endpoints\/(\d+)$/)
    if (method === "GET" && endpointMatch) {
      const endpoint = getMockEndpointById(Number.parseInt(endpointMatch[1], 10))
      return endpoint ? json(endpoint) : json({ error: "Endpoint not found" }, { status: 404 })
    }
    if (method === "DELETE" && endpointMatch) {
      const endpointId = Number.parseInt(endpointMatch[1], 10)
      return deleteMockEndpoint(endpointId)
        ? empty()
        : json({ error: "Endpoint not found" }, { status: 404 })
    }
  }

  if (method === "GET" && path === "/domains") {
    return json(buildSubdomainResponse(url, "all"))
  }
  {
    const subdomainMatch = path.match(/^\/domains\/(\d+)$/)
    if (method === "GET" && subdomainMatch) {
      const subdomain = getMockSubdomainById(Number.parseInt(subdomainMatch[1], 10))
      return subdomain ? json(subdomain) : json({ error: "Subdomain not found" }, { status: 404 })
    }
  }

  if (method === "GET" && path === "/scheduledScans:summarize") {
    if (url.searchParams.has("timeZone")) {
      return json({ error: { code: "INVALID_ARGUMENT", message: "timeZone query parameter is no longer supported; schedules use UTC" } }, { status: 400 })
    }
    return json(getMockScheduledScanOverviewSummary())
  }
  if (method === "GET" && path === "/scheduledScans") {
    return json(getMockScheduledScans({
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
      pageToken: url.searchParams.get("pageToken") || undefined,
      search: url.searchParams.get("filter") || url.searchParams.get("search") || undefined,
      targetId: parseNumeric(url.searchParams.get("targetId")),
      organizationId: parseNumeric(url.searchParams.get("organizationId")),
    }))
  }
  if (method === "POST" && path === "/scheduledScans:batchUpdate") {
    const body = (await request.json()) as {
      requests?: Array<{ name?: string; isEnabled?: boolean; updateMask?: string }>
    }
    const requests = body.requests
    if (
      !Array.isArray(requests) ||
      requests.length < 1 ||
      requests.length > 100 ||
      requests.some((item) => (
        !/^scheduledScans\/[1-9]\d*$/.test(item.name ?? '') ||
        typeof item.isEnabled !== 'boolean' ||
        item.updateMask !== 'isEnabled'
      ))
    ) {
      return json({ error: "Invalid scheduled scan batch update request" }, { status: 400 })
    }

    const ids = requests.map((item) => Number.parseInt(item.name!.slice("scheduledScans/".length), 10))
    if (new Set(ids).size !== ids.length) {
      return json({ error: "Scheduled scan batch update request contains duplicate names" }, { status: 400 })
    }

    const schedules = ids.map((id) => getMockScheduledScanById(id))
    if (schedules.some((schedule) => schedule === undefined)) {
      return json({ error: "Scheduled scan not found" }, { status: 404 })
    }

    const now = new Date()
    for (let index = 0; index < requests.length; index += 1) {
      const schedule = schedules[index]!
      const isEnabled = requests[index]!.isEnabled!
      schedule.isEnabled = isEnabled
      schedule.nextRunTime = isEnabled
        ? getNextCronExecutions(schedule.cronExpression, now, 1)[0]?.toISOString() ?? null
        : null
      schedule.updatedAt = now.toISOString()
    }

    return json({ updatedCount: requests.length })
  }
  if (method === "POST" && path === "/scheduledScans") {
    const body = (await request.json()) as {
      displayName?: string
      scanWorkflow?: string
      target?: string
      organization?: string
      cronExpression?: string
      isEnabled?: boolean
      configuration?: ScheduledScan["configuration"]
      inputSource?: unknown
    }
    if ("timeZone" in (body as Record<string, unknown>)) {
      return json({ error: { code: "INVALID_ARGUMENT", message: "timeZone is no longer supported; schedules use UTC" } }, { status: 400 })
    }
    const now = new Date().toISOString()
    const workflow = body.scanWorkflow ? getMockScanWorkflowByName(body.scanWorkflow) : undefined
    if (!workflow || body.configuration === undefined || !isScanInputSource(body.inputSource)) {
      return json({ error: "scanWorkflow, complete configuration, and inputSource are required" }, { status: 400 })
    }
    if (!body.cronExpression || !isCronExpressionValid(body.cronExpression)) {
      return json({ error: "five-field cronExpression must be valid" }, { status: 400 })
    }
    let configuration: ScheduledScan["configuration"]
    try {
      configuration = serializeMockWorkflowConfiguration(body.configuration, workflow)
    } catch (error) {
      return json({ error: error instanceof Error ? error.message : "invalid workflow configuration" }, { status: 400 })
    }
    const targetId = parseIdFromResourceName(body.target, "targets") ?? null
    const organizationId = parseIdFromResourceName(body.organization, "organizations") ?? null
    const id = Math.max(0, ...mockScheduledScans.map((item) => item.id)) + 1
    const isEnabled = body.isEnabled ?? true
    const nextRunTime = isEnabled
      ? getNextCronExecutions(body.cronExpression, new Date(), 1)[0]?.toISOString() ?? null
      : null
    const scheduledScan: ScheduledScan = {
      id,
      name: `scheduledScans/${id}`,
      displayName: body.displayName || "Mock scheduled scan",
      scanWorkflow: body.scanWorkflow || "scanWorkflows/default",
      inputSource: body.inputSource,
      configuration,
      organization: body.organization ?? null,
      organizationId,
      organizationName: organizationId ? `Organization ${organizationId}` : null,
      target: body.target ?? null,
      targetId,
      targetName: targetId ? `Target ${targetId}` : null,
      scanMode: targetId ? "target" as const : "organization" as const,
      cronExpression: body.cronExpression,
      isEnabled,
      nextRunTime,
      lastRunTime: null,
      runCount: 0,
      successfulHandoffCount: 0,
      failedHandoffCount: 0,
      createdAt: now,
      updatedAt: now,
    }
    mockScheduledScans.unshift(scheduledScan)
    return json(scheduledScan, { status: 201 })
  }
  {
    const scheduledScanMatch = path.match(/^\/scheduledScans\/(\d+)$/)
    if (method === "GET" && scheduledScanMatch) {
      const scheduledScan = getMockScheduledScanById(Number.parseInt(scheduledScanMatch[1], 10))
      return scheduledScan ? json(scheduledScan) : json({ error: "Scheduled scan not found" }, { status: 404 })
    }
    if (method === "PATCH" && scheduledScanMatch) {
      const scheduledScanId = Number.parseInt(scheduledScanMatch[1], 10)
      const scheduledScan = getMockScheduledScanById(scheduledScanId)
      if (!scheduledScan) return json({ error: "Scheduled scan not found" }, { status: 404 })
      const body = (await request.json()) as Partial<typeof scheduledScan>
      if ("timeZone" in (body as Record<string, unknown>)) {
        return json({ error: { code: "INVALID_ARGUMENT", message: "timeZone is no longer supported; schedules use UTC" } }, { status: 400 })
      }
      if (body.configuration !== undefined && body.scanWorkflow === undefined) {
        return json({ error: "scanWorkflow is required when configuration is updated" }, { status: 400 })
      }
      if (body.inputSource !== undefined && !isScanInputSource(body.inputSource)) {
        return json({ error: "inputSource must be scanSnapshot or targetInventory" }, { status: 400 })
      }
      let configuration = scheduledScan.configuration
      if (body.configuration !== undefined) {
        const workflow = getMockScanWorkflowByName(body.scanWorkflow ?? scheduledScan.scanWorkflow)
        if (!workflow) return json({ error: "scanWorkflow not found" }, { status: 400 })
        try {
          configuration = serializeMockWorkflowConfiguration(body.configuration, workflow)
        } catch (error) {
          return json({ error: error instanceof Error ? error.message : "invalid workflow configuration" }, { status: 400 })
        }
      }
      const cronExpression = body.cronExpression ?? scheduledScan.cronExpression
      if (!isCronExpressionValid(cronExpression)) {
        return json({ error: "five-field cronExpression must be valid" }, { status: 400 })
      }
      const isEnabled = body.isEnabled ?? scheduledScan.isEnabled
      const timeRuleChanged = body.cronExpression !== undefined
      const nextRunTime = !isEnabled
        ? null
        : timeRuleChanged || body.isEnabled === true
          ? getNextCronExecutions(cronExpression, new Date(), 1)[0]?.toISOString() ?? null
          : scheduledScan.nextRunTime
      Object.assign(scheduledScan, {
        ...body,
        ...(body.configuration !== undefined ? { configuration } : {}),
        cronExpression,
        isEnabled,
        nextRunTime,
        name: `scheduledScans/${scheduledScanId}`,
        updatedAt: new Date().toISOString(),
      })
      return json(scheduledScan)
    }
    if (method === "DELETE" && scheduledScanMatch) {
      const scheduledScanId = Number.parseInt(scheduledScanMatch[1], 10)
      return deleteMockScheduledScan(scheduledScanId)
        ? empty()
        : json({ error: "Scheduled scan not found" }, { status: 404 })
    }
  }

  if (method === "GET" && path === "/scanWorkflows") {
    const pageSize = Math.min(Math.max(Number.parseInt(url.searchParams.get("pageSize") || "20", 10) || 20, 1), 50)
    const page = parseMockPageToken(url.searchParams.get("pageToken"), "mock-scan-workflow-page")
    const filter = url.searchParams.get("filter")?.trim().toLowerCase() || ""
    const workflows = getMockScanWorkflows().filter((workflow) => (
      !filter || [workflow.name, workflow.displayName, workflow.description].some((value) => value.toLowerCase().includes(filter))
    ))
    const start = (page - 1) * pageSize
    const nextPage = start + pageSize < workflows.length ? page + 1 : undefined
    return json({
      results: workflows.slice(start, start + pageSize),
      totalSize: workflows.length,
      ...(nextPage ? { nextPageToken: `mock-scan-workflow-page-${nextPage}` } : {}),
    })
  }
  if (method === "POST" && path === "/scanWorkflows") {
    const body = await request.json() as {
      scanWorkflow?: Pick<ScanWorkflow, "displayName" | "description" | "stages">
      scanWorkflowId?: string
    }
    const definition = body.scanWorkflow
    if (!definition?.displayName || !hasExplicitWorkflowStepDefaults(definition.stages)) {
      return json({ error: "A display name and non-empty workflow topology are required" }, { status: 400 })
    }
    const sequence = mockScanWorkflows.length
    const scanWorkflowId = body.scanWorkflowId || `wf-00000000-0000-4000-8000-${String(sequence).padStart(12, "0")}`
    const name = `scanWorkflows/${scanWorkflowId}`
    if (getMockScanWorkflowByName(name)) {
      return json({ error: "Scan workflow already exists" }, { status: 409 })
    }
    const now = new Date().toISOString()
    const stages = definition.stages as ScanWorkflowStageView[]
    const workflow: ScanWorkflow = {
      name,
      displayName: definition.displayName,
      description: definition.description || "",
      stages,
      steps: stages.flatMap((stage) => stage.steps),
      isBuiltin: false,
      isExecutable: true,
      etag: `mock-${scanWorkflowId}-v1`,
      createTime: now,
      updateTime: now,
    }
    mockScanWorkflows.push(workflow)
    return json(workflow, { status: 201 })
  }
  if (method === "GET" && path === "/engines") {
    return json(getMockEngineCatalog())
  }
  if (method === "POST" && path === "/engines:install") {
    const result = installMockEngine(await request.json())
    if ("engine" in result) return json(result.engine)
    if (result.conflict) {
      return json({
        error: {
          code: "ALREADY_EXISTS",
          message: result.error,
          details: [
            { field: "engineId", message: result.conflict.engineId },
            { field: "currentPackageDigest", message: result.conflict.currentPackageDigest },
            { field: "proposedPackageDigest", message: result.conflict.proposedPackageDigest },
          ],
        },
      }, { status: 409 })
    }
    return json({ error: { code: "INVALID_ARGUMENT", message: result.error } }, { status: 400 })
  }
  {
    const engineMatch = path.match(/^\/engines\/([^/]+)$/)
    if (method === "GET" && engineMatch) {
      const engine = getMockEngineCatalogDetail(decodeURIComponent(engineMatch[1]))
      return engine ? json(engine) : json({ error: "Engine not found" }, { status: 404 })
    }
  }
  {
    const scanWorkflowProfileMatch = path.match(/^\/scanWorkflows\/([^/]+)\/profile$/)
    if (method === "GET" && scanWorkflowProfileMatch) {
      const profile = getMockScanWorkflowProfile(decodeURIComponent(scanWorkflowProfileMatch[1]))
      return profile ? json(profile) : json({ error: "Scan workflow profile not found" }, { status: 404 })
    }
    const scanWorkflowMatch = path.match(/^\/scanWorkflows\/([^/]+)$/)
    if (method === "GET" && scanWorkflowMatch) {
      const workflow = getMockScanWorkflowByName(`scanWorkflows/${decodeURIComponent(scanWorkflowMatch[1])}`)
      return workflow ? json(workflow) : json({ error: "Scan workflow not found" }, { status: 404 })
    }
    if (method === "PATCH" && scanWorkflowMatch) {
      const workflow = getMockScanWorkflowByName(`scanWorkflows/${decodeURIComponent(scanWorkflowMatch[1])}`)
      if (!workflow) return json({ error: "Scan workflow not found" }, { status: 404 })
      if (workflow.isBuiltin) return json({ error: "BUILTIN_SCAN_WORKFLOW_IMMUTABLE" }, { status: 400 })
      const body = await request.json() as { scanWorkflow?: Pick<ScanWorkflow, "displayName" | "description" | "stages" | "etag"> }
      const update = body.scanWorkflow
      if (!update || update.etag !== workflow.etag || !hasExplicitWorkflowStepDefaults(update.stages)) {
        return json({ error: "Workflow ETag or topology is invalid" }, { status: 409 })
      }
      workflow.displayName = update.displayName
      workflow.description = update.description
      workflow.stages = update.stages
      workflow.steps = update.stages.flatMap((stage) => stage.steps)
      workflow.etag = `${workflow.etag}-next`
      workflow.updateTime = new Date().toISOString()
      return json(workflow)
    }
  }

  if (method === "GET" && path === "/admin/agents/filterOptions") {
    const field = url.searchParams.get("field")
    if (field !== "status" && field !== "healthState") {
      return json({ error: "Unsupported agent filter option field" }, { status: 400 })
    }
    return json(getMockAgentFilterOptions(field))
  }

  if (method === "GET" && path === "/admin/agents") {
    return json(getMockAgents({
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
      pageToken: url.searchParams.get("pageToken") || undefined,
      filter: url.searchParams.get("filter") || undefined,
      orderBy: url.searchParams.get("orderBy") || undefined,
    }))
  }
  if (method === "GET" && path === "/admin/agentClusterSummaries/current") {
    return json(getMockAgentClusterSummary())
  }
  if (method === "GET" && path === "/admin/agentLocationMaps/current") {
    return json(getMockAgentLocationMap())
  }
  {
    const agentNodeMatch = path.match(/^\/admin\/agents\/(\d+)$/)
    if (method === "GET" && agentNodeMatch) {
      const node = getMockAgentById(Number.parseInt(agentNodeMatch[1], 10))
      return node ? json(node) : json({ error: "Distributed node not found" }, { status: 404 })
    }
    if (method === "DELETE" && agentNodeMatch) {
      const nodeId = Number.parseInt(agentNodeMatch[1], 10)
      return deleteMockAgent(nodeId)
        ? empty()
        : json({ error: "Distributed node not found" }, { status: 404 })
    }
  }
  if (method === "POST" && path === "/admin/agentRegistrationTokens") {
    return json(getMockRegistrationToken(), { status: 201 })
  }
  {
    const registrationTokenMatch = path.match(/^\/admin\/agentRegistrationTokens\/(\d+)$/)
    if (method === "GET" && registrationTokenMatch) {
      const token = getMockRegistrationTokenById(Number.parseInt(registrationTokenMatch[1], 10))
      return token ? json(token) : json({ error: "Registration token not found" }, { status: 404 })
    }
  }
  {
    const agentPatchMatch = path.match(/^\/admin\/agents\/(\d+)$/)
    if (method === "PATCH" && agentPatchMatch) {
      const nodeId = Number.parseInt(agentPatchMatch[1], 10)
      const node = getMockAgentById(nodeId)
      return node ? json(node) : json({ error: "Distributed node not found" }, { status: 404 })
    }
  }
  {
    const agentLogsMatch = path.match(/^\/admin\/agents\/(\d+)\/logEntries$/)
    if (method === "GET" && agentLogsMatch) {
      const ts = new Date().toISOString()
      const tsNs = `${Date.now()}000000`
      return json({
        results: [
          {
            id: `mock:${agentLogsMatch[1]}:${tsNs}`,
            ts,
            tsNs,
            stream: "stdout",
            line: `[mock:${getMockScenario()}] log polling is enabled in mock mode`,
            truncated: false,
          },
        ],
        nextPageToken: `mock-cursor:${agentLogsMatch[1]}:${tsNs}`,
        previousPageToken: "",
        hasOlder: false,
        hasNewer: false,
        caughtUp: true,
        gap: false,
        gapReason: "",
      })
    }
  }

  if (method === "GET" && path === "/assets:search") {
    const assetType = url.searchParams.get("assetType")
    const rawPageSize = url.searchParams.get("pageSize")
    const pageSize = rawPageSize === null ? undefined : Number(rawPageSize)
    if ((assetType !== "website" && assetType !== "endpoint") || (pageSize !== undefined && (!Number.isInteger(pageSize) || pageSize < 1 || pageSize > 100))) {
      return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "Invalid global asset search request" } }, { status: 400 })
    }
    try {
      return json(getMockSearchResults({
        q: url.searchParams.get("q") ?? "",
        assetType,
        pageSize,
        pageToken: url.searchParams.get("pageToken") ?? undefined,
      }))
    } catch {
      return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "Invalid global asset search query" } }, { status: 400 })
    }
  }

  if (method === "GET" && path === "/tools") {
    return json(buildTools({
      page: parseNumeric(url.searchParams.get("page")) || 1,
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
    }))
  }
  if (method === "POST" && path === "/tools/create") {
    return json({ tool: createMockTool(await request.json()) })
  }
  {
    const toolMatch = path.match(/^\/tools\/(\d+)$/)
    if (toolMatch) {
      const toolId = Number.parseInt(toolMatch[1], 10)
      if (method === "PUT") {
        return json({ tool: updateMockTool(toolId, await request.json()) })
      }
      if (method === "DELETE") {
        deleteMockTool(toolId)
        return empty()
      }
      if (method === "GET") {
        const tool = getMockToolById(toolId)
        return tool ? json({ tool }) : json({ error: "Tool not found" }, { status: 404 })
      }
    }
  }

  if (method === "GET" && path === "/commands") {
    return json(getMockCommands({
      page: parseNumeric(url.searchParams.get("page")) || 1,
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
      toolId: parseNumeric(url.searchParams.get("toolId")),
    }))
  }
  if (method === "POST" && path === "/commands/create") {
    return json(createMockCommand(await request.json()))
  }
  if (method === "POST" && path === "/commands/batch-delete") {
    const body = (await request.json()) as { ids?: number[] }
    return json(batchDeleteMockCommands(body.ids || []))
  }
  {
    const commandMatch = path.match(/^\/commands\/(\d+)$/)
    if (commandMatch) {
      const commandId = Number.parseInt(commandMatch[1], 10)
      if (method === "GET") {
        const command = getMockCommandById(commandId)
        return command ? json({ command }) : json({ error: "Command not found" }, { status: 404 })
      }
      if (method === "PUT") {
        return json(updateMockCommand(commandId, await request.json()))
      }
      if (method === "DELETE") {
        deleteMockCommand(commandId)
        return empty()
      }
    }
  }

  if (method === "GET" && path === "/wordlists") {
    return json(getMockWordlists({
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || 10,
      pageToken: url.searchParams.get("pageToken"),
      filter: url.searchParams.get("filter"),
      orderBy: url.searchParams.get("orderBy"),
    }))
  }
  if (method === "POST" && path === "/wordlists") {
    const formData = await request.formData()
    const file = formData.get("file")
    if (!(file instanceof File)) {
      return json({ error: "File is required" }, { status: 400 })
    }
    const description = formData.get("description")
    const tags = formData.get("tags")
    return json(createMockWordlist({
      file,
      description: typeof description === "string" ? description : undefined,
      tags: typeof tags === "string" && tags ? tags.split(",") : undefined,
    }), { status: 201 })
  }
  if (method === "GET" && path === "/wordlistTags") {
    return json(getMockWordlistTags({
      pageSize: parseNumeric(url.searchParams.get("pageSize")) || 20,
      pageToken: url.searchParams.get("pageToken"),
      filter: url.searchParams.get("filter"),
    }))
  }
  {
    const wordlistMatch = path.match(/^\/wordlists\/(\d+)$/)
    if (method === "PATCH" && wordlistMatch) {
      const wordlistId = Number.parseInt(wordlistMatch[1], 10)
      const body = (await request.json()) as { updateMask?: string; description?: string; tags?: string[] }
      if (body.updateMask?.split(",").map((field) => field.trim()).includes("displayName")) {
        return json({ error: "unsupported updateMask field" }, { status: 400 })
      }
      const wordlist = updateMockWordlistMetadata(wordlistId, body)
      return wordlist ? json(wordlist) : json({ error: "Wordlist not found" }, { status: 404 })
    }

    const wordlistContentMatch = path.match(/^\/wordlists\/(\d+)\/text$/)
    if (method === "GET" && wordlistContentMatch) {
      const wordlistId = Number.parseInt(wordlistContentMatch[1], 10)
      if (!isMockWordlistEditable(wordlistId)) {
        return json({ error: "File too large for online editing (max 5MB), please download and edit locally" }, { status: 400 })
      }
      return json({ content: getMockWordlistContent() })
    }
    if (method === "PATCH" && wordlistContentMatch) {
      const wordlistId = Number.parseInt(wordlistContentMatch[1], 10)
      if (!isMockWordlistEditable(wordlistId)) {
        return json({ error: "File too large for online editing (max 5MB), please re-upload the file" }, { status: 400 })
      }
      return json({ name: `wordlists/${wordlistId}/text`, content: getMockWordlistContent(), updateTime: new Date().toISOString() })
    }
  }

  if (method === "GET" && path === "/nucleiPocSources/current") {
    const source = getMockNucleiPocSource(getMockScenario() === "empty")
    return source
      ? json(source)
      : json({ error: { code: 404, status: "NOT_FOUND", message: "Current Nuclei POC source not found." } }, { status: 404 })
  }
  if (method === "POST" && path === "/nucleiPocSources:sync") {
    const body = await request.json() as Record<string, unknown>
    if (!hasOnlyKeys(body, ["requestId", "sourceType", "repoUrl"]) || typeof body.requestId !== "string" || typeof body.sourceType !== "string" || typeof body.repoUrl !== "string") {
      return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "requestId, sourceType and repoUrl are required." } }, { status: 400 })
    }
    try {
      const result = createMockNucleiPocSync({
        requestId: body.requestId,
        sourceType: body.sourceType as never,
        repoUrl: body.repoUrl,
      }, getMockScenario() === "error")
      if (result.conflictTaskName) {
        return json({ error: { code: 409, status: "ABORTED", message: "Another Nuclei POC sync is already running.", details: [{ "@type": "type.googleapis.com/google.rpc.ErrorInfo", reason: "SYNC_ALREADY_RUNNING", domain: "lunafox", metadata: { task: result.conflictTaskName } }] } }, { status: 409 })
      }
      return json(result.task, { status: 201 })
    } catch (error) {
      const typed = error as { status?: number; code?: string; message?: string }
      return json({ error: { code: typed.status ?? 400, status: typed.code ?? "INVALID_ARGUMENT", message: typed.message ?? "Invalid Nuclei POC sync request." } }, { status: typed.status ?? 400 })
    }
  }
  {
    const nucleiPocTaskMatch = path.match(/^\/nucleiPocSyncTasks\/([^/]+)$/)
    if (method === "GET" && nucleiPocTaskMatch) {
      const task = getMockNucleiPocSyncTask(`nucleiPocSyncTasks/${nucleiPocTaskMatch[1]}`)
      return task
        ? json(task)
        : json({ error: { code: 404, status: "NOT_FOUND", message: "Nuclei POC sync task not found." } }, { status: 404 })
    }
  }
  if (method === "GET" && path === "/nucleiPocs/filterOptions") {
    const fields = url.searchParams.getAll("field")
    if (![...url.searchParams.keys()].every((key) => key === "field") || fields.length !== 1 || fields[0] !== "tags") {
      return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "field must be tags." } }, { status: 400 })
    }
    if (getMockScenario() === "empty") return json({ results: [] })
    try {
      return json(getMockNucleiPocFilterOptions("tags"))
    } catch (error) {
      return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: error instanceof Error ? error.message : "Invalid Nuclei POC filter-options query." } }, { status: 400 })
    }
  }
  if (method === "GET" && path === "/nucleiPocs") {
    if (getMockScenario() === "empty") return json({ results: [], totalSize: 0 })
    if (![...url.searchParams.keys()].every((key) => ["pageSize", "pageToken", "filter", "orderBy"].includes(key))) {
      return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "Unsupported Nuclei POC query field." } }, { status: 400 })
    }
    try {
      return json(getMockNucleiPocs({
        pageSize: parseNumeric(url.searchParams.get("pageSize")),
        pageToken: url.searchParams.get("pageToken") || undefined,
        filter: url.searchParams.get("filter") || undefined,
        orderBy: url.searchParams.get("orderBy") || undefined,
      }))
    } catch (error) {
      return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: error instanceof Error ? error.message : "Invalid Nuclei POC query." } }, { status: 400 })
    }
  }
  if (method === "POST" && path === "/nucleiPocs:setActivation") {
    const body = await request.json() as Record<string, unknown>
    if (!hasOnlyKeys(body, ["enabled", "names"]) || typeof body.enabled !== "boolean") {
      return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "enabled is required and must be boolean." } }, { status: 400 })
    }
    try {
      return json(setMockNucleiPocActivation(body.enabled, getMockScenario() === "error", getMockScenario() === "empty", body.names))
    } catch (error) {
      const typed = error as { status?: number; code?: string; message?: string; taskName?: string }
      if (typed.code === "SYNC_ALREADY_RUNNING" && typed.taskName) {
        return json({
          error: {
            code: 409,
            status: "ABORTED",
            message: typed.message ?? "Another Nuclei POC sync is already running.",
            details: [{
              "@type": "type.googleapis.com/google.rpc.ErrorInfo",
              reason: "SYNC_ALREADY_RUNNING",
              domain: "lunafox",
              metadata: { task: typed.taskName },
            }],
          },
        }, { status: 409 })
      }
      return json({ error: { code: typed.status ?? 500, status: typed.code ?? "INTERNAL", message: typed.message ?? "Nuclei POC activation failed." } }, { status: typed.status ?? 500 })
    }
  }
  {
    const nucleiPocMatch = path.match(/^\/nucleiPocs\/([^/]+)$/)
    if (nucleiPocMatch) {
      const resourceName = `nucleiPocs/${decodeURIComponent(nucleiPocMatch[1])}`
      if (method === "GET") {
        const detail = getMockNucleiPoc(resourceName, getMockScenario() === "edge" && resourceName.endsWith("http-missing-security-headers"))
        return detail
          ? json(detail)
          : json({ error: { code: 404, status: "NOT_FOUND", message: "Nuclei POC not found." } }, { status: 404 })
      }
      if (method === "PATCH") {
        const body = await request.json() as Record<string, unknown>
        if (!hasOnlyKeys(body, ["name", "isEnabled", "updateMask"]) || body.name !== resourceName || !Array.isArray(body.updateMask) || body.updateMask.length !== 1 || body.updateMask[0] !== "isEnabled" || typeof body.isEnabled !== "boolean") {
          return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "updateMask must contain only isEnabled." } }, { status: 400 })
        }
        const updated = updateMockNucleiPocEnabled(resourceName, body.isEnabled)
        return updated
          ? json({
              name: updated.name,
              templateId: updated.templateId,
              displayName: updated.displayName,
              severity: updated.severity,
              tags: updated.tags,
              author: updated.author,
              description: updated.description,
              cve: updated.cve,
              cwe: updated.cwe,
              references: updated.references,
              remediation: updated.remediation,
              relativePath: updated.relativePath,
              contentSha256: updated.contentSha256,
              isEnabled: updated.isEnabled,
              createdAt: updated.createdAt,
              updatedAt: updated.updatedAt,
            })
          : json({ error: { code: 404, status: "NOT_FOUND", message: "Nuclei POC not found." } }, { status: 404 })
      }
    }
  }

  if (method === "POST" && path === "/screenshots:batchDelete") {
    const body = (await request.json()) as { names?: string[] }
    const ids = parseNestedIdsFromResourceNames(body.names, "targets", "screenshots")
    return json(bulkDeleteMockScreenshots(ids))
  }
  {
    const screenshotBlobMatch = path.match(/^\/screenshots\/(\d+)\/blob$/)
    if (method === "GET" && screenshotBlobMatch) {
      return text(getMockScreenshotImageSvg(Number.parseInt(screenshotBlobMatch[1], 10)), "image/svg+xml")
    }
  }
  {
    const scanScreenshotBlobMatch = path.match(/^\/scans\/(\d+)\/screenshotSnapshots\/(\d+)\/blob$/)
    if (method === "GET" && scanScreenshotBlobMatch) {
      return text(
        getMockScreenshotImageSvg(
          Number.parseInt(scanScreenshotBlobMatch[2], 10),
          Number.parseInt(scanScreenshotBlobMatch[1], 10)
        ),
        "image/svg+xml"
      )
    }
  }

  if (method === "GET" && path === "/fingerprintLibraryStatistics") {
    return json(getMockFingerprintStats())
  }

  if (path.startsWith("/fingerprintLibraries/")) {
    const source = getFingerprintSource(path)
    if (!source) {
      return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "Unsupported fingerprint library." } }, { status: 400 })
    }

    const collectionPath = `/fingerprintLibraries/${source}/fingerprints`
    const libraryPath = `/fingerprintLibraries/${source}`

    if (method === "GET" && path === `${libraryPath}/exportFiles/current`) {
		return text(exportMockFingerprints(source), "application/json; charset=utf-8", {
			headers: { "Content-Disposition": "attachment; filename=\"fingerprinthub_web.json\"" },
      })
    }

    if (method === "GET" && path === collectionPath) {
      return json(buildFingerprintList(source, {
        pageSize: parseNumeric(url.searchParams.get("pageSize")) || 20,
        pageToken: url.searchParams.get("pageToken") || undefined,
        filter: url.searchParams.get("filter") || undefined,
        orderBy: url.searchParams.get("orderBy") || undefined,
      }))
    }

    if (method === "GET" && path === `${collectionPath}/filterOptions`) {
      const field = url.searchParams.get("field")
      const options = field
        ? getMockFingerprintFilterOptions(source, field as never)
        : undefined
      if (!options) {
        return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "Unsupported fingerprint filter field." } }, { status: 400 })
      }
      return json(options)
    }

    if (method === "GET" && path.startsWith(`${collectionPath}/`)) {
      const canonicalName = path.slice(1)
      if (!isCanonicalFingerprintName(source, canonicalName)) {
        return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "Invalid fingerprint resource name." } }, { status: 400 })
      }
      const item = getMockFingerprintByName(source, canonicalName)
      return item ? json(item) : json({ error: { code: 404, status: "NOT_FOUND", message: "Fingerprint not found." } }, { status: 404 })
    }

    if (method === "POST" && path === `${libraryPath}:import`) {
      const contentType = request.headers.get("content-type") ?? ""
      if (!contentType.toLowerCase().startsWith("multipart/form-data")) {
        return fingerprintImportError(source, 415, "TRANSPORT", "FINGERPRINT_IMPORT_UNSUPPORTED_MEDIA_TYPE", "UNSUPPORTED_MEDIA_TYPE")
      }

      try {
        // FormData parsers may replace malformed bytes, so validate the raw
        // multipart copy before extracting the file part.
        new TextDecoder("utf-8", { fatal: true }).decode(await request.clone().arrayBuffer())
      } catch {
        return fingerprintImportError(source, 400, "ENCODING", "FINGERPRINT_IMPORT_INVALID", "INVALID_UTF8")
      }

      let formData: FormData
      try {
        formData = await request.formData()
      } catch {
        return fingerprintImportError(source, 400, "TRANSPORT", "FINGERPRINT_IMPORT_INVALID", "MULTIPART_INVALID")
      }
      const entries = Array.from(formData.entries())
      const file = formData.get("file")
      if (entries.length !== 1 || !isFingerprintUploadFile(file)) {
        return fingerprintImportError(source, 400, "TRANSPORT", "FINGERPRINT_IMPORT_INVALID", "FILE_REQUIRED")
      }
      if (file.size === 0) {
        return fingerprintImportError(source, 400, "TRANSPORT", "FINGERPRINT_IMPORT_INVALID", "FILE_REQUIRED")
      }
      if (file.size > 30 * 1024 * 1024) {
        return fingerprintImportError(source, 413, "TRANSPORT", "FINGERPRINT_IMPORT_FILE_TOO_LARGE", "FILE_TOO_LARGE")
      }

		const allowedTypes = new Set(["", "application/json", "application/octet-stream"])
      const fileMediaType = file.type.split(";", 1)[0]?.trim().toLowerCase() ?? ""
      if (!allowedTypes.has(fileMediaType)) {
        return fingerprintImportError(source, 415, "TRANSPORT", "FINGERPRINT_IMPORT_UNSUPPORTED_MEDIA_TYPE", "UNSUPPORTED_MEDIA_TYPE")
      }
      let fileContent: string
      try {
        fileContent = new TextDecoder("utf-8", { fatal: true }).decode(await file.arrayBuffer())
      } catch {
        return fingerprintImportError(source, 400, "ENCODING", "FINGERPRINT_IMPORT_INVALID", "INVALID_UTF8")
      }
      const parsed = parseMockFingerprintImport(source, fileContent)
      if ("diagnostic" in parsed) {
        return fingerprintImportError(
          source,
          400,
          parsed.diagnostic.kind,
          "FINGERPRINT_IMPORT_INVALID",
          parsed.diagnostic.reason,
          {
            ...(parsed.diagnostic.recordIndex !== undefined ? { recordIndex: parsed.diagnostic.recordIndex } : {}),
            ...(parsed.diagnostic.fieldPath ? { fieldPath: parsed.diagnostic.fieldPath } : {}),
            ...(parsed.diagnostic.line !== undefined ? { line: parsed.diagnostic.line } : {}),
            ...(parsed.diagnostic.column !== undefined ? { column: parsed.diagnostic.column } : {}),
          }
        )
      }

      return json(importMockFingerprints(source, parsed.records))
    }

    if (method === "POST" && path === `${collectionPath}:batchDelete`) {
      const body = (await request.json()) as { names?: unknown }
      const names = Array.isArray(body.names) && body.names.every((name): name is string => typeof name === "string")
        ? body.names
        : []
      // Canonical resource names are API identifiers, not URL paths, so they
      // intentionally omit the request path's leading slash.
      const expectedPrefix = `${collectionPath.slice(1)}/`
      if (names.length === 0 || names.length > 1000 || new Set(names).size !== names.length || names.some((name) => !name.startsWith(expectedPrefix) || !isCanonicalFingerprintName(source, name))) {
        return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "Invalid fingerprint names." } }, { status: 400 })
      }
      return json(batchDeleteMockFingerprints(source, names))
    }

    if (method === "POST" && path === `${libraryPath}:clear`) {
      if (!await hasEmptyClearRequestBody(request)) {
        return json({ error: { code: 400, status: "INVALID_ARGUMENT", message: "Fingerprint clear request must not contain fields." } }, { status: 400 })
      }
      return json(clearMockFingerprints(source))
    }
  }

  return json(
    {
      error: {
        code: "mock_route_not_implemented",
        message: `No mock handler implemented for ${method} ${path}`,
      },
    },
    { status: 501 }
  )
}

function hasExplicitWorkflowStepDefaults(value: unknown): value is ScanWorkflowStageView[] {
  return Array.isArray(value) && value.length > 0 && value.every((stage) => (
    typeof stage === "object" && stage !== null && !Array.isArray(stage) &&
    typeof (stage as { stageId?: unknown }).stageId === "string" &&
    Array.isArray((stage as { steps?: unknown }).steps) &&
    (stage as { steps: unknown[] }).steps.length > 0 &&
    (stage as { steps: unknown[] }).steps.every((step) => (
      typeof step === "object" && step !== null && !Array.isArray(step) &&
      typeof (step as { stepId?: unknown }).stepId === "string" &&
      typeof (step as { engineId?: unknown }).engineId === "string" &&
      typeof (step as { profileDefaultEnabled?: unknown }).profileDefaultEnabled === "boolean"
    ))
  ))
}

export const mockHandlers = [
  http.all(/\/v1\/.*/, async ({ request }) => {
    try {
      return await resolveMockApi(request)
    } catch (error) {
      if (error instanceof TypeError) {
        return json({ error: { code: "INVALID_ARGUMENT", message: error.message } }, { status: 400 })
      }
      throw error
    }
  }),
]
