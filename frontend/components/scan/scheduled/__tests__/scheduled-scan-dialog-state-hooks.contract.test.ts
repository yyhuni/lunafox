import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-dialog-state-hooks.ts"), "utf8")

describe("scheduled-scan-dialog-state-hooks contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScheduledScanSearch")
    expect(source).toContain("from \"react\"")
  })

  it("keeps target search and pagination ownership in the scheduled-scan caller", () => {
    expect(source).toContain("const [targetPage, setTargetPageState] = React.useState(1)")
    expect(source).toContain("const [targetPageSize, setTargetPageSizeState] = React.useState(10)")
    expect(source).toContain("pageToken: targetPageTokens[targetPage]")
    expect(source).toContain("setTargetSearch(value)")
    expect(source).toContain("resetTargetPaging()")
    expect(source).toContain("targetPaginationNavigation")
    expect(source).toContain("getCursorPageTransition")
    expect(source).toContain("onPreviousTargetPage")
    expect(source).toContain("onNextTargetPage")
  })
})
