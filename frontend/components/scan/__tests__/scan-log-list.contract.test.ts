import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-log-list.tsx"), "utf8")

describe("scan-log-list contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanLogList")
    expect(source).toContain("from \"@/components/scan/scan-log-list-sections\"")
  })
})
