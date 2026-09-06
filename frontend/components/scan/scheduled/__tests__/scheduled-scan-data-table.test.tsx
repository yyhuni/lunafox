import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { fireEvent, screen, within } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import {
  ScheduledScanDataTable,
  type ScheduledScanQuickFilter,
} from "@/components/scan/scheduled/scheduled-scan-data-table"
import { Checkbox } from "@/components/ui/checkbox"
import { renderWithProviders } from "@/test/utils/render-with-providers"
import type { ScheduledScan } from "@/types/scheduled-scan.types"

const columns: ColumnDef<ScheduledScan>[] = [
  {
    accessorKey: "name",
    header: "任务名称",
    meta: { title: "任务名称" },
  },
]

const quickFilterLabels: Record<ScheduledScanQuickFilter, string> = {
  all: "全部",
  enabled: "已启用",
  paused: "已暂停",
}

const scheduledScan: ScheduledScan = {
  id: 1,
  name: "Daily scan",
  displayName: "Daily scan",
  scanWorkflow: "scanWorkflows/default",
  organizationId: 1,
  organizationName: "Acme",
  targetId: null,
  targetName: null,
  scanMode: "organization",
  inputSource: "scanSnapshot",
  cronExpression: "0 2 * * *",
  isEnabled: true,
  nextRunTime: "2026-08-16T02:00:00Z",
  lastRunTime: null,
  runCount: 0,
  successfulHandoffCount: 0,
  failedHandoffCount: 0,
  createdAt: "2026-08-15T00:00:00Z",
  updatedAt: "2026-08-15T00:00:00Z",
}

describe("ScheduledScanDataTable", () => {
  it("renders quick filter counts next to the scheduled scan tab labels", () => {
    renderWithProviders(
      <ScheduledScanDataTable
        data={[]}
        columns={columns}
        searchPlaceholder="搜索定时扫描"
        addButtonText="新建定时扫描"
        quickFilterLabels={quickFilterLabels}
        quickFilterCounts={{
          all: 4,
          enabled: 3,
          paused: 1,
        }}
      />
    )

    const tablist = screen.getByRole("tablist")

    expect(within(tablist).getByRole("tab", { name: "全部 4" })).toBeInTheDocument()
    expect(within(tablist).getByRole("tab", { name: "已启用 3" })).toBeInTheDocument()
    expect(within(tablist).getByRole("tab", { name: "已暂停 1" })).toBeInTheDocument()
  })

  it("does not render a duplicate filter dropdown when quick filter tabs are visible", () => {
    renderWithProviders(
      <ScheduledScanDataTable
        data={[]}
        columns={columns}
        searchPlaceholder="搜索定时扫描"
        addButtonText="新建定时扫描"
        quickFilterLabels={quickFilterLabels}
        quickFilterCounts={{
          all: 4,
          enabled: 3,
          paused: 1,
        }}
      />
    )

    expect(screen.queryByRole("button", { name: "Filter" })).not.toBeInTheDocument()
  })

  it("renders initial toolbar chrome without actionable controls", () => {
    const { container } = renderWithProviders(
      <ScheduledScanDataTable
        data={[]}
        columns={columns}
        searchPlaceholder="搜索定时扫描"
        addButtonText="新建定时扫描"
        quickFilterLabels={quickFilterLabels}
        quickFilterCounts={{ all: 4, enabled: 3, paused: 1 }}
        loading
        initialLoading
        loadingRowCount={2}
      />
    )

    const toolbar = container.querySelector<HTMLElement>('[data-slot="scheduled-scan-table-toolbar"]')
    const table = container.querySelector<HTMLElement>('[data-slot="unified-data-table"]')

    expect(toolbar).toHaveAttribute("data-loading-presentation", "initial")
    expect(toolbar).toHaveAttribute("inert")
    expect(table).toHaveAttribute("data-loading-presentation", "initial")
    expect(toolbar?.querySelector('[data-slot="search-toolbar-skeleton"]')).toBeInTheDocument()
    expect(toolbar?.querySelector('[data-slot="action-skeleton"]')).toBeInTheDocument()
    for (const tab of within(toolbar as HTMLElement).getAllByRole("tab")) {
      expect(tab).toHaveAttribute("aria-disabled", "true")
    }
    expect(within(toolbar as HTMLElement).queryByRole("button", { name: "新建定时扫描" })).not.toBeInTheDocument()
  })

  it("keeps the empty scheduled scan state compact instead of rendering spacer rows", () => {
    const { container } = renderWithProviders(
      <ScheduledScanDataTable
        data={[]}
        columns={columns}
        searchPlaceholder="搜索定时扫描"
        addButtonText="新建定时扫描"
        page={1}
        pageSize={10}
        total={0}
        totalPages={1}
        showQuickFilters={false}
      />
    )

    expect(screen.getByText("noData")).toBeInTheDocument()
    expect(container.querySelectorAll('[data-slot="data-table-spacer-row"]')).toHaveLength(0)
  })

  it("uses adjacent cursor actions without numbered page controls", () => {
    const onPageChange = vi.fn()

    renderWithProviders(
      <ScheduledScanDataTable
        data={[scheduledScan]}
        columns={columns}
        page={2}
        pageSize={10}
        total={21}
        cursorPaginationSummary={{ total: 21 }}
        paginationNavigation={{
          mode: "cursor",
          canFirstPage: true,
          canPreviousPage: true,
          canNextPage: true,
        }}
        onPageChange={onPageChange}
        showQuickFilters={false}
        enableRowSelection={false}
      />
    )

    expect(screen.getByRole("button", { name: "first" })).toBeEnabled()
    expect(screen.queryByRole("button", { name: "last" })).not.toBeInTheDocument()
    expect(screen.queryByText(/page:\{/)).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "next" }))
    expect(onPageChange).toHaveBeenCalledWith(3)
  })

  it("activates a scheduled scan row by pointer and keyboard", () => {
    const onRowClick = vi.fn()

    renderWithProviders(
      <ScheduledScanDataTable
        data={[scheduledScan]}
        columns={columns}
        onRowClick={onRowClick}
        showQuickFilters={false}
        enableRowSelection={false}
      />
    )

    const row = screen.getByText("Daily scan").closest("tr")
    expect(row).not.toBeNull()

    fireEvent.click(screen.getByText("Daily scan"))
    fireEvent.focus(row!)
    fireEvent.keyDown(row!, { key: "Enter" })
    fireEvent.keyDown(row!, { key: " " })

    expect(onRowClick).toHaveBeenCalledTimes(3)
    expect(onRowClick).toHaveBeenLastCalledWith(scheduledScan)
  })

  it("keeps interactive row controls isolated from row activation", () => {
    const onRowClick = vi.fn()
    const interactiveColumns: ColumnDef<ScheduledScan>[] = [
      ...columns,
      {
        id: "selection",
        header: "选择",
        cell: () => <Checkbox aria-label="选择任务" />,
        meta: { title: "选择" },
      },
      {
        id: "actions",
        header: "操作",
        cell: () => <button type="button">任务操作</button>,
        meta: { title: "操作" },
      },
    ]

    renderWithProviders(
      <ScheduledScanDataTable
        data={[scheduledScan]}
        columns={interactiveColumns}
        onRowClick={onRowClick}
        showQuickFilters={false}
        enableRowSelection={false}
      />
    )

    fireEvent.click(screen.getByRole("checkbox", { name: "选择任务" }))
    fireEvent.click(screen.getByRole("button", { name: "任务操作" }))

    expect(onRowClick).not.toHaveBeenCalled()
  })


  it("renders a caller-owned selected-row status action", () => {
    const onEnable = vi.fn()
    const selectionColumns: ColumnDef<ScheduledScan>[] = [
      {
        id: "select",
        header: ({ table }) => (
          <Checkbox
            aria-label="全选任务"
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(Boolean(value))}
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            aria-label="选择任务"
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(Boolean(value))}
          />
        ),
      },
      ...columns,
    ]

    renderWithProviders(
      <ScheduledScanDataTable
        data={[scheduledScan]}
        columns={selectionColumns}
        selectedRowActions={[{
          key: "enable",
          label: "批量启用",
          onClick: onEnable,
        }]}
        showQuickFilters={false}
      />
    )

    fireEvent.click(screen.getByRole("checkbox", { name: "选择任务" }))
    fireEvent.click(screen.getByRole("button", { name: "批量启用" }))

    expect(onEnable).toHaveBeenCalledTimes(1)
  })
})
