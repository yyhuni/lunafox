import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-history-stat-cards.tsx"), "utf8")
const enMessages = JSON.parse(
  readFileSync(path.resolve(process.cwd(), "messages/en.json"), "utf8")
) as {
  scan?: {
    history?: {
      stats?: {
        pendingScans?: string
        runningScans?: string
        completedScans?: string
        failedScans?: string
        cancelledScans?: string
        pendingShort?: string
        failedNeedsAttention?: string
        cancelledShort?: string
        completedAll?: string
      }
    }
  }
}

describe("scan-history-stat-cards contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanHistoryStatCards")
    expect(source).toContain("export function ScanHistoryStatCardsLoadingState")
    expect(source).toContain("className")
    expect(source).toContain("from \"next-intl\"")
  })

  it("shares one metric-strip renderer between query loading and route loading", () => {
    expect(source).toContain("function ScanHistoryStatCardsContent")
    expect(source).toContain("<ScanHistoryStatCardsContent data={data} isLoading={isLoading} />")
    expect(source).toContain("<ScanHistoryStatCardsContent isLoading />")
    expect(source).not.toContain("ContentHandoff")
  })

  it("uses semantic status tones instead of local raw status colors", () => {
    expect(source).toContain("statTone")
    expect(source).toContain("getScanStatusMetricTone")
    expect(source).not.toContain("--scan-history-running")
    expect(source).not.toContain("#0a84ff")
    expect(source).not.toContain('label: "text-muted-foreground"')
    expect(source).not.toContain('dot: "bg-muted-foreground"')
    expect(source).not.toContain('label: "text-error"')
    expect(source).not.toContain('dot: "bg-error"')
    expect(source).not.toContain("text-[#333]")
    expect(source).not.toContain("text-blue-500")
    expect(source).not.toContain("text-amber-500")
    expect(source).not.toContain("text-emerald-500")
    expect(source).not.toContain("text-red-500")
    expect(source).not.toContain("text-slate-400")
  })

  it("renders scan history stats as a single divided strip", () => {
    expect(source).toContain("StatMetricRow")
    expect(source).toContain('from "@/components/shared/metrics/stat-metric-row"')
    expect(source).toContain("className=\"scan-history-stats-row\"")
    expect(source).toContain('dataSlot={isLoading ? "scan-history-stats-skeleton" : undefined}')
    expect(source.match(/loading:\s*isLoading/g)?.length).toBe(5)
    expect(source).toContain('featuredKey="running"')
    expect(source).not.toContain("featured: true")
    expect(source).not.toContain("ContentHandoff")
    expect(source).not.toContain("CardFooter")
    expect(source).not.toContain("Card variant=\"stat\"")
    expect(source).not.toContain("getScanStatusBadgeVariant")
    expect(source).not.toContain("@[200px]/card:p-6")
    expect(source).not.toContain("@[250px]/card:text-3xl")
  })

  it("does not pass per-card index markers into the stat cards", () => {
    expect(source).not.toContain("data-card-index")
    expect(source).not.toContain("padStart(2, \"0\")")
    expect(source).not.toContain("index={1}")
  })

  it("orders the summary metrics to match the scan history reference strip", () => {
    const runningIndex = source.indexOf('key: "running"')
    const pendingIndex = source.indexOf('key: "pending"')
    const failedIndex = source.indexOf('key: "failed"')
    const cancelledIndex = source.indexOf('key: "cancelled"')
    const succeededIndex = source.indexOf('key: "succeeded"')

    expect(runningIndex).toBeGreaterThan(-1)
    expect(runningIndex).toBeLessThan(pendingIndex)
    expect(pendingIndex).toBeLessThan(failedIndex)
    expect(failedIndex).toBeLessThan(cancelledIndex)
    expect(cancelledIndex).toBeLessThan(succeededIndex)
    expect(source).not.toContain("percentage")
    expect(source).not.toContain("data?.total")
  })

  it("keeps english scan history stat footer copy complete", () => {
    expect(enMessages.scan?.history?.stats?.pendingScans).toBe("Pending Scans")
    expect(enMessages.scan?.history?.stats?.runningScans).toBe("Running Scans")
    expect(enMessages.scan?.history?.stats?.completedScans).toBe("Completed Scans")
    expect(enMessages.scan?.history?.stats?.failedScans).toBe("Failed Scans")
    expect(enMessages.scan?.history?.stats?.cancelledScans).toBe("Cancelled Scans")
    expect(enMessages.scan?.history?.stats?.pendingShort).toBe("To process")
    expect(enMessages.scan?.history?.stats?.failedNeedsAttention).toBe("Needs attention")
    expect(enMessages.scan?.history?.stats?.cancelledShort).toBe("Cancelled")
    expect(enMessages.scan?.history?.stats?.completedAll).toBe("All complete")
  })
})
