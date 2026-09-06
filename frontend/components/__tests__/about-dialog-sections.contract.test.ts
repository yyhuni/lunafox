import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/about-dialog-sections.tsx"), "utf8")

describe("about-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AboutDialogHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/icons\"")
  })

  it("adds a support-author shortcut from the about dialog", () => {
    expect(source).toContain("supportAuthor")
    expect(source).toContain("/settings/support/")
  })

  it("uses the shared brand mark instead of a page-local logo image", () => {
    expect(source).toContain('from "@/components/brand/lunafox-mark"')
    expect(source).toContain("<LunaFoxMark")
    expect(source).not.toContain("logoSrc")
    expect(source).not.toContain("<img")
  })
})
