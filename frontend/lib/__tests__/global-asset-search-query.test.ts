import { describe, expect, it } from "vitest"
import {
  GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE,
  globalAssetSearchQueryFingerprint,
  normalizeGlobalAssetSearchPageSize,
  parseGlobalAssetSearchQuery,
} from "@/lib/global-asset-search-query"

describe("global asset search query", () => {
  it("keeps ordinary URL text out of the structured DSL", () => {
    expect(parseGlobalAssetSearchQuery("https://example.test/?foo=bar%zz ")).toEqual({
      mode: "plainUrl",
      value: "https://example.test/?foo=bar%zz ",
    })
    expect(parseGlobalAssetSearchQuery("HTTPS://example.test/path with=literal&&payload")).toEqual({
      mode: "plainUrl",
      value: "HTTPS://example.test/path with=literal&&payload",
    })
  })

  it("parses the approved fields with their typed values", () => {
    const query = parseGlobalAssetSearchQuery('host="api" && title=="Admin" && statusCode="200" && tech="nginx"')
    expect(query).toEqual({
      mode: "structured",
      conditions: [
        { field: "host", operator: "=", value: "api" },
        { field: "title", operator: "==", value: "Admin" },
        { field: "statusCode", operator: "=", value: 200 },
        { field: "tech", operator: "=", value: "nginx" },
      ],
    })
  })

  it("fast-fails unsafe, oversized, or too-broad queries", () => {
    const elevenConditions = Array.from({ length: 11 }, () => 'tech="nginx"').join(" && ")
    for (const query of [
      " ",
      "ab",
      elevenConditions,
      'host="api" || tech="nginx"',
      'host!="api"',
      'responseBody="secret"',
      "a".repeat(2049),
    ]) {
      expect(() => parseGlobalAssetSearchQuery(query)).toThrow()
    }
    expect(parseGlobalAssetSearchQuery('url="ab"')).toEqual({
      mode: "structured",
      conditions: [{ field: "url", operator: "==", value: "ab" }],
    })
    expect(() => parseGlobalAssetSearchQuery('title=="A"')).not.toThrow()
    expect(() => parseGlobalAssetSearchQuery('tech="ng"')).not.toThrow()
  })

  it("normalizes the page size and binds a stable query fingerprint", () => {
    const left = parseGlobalAssetSearchQuery('host="api" && tech="nginx"')
    const right = parseGlobalAssetSearchQuery('tech="nginx" && host="api"')
    expect(normalizeGlobalAssetSearchPageSize(undefined)).toBe(GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE)
    expect(normalizeGlobalAssetSearchPageSize(100)).toBe(100)
    expect(() => normalizeGlobalAssetSearchPageSize(0)).toThrow()
    expect(globalAssetSearchQueryFingerprint(left, "website", 10)).toBe(
      globalAssetSearchQueryFingerprint(right, "website", 10)
    )
  })
})
