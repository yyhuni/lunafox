import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/common/page-refresh-status-button.tsx"),
  "utf8"
)

describe("page refresh status button contract", () => {
  it("owns the shared page refresh presentation without business query knowledge", () => {
    expect(source).toContain("export function PageRefreshStatusButton")
    expect(source).toContain("useRefreshFeedback")
    expect(source).toContain("RefreshSpinner")
    expect(source).toContain('variant="ghost"')
    expect(source).toContain('display?: "full" | "icon-only"')
    expect(source).toContain("aria-busy={isRefreshFeedbackVisible}")
    expect(source).not.toContain("useQueryClient")
    expect(source).not.toContain("scanKeys")
    expect(source).not.toContain("agentKeys")
    expect(source).not.toContain("overviewKeys")
  })

  it("owns the fixed local timestamp format", () => {
    expect(source).toContain("export function formatPageRefreshTimestamp")
    expect(source).toContain('String(value).padStart(2, "0")')
    expect(source).toContain('].join("/") + " " + [')
    expect(source).toContain('].join(":")')
    expect(source).not.toContain("Intl.DateTimeFormat")
    expect(source).not.toContain("dateStyle")
  })
})
