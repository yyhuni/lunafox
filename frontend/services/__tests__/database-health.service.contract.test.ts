import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "services/database-health.service.ts"), "utf8")

describe("database-health.service contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from '@/lib/api-client'")
  })

  it("requests the canonical backend database health report", () => {
    expect(source).toContain("api.get<DatabaseHealthSnapshot>('/databaseHealthReports/current')")
    expect(source).not.toContain("/system/database-health")
  })
})
