import type { Endpoint, EndpointFilterOptionField, GetEndpointsRequest, GetEndpointsResponse } from "@/types/endpoint.types"
import { extractWebsiteURLScopeFilter, matchesWebsiteURLScope, removeLeadingWebsiteScopeFilter } from "@/lib/website-scope"
import { mockEndpoints } from "../data/endpoints"
import { getMockScanById } from "../data/scans"
import { getMockTargetById } from "../data/targets"
import { getMockScenario } from "../scenarios"

type EndpointSource = "all" | "target" | "scan"

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function paginateEndpoints(items: Endpoint[], page = 1, pageSize = 10): GetEndpointsResponse {
  const total = items.length
  const totalPages = total === 0 ? 0 : Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const nextPage = page < totalPages ? page + 1 : undefined

  return {
    results: items.slice(start, start + pageSize),
    total,
    page,
    pageSize,
    totalPages,
    totalSize: total,
    ...(nextPage && { nextPageToken: encodeMockEndpointPageToken(nextPage) }),
  }
}

function encodeMockEndpointPageToken(page: number) {
  return `mock-endpoint-page-${page}`
}

function decodeMockEndpointPageToken(pageToken?: string) {
  const match = pageToken?.match(/^mock-endpoint-page-(\d+)$/)
  return match ? Number.parseInt(match[1], 10) : 1
}

function buildStressEndpoints(): Endpoint[] {
  return [
    {
      id: 9701,
      url: "https://acme.com/platform/customer-facing-shared-security-center/tenant/onboarding/step/enterprise-acquisition-regional-control-plane?region=apac&feature=post-authentication-risk-evaluation",
      method: "GET",
      statusCode: 200,
      title: "Acme Enterprise Shared Security Center - Tenant Onboarding Control Plane",
      contentLength: 245_678,
      contentType: "text/html; charset=utf-8",
      host: "acme.com",
      webserver: "nginx/1.25.4",
      tech: ["React", "Next.js", "TypeScript", "Node.js", "Tailwind CSS"],
      responseBody: "Large HTML document placeholder for layout stress testing.",
      responseHeaders: "server: nginx\ncache-control: private\ncontent-security-policy: default-src 'self'",
      createdAt: "2026-05-27T08:30:00Z",
    },
    {
      id: 9702,
      url: "https://api.acme.com/v2/internal/audit/events/export?window=quarterly&include=relationships,owners,controls,exceptions",
      method: "POST",
      statusCode: 401,
      title: "",
      contentLength: 4_096,
      contentType: "application/json",
      host: "api.acme.com",
      webserver: "envoy",
      tech: ["Go", "gRPC", "PostgreSQL", "Redis"],
      responseBody: "{\"error\":\"unauthorized\"}",
      createdAt: "2026-05-27T08:31:00Z",
    },
    {
      id: 9703,
      url: "https://acme.com/.well-known/change-password",
      method: "GET",
      statusCode: 302,
      title: "",
      contentLength: 0,
      contentType: "text/html; charset=utf-8",
      location: "https://auth.acme.com/account/password",
      host: "acme.com",
      webserver: "nginx/1.25.4",
      tech: ["React", "Next.js"],
      createdAt: "2026-05-27T08:32:00Z",
    },
  ]
}

function buildEdgeEndpoints(): Endpoint[] {
  return [
    {
      id: 9704,
      url: "https://edge.acme.com/callback?return=%2Fdashboard%2F",
      method: "GET",
      statusCode: null,
      title: "",
      contentLength: null,
      contentType: null,
      host: "edge.acme.com",
      location: "https://acme.com/dashboard",
      webserver: "",
      tech: [],
      vhost: null,
      responseHeaders: "",
      responseBody: "",
      createdAt: "2026-05-27T09:00:00Z",
    },
    {
      id: 9705,
      url: "https://api.acme.com/v1/legacy-export",
      method: "GET",
      statusCode: 403,
      title: "",
      contentLength: 0,
      contentType: null,
      host: "api.acme.com",
      webserver: "envoy",
      tech: ["Go"],
      responseBody: "",
      createdAt: "2026-05-27T09:01:00Z",
    },
  ]
}

function getScenarioEndpoints(): Endpoint[] {
  const scenario = getMockScenario()

  if (scenario === "empty") {
    return []
  }

  const base = clone(mockEndpoints)

  if (scenario === "stress") {
    return [...buildStressEndpoints(), ...base]
  }

  if (scenario === "edge") {
    return [...buildEdgeEndpoints(), ...base]
  }

  return base
}

function getSourcePattern(source: EndpointSource, sourceId?: number) {
  if (source === "target") {
    return (getMockTargetById(sourceId ?? 0)?.name || "").toLowerCase()
  }

  if (source === "scan") {
    return (getMockScanById(sourceId ?? 0)?.target?.name || "").toLowerCase()
  }

  return ""
}

function filterEndpoints(
  items: Endpoint[],
  source: EndpointSource,
  filter?: string,
  sourceId?: number
) {
  const pattern = getSourcePattern(source, sourceId)

  let filtered = items

  if (pattern) {
    filtered = filtered.filter((endpoint) => {
      const url = endpoint.url.toLowerCase()
      const host = (endpoint.host || "").toLowerCase()
      return url.includes(pattern) || host.includes(pattern)
    })
  } else if (source !== "all") {
    filtered = []
  }

  return applyEndpointFilter(filtered, filter)
}

function parseFilterValues(filter: string | undefined, field: string) {
  const values: string[] = []
  if (!filter) return values
  const pattern = new RegExp(`${field}(?:==|=)"((?:\\\\.|[^"\\\\])*)"`, "g")
  let match: RegExpExecArray | null
  while ((match = pattern.exec(filter)) !== null) {
    values.push(match[1].replace(/\\"/g, '"').replace(/\\\\/g, "\\"))
  }
  return values
}

function applyEndpointFilter(items: Endpoint[], filter?: string) {
  let filtered = items
  const websiteScope = extractWebsiteURLScopeFilter(filter)
  const userFilter = removeLeadingWebsiteScopeFilter(filter, "websiteUrl")
  if (websiteScope) {
    filtered = filtered.filter((endpoint) => matchesWebsiteURLScope(websiteScope, endpoint.url))
  }

  const urlValues = parseFilterValues(userFilter, "url")
  for (const value of urlValues) {
    filtered = filtered.filter((endpoint) => endpoint.url === value)
  }

  const statusValues = parseFilterValues(userFilter, "statusCode")
  if (statusValues.length > 0) {
    filtered = filtered.filter((endpoint) => endpoint.statusCode !== null && statusValues.includes(String(endpoint.statusCode)))
  }

  const webserverValues = parseFilterValues(userFilter, "webserver")
  if (webserverValues.length > 0) {
    filtered = filtered.filter((endpoint) => webserverValues.includes(endpoint.webserver || ""))
  }

  const contentTypeValues = parseFilterValues(userFilter, "contentType")
  if (contentTypeValues.length > 0) {
    filtered = filtered.filter((endpoint) => contentTypeValues.includes(endpoint.contentType || ""))
  }

  const techValues = parseFilterValues(userFilter, "tech")
  if (techValues.length > 0) {
    filtered = filtered.filter((endpoint) => (endpoint.tech ?? []).some((value) => techValues.includes(value)))
  }

  const vhostValues = parseFilterValues(userFilter, "vhost").filter((value) => value === "true" || value === "false")
  if (vhostValues.length === 1) {
    filtered = filtered.filter((endpoint) => String(endpoint.vhost) === vhostValues[0])
  }

  return filtered
}

function sortEndpoints(items: Endpoint[], orderBy?: string) {
  const [field, direction = "asc"] = (orderBy || "createdAt desc").split(/\s+/)
  const multiplier = direction === "desc" ? -1 : 1
  const sorted = [...items]

  sorted.sort((left, right) => {
    const result = compareEndpointField(left, right, field)
    if (result !== 0) return result * multiplier
    return (left.id - right.id) * multiplier
  })

  return sorted
}

function compareEndpointField(left: Endpoint, right: Endpoint, field?: string) {
  if (field === "statusCode") return compareNullableNumber(left.statusCode, right.statusCode)
  if (field === "contentLength") return compareNullableNumber(left.contentLength, right.contentLength)
  return new Date(left.createdAt ?? "").getTime() - new Date(right.createdAt ?? "").getTime()
}

function compareNullableNumber(left: number | null | undefined, right: number | null | undefined) {
  if (left == null && right == null) return 0
  if (left == null) return 1
  if (right == null) return -1
  return left - right
}

export function buildEndpoints(params?: GetEndpointsRequest): GetEndpointsResponse {
  const page = decodeMockEndpointPageToken(params?.pageToken)
  const pageSize = params?.pageSize || 10
  const filtered = sortEndpoints(filterEndpoints(getScenarioEndpoints(), "all", params?.filter), params?.orderBy)

  return paginateEndpoints(filtered, page, pageSize)
}

export function buildTargetEndpoints(
  targetId: number,
  params?: GetEndpointsRequest & { filter?: string }
): GetEndpointsResponse {
  const page = decodeMockEndpointPageToken(params?.pageToken)
  const pageSize = params?.pageSize || 10
  const filtered = sortEndpoints(filterEndpoints(
    getScenarioEndpoints(),
    "target",
    params?.filter,
    targetId
  ), params?.orderBy)

  return paginateEndpoints(filtered, page, pageSize)
}

export function buildEndpointResults(params: {
  source: "target" | "scan"
  sourceId: number
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}): GetEndpointsResponse {
  const page = decodeMockEndpointPageToken(params.pageToken)
  const pageSize = params.pageSize || 10
  const filtered = sortEndpoints(filterEndpoints(
    getScenarioEndpoints(),
    params.source,
    params.filter,
    params.sourceId
  ), params.orderBy)

  return paginateEndpoints(filtered, page, pageSize)
}

export function buildEndpointFilterOptions(params: {
  source: "target" | "scan"
  sourceId: number
  field: EndpointFilterOptionField
}) {
  const items = filterEndpoints(getScenarioEndpoints(), params.source, undefined, params.sourceId)
  const counts = new Map<string, number>()

  for (const endpoint of items) {
    for (const value of getEndpointOptionValues(endpoint, params.field)) {
      if (!value) continue
      counts.set(value, (counts.get(value) ?? 0) + 1)
    }
  }

  return {
    results: Array.from(counts.entries())
      .map(([value, count]) => ({ value, label: value, count }))
      .sort((left, right) => left.label.localeCompare(right.label, undefined, { numeric: true })),
  }
}

function getEndpointOptionValues(endpoint: Endpoint, field: EndpointFilterOptionField) {
  if (field === "statusCode") return endpoint.statusCode == null ? [] : [String(endpoint.statusCode)]
  if (field === "tech") return endpoint.tech ?? []
  if (field === "webserver") return endpoint.webserver ? [endpoint.webserver] : []
  if (field === "contentType") return endpoint.contentType ? [endpoint.contentType] : []
  if (field === "vhost") return endpoint.vhost == null ? [] : [String(endpoint.vhost)]
  return []
}
