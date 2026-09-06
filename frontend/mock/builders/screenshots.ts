import type {
  Screenshot,
  ScreenshotFilterOption,
  ScreenshotFilterOptionField,
  ScreenshotListQueryParams,
  ScreenshotListResponse,
  ScreenshotSnapshot,
} from "@/types/screenshot.types"
import { mockScreenshots, mockScreenshotSnapshots } from "../data/screenshots"
import { getMockScenario } from "../scenarios"

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function encodeMockScreenshotPageToken(page: number) {
  return `mock-screenshot-page-${page}`
}

function decodeMockScreenshotPageToken(pageToken?: string) {
  const match = pageToken?.match(/^mock-screenshot-page-(\d+)$/)
  return match ? Number.parseInt(match[1], 10) : 1
}

function getQuotedFilterValues(filter: string | undefined, field: string) {
  if (!filter) return []
  const regexp = new RegExp(`${field}(?:==|=)"((?:\\\\.|[^"\\\\])*)"`, "g")
  return Array.from(filter.matchAll(regexp), (match) => (
    (match[1] ?? "").replace(/\\"/g, '"').replace(/\\\\/g, "\\")
  )).filter(Boolean)
}

function filterScreenshots<T extends { url: string; statusCode: number | null }>(items: T[], filter?: string) {
  const urlTerms = getQuotedFilterValues(filter, "url")
  const statusCodes = new Set(getQuotedFilterValues(filter, "statusCode"))

  return items.filter((item) => {
    const matchesURL = urlTerms.length === 0 || urlTerms.every((term) => item.url === term)
    const matchesStatus = statusCodes.size === 0 || (item.statusCode !== null && statusCodes.has(String(item.statusCode)))
    return matchesURL && matchesStatus
  })
}

function sortScreenshots<T extends { id: number; statusCode: number | null; createdAt: string }>(items: T[], orderBy?: string) {
  const normalized = (orderBy || "createdAt desc").trim()
  const [field, direction = "asc"] = normalized.split(/\s+/)
  const multiplier = direction === "desc" ? -1 : 1

  return [...items].sort((left, right) => {
    if (field === "statusCode") {
      const leftValue = left.statusCode
      const rightValue = right.statusCode
      if (leftValue === null && rightValue === null) return (left.id - right.id) * multiplier
      if (leftValue === null) return 1
      if (rightValue === null) return -1
      if (leftValue !== rightValue) return (leftValue - rightValue) * multiplier
      return (left.id - right.id) * multiplier
    }
    const leftTime = Date.parse(left.createdAt)
    const rightTime = Date.parse(right.createdAt)
    if (leftTime !== rightTime) return (leftTime - rightTime) * multiplier
    return (left.id - right.id) * multiplier
  })
}

function paginate<T extends { id: number; url: string; statusCode: number | null; createdAt: string }>(
  items: T[],
  params?: ScreenshotListQueryParams
): ScreenshotListResponse<T> {
  const page = decodeMockScreenshotPageToken(params?.pageToken)
  const pageSize = params?.pageSize || 12
  const filtered = sortScreenshots(filterScreenshots(items, params?.filter), params?.orderBy)
  const total = filtered.length
  const totalPages = total === 0 ? 0 : Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const nextPage = page < totalPages ? page + 1 : undefined

  return {
    results: filtered.slice(start, start + pageSize),
    total,
    page,
    pageSize,
    totalPages,
    totalSize: total,
    ...(nextPage && { nextPageToken: encodeMockScreenshotPageToken(nextPage) }),
  }
}

function buildStatusCodeOptions<T extends { statusCode: number | null }>(items: T[]): ScreenshotFilterOption[] {
  const counts = new Map<string, number>()
  for (const item of items) {
    if (item.statusCode === null) continue
    const value = String(item.statusCode)
    counts.set(value, (counts.get(value) ?? 0) + 1)
  }
  return Array.from(counts.entries())
    .sort(([left], [right]) => left.localeCompare(right, undefined, { numeric: true }))
    .map(([value, count]) => ({ value, label: value, count }))
}

function buildStressScreenshots(): Screenshot[] {
  return Array.from({ length: 8 }, (_, index) => ({
    id: 9601 + index,
    url: `https://app.example.test/workspaces/customer-facing-shared-control-plane/segments/${index + 1}/experience/regional-onboarding-flow?variant=enterprise-acquisition-${index + 1}`,
    statusCode: index % 3 === 0 ? 302 : 200,
    createdAt: `2026-05-27T0${index}:00:00Z`,
    updatedAt: `2026-05-27T0${index}:10:00Z`,
  }))
}

function buildEdgeScreenshots(): Screenshot[] {
  return [
    {
      id: 9611,
      url: "https://admin.example.test/login?return=%2Fbilling%2Fexports%2F",
      statusCode: null,
      createdAt: "2026-05-27T09:40:00Z",
      updatedAt: "2026-05-27T09:40:00Z",
    },
    {
      id: 9612,
      url: "https://edge.example.test/redirect",
      statusCode: 302,
      createdAt: "2026-05-27T09:41:00Z",
      updatedAt: "2026-05-27T09:41:00Z",
    },
  ]
}

function getScenarioTargetScreenshots(): Screenshot[] {
  const scenario = getMockScenario()

  if (scenario === "empty") {
    return []
  }

  const base = clone(mockScreenshots).map(({ targetId: _, ...screenshot }) => screenshot)

  if (scenario === "stress") {
    return [...buildStressScreenshots(), ...base]
  }

  if (scenario === "edge") {
    return [...buildEdgeScreenshots(), ...base]
  }

  return base
}

function getScenarioScanScreenshots(): ScreenshotSnapshot[] {
  const scenario = getMockScenario()

  if (scenario === "empty") {
    return []
  }

  const base = clone(mockScreenshotSnapshots)

  if (scenario === "stress") {
    return buildStressScreenshots().map((item) => {
      const { updatedAt, ...snapshot } = item
      void updatedAt
      return snapshot
    }).concat(base)
  }

  if (scenario === "edge") {
    return buildEdgeScreenshots().map((item) => {
      const { updatedAt, ...snapshot } = item
      void updatedAt
      return snapshot
    }).concat(base)
  }

  return base
}

export function buildScreenshotsByTarget(
  _targetId: number,
  params?: ScreenshotListQueryParams
): ScreenshotListResponse<Screenshot> {
  return paginate(getScenarioTargetScreenshots(), params)
}

export function buildScreenshotsByScan(
  _scanId: number,
  params?: ScreenshotListQueryParams
): ScreenshotListResponse<ScreenshotSnapshot> {
  return paginate(getScenarioScanScreenshots(), params)
}

export function buildTargetScreenshotFilterOptions(
  _targetId: number,
  field: ScreenshotFilterOptionField
) {
  if (field !== "statusCode") {
    return { results: [] }
  }
  return { results: buildStatusCodeOptions(getScenarioTargetScreenshots()) }
}

export function buildScanScreenshotFilterOptions(
  _scanId: number,
  field: ScreenshotFilterOptionField
) {
  if (field !== "statusCode") {
    return { results: [] }
  }
  return { results: buildStatusCodeOptions(getScenarioScanScreenshots()) }
}
