import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "hooks/_shared/use-page-refresh-timestamp.ts"),
  "utf8"
)

describe("use-page-refresh-timestamp contract", () => {
  it("owns client mount and manual completion timestamps", () => {
    expect(source).toContain('"use client"')
    expect(source).toContain("export function usePageRefreshTimestamp")
    expect(source).toContain("React.useEffect")
    expect(source).toContain("setLastRefreshedAt(new Date())")
    expect(source).toContain("markRefreshCompleted")
  })
})
