import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { ScanConfigEditorLayout } from "@/components/scan/scan-config-editor-sections"

describe("ScanConfigEditorLayout schema feedback", () => {
  it("shows a schema validation reason in the existing YAML error details", () => {
    render(
      <ScanConfigEditorLayout
        state={{ capabilities: [], capabilityStyles: [] }}
        configuration="steps: {}"
        onChange={vi.fn()}
        validationError={'configuration.steps["discovery"].engineConfig: complete Engine configuration is required'}
        showCapabilities={false}
      />
    )

    expect(screen.getByText("configBlockingSummary:{\"count\":1}")).toBeVisible()
    fireEvent.click(screen.getByRole("button", { name: "configExpandDetails" }))
    expect(screen.getByText(/complete Engine configuration is required/)).toBeVisible()
  })
})
