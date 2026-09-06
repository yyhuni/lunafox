import React from "react"
import { render, screen, within } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { BulkLineValidationInput } from "@/components/common/bulk-line-validation-input"

function renderBulkLineValidationInput() {
  return render(
    <BulkLineValidationInput
      id="items"
      name="items"
      label="Items"
      required
      placeholder="One item per line"
      value=""
      lineCount={8}
      lineNumbersRef={React.createRef<HTMLDivElement>()}
      textareaRef={React.createRef<HTMLTextAreaElement>()}
      onValueChange={() => {}}
      onScroll={() => {}}
      helper="Use one item per line."
      example="Example: example.com"
      emptySummary="No content entered yet"
      validSummary="All items are valid"
      blockingSummary="Errors block submit"
      collapseDetails="Collapse details"
      expandDetails="Expand details"
      validationResult={null}
    />
  )
}

describe("BulkLineValidationInput", () => {
  it("keeps helper and example copy inside the empty feedback panel", () => {
    renderBulkLineValidationInput()

    const feedbackPanel = screen.getByText("No content entered yet").parentElement

    expect(feedbackPanel).not.toBeNull()
    expect(within(feedbackPanel!).getByText("Use one item per line.")).toBeInTheDocument()
    expect(within(feedbackPanel!).getByText("Example: example.com")).toBeInTheDocument()
  })

  it("keeps blocking issue details collapsed by default", () => {
    render(
      <BulkLineValidationInput
        id="items"
        name="items"
        label="Items"
        placeholder="One item per line"
        value="bad"
        lineCount={8}
        lineNumbersRef={React.createRef<HTMLDivElement>()}
        textareaRef={React.createRef<HTMLTextAreaElement>()}
        onValueChange={() => {}}
        onScroll={() => {}}
        helper="Use one item per line."
        example="Example: example.com"
        emptySummary="No content entered yet"
        validSummary="All items are valid"
        blockingSummary="1 error blocks submit"
        collapseDetails="Collapse details"
        expandDetails="Expand details"
        validationResult={{
          validCount: 0,
          blockingIssueCount: 1,
          advisoryIssueCount: 0,
          lineIssues: [{
            id: "invalid-1",
            lineNumber: 1,
            tone: "error",
            badgeLabel: "Invalid",
            message: "Line 1: invalid item",
          }],
        }}
      />
    )

    expect(screen.getByText("Expand details")).toBeInTheDocument()
    expect(screen.queryByText("Line 1: invalid item")).not.toBeInTheDocument()
  })

  it("can suppress the success panel when its workbench owns a compact validation summary", () => {
    render(
      <BulkLineValidationInput
        id="items"
        name="items"
        label="Items"
        placeholder="One item per line"
        value="good"
        lineCount={8}
        lineNumbersRef={React.createRef<HTMLDivElement>()}
        textareaRef={React.createRef<HTMLTextAreaElement>()}
        onValueChange={() => {}}
        onScroll={() => {}}
        helper="Use one item per line."
        example="Example: example.com"
        emptySummary="No content entered yet"
        validSummary="All items are valid"
        blockingSummary="Errors block submit"
        collapseDetails="Collapse details"
        expandDetails="Expand details"
        showSuccessSummary={false}
        validationResult={{
          validCount: 1,
          blockingIssueCount: 0,
          advisoryIssueCount: 0,
          lineIssues: [],
        }}
      />
    )

    expect(screen.queryByText("All items are valid")).not.toBeInTheDocument()
  })

  it("can suppress the empty panel when a loading owner only needs the editor shell", () => {
    render(
      <BulkLineValidationInput
        id="items"
        name="items"
        label="Items"
        placeholder="One item per line"
        value=""
        lineCount={8}
        lineNumbersRef={React.createRef<HTMLDivElement>()}
        textareaRef={React.createRef<HTMLTextAreaElement>()}
        onValueChange={() => {}}
        onScroll={() => {}}
        helper="Use one item per line."
        example="Example: example.com"
        emptySummary="No content entered yet"
        validSummary="All items are valid"
        blockingSummary="Errors block submit"
        collapseDetails="Collapse details"
        expandDetails="Expand details"
        validationResult={null}
        showEmptySummary={false}
      />
    )

    expect(screen.queryByText("No content entered yet")).not.toBeInTheDocument()
  })

  it("shows line numbers for every entered line even when the caller passes a smaller line count", () => {
    render(
      <BulkLineValidationInput
        id="items"
        name="items"
        label="Items"
        placeholder="One item per line"
        value={Array.from({ length: 100 }, () => "item").join("\n")}
        lineCount={8}
        lineNumbersRef={React.createRef<HTMLDivElement>()}
        textareaRef={React.createRef<HTMLTextAreaElement>()}
        onValueChange={() => {}}
        onScroll={() => {}}
        helper="Use one item per line."
        example="Example: example.com"
        emptySummary="No content entered yet"
        validSummary="All items are valid"
        blockingSummary="Errors block submit"
        collapseDetails="Collapse details"
        expandDetails="Expand details"
        validationResult={null}
      />
    )

    expect(screen.getByText("100")).toBeInTheDocument()
  })
})
