import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/organizations.ts"), "utf8")

describe("organizations contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockOrganizations")
    expect(source).toContain("from '@/types/organization.types'")
  })

  it("keeps enough default organizations to exercise compact picker pagination and truncation", () => {
    expect(source).toContain("North America Enterprise Security Integration Office")
    expect(source).toContain("Partner Cloud Exchange")
    expect(source).toContain("Zero Trust Pilot Group")
    expect(source).toContain("targetCount")
    expect(source).toContain("domainCount")
    expect(source).toContain("endpointCount")
    expect(source).toContain("updatedAt")
  })
})
