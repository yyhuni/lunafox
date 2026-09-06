import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/target-overview-state.ts"), "utf8")

describe("target-overview-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useTargetOverviewState")
    expect(source).toContain("from \"react\"")
  })

  it("links subdomain summary cards through the canonical plural child route", () => {
    expect(source).toContain("`/targets/${targetId}/subdomains/`")
    expect(source).not.toContain("`/targets/${targetId}/subdomain/`")
  })
})
