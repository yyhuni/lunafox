import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/database-health.ts"), "utf8")

describe("database-health contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from '@/types/database-health.types'")
  })

  it("does not include fields that the backend MVP report does not return", () => {
    expect(source).toContain("databaseSizeBytes")
    expect(source).toContain("findings")
    expect(source).toContain("recommendation")
    expect(source).not.toContain("databaseName")
    expect(source).not.toContain("storageUsedBytes")
    expect(source).not.toContain("storageTotalBytes")
  })
})
