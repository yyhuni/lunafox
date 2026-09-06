import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-step.ts"), "utf8")

describe("use-step contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useStep")
    expect(source).toContain("from 'react'")
  })
})
