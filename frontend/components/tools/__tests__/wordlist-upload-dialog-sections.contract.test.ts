import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/wordlist-upload-dialog-sections.tsx"), "utf8")

describe("wordlist-upload-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function WordlistUploadHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/icons\"")
  })

  it("renders upload metadata without a separate name input", () => {
    expect(source).toContain("export function WordlistUploadFields")
    expect(source).not.toContain("onNameChange")
    expect(source).not.toContain('name="name"')
    expect(source).not.toContain('id="name"')
    expect(source).not.toContain('tWordlists("name")')
  })
})
