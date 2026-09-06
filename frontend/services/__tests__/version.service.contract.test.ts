import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "services/version.service.ts"), "utf8")

describe("version.service contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from '@/lib/api-client'")
  })
})
