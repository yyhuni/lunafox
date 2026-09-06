import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/config/add-tool-dialog.tsx"), "utf8")

describe("add-tool-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AddToolDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
