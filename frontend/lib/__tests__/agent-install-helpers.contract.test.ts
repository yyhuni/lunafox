import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/agent-install-helpers.ts"), "utf8")

describe("agent-install-helpers contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export const normalizeOrigin = (value: string): string => value.replace(/\\/+$/, ")
  })
})
