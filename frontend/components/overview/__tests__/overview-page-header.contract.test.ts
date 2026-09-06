import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/overview-page-header.tsx"), "utf8")

describe("overview-page-header contract", () => {
  it("renders the overview page header through the shared header component", () => {
    expect(source).toContain("export function OverviewPageHeader")
    expect(source).toContain("PageHeader")
    expect(source).toContain('useTranslations("overview.pageHeader")')
    expect(source).toContain('code="overview"')
    expect(source).toContain('title={t("systemOverview")}')
    expect(source).toContain("middle={(")
    expect(source).toContain('t("updatedAtLabel")')
    expect(source).toContain("PageRefreshStatusButton")
    expect(source).toContain('display="icon-only"')
    expect(source).not.toContain("textRole.metadataLabel")
    expect(source).not.toContain("textRole.metadataValue")
    expect(source).not.toContain("Badge")
    expect(source).not.toContain("border-b")
    expect(source).not.toContain('description={t("realtimeSummary")}')
  })

  it("uses existing command components for the dashboard entry actions", () => {
    expect(source).toContain("QuickScanDialog")
    expect(source).toContain("AddTargetDialog")
    expect(source).toContain('import { Button } from "@/components/ui/button"')
    expect(source).toContain('t("quickScan")')
    expect(source).toContain('t("addTarget")')
    expect(source).toContain('variant="outline"')
    expect(source).toContain('variant="primary"')
    expect(source).not.toContain("PageActionButton")
    expect(source).not.toContain("DropdownMenu")
  })

  it("owns page-level refresh feedback without reading a source timestamp", () => {
    expect(source).toContain('t("refresh")')
    expect(source).toContain("lastRefreshedAt={lastRefreshedAt}")
    expect(source).not.toContain("useRefreshFeedback")
    expect(source).not.toContain("Intl.DateTimeFormat")
    expect(source).not.toContain("useAssetStatistics")
    expect(source).not.toContain("window.location.reload")
  })

  it("does not retain Bauhaus runtime branching", () => {
    expect(source).not.toContain("bauhaus")
    expect(source).not.toContain("useColorTheme")
    expect(source).not.toContain("setInterval")
  })
})
