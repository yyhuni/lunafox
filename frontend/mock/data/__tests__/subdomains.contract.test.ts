import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/subdomains.ts"), "utf8")

describe("subdomains contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockSubdomains")
    expect(source).toContain("from '@/types/subdomain.types'")
  })
})
