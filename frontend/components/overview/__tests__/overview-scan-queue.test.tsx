import { cloneElement, isValidElement, type ReactElement, type ReactNode } from "react"
import { render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { OverviewScanQueue } from "@/components/overview/overview-runtime-details"
import type { GetScansResponse, ScanListRecord } from "@/types/scan.types"

const hookMocks = vi.hoisted(() => ({
  scans: vi.fn(),
  statistics: vi.fn(),
}))

vi.mock("@/hooks/use-scans", () => ({
  useScans: hookMocks.scans,
  useScanStatistics: hookMocks.statistics,
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh-CN",
  useTranslations: () => (key: string) => key,
}))

vi.mock("next/link", () => ({
  default: ({ href, children, ...props }: { href: string; children: ReactNode }) => <a href={href} {...props}>{children}</a>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, render: renderElement }: { children: ReactNode; render?: ReactNode }) => (
    isValidElement(renderElement)
      ? cloneElement(renderElement as ReactElement<{ children?: ReactNode }>, undefined, children)
      : <button>{children}</button>
  ),
}))

vi.mock("@/components/ui/scroll-area", () => ({
  ScrollArea: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
  TooltipContent: ({ children }: { children: ReactNode }) => <>{children}</>,
  TooltipTrigger: ({ render: renderElement }: { render: ReactNode }) => <>{renderElement}</>,
}))

const statistics = {
  total: 5,
  pending: 1,
  running: 1,
  succeeded: 1,
  failed: 1,
  cancelled: 1,
  totalVulns: 0,
  totalSubdomains: 0,
  totalEndpoints: 0,
  totalWebsites: 0,
  totalAssets: 0,
  retentionPolicy: { minimumRetentionSeconds: 0, automaticCleanupEnabled: false },
}

const recentScan: ScanListRecord = {
  id: 42,
  targetId: 7,
  target: { id: 7, name: "targets/7", displayName: "latest.example.com", type: "domain" },
  plannedEngineIds: [],
  triggerType: "manual",
  inputSource: "scanSnapshot",
  createdAt: "2026-08-07T08:00:00Z",
  status: "succeeded",
  progress: 100,
}

function scanResponse(results: ScanListRecord[]): GetScansResponse {
  return { results, total: results.length, page: 1, pageSize: 6, totalPages: 1 }
}

function queryState<T>(data: T, overrides: Record<string, unknown> = {}) {
  return { data, isError: false, isLoading: false, ...overrides }
}

describe("OverviewScanQueue", () => {
  beforeEach(() => {
    hookMocks.statistics.mockReturnValue(queryState(statistics))
    hookMocks.scans.mockReturnValue(queryState(scanResponse([recentScan])))
  })

  it("uses the scan-history default ordering and exposes terminal recent scans", () => {
    render(<OverviewScanQueue />)

    expect(hookMocks.scans).toHaveBeenCalledWith({ page: 1, pageSize: 6, orderBy: "createdAt desc" })
    expect(screen.getByRole("heading", { name: "scans.recentTitle" })).toBeInTheDocument()
    expect(screen.queryByText("recentTitle")).not.toBeInTheDocument()
    expect(screen.getByText("latest.example.com")).toBeInTheDocument()
    expect(screen.getByRole("img", { name: "succeeded" })).toBeInTheDocument()
    expect(screen.queryByText("succeeded")).not.toBeInTheDocument()
    expect(screen.getByRole("link", { name: "viewRecent" })).toHaveAttribute("href", "/scan/history/42/overview/")
  })

  it.each([
    ["loading", queryState(undefined, { isLoading: true }), "recentLoading"],
    ["unavailable", queryState(undefined, { isError: true }), "recentUnavailable"],
    ["empty", queryState(scanResponse([])), "noRecent"],
  ])("keeps statistics visible while the recent list is %s", (_state, recentQuery, expectedMessage) => {
    hookMocks.scans.mockReturnValue(recentQuery)

    render(<OverviewScanQueue />)

    expect(screen.getByText(expectedMessage)).toBeInTheDocument()
    expect(screen.getAllByText("1")).toHaveLength(5)
    expect(screen.queryByText("noActive")).not.toBeInTheDocument()
  })
})
