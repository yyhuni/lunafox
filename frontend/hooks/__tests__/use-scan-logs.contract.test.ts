import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-task-progress-logs.ts"), "utf8")

describe("use-scan-logs contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScanLogs")
    expect(source).toContain("from 'react'")
    expect(source).toContain("getScanLogs")
  })
})
