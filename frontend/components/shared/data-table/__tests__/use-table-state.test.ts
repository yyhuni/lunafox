import type { ColumnDef } from "@tanstack/react-table"
import { act, renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { useTableState } from "@/components/shared/data-table/use-table-state"
import { getBadgeMinWidthPx, getColumnHeaderMinWidthPx } from "@/lib/table-utils"

type Row = {
  id: number
  name: string
}

type NarrowHeaderRow = {
  id: number
  vhost: boolean
  status: number
  webserver: string
}

type BadgeRow = {
  id: number
  contentType: string
}

const KNOWN_BADGE_LABEL = "application/javascript"

const columns: ColumnDef<Row>[] = [
  {
    accessorKey: "name",
    header: "Name",
    meta: { title: "Name" },
  },
]

const serverSortableColumns: ColumnDef<Row>[] = [
  {
    accessorKey: "name",
    header: "Name",
    meta: {
      title: "Name",
      orderBy: "displayName",
      serverSortPerformance: "covered by wordlist name btree index",
    },
  },
]

const rows: Row[] = [
  { id: 1, name: "alpha" },
  { id: 2, name: "beta" },
]

const unsortedRows: Row[] = [
  { id: 1, name: "beta" },
  { id: 2, name: "alpha" },
]

const narrowHeaderColumns: ColumnDef<NarrowHeaderRow>[] = [
  {
    accessorKey: "vhost",
    header: "虚拟主机",
    meta: { title: "虚拟主机" },
    size: 66,
    minSize: 60,
    maxSize: 88,
  },
  {
    accessorKey: "status",
    header: "状态",
    meta: { title: "状态" },
    size: 72,
    minSize: 64,
    maxSize: 90,
  },
  {
    accessorKey: "webserver",
    header: "Web 服务器",
    meta: { title: "Web 服务器" },
    size: 100,
    minSize: 88,
    maxSize: 160,
  },
]

const singleBadgeColumns: ColumnDef<BadgeRow>[] = [
  {
    accessorKey: "contentType",
    header: "内容类型",
    meta: {
      title: "内容类型",
      singleBadge: true,
      singleBadgeValues: [KNOWN_BADGE_LABEL],
    },
    size: 96,
    minSize: 80,
    maxSize: 220,
  },
]

describe("useTableState", () => {
  it("外部分页受控时会通过 onPaginationChange 回传更新", () => {
    const onPaginationChange = vi.fn()
    const setPagination = vi.fn()

    const { result } = renderHook(() =>
      useTableState({
        data: rows,
        columns,
        pagination: { pageIndex: 0, pageSize: 10 },
        onPaginationChange,
        setPagination,
      })
    )

    act(() => {
      result.current.table.setPageIndex(1)
    })

    expect(onPaginationChange).toHaveBeenCalledWith({ pageIndex: 1, pageSize: 10 })
    expect(setPagination).not.toHaveBeenCalled()
  })

  it("外部选择受控时会通过 onRowSelectionChange 回传选择状态", () => {
    const onRowSelectionChange = vi.fn()

    const { result } = renderHook(() =>
      useTableState({
        data: rows,
        columns,
        rowSelection: {},
        onRowSelectionChange,
      })
    )

    act(() => {
      result.current.table.getRowModel().rows[0]?.toggleSelected(true)
    })

    expect(onRowSelectionChange).toHaveBeenCalled()
    const selection = onRowSelectionChange.mock.calls.at(-1)?.[0]
    expect(selection).toMatchObject({ "1": true })
  })

  it("会把可排序列头标题的最小宽度折算进共享列宽下限", () => {
    const { result } = renderHook(() =>
      useTableState({
        data: [{ id: 1, vhost: false, status: 200, webserver: "nginx/1.24.0" }],
        columns: narrowHeaderColumns,
      })
    )

    const vhostHeader = result.current.table.getFlatHeaders().find((header) => header.column.id === "vhost")
    const statusHeader = result.current.table.getFlatHeaders().find((header) => header.column.id === "status")
    const webserverHeader = result.current.table.getFlatHeaders().find((header) => header.column.id === "webserver")

    expect(vhostHeader?.getSize()).toBeGreaterThan(66)
    expect(statusHeader?.getSize()).toBeGreaterThan(72)
    expect(webserverHeader?.getSize()).toBeGreaterThan(100)
  })

  it("会把中文状态码列头的完整排序触发器宽度折算进共享列宽下限", () => {
    const columns: ColumnDef<NarrowHeaderRow>[] = [
      {
        accessorKey: "status",
        header: "状态码",
        meta: { title: "状态码" },
        size: 72,
        minSize: 64,
        maxSize: 90,
      },
    ]

    const { result } = renderHook(() =>
      useTableState({
        data: [{ id: 1, vhost: false, status: 200, webserver: "nginx/1.24.0" }],
        columns,
      })
    )

    const statusHeader = result.current.table.getFlatHeaders().find((header) => header.column.id === "status")

    expect(statusHeader?.getSize()).toBeGreaterThan(90)
    expect(statusHeader?.getSize()).toBeGreaterThanOrEqual(getColumnHeaderMinWidthPx("状态码"))
  })

  it("会把单 badge 列的最长标签宽度折算进共享列宽下限", () => {
    const { result } = renderHook(() =>
      useTableState({
        data: [{ id: 1, contentType: KNOWN_BADGE_LABEL }],
        columns: singleBadgeColumns,
      })
    )

    const header = result.current.table.getFlatHeaders().find((item) => item.column.id === "contentType")

    expect(header?.getSize()).toBeGreaterThanOrEqual(getBadgeMinWidthPx(KNOWN_BADGE_LABEL))
  })

  it("会在初始空数据时保留单 badge 列的已知标签宽度", () => {
    const { result } = renderHook(() =>
      useTableState({
        data: [],
        columns: singleBadgeColumns,
      })
    )

    const header = result.current.table.getFlatHeaders().find((item) => item.column.id === "contentType")

    expect(header?.getSize()).toBeGreaterThanOrEqual(getBadgeMinWidthPx(KNOWN_BADGE_LABEL))
  })

  it("server sorting 模式不会重排后端已经返回的一页数据", () => {
    const { result } = renderHook(() =>
      useTableState({
        data: unsortedRows,
        columns: serverSortableColumns,
        sortingMode: "server",
        sorting: [{ id: "name", desc: false }],
        paginationInfo: { total: 2, totalPages: 1 },
      })
    )

    expect(result.current.table.getRowModel().rows.map((row) => row.original.name)).toEqual(["beta", "alpha"])
  })

  it("server sorting 模式只允许声明了后端 orderBy 和性能依据的列排序", () => {
    const { result: missingMeta } = renderHook(() =>
      useTableState({
        data: unsortedRows,
        columns,
        sortingMode: "server",
      })
    )
    expect(missingMeta.current.table.getColumn("name")?.getCanSort()).toBe(false)

    const { result: withMeta } = renderHook(() =>
      useTableState({
        data: unsortedRows,
        columns: serverSortableColumns,
        sortingMode: "server",
      })
    )
    expect(withMeta.current.table.getColumn("name")?.getCanSort()).toBe(true)
  })

  it("none sorting 模式会关闭所有列排序能力", () => {
    const { result } = renderHook(() =>
      useTableState({
        data: unsortedRows,
        columns,
        sortingMode: "none",
      })
    )

    expect(result.current.table.getAllLeafColumns().every((column) => !column.getCanSort())).toBe(true)
  })

  it("后端分页列表未声明排序模式时默认关闭当前页排序", () => {
    const { result } = renderHook(() =>
      useTableState({
        data: unsortedRows,
        columns,
        paginationInfo: { total: 2, totalPages: 1 },
      })
    )

    expect(result.current.table.getAllLeafColumns().every((column) => !column.getCanSort())).toBe(true)
    expect(result.current.table.getRowModel().rows.map((row) => row.original.name)).toEqual(["beta", "alpha"])
  })
})
