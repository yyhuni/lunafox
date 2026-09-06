import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const pageSource = readFileSync(path.resolve(process.cwd(), "app/scan/history/page.tsx"), "utf8")
const refreshSource = readFileSync(
  path.resolve(process.cwd(), "components/scan/history/scan-history-page-refresh.tsx"),
  "utf8"
)
const refreshHookSource = readFileSync(
  path.resolve(process.cwd(), "hooks/use-scan-history-refresh.ts"),
  "utf8"
)

describe("scan-history page refresh contract", () => {
  it("places the sole refresh action in the shared page-header description row", () => {
    expect(pageSource).toContain("ScanHistoryPageRefresh")
    expect(pageSource).toContain("descriptionAction={<ScanHistoryPageRefresh />}")
    expect(refreshSource).toContain("useScanHistoryRefresh")
    expect(refreshSource).toContain("PageRefreshStatusButton")
    expect(refreshSource).toContain('t("updatedAtLabel")')
    expect(refreshSource).toContain('t("updatedAtUnavailable")')
    expect(refreshSource).toContain("lastRefreshedAt")
    expect(refreshSource).not.toContain("useRefreshFeedback")
    expect(refreshSource).not.toContain("Intl.DateTimeFormat")
    expect(refreshSource).not.toContain("window.location.reload")
    expect(refreshSource).not.toContain("setInterval")
    expect(refreshHookSource).toContain("SCAN_HISTORY_AUTO_REFRESH_MS")
    expect(refreshHookSource).toContain("document.visibilityState")
    expect(refreshHookSource).toContain("window.setInterval")
    expect(refreshHookSource).toContain("scanStatistics?.pending")
    expect(refreshHookSource).toContain("scanStatistics?.running")
  })
})
