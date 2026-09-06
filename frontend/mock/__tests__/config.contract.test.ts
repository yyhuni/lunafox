import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/config.ts"), "utf8")

describe("config contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function mockDelay")
  })
})
