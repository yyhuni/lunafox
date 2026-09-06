import { describe, expect, it } from "vitest"

import {
  composeWebsiteScopeFilter,
  extractExactWebsiteHostFilter,
  extractWebsiteURLScopeFilter,
  matchesExactWebsiteHost,
  matchesWebsiteURLScope,
  removeLeadingWebsiteScopeFilter,
} from "@/lib/website-scope"
import { WEBSITE_ASSET_SCOPE, WEBSITE_URL_SCOPE_CASES } from "@/test/fixtures/website-scope"

describe("Website asset scope", () => {
  it("always prefixes the non-removable scope before the user filter", () => {
    expect(composeWebsiteScopeFilter(WEBSITE_ASSET_SCOPE, "websiteUrl", 'statusCode="200"')).toBe(
      'websiteUrl=="https://api.acme.com" && (statusCode="200")',
    )
    expect(composeWebsiteScopeFilter(WEBSITE_ASSET_SCOPE, "host", 'port="443"')).toBe(
      'host=="api.acme.com" && (port="443")',
    )
  })

  it.each(WEBSITE_URL_SCOPE_CASES)("$name", ({ scope, candidate, matches }) => {
    expect(matchesWebsiteURLScope(scope, candidate)).toBe(matches)
  })

  it("derives relation scope without parsing or rewriting payload bytes", () => {
    const scope = "HTTPS://Api.Acme.COM:443/a%2Fb?payload=%0d%0a#fragment"
    expect(matchesWebsiteURLScope(scope, "https://api.acme.com:443/a%2Fb/child?x=%zz#other")).toBe(true)
    expect(matchesWebsiteURLScope(scope, "https://api.acme.com/a/b/child?x=%zz")).toBe(false)
  })

  it("fast-fails invalid Website URLs and rejects invalid scope expressions", () => {
    expect(() => composeWebsiteScopeFilter({ ...WEBSITE_ASSET_SCOPE, url: "api.acme.com" }, "websiteUrl")).toThrow(
      "Website URL must be an absolute HTTP(S) URL",
    )
    expect(() => extractWebsiteURLScopeFilter('websiteUrl="https://api.acme.com"')).toThrow("requires ==")
    expect(() => extractWebsiteURLScopeFilter('websiteUrl=="https://api.acme.com" || statusCode="200"')).toThrow(
      "leading AND clause",
    )
    expect(() => extractWebsiteURLScopeFilter('statusCode="200" && websiteUrl=="https://api.acme.com"')).toThrow(
      "leading AND clause",
    )
  })

  it("keeps exact host matching separate from ordinary contains filters", () => {
    const filter = 'host=="api.acme.com" && (port="443")'

    expect(extractExactWebsiteHostFilter(filter)).toBe("api.acme.com")
    expect(removeLeadingWebsiteScopeFilter(filter, "host")).toBe('(port="443")')
    expect(matchesExactWebsiteHost("API.ACME.COM", "api.acme.com")).toBe(true)
    expect(matchesExactWebsiteHost("api.acme.com", "api.acme.com.evil")).toBe(false)
  })
})
