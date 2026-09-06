import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/config/add-custom-tool-dialog-sections.tsx"), "utf8")

describe("add-custom-tool-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AddCustomToolDialogHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
