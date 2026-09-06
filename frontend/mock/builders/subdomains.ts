import type { GetAllSubdomainsResponse, GetSubdomainsResponse, Subdomain } from "@/types/subdomain.types"
import type { PaginatedResponse } from "@/types/api-response.types"
import { mockOrganizations } from "../data/organizations"
import { getMockScanById } from "../data/scans"
import { mockSubdomains } from "../data/subdomains"
import { getMockTargetById } from "../data/targets"
import { getMockScenario } from "../scenarios"

type SubdomainSource = "all" | "organization" | "target" | "scan"

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function isDomainLike(value: string) {
  return value.includes(".") && !/^\d{1,3}(?:\.\d{1,3}){3}(?:\/\d+)?$/.test(value) && !value.includes("/")
}

function matchesDomainPattern(name: string, pattern: string) {
  return name === pattern || name.endsWith(`.${pattern}`)
}

function paginateDomains(items: Subdomain[], page = 1, pageSize = 10): GetAllSubdomainsResponse {
  const total = items.length
  const totalPages = total === 0 ? 0 : Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize

  return {
    domains: items.slice(start, start + pageSize),
    total,
    page,
    pageSize,
    totalPages,
  }
}

function paginateResults(items: Subdomain[], page = 1, pageSize = 10): PaginatedResponse<Subdomain> {
  const total = items.length
  const totalPages = total === 0 ? 0 : Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize

  return {
    results: items.slice(start, start + pageSize),
    total,
    page,
    pageSize,
    totalPages,
  }
}

function buildStressSubdomains(): Subdomain[] {
  return [
    {
      id: 9801,
      name: "customer-facing-shared-security-platform-api.acme.com",
      dnsName: "customer-facing-shared-security-platform-api.acme.com",
      createdAt: "2026-05-27T08:00:00Z",
    },
    {
      id: 9802,
      name: "north-america-enterprise-acquisition-integration.acme.io",
      dnsName: "north-america-enterprise-acquisition-integration.acme.io",
      createdAt: "2026-05-27T08:01:00Z",
    },
    {
      id: 9803,
      name: "global-shared-platform-auth-gateway.techstart.io",
      dnsName: "global-shared-platform-auth-gateway.techstart.io",
      createdAt: "2026-05-27T08:02:00Z",
    },
    {
      id: 9804,
      name: "partner-extranet-customer-support-portal.globalfinance.com",
      dnsName: "partner-extranet-customer-support-portal.globalfinance.com",
      createdAt: "2026-05-27T08:03:00Z",
    },
    {
      id: 9805,
      name: "video-streaming-edge-control-plane.mediastream.tv",
      dnsName: "video-streaming-edge-control-plane.mediastream.tv",
      createdAt: "2026-05-27T08:04:00Z",
    },
  ]
}

function buildEdgeSubdomains(): Subdomain[] {
  return [
    {
      id: 9806,
      name: "_acme-challenge.acme.com",
      dnsName: "_acme-challenge.acme.com",
      createdAt: "2026-05-27T09:00:00Z",
    },
    {
      id: 9807,
      name: "xn--customer-portal-6f5.acme.com",
      dnsName: "xn--customer-portal-6f5.acme.com",
      createdAt: "2026-05-27T09:01:00Z",
    },
  ]
}

function getScenarioSubdomains(): Subdomain[] {
  const scenario = getMockScenario()

  if (scenario === "empty") {
    return []
  }

  const base = clone(mockSubdomains)

  if (scenario === "stress") {
    return [...buildStressSubdomains(), ...base]
  }

  if (scenario === "edge") {
    return [...buildEdgeSubdomains(), ...base]
  }

  return base
}

function getSourcePatterns(source: SubdomainSource, sourceId?: number): string[] {
  if (source === "organization") {
    const organization = mockOrganizations.find((item) => item.id === sourceId)
    return (organization?.targets || [])
      .map((target) => target.name.toLowerCase())
      .filter(isDomainLike)
  }

  if (source === "target") {
    const target = getMockTargetById(sourceId ?? 0)
    const name = (target?.name || "").toLowerCase()
    return isDomainLike(name) ? [name] : []
  }

  if (source === "scan") {
    const scan = getMockScanById(sourceId ?? 0)
    const name = (scan?.target?.name || "").toLowerCase()
    return isDomainLike(name) ? [name] : []
  }

  return []
}

function filterSubdomains(
  items: Subdomain[],
  source: SubdomainSource,
  sourceId?: number,
  searchOrFilter?: string
) {
  const keyword = extractDnsNameSearch(searchOrFilter).toLowerCase()
  const patterns = getSourcePatterns(source, sourceId)

  let filtered = items

  if (patterns.length > 0) {
    filtered = filtered.filter((item) => {
      const name = item.name.toLowerCase()
      return patterns.some((pattern) => matchesDomainPattern(name, pattern))
    })
  } else if (source !== "all") {
    filtered = []
  }

  if (keyword) {
    filtered = filtered.filter((item) => item.name.toLowerCase().includes(keyword))
  }

  return filtered
}

function extractDnsNameSearch(searchOrFilter?: string) {
  const value = (searchOrFilter || "").trim()
  const match = value.match(/^dnsName="([^"]*)"$/i)
  return match ? match[1] : value
}

function sortSubdomainResults(items: Subdomain[], orderBy?: string) {
  const normalized = (orderBy || "createdAt desc").trim()
  const sorted = [...items]
  switch (normalized) {
    case "dnsName":
    case "dnsName asc":
      return sorted.sort((a, b) => a.dnsName.localeCompare(b.dnsName) || a.id - b.id)
    case "dnsName desc":
      return sorted.sort((a, b) => b.dnsName.localeCompare(a.dnsName) || b.id - a.id)
    case "createdAt":
    case "createdAt asc":
      return sorted.sort((a, b) => a.createdAt.localeCompare(b.createdAt) || a.id - b.id)
    case "createdAt desc":
    default:
      return sorted.sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id - a.id)
  }
}

export function buildAllSubdomains(params?: {
  page?: number
  pageSize?: number
  search?: string
}): GetAllSubdomainsResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const filtered = filterSubdomains(getScenarioSubdomains(), "all", undefined, params?.search)

  return paginateDomains(filtered, page, pageSize)
}

export function buildOrganizationSubdomains(
  organizationId: number,
  params?: {
    page?: number
    pageSize?: number
  }
): GetSubdomainsResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const filtered = filterSubdomains(getScenarioSubdomains(), "organization", organizationId)

  return paginateDomains(filtered, page, pageSize)
}

export function buildSubdomainResults(params: {
  source: "target" | "scan"
  sourceId: number
  page?: number
  pageSize?: number
  filter?: string
  orderBy?: string
}): PaginatedResponse<Subdomain> {
  const page = params.page || 1
  const pageSize = params.pageSize || 10
  const filtered = filterSubdomains(
    getScenarioSubdomains(),
    params.source,
    params.sourceId,
    params.filter
  )

  return paginateResults(sortSubdomainResults(filtered, params.orderBy), page, pageSize)
}
