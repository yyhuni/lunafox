import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-config-editor-state.ts"), "utf8")

describe("scan-config-editor-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScanConfigEditorState")
    expect(source).toContain("from \"react\"")
  })
})
