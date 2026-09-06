import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-overview-state.ts"), "utf8")

describe("scan-overview-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScanOverviewState")
    expect(source).toContain("from \"react\"")
  })

  it("lets runtime detail screens disable log polling when the logs tab is not active", () => {
    expect(source).toContain("logsEnabled?: boolean")
    expect(source).toContain("const resolvedLogsEnabled = logsEnabled ?? activeTab === \"logs\"")
    expect(source).toContain("const hasRuntimeLogs = React.useMemo(")
    expect(source).toContain("enabled: Boolean(scan && resolvedLogsEnabled && hasRuntimeLogs)")
  })

  it("keeps active runtime detail polling scoped to the drawer lifecycle", () => {
    expect(source).toContain("refreshEnabled?: boolean")
    expect(source).toContain("SCAN_RUNTIME_DETAIL_REFRESH_MS")
    expect(source).toContain("window.setInterval")
    expect(source).toContain("if (isFetching) return")
  })

  it("links subdomain summary cards through the canonical plural child route", () => {
    expect(source).toContain("`/scan/history/${scanId}/subdomains/`")
    expect(source).not.toContain("`/scan/history/${scanId}/subdomain/`")
  })

  it("uses canonical scan record fields instead of legacy summary aliases", () => {
    expect(source).not.toContain("ScanRecordWithLegacy")
    expect(source).not.toContain("LegacyScanSummary")
    expect(source).toContain("stats?.subdomainsCount ?? 0")
    expect(source).toContain("scan?.stoppedAt")
  })
})
