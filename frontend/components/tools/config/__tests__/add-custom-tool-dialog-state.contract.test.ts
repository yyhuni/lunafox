import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/config/add-custom-tool-dialog-state.ts"), "utf8")

describe("add-custom-tool-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useAddCustomToolDialogState")
    expect(source).toContain("from \"react\"")
  })
})
