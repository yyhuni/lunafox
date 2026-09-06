import {
  globalAssetSearchQueryFingerprint,
  normalizeGlobalAssetSearchPageSize,
  parseGlobalAssetSearchQuery,
  type GlobalAssetSearchCondition,
  type GlobalAssetSearchQuery,
} from "@/lib/global-asset-search-query"
import type {
  EndpointSearchResult,
  SearchParams,
  SearchResponse,
  SearchResult,
  WebsiteSearchResult,
} from "@/types/search.types"
import { mockEndpoints } from "./endpoints"
import { mockWebsites } from "./websites"

interface MockSearchPageToken {
  fingerprint: string
  createdAt: string
  id: number
}

function websiteToSearchResult(website: typeof mockWebsites[number]): WebsiteSearchResult {
  return {
    id: website.id,
    name: `targets/${website.target}/websites/${website.id}`,
    url: website.url,
    host: website.host,
    title: website.title,
    tech: website.tech ?? [],
    statusCode: website.statusCode,
    contentLength: website.contentLength,
    contentType: website.contentType,
    webserver: website.webserver,
    location: website.location,
    vhost: website.vhost ?? null,
    responseHeaders: "",
    responseBody: website.responseBody ?? "",
    createdAt: website.createdAt ?? "",
  }
}

function endpointToSearchResult(endpoint: typeof mockEndpoints[number]): EndpointSearchResult {
  return {
    id: endpoint.id,
    name: `targets/1/endpoints/${endpoint.id}`,
    targetId: 1,
    url: endpoint.url,
    host: endpoint.host ?? "",
    title: endpoint.title,
    tech: endpoint.tech ?? [],
    statusCode: endpoint.statusCode,
    contentLength: endpoint.contentLength,
    contentType: endpoint.contentType ?? "",
    webserver: endpoint.webserver ?? "",
    location: endpoint.location ?? "",
    vhost: endpoint.vhost ?? null,
    responseHeaders: endpoint.responseHeaders ?? "",
    responseBody: endpoint.responseBody ?? "",
    createdAt: endpoint.createdAt ?? "",
  }
}

function matchesSearchQuery(record: SearchResult, query: GlobalAssetSearchQuery): boolean {
  if (query.mode === "plainUrl") {
    return record.url === query.value
  }
  return query.conditions.every((condition) => matchesCondition(record, condition))
}

function matchesCondition(record: SearchResult, condition: GlobalAssetSearchCondition): boolean {
  if (condition.field === "statusCode") {
    return record.statusCode === condition.value
  }
  if (condition.field === "tech") {
    return record.tech.includes(String(condition.value))
  }

  const fieldValue = record[condition.field]
  if (typeof fieldValue !== "string") return false
  const expected = String(condition.value)
  if (condition.field === "url") {
    return fieldValue === expected
  }
  return condition.operator === "=="
    ? fieldValue === expected
    : fieldValue.toLocaleLowerCase().includes(expected.toLocaleLowerCase())
}

function sortByCreatedAtAndID(records: SearchResult[]): SearchResult[] {
  return [...records].sort((left, right) => {
    const byCreatedAt = Date.parse(right.createdAt) - Date.parse(left.createdAt)
    return byCreatedAt || right.id - left.id
  })
}

function encodePageToken(payload: MockSearchPageToken): string {
  return `mock-search.v1.${encodeURIComponent(JSON.stringify(payload))}`
}

function decodePageToken(raw: string | undefined, fingerprint: string): MockSearchPageToken | undefined {
  if (!raw) return undefined
  const encoded = raw.startsWith("mock-search.v1.") ? raw.slice("mock-search.v1.".length) : ""
  if (!encoded) throw new Error("invalid global asset search pageToken")
  try {
    const token = JSON.parse(decodeURIComponent(encoded)) as MockSearchPageToken
    if (!token.fingerprint || !token.createdAt || !Number.isInteger(token.id) || token.id < 1 || token.fingerprint !== fingerprint) {
      throw new Error("invalid global asset search pageToken")
    }
    return token
  } catch {
    throw new Error("invalid global asset search pageToken")
  }
}

export function getMockSearchResults(params: SearchParams): SearchResponse {
  const query = parseGlobalAssetSearchQuery(params.q)
  const pageSize = normalizeGlobalAssetSearchPageSize(params.pageSize)
  const fingerprint = globalAssetSearchQueryFingerprint(query, params.assetType, pageSize)
  const cursor = decodePageToken(params.pageToken, fingerprint)
  const source = params.assetType === "website"
    ? mockWebsites.map(websiteToSearchResult)
    : mockEndpoints.map(endpointToSearchResult)
  const matchingRecords = sortByCreatedAtAndID(source.filter((record) => matchesSearchQuery(record, query)))
  const cursorIndex = cursor
    ? matchingRecords.findIndex((record) => record.id === cursor.id && record.createdAt === cursor.createdAt)
    : -1
  if (cursor && cursorIndex < 0) {
    throw new Error("invalid global asset search pageToken")
  }

  const page = matchingRecords.slice(cursorIndex + 1, cursorIndex + 1 + pageSize + 1)
  const hasNextPage = page.length > pageSize
  const results = hasNextPage ? page.slice(0, pageSize) : page
  const last = results.at(-1)

  return {
    results,
    ...(hasNextPage && last ? {
      nextPageToken: encodePageToken({ fingerprint, createdAt: last.createdAt, id: last.id }),
    } : {}),
  }
}
