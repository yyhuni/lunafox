import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/wordlist-tag-picker.tsx"), "utf8")

describe("wordlist-tag-picker contract", () => {
  it("renders recommended tag suggestions as clickable badges instead of button chips", () => {
    expect(source).toContain("export function WordlistTagPicker")
    expect(source).toContain("suggestions.map")
    expect(source).toContain('variant={isSelected ? "secondary" : "outline"}')
    expect(source).toContain("render={(")
    expect(source).toContain('type="button"')
    expect(source).toContain("onClick={() => addTag(tag.displayName)}")
    expect(source).toContain("aria-pressed={isSelected}")
    expect(source).not.toContain('size="sm"\n              onClick={() => addTag(tag.displayName)}')
  })

  it("keeps manual tag creation beside selected tags instead of common recommendations", () => {
    expect(source).toContain("isAddingTag")
    expect(source).toContain("renderAddTagControl")
    expect(source).toContain('name="tagPickerInlineInput"')
    expect(source).toContain("setIsAddingTag(true)")
    expect(source).toContain("commitDraftTag")
    expect(source).toContain('<div className="flex min-h-7 flex-wrap items-center gap-1.5">')
    expect(source.indexOf("{renderAddTagControl()}")).toBeLessThan(
      source.indexOf('<div className="space-y-2 rounded-md border bg-muted/20 p-2.5">')
    )
    expect(source).not.toContain('name="tagPickerSearch"')
    expect(source).not.toContain('variant="outline" onClick={() => addTag(query)}')
  })

  it("separates selected tags from recommended tag suggestions", () => {
    expect(source).not.toContain('t("selectedTags")')
    expect(source).toContain('t("recommendedTags")')
    expect(source).toContain("rounded-md border bg-muted/20")
  })
})
