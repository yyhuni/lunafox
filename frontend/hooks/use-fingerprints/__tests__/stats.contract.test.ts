import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-fingerprints/stats.ts"), "utf8")

describe("stats contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useFingerprintStats")
    expect(source).toContain("from \"@tanstack/react-query\"")
  })
})
