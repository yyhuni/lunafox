import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-route-prefetch.ts"), "utf8")

const extractStringArray = (name: string) => {
  const match = source.match(new RegExp(`const ${name} = \\[([\\s\\S]*?)\\] as const`))
  expect(match).not.toBeNull()
  return Array.from(match?.[1].matchAll(/['"]([^'"]+)['"]/g) ?? [], ([, value]) => value)
}

describe("use-route-prefetch contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useRoutePrefetch")
    expect(source).toContain("from 'react'")
  })

  it("keeps global route prefetching bounded and network-aware", () => {
    expect(extractStringArray("BASE_CRITICAL_ROUTES")).toHaveLength(1)
    expect(extractStringArray("BASE_SECONDARY_ROUTES")).toEqual([])
    expect(extractStringArray("BASE_LOW_PRIORITY_ROUTES")).toEqual([])
    expect(source).not.toContain("'/targets/'")
    expect(source).not.toContain("'/scan/history/'")
    expect(source).toContain("requestIdleCallback")
    expect(source).toContain("saveData")
    expect(source).toContain("slow-2g")
    expect(source).toContain("effectiveType === '4g'")
    expect(source).toContain("currentPath && normalizePath(currentPath) === normalizedPath")
  })
})
