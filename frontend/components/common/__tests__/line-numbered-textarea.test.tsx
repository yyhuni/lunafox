import React from "react"
import { fireEvent, render } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import {
  countLineNumberedTextareaLines,
  LineNumberedTextarea,
  getLineNumberGutterWidth,
} from "@/components/common/line-numbered-textarea"

describe("LineNumberedTextarea", () => {
  it("counts logical lines without allocating a split array", () => {
    expect(countLineNumberedTextareaLines("")).toBe(1)
    expect(countLineNumberedTextareaLines("one\ntwo\nthree")).toBe(3)
  })

  it("sizes the line-number gutter from the displayed line count", () => {
    expect(getLineNumberGutterWidth(8)).toBe("calc(2ch + 1rem)")
    expect(getLineNumberGutterWidth(10)).toBe("calc(2ch + 1rem)")
    expect(getLineNumberGutterWidth(100)).toBe("calc(3ch + 1rem)")
  })

  it("updates the gutter width when the line count crosses a digit boundary", () => {
    const lineNumbersRef = React.createRef<HTMLDivElement>()
    const textareaRef = React.createRef<HTMLTextAreaElement>()
    const { rerender } = render(
      <LineNumberedTextarea
        lineCount={1}
        lineNumbersRef={lineNumbersRef}
        textareaRef={textareaRef}
        value=""
        onChange={() => {}}
      />
    )

    expect(lineNumbersRef.current?.parentElement).toHaveStyle({ width: "calc(2ch + 1rem)" })

    rerender(
      <LineNumberedTextarea
        lineCount={100}
        lineNumbersRef={lineNumbersRef}
        textareaRef={textareaRef}
        value=""
        onChange={() => {}}
      />
    )

    expect(lineNumbersRef.current?.parentElement).toHaveStyle({ width: "calc(3ch + 1rem)" })
  })

  it("keeps large line counts to a viewport-sized set of presentation rows", () => {
    const lineNumbersRef = React.createRef<HTMLDivElement>()
    const textareaRef = React.createRef<HTMLTextAreaElement>()
    const { container } = render(
      <LineNumberedTextarea
        lineCount={10_000}
        lineNumbersRef={lineNumbersRef}
        textareaRef={textareaRef}
        value=""
        onChange={() => {}}
        lineHighlights={[{ lineNumber: 5_000, tone: "error", label: "Invalid" }]}
      />
    )

    const highlightLayer = container.querySelector('[aria-hidden="true"]')
    expect(lineNumbersRef.current?.querySelectorAll('[data-slot="line-numbered-textarea-number-row"]').length).toBeLessThan(100)
    expect(highlightLayer?.querySelectorAll('[data-slot="line-numbered-textarea-highlight-row"]').length).toBeLessThan(100)

    const textarea = textareaRef.current
    expect(textarea).not.toBeNull()
    Object.defineProperty(textarea!, "scrollTop", { configurable: true, value: 99_980 })
    fireEvent.scroll(textarea!)

    expect(lineNumbersRef.current).toHaveTextContent("5000")
    expect(highlightLayer).toHaveTextContent("Invalid")
  })
})
