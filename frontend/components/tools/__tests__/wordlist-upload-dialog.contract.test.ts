import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/wordlist-upload-dialog.tsx"), "utf8")

describe("wordlist-upload-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function WordlistUploadDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("does not wire a separate wordlist name field", () => {
    expect(source).not.toContain("name,")
    expect(source).not.toContain("setName")
    expect(source).not.toContain("onNameChange")
    expect(source).toContain("canSubmit={!!file}")
  })
})
