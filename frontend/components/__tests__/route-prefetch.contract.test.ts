import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/route-prefetch.tsx"), "utf8")

describe("route-prefetch contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function RoutePrefetch")
    expect(source).toContain("from 'next/navigation'")
  })
})
