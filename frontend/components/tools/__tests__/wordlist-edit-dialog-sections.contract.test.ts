import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/wordlist-edit-dialog-sections.tsx"), "utf8")

describe("wordlist-edit-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function WordlistEditHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/icons\"")
  })

  it("does not render an editable display name input", () => {
    expect(source).toContain("export function WordlistEditMetadata")
    expect(source).not.toContain("onDisplayNameChange")
    expect(source).not.toContain("wordlist-display-name")
    expect(source).not.toContain("displayName.trim")
  })

  it("uses a single footer save action instead of metadata-local save", () => {
    expect(source).not.toContain("saveMetadata")
    expect(source).not.toContain("onSave: () => void")
    expect(source).toContain("hasMetadataChanges")
    expect(source).toContain("isSavingMetadata")
    expect(source).toContain("hasChanges || hasMetadataChanges")
  })
})
