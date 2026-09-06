import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/wordlist-edit-dialog-state.ts"), "utf8")

describe("wordlist-edit-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useWordlistEditDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("keeps wordlist display name immutable after upload", () => {
    expect(source).not.toContain("setDisplayName")
    expect(source).not.toContain("displayName,")
    expect(source).not.toContain('updateMask: "displayName,description,tags"')
    expect(source).toContain('updateMask: "description,tags"')
  })

  it("saves metadata and content through one dialog save handler", () => {
    expect(source).toContain("handleSaveDialog")
    expect(source).toContain("hasMetadataChanges")
    expect(source).toContain("hasChanges")
    expect(source).toContain("saveCompletionCount")
  })

  it("keeps the shared drawer close guard and successful-save terminal path", () => {
    expect(source).toContain("handleClose")
    expect(source).toContain('window.confirm(t("confirmClose"))')
    expect(source).toContain("onOpenChange(false)")
    expect(source).not.toContain("onError: () => onOpenChange(false)")
  })

  it("gates content queries behind explicit edit intent", () => {
    expect(source).toContain("contentEnabled?: boolean")
    expect(source).toContain("contentEnabled && open && wordlist && canEditContent")
    expect(source).toContain("contentEnabled && open")
  })
})
