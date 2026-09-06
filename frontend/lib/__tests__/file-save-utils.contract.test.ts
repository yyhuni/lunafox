import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/file-save-utils.ts"), "utf8")

describe("file-save-utils contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function saveBlobAsFile")
  })
})
