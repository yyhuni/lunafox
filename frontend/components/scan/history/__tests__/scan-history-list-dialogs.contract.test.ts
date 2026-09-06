import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-history-list-dialogs.tsx"), "utf8")

describe("scan-history-list-dialogs contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanHistoryDialogs")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps bulk scan delete confirmation summary-only by default", () => {
    expect(source).toContain("bulkDeleteDialogOpen")
    expect(source).toContain("bulkDeleteScanMessage")
    expect(source).toContain("confirmDelete")
    expect(source).not.toContain("selectedScans.map")
  })
})
