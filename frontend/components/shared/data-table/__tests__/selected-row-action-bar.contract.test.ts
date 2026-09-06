import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import { join } from "node:path"
import * as React from "react"
import { render, screen } from "@testing-library/react"

import { SelectedRowActionBar } from "../selected-row-action-bar"

const componentPath = join(process.cwd(), "components/shared/data-table/selected-row-action-bar.tsx")
const indexPath = join(process.cwd(), "components/shared/data-table/index.ts")

const TestIcon = ({
  className,
  ...props
}: {
  className?: string
  "aria-hidden"?: "true"
}) => React.createElement("svg", { className, ...props })

describe("SelectedRowActionBar contract", () => {
  it("exports the shared selected-row action bar from the data-table entrypoint", () => {
    const indexSource = readFileSync(indexPath, "utf8")

    expect(indexSource).toContain('export { SelectedRowActionBar } from "./selected-row-action-bar"')
    expect(indexSource).toContain("SelectedRowActionBarAction")
  })

  it("owns the selected-row toolbar shell through shared primitives", () => {
    const source = readFileSync(componentPath, "utf8")

    expect(source).toContain('import { Button } from "@/components/ui/button"')
    expect(source).toContain('role="toolbar"')
    expect(source).toContain("aria-label={ariaLabel}")
    expect(source).toContain("countLabel")
    expect(source).toContain("onClearSelection")
    expect(source).toContain("clearSelectionLabel")
  })

  it("keeps action semantics configurable without leaking route business logic", () => {
    const source = readFileSync(componentPath, "utf8")

    expect(source).toContain('tone?: "default" | "success" | "muted" | "destructive"')
    expect(source).toContain("group?: string")
    expect(source).toContain("getActionClassName")
    expect(source).not.toContain("markAsReviewed")
    expect(source).not.toContain("markAsPending")
    expect(source).not.toContain("vulnerabil")
  })

  it("uses sidebar-aligned neutral feedback for non-destructive actions and keeps danger distinct", () => {
    render(
      React.createElement(SelectedRowActionBar, {
        selectedCount: 4,
        ariaLabel: "4 selected",
        countLabel: "4 selected",
        actions: [
          { key: "default", label: "Default", icon: TestIcon, onClick: () => {} },
          { key: "success", label: "Success", icon: TestIcon, tone: "success", onClick: () => {} },
          { key: "muted", label: "Muted", icon: TestIcon, tone: "muted", onClick: () => {} },
          { key: "delete", label: "Delete", icon: TestIcon, tone: "destructive", onClick: () => {} },
        ],
      })
    )

    for (const name of ["Default", "Success", "Muted"]) {
      const action = screen.getByRole("button", { name })

      expect(action).toHaveClass(
        "text-sidebar-foreground/65",
        "hover:bg-sidebar-accent",
        "hover:text-sidebar-accent-foreground",
        "dark:hover:bg-sidebar-accent"
      )
      expect(action).not.toHaveClass("hover:bg-transparent", "dark:hover:bg-transparent", "active:bg-transparent")
    }

    const destructiveAction = screen.getByRole("button", { name: "Delete" })
    expect(destructiveAction).toHaveClass("text-destructive", "hover:bg-destructive/10")
    expect(destructiveAction).not.toHaveClass("hover:bg-accent")

    expect(screen.getByRole("button", { name: "Success" }).querySelector("svg")).toHaveClass("text-success")
    expect(screen.getByRole("button", { name: "Muted" }).querySelector("svg")).toHaveClass("text-muted-foreground")
  })

  it("highlights the selected count when pages pass a translated string label", () => {
    render(
      React.createElement(SelectedRowActionBar, {
        selectedCount: 3,
        ariaLabel: "3 selected",
        countLabel: "3 selected",
        actions: [{
          key: "delete",
          label: "Delete",
          onClick: () => {},
        }],
      })
    )

    expect(screen.getByText("3")).toHaveClass("text-highlight")
    expect(screen.getByText("selected")).toBeInTheDocument()
  })

  it("keeps disabled actions visible and exposes their explanation", () => {
    render(
      React.createElement(SelectedRowActionBar, {
        selectedCount: 2,
        ariaLabel: "2 selected",
        countLabel: "2 selected",
        actions: [{
          key: "stop",
          label: "Stop",
          disabled: true,
          disabledReason: "All selected scans are terminal.",
          onClick: () => {},
        }],
      })
    )

    const action = screen.getByRole("button", { name: "Stop" })
    expect(action).toBeDisabled()
    expect(screen.getByTitle("All selected scans are terminal.")).toBeInTheDocument()
  })
})
