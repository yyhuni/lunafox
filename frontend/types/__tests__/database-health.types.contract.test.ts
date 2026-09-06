import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/database-health.types.ts"), "utf8")

describe("database-health.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export type DatabaseHealthStatus = 'online' | 'degraded' | 'maintenance' | 'offl")
  })

  it("keeps the snapshot type aligned to the backend MVP response", () => {
    expect(source).toContain("databaseSizeBytes: number | null")
    expect(source).toContain("export interface DatabaseHealthFinding")
    expect(source).toContain("findings: DatabaseHealthFinding[]")
    expect(source).toContain("evidence: string[]")
    expect(source).toContain("recommendation: string")
    expect(source).not.toContain("region:")
    expect(source).not.toContain("databaseName")
    expect(source).not.toContain("storageUsedBytes")
    expect(source).not.toContain("storageTotalBytes")
  })
})
