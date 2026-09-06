import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-version.ts"), "utf8")

describe("use-version contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useVersion")
    expect(source).toContain("from '@tanstack/react-query'")
  })
})
