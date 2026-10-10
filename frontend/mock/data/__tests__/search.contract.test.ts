import { describe, expect, it } from "vitest"
import { GlobalAssetSearchQueryError } from "@/lib/global-asset-search-query"
import { getMockSearchResults } from "@/mock/data/search"

describe("search contract", () => {
  it("uses the production query semantics and cursor shape", () => {
    const structured = getMockSearchResults({
      q: 'host=="acme.com" && tech="React"',
      assetType: "website",
    })

    expect(structured.results).toHaveLength(1)
    expect(structured.results[0]?.host).toBe("acme.com")
    expect(structured.results[0]?.tech).toContain("React")

    const plainURL = getMockSearchResults({ q: "acme", assetType: "website" })
    expect(plainURL.results.length).toBeGreaterThan(1)
    expect(plainURL.results.every((result) => result.url.toLowerCase().includes("acme"))).toBe(true)

    const containsURL = getMockSearchResults({ q: 'url="ac"', assetType: "website" })
    expect(containsURL.results.length).toBeGreaterThan(1)

    const containsHost = getMockSearchResults({ q: 'host="ac"', assetType: "website" })
    expect(containsHost.results.length).toBeGreaterThan(1)

    const containsTitle = getMockSearchResults({ q: 'title="Ac"', assetType: "website" })
    expect(containsTitle.results.length).toBeGreaterThan(1)

    const exactURL = getMockSearchResults({ q: 'url=="https://acme.com"', assetType: "website" })
    expect(exactURL.results).toHaveLength(1)
    expect(exactURL.results[0]?.url).toBe("https://acme.com")

    expect(getMockSearchResults({ q: "jd", assetType: "website" }).results).toHaveLength(0)
    expect(() => getMockSearchResults({ q: 'host="a"', assetType: "website" })).toThrow(GlobalAssetSearchQueryError)
  })

  it("matches either technology for a flat OR and still rejects mixed connectors", () => {
    const either = getMockSearchResults({
      q: 'tech="React" || tech="Vue.js"',
      assetType: "website",
    })
    expect(either.results.length).toBeGreaterThan(1)
    expect(either.results.some((result) => result.tech.includes("React"))).toBe(true)
    expect(either.results.some((result) => result.tech.includes("Vue.js"))).toBe(true)
    expect(either.results.every((result) => result.tech.includes("React") || result.tech.includes("Vue.js"))).toBe(true)

    expect(() => getMockSearchResults({
      q: 'tech="React" && tech="Vue.js" || host="api"',
      assetType: "website",
    })).toThrow(GlobalAssetSearchQueryError)
    expect(getMockSearchResults({
      q: 'tech="react"',
      assetType: "website",
    }).results).toHaveLength(0)
  })
})
