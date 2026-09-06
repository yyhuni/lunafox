import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/table-actions.tsx"), "utf8")

describe("table-actions contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TableActions")
    expect(source).toContain("className")
    expect(source).toContain('from "@/components/shared/loading/use-refresh-feedback"')
  })

  it("supports standard page-level refresh without selected-row bulk actions", () => {
    expect(source).toContain("onRefresh")
    expect(source).toContain("refreshLabel")
    expect(source).toContain("IconRefresh")
    expect(source).toContain("isRefreshing")
    expect(source).toContain("useRefreshFeedback")
    expect(source).toContain("RefreshSpinner")
    expect(source).toContain('const refreshButtonSize = toolbarDensity === "standard" ? "icon" : "icon-sm"')
    expect(source).toContain("size={refreshButtonSize}")
    expect(source).toContain("aria-busy={isRefreshFeedbackVisible}")
    expect(source).not.toContain("loadingIndicator")
    expect(source).not.toContain("loadingLabel")
    expect(source).not.toContain("onBulkDelete")
    expect(source).not.toContain("bulkDeleteLabel")
    expect(source).not.toContain("deleteConfirmation")
    expect(source).not.toContain("AlertDialog")
  })

  it("lets page-level data-table toolbars use standard button density", () => {
    expect(source).toContain("toolbarDensity")
    expect(source).toContain('toolbarDensity = "compact"')
    expect(source).toContain('toolbarDensity === "standard" ? "default" : "sm"')
    expect(source).toContain("size={actionButtonSize}")
  })

  it("treats keyboard focus as the same bounded add-panel preload intent as hover", () => {
    expect(source).toContain("onMouseEnter={onAddHover}")
    expect(source).toContain("onFocus={onAddHover}")
  })
})
