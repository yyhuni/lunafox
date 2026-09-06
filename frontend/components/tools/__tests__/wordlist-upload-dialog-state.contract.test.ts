import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/wordlist-upload-dialog-state.ts"), "utf8")

describe("wordlist-upload-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useWordlistUploadDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("uses selected file as the only required upload naming input", () => {
    expect(source).not.toContain("normalizeNameFromFile")
    expect(source).not.toContain("setName")
    expect(source).not.toContain("name,")
    expect(source).not.toContain("{ name,")
    expect(source).toContain("if (!file) return")
  })
})
