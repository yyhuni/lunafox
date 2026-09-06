import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/wordlist-edit-dialog.tsx"), "utf8")

describe("wordlist-edit-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function WordlistEditDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"next-intl\"")
  })

  it("uses the shared editor dialog geometry", () => {
    expect(source).toContain("editorDialogPanelClassName")
    expect(source).toContain('cn(editorDialogPanelClassName, "sm:max-w-6xl")')
    expect(source).not.toContain("max-w-[calc(100%-2rem)]")
  })
})
