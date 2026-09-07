import { describe, expect, it } from "vitest"
import {
  parseBlacklistRule,
  parseBlacklistRules,
  submittedBlacklistPatterns,
} from "@/components/settings/blacklist/blacklist-rule-parser"

describe("blacklist rule parser", () => {
  it("accepts the policy language supported by the editor", () => {
    expect(parseBlacklistRule("example.com")).toBe("domain")
    expect(parseBlacklistRule("*.example.com")).toBe("domain")
    expect(parseBlacklistRule("192.0.2.1")).toBe("ipv4")
    expect(parseBlacklistRule("192.0.2.0/24")).toBe("cidr")
  })

  it.each([
    "*keyword*",
    "api.*.example.com",
    "2001:db8::1",
    "2001:db8::/32",
    "https://example.com/path",
    "localhost",
  ])("blocks unsupported pattern %s", (pattern) => {
    expect(parseBlacklistRule(pattern)).toBe("invalid")
  })

  it("preserves non-empty user-entered lines and identifies invalid line numbers", () => {
    const rules = parseBlacklistRules("# comment\nexample.com\napi.*.example.com\n192.0.2.1")

    expect(rules).toEqual([
      { line: 2, value: "example.com", kind: "domain", valid: true },
      { line: 3, value: "api.*.example.com", kind: "invalid", valid: false },
      { line: 4, value: "192.0.2.1", kind: "ipv4", valid: true },
    ])
    expect(submittedBlacklistPatterns(rules)).toEqual([
      "example.com",
      "api.*.example.com",
      "192.0.2.1",
    ])
  })
})
