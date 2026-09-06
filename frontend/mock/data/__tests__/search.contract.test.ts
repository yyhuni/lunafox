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

    const exactURL = getMockSearchResults({ q: "https://acme.com", assetType: "website" })
    expect(exactURL.results).toHaveLength(1)
    expect(exactURL.results[0]?.url).toBe("https://acme.com")

    expect(getMockSearchResults({ q: "acme", assetType: "website" }).results).toHaveLength(0)
  })

  it("does not retain the permissive legacy query syntax", () => {
    expect(() => getMockSearchResults({
      q: 'tech="React" || tech="Vue.js"',
      assetType: "website",
    })).toThrow(GlobalAssetSearchQueryError)
    expect(getMockSearchResults({
      q: 'tech="react"',
      assetType: "website",
    }).results).toHaveLength(0)
  })
})
