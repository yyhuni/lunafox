import { waitFor } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { ScanHistoryDataTable } from "@/components/scan/history/scan-history-data-table"
import { createScanHistoryColumns } from "@/components/scan/history/scan-history-columns"
import { getMockScans } from "@/mock/data/scans"
import { renderWithProviders } from "@/test/utils/render-with-providers"
import type { ScanRecord } from "@/types/scan.types"

describe("ScanHistoryDataTable engine display", () => {
  it("renders resolved executed-engine labels through the shared table state", async () => {
    const scan = getMockScans({ pageSize: 1 }).results[0] as ScanRecord
    const columns = createScanHistoryColumns({
      formatDate: (value) => value,
      handleDelete: () => {},
      handleStop: () => {},
      t: {
        columns: {
          target: "Target",
          summary: "Summary",
          executedEngines: "Engines",
          triggerType: "Trigger",
          createdAt: "Created",
          status: "Status",
          progress: "Progress",
        },
        actions: {
          scanDetail: "Detail",
          runtimeDetail: "Runtime",
          openMenu: "Menu",
          stop: "Stop",
          stopScanPending: "Stopping",
          delete: "Delete",
          selectAll: "Select all",
          selectRow: "Select row",
        },
        tooltips: { viewProgress: "Progress" },
        status: {
          cancelled: "Cancelled",
          succeeded: "Succeeded",
          failed: "Failed",
          pending: "Pending",
          running: "Running",
        },
        summary: {
          subdomains: "Subdomains",
          websites: "Websites",
          ipAddresses: "IPs",
          endpoints: "Endpoints",
          vulnerabilities: "Vulnerabilities",
        },
        triggerTypes: { manual: "Manual", scheduled: "Scheduled", ai: "AI" },
      },
      executedEngineNamesByScanId: new Map([[scan.id, ["Subdomain Discovery", "Port Scan"]]]),
      executedEngineDescriptionsByScanId: new Map([[
        scan.id,
        ["Discover subdomains through reconnaissance, optional dictionary brute force, and DNS resolution.", "Identify open ports on the target surface."],
      ]]),
    })

    const { container } = renderWithProviders(
      <ScanHistoryDataTable
        data={[scan]}
        columns={columns}
        hideToolbar
        hidePagination
      />
    )

    await waitFor(() => {
      expect(container.querySelectorAll('[data-scan-engine-badges] [data-badge-type="engine"]')).toHaveLength(2)
    })

    expect(container.querySelector('[data-scan-engine-badges]')?.textContent).toContain("Subdomain Discovery")
    expect(container.querySelector('[data-scan-engine-badges]')?.textContent).toContain("Port Scan")

    const engineBadge = container.querySelector('[data-scan-engine-badges] [data-badge-type="engine"]')!
    expect(engineBadge.closest('[data-slot="tooltip-trigger"]')).not.toBeNull()
    expect(engineBadge.closest('[data-slot="tooltip-trigger"]')).toHaveAttribute("tabindex", "0")
  })

  it("refreshes the engine column when catalog labels resolve after the first render", async () => {
    const scan = getMockScans({ pageSize: 1 }).results[0] as ScanRecord
    const buildColumns = (names: string[], descriptions: string[]) => createScanHistoryColumns({
      formatDate: (value) => value,
      handleDelete: () => {},
      handleStop: () => {},
      t: {
        columns: {
          target: "Target",
          summary: "Summary",
          executedEngines: "Engines",
          triggerType: "Trigger",
          createdAt: "Created",
          status: "Status",
          progress: "Progress",
        },
        actions: {
          scanDetail: "Detail",
          runtimeDetail: "Runtime",
          openMenu: "Menu",
          stop: "Stop",
          stopScanPending: "Stopping",
          delete: "Delete",
          selectAll: "Select all",
          selectRow: "Select row",
        },
        tooltips: { viewProgress: "Progress" },
        status: {
          cancelled: "Cancelled",
          succeeded: "Succeeded",
          failed: "Failed",
          pending: "Pending",
          running: "Running",
        },
        summary: {
          subdomains: "Subdomains",
          websites: "Websites",
          ipAddresses: "IPs",
          endpoints: "Endpoints",
          vulnerabilities: "Vulnerabilities",
        },
        triggerTypes: { manual: "Manual", scheduled: "Scheduled", ai: "AI" },
      },
      executedEngineNamesByScanId: new Map([[scan.id, names]]),
      executedEngineDescriptionsByScanId: new Map([[scan.id, descriptions]]),
    })

    const { container, rerender } = renderWithProviders(
      <ScanHistoryDataTable data={[scan]} columns={buildColumns([], [])} hideToolbar hidePagination />
    )

    rerender(
      <ScanHistoryDataTable
        data={[scan]}
        columns={buildColumns(
          ["Subdomain Discovery", "Port Scan"],
          ["Discover subdomains through reconnaissance, optional dictionary brute force, and DNS resolution.", "Identify open ports on the target surface."]
        )}
        hideToolbar
        hidePagination
      />
    )

    await waitFor(() => {
      expect(container.querySelectorAll('[data-scan-engine-badges] [data-badge-type="engine"]')).toHaveLength(2)
    })
  })
})
