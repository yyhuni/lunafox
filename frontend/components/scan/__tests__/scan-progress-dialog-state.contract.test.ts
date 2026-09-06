import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-progress-dialog-state.ts"), "utf8")

describe("scan-progress-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScanProgressDialogState")
    expect(source).toContain("from \"react\"")
  })
})
