import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/vulnerabilities.ts"), "utf8")

describe("vulnerabilities contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockVulnerabilities")
    expect(source).toContain("from '@/types/vulnerability.types'")
  })
})
