import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/about-dialog-state.ts"), "utf8")

describe("about-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useAboutDialogState")
    expect(source).toContain("from \"react\"")
  })
})
