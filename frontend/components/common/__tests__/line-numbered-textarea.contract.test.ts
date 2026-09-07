import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/line-numbered-textarea.tsx"), "utf8")

describe("line-numbered-textarea contract", () => {
  it("owns repeated line-number editor geometry", () => {
    expect(source).toContain("lineNumberedTextareaViewportClassName")
    expect(source).toContain("lineNumberedTextareaResponsiveViewportClassName")
    expect(source).toContain("lineNumberedTextareaFillViewportClassName")
    expect(source).toContain("lineNumberedTextareaRowClassName")
    expect(source).toContain("lineNumberedTextareaClassName")
    expect(source).toContain("getLineNumberGutterWidth")
    expect(source).toContain("countLineNumberedTextareaLines")
    expect(source).toContain("style={{ width: getLineNumberGutterWidth(normalizedLineCount) }}")
  })

  it("lets callers tune the line-number gutter without changing global defaults", () => {
    expect(source).toContain("numbersClassName?: string")
    expect(source).toContain("numbersClassName")
    expect(source).toContain("cn(lineNumberedTextareaNumbersClassName, numbersClassName)")
  })

  it("supports line-level validation highlighting without replacing the textarea primitive", () => {
    expect(source).toContain("lineHighlights?: LineNumberedTextareaHighlight[]")
    expect(source).toContain("lineNumberedTextareaHighlightToneClassNames")
    expect(source).toContain("lineNumberedTextareaBadgeToneClassNames")
    expect(source).toContain("highlightRowsRef")
  })

  it("keeps line numbers and textarea text on the same baseline rhythm", () => {
    expect(source).toContain("lineNumberedTextareaRowClassName = \"h-5\"")
    expect(source).toContain("lineNumberedTextareaNumbersClassName =\n  \"font-mono h-full leading-5 overflow-y-auto py-3 scrollbar-hide text-muted-foreground text-right text-sm\"")
    expect(source).toContain("lineNumberedTextareaClassName =\n  \"border-0 focus-visible:ring-0 focus-visible:ring-offset-0 font-mono h-full leading-5 overflow-y-auto py-3 resize-none text-sm\"")
    expect(source).not.toContain("leading-[1.4]")
  })
})
