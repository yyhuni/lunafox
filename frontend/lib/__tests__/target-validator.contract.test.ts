import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/target-validator.ts"), "utf8")

describe("target-validator contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from 'validator'")
  })

  it("keeps line-number-preserving batch parsing available for UI validation", () => {
    expect(source).toContain("export interface ParsedTargetLine")
    expect(source).toContain("static parseLines")
    expect(source).toContain("lineNumber: index + 1")
    expect(source).toContain("lineNumber: number; originalTarget: string")
  })
})
