import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-tools.ts"), "utf8")

describe("use-tools contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useTools")
    expect(source).toContain("from \"@tanstack/react-query\"")
  })
})
