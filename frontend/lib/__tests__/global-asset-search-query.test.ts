import { describe, expect, it } from "vitest"
import {
  GLOBAL_ASSET_SEARCH_FEATURED_GUIDANCE,
  GLOBAL_ASSET_SEARCH_GUIDANCE,
  GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE,
  getGlobalAssetSearchDiagnosticCode,
  GlobalAssetSearchQueryError,
  globalAssetSearchQueryFingerprint,
  normalizeGlobalAssetSearchPageSize,
  parseGlobalAssetSearchQuery,
} from "@/lib/global-asset-search-query"

describe("global asset search query", () => {
  it("keeps ordinary URL text out of the structured DSL", () => {
    expect(parseGlobalAssetSearchQuery("https://example.test/?foo=bar%zz ")).toEqual({
      mode: "plainUrl",
      value: "https://example.test/?foo=bar%zz",
    })
    expect(parseGlobalAssetSearchQuery("HTTPS://example.test/path with=literal&&payload")).toEqual({
      mode: "plainUrl",
      value: "HTTPS://example.test/path with=literal&&payload",
    })
    expect(parseGlobalAssetSearchQuery(" jd ")).toEqual({
      mode: "plainUrl",
      value: "jd",
    })
    expect(parseGlobalAssetSearchQuery("login and admin")).toEqual({
      mode: "plainUrl",
      value: "login and admin",
    })
    expect(parseGlobalAssetSearchQuery("login or admin")).toEqual({
      mode: "plainUrl",
      value: "login or admin",
    })
  })

  it("treats the word and as the same flat conjunction as &&", () => {
    const expected = {
      mode: "structured",
      combinator: "and",
      conditions: [
        { field: "url", operator: "=", value: "admin" },
        { field: "host", operator: "=", value: "api" },
        { field: "title", operator: "==", value: "Login" },
      ],
    }
    for (const query of [
      'url="admin" and host="api" and title=="Login"',
      'url="admin" AND host="api" And title=="Login"',
      'url="admin"&&host="api" and title=="Login"',
    ]) {
      expect(parseGlobalAssetSearchQuery(query)).toEqual(expected)
    }
    expect(parseGlobalAssetSearchQuery('title="foo and bar"')).toEqual({
      mode: "structured",
      combinator: "and",
      conditions: [{ field: "title", operator: "=", value: "foo and bar" }],
    })
    const withAnd = parseGlobalAssetSearchQuery('host="api" and tech="nginx"')
    const withSymbol = parseGlobalAssetSearchQuery('host="api" && tech="nginx"')
    expect(globalAssetSearchQueryFingerprint(withAnd, "website", 10)).toBe(
      globalAssetSearchQueryFingerprint(withSymbol, "website", 10)
    )
  })

  it("treats the word or as the same flat disjunction as ||", () => {
    const expected = {
      mode: "structured",
      combinator: "or",
      conditions: [
        { field: "url", operator: "=", value: "admin" },
        { field: "host", operator: "=", value: "api" },
        { field: "title", operator: "==", value: "Login" },
      ],
    }
    for (const query of [
      'url="admin" or host="api" or title=="Login"',
      'url="admin" OR host="api" Or title=="Login"',
      'url="admin"||host="api" or title=="Login"',
    ]) {
      expect(parseGlobalAssetSearchQuery(query)).toEqual(expected)
    }
    expect(parseGlobalAssetSearchQuery('title="foo or bar"')).toEqual({
      mode: "structured",
      combinator: "and",
      conditions: [{ field: "title", operator: "=", value: "foo or bar" }],
    })
    const withOr = parseGlobalAssetSearchQuery('host="api" or tech="nginx"')
    const withSymbol = parseGlobalAssetSearchQuery('host="api" || tech="nginx"')
    const withAnd = parseGlobalAssetSearchQuery('host="api" && tech="nginx"')
    expect(globalAssetSearchQueryFingerprint(withOr, "website", 10)).toBe(
      globalAssetSearchQueryFingerprint(withSymbol, "website", 10)
    )
    expect(globalAssetSearchQueryFingerprint(withOr, "website", 10)).not.toBe(
      globalAssetSearchQueryFingerprint(withAnd, "website", 10)
    )
  })

  it("parses the approved fields with their typed values", () => {
    const query = parseGlobalAssetSearchQuery('host="api" && title=="Admin" && statusCode="200" && tech="nginx"')
    expect(query).toEqual({
      mode: "structured",
      combinator: "and",
      conditions: [
        { field: "host", operator: "=", value: "api" },
        { field: "title", operator: "==", value: "Admin" },
        { field: "statusCode", operator: "=", value: 200 },
        { field: "tech", operator: "=", value: "nginx" },
      ],
    })
  })

  it("parses hasScreenshot as an exact boolean with both operators", () => {
    expect(parseGlobalAssetSearchQuery('hasScreenshot="true"')).toEqual({
      mode: "structured",
      combinator: "and",
      conditions: [{ field: "hasScreenshot", operator: "=", value: "true" }],
    })
    expect(parseGlobalAssetSearchQuery('hasScreenshot=="false"')).toEqual({
      mode: "structured",
      combinator: "and",
      conditions: [{ field: "hasScreenshot", operator: "==", value: "false" }],
    })
    expect(parseGlobalAssetSearchQuery('host="api" && hasScreenshot=="true"')).toEqual({
      mode: "structured",
      combinator: "and",
      conditions: [
        { field: "host", operator: "=", value: "api" },
        { field: "hasScreenshot", operator: "==", value: "true" },
      ],
    })
    for (const query of ['hasScreenshot=="TRUE"', 'hasScreenshot="1"', 'hasScreenshot==""', 'hasScreenshot=" true "']) {
      expect(() => parseGlobalAssetSearchQuery(query)).toThrow(GlobalAssetSearchQueryError)
    }
  })

  it("fast-fails unsafe, oversized, or too-broad queries", () => {
    const elevenConditions = Array.from({ length: 11 }, () => 'tech="nginx"').join(" && ")
    for (const query of [
      " ",
      elevenConditions,
      'host="api" && tech="nginx" || title="Login"',
      'host="api" or tech="nginx" && title="Login"',
      'host!="api"',
      'responseBody="secret"',
      "a".repeat(2049),
      "a",
      'url="a"',
      'host="a"',
      'title="a"',
    ]) {
      expect(() => parseGlobalAssetSearchQuery(query)).toThrow()
    }
    expect(parseGlobalAssetSearchQuery('url="jd"')).toEqual({
      mode: "structured",
      combinator: "and",
      conditions: [{ field: "url", operator: "=", value: "jd" }],
    })
    expect(parseGlobalAssetSearchQuery('host="ab"')).toEqual({
      mode: "structured",
      combinator: "and",
      conditions: [{ field: "host", operator: "=", value: "ab" }],
    })
    expect(parseGlobalAssetSearchQuery('title="登录"')).toEqual({
      mode: "structured",
      combinator: "and",
      conditions: [{ field: "title", operator: "=", value: "登录" }],
    })
    expect(parseGlobalAssetSearchQuery('url=="a"')).toEqual({
      mode: "structured",
      combinator: "and",
      conditions: [{ field: "url", operator: "==", value: "a" }],
    })
    expect(() => parseGlobalAssetSearchQuery('title=="A"')).not.toThrow()
    expect(() => parseGlobalAssetSearchQuery('tech="ng"')).not.toThrow()
  })

  it("keeps the guidance catalog parser-valid and covers every featured field", () => {
    expect(GLOBAL_ASSET_SEARCH_GUIDANCE.length).toBeGreaterThan(6)
    expect(() => GLOBAL_ASSET_SEARCH_GUIDANCE.forEach((entry) => parseGlobalAssetSearchQuery(entry.query))).not.toThrow()
    expect(GLOBAL_ASSET_SEARCH_FEATURED_GUIDANCE.map((entry) => entry.query)).toEqual([
      'url="admin"',
      'host="api"',
      'title="Login"',
      'statusCode=="200"',
      'tech="nginx"',
      'host="api" && statusCode=="200"',
    ])
  })

  it("assigns stable diagnostic codes and hides unknown parser details", () => {
    const cases = [
      ["", "required"],
      ["a".repeat(2049), "queryTooLong"],
      ["domain=\"example\"", "unsupportedField"],
      ["host!=\"api\"", "invalidOperator"],
      ["statusCode=\"ok\"", "invalidStatusCode"],
      ["hasScreenshot=\"yes\"", "invalidBoolean"],
      ["host=\"a\"", "containsValueTooShort"],
      ["host=api", "missingQuotedValue"],
      ["host=\"api", "unterminatedQuotedValue"],
      ["host=\"api\" && tech=\"nginx\" or title=\"Login\"", "invalidConnector"],
      ["host=\"api\" or tech=\"nginx\" && title=\"Login\"", "invalidConnector"],
      ["host=\"api\" android=\"x\"", "invalidConnector"],
      ["host=\"api\" origin=\"x\"", "invalidConnector"],
      ["host=\"api\" &&", "missingCondition"],
      ["host=\"api\" and", "missingCondition"],
      ["host=\"api\" ||", "missingCondition"],
      ["host=\"api\" or", "missingCondition"],
    ] as const

    for (const [query, code] of cases) {
      try {
        parseGlobalAssetSearchQuery(query)
        throw new Error(`expected ${query} to fail`)
      } catch (error) {
        expect(getGlobalAssetSearchDiagnosticCode(error)).toBe(code)
      }
    }

    expect(getGlobalAssetSearchDiagnosticCode(new Error("internal parser detail"))).toBe("unknown")
    expect(getGlobalAssetSearchDiagnosticCode(new GlobalAssetSearchQueryError("bad", "unknown"))).toBe("unknown")
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
