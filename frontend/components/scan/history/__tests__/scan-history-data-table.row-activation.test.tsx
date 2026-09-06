import type { ColumnDef } from "@tanstack/react-table"
import { fireEvent, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { ScanHistoryDataTable } from "@/components/scan/history/scan-history-data-table"
import { getMockScans } from "@/mock/data/scans"
import { renderWithProviders } from "@/test/utils/render-with-providers"
import type { ScanRecord } from "@/types/scan.types"

describe("ScanHistoryDataTable row activation", () => {
  it("opens runtime details from the row while exempting links", () => {
    const scan = getMockScans({ pageSize: 1 }).results[0] as ScanRecord
    const onRowClick = vi.fn()
    const columns: ColumnDef<ScanRecord>[] = [
      {
        id: "target",
        header: "Target",
        accessorFn: (row) => row.target?.displayName,
        cell: () => (
          <a href={`/targets/${scan.targetId}`} onClick={(event) => event.preventDefault()}>
            {scan.target?.displayName}
          </a>
        ),
        meta: { title: "Target" },
      },
    ]

    renderWithProviders(
      <ScanHistoryDataTable
        data={[scan]}
        columns={columns}
        hideToolbar
        hidePagination
        onRowClick={onRowClick}
        rowActionLabel="Open runtime details"
      />
    )

    const row = screen.getByText(scan.target?.displayName ?? "").closest("tr")
    expect(row).not.toBeNull()
    expect(row).toHaveAttribute("tabindex", "0")
    expect(row).toHaveAttribute("aria-label", "Open runtime details")

    fireEvent.click(row!)
    expect(onRowClick).toHaveBeenCalledTimes(1)
    expect(onRowClick).toHaveBeenLastCalledWith(scan)

    fireEvent.keyDown(row!, { key: "Enter" })
    expect(onRowClick).toHaveBeenCalledTimes(2)

    fireEvent.click(screen.getByRole("link", { name: scan.target?.displayName }))
    expect(onRowClick).toHaveBeenCalledTimes(2)
  })
})
