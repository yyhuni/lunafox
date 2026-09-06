import { describe, expect, it } from "vitest"

import { getOrganizationInitial } from "../organization-initial"

describe("getOrganizationInitial", () => {
  it("keeps the first source character for English and Chinese names", () => {
    expect(getOrganizationInitial("Acme Corporation")).toBe("A")
    expect(getOrganizationInitial("阿里云")).toBe("阿")
  })

  it("trims surrounding whitespace without changing case or script", () => {
    expect(getOrganizationInitial("  acme  ")).toBe("a")
    expect(getOrganizationInitial("  中文组织  ")).toBe("中")
  })

  it("fails fast for an empty required name", () => {
    expect(() => getOrganizationInitial("   ")).toThrow(
      "Organization name is required to render its identity marker."
    )
  })
})
