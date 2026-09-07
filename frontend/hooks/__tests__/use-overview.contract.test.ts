import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-overview.ts"), "utf8")

describe("use-overview contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useOverviewStats")
    expect(source).toContain("from '@tanstack/react-query'")
  })

  it("exposes a server runtime metrics query", () => {
    expect(source).toContain("runtimeMetrics: () => [...overviewKeys.all, 'runtimeMetrics'] as const")
    expect(source).toContain("export function useServerRuntimeMetrics")
    expect(source).toContain("queryFn: getServerRuntimeMetrics")
    expect(source).toContain("refetchInterval: 2000")
  })
})
