import type { ColumnDef } from "@tanstack/react-table"
import { act, renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { useTableState } from "@/components/ui/data-table/use-table-state"

type Row = {
  id: number
  name: string
}

const columns: ColumnDef<Row>[] = [
  {
    accessorKey: "name",
    header: "Name",
    meta: { title: "Name" },
  },
]

const rows: Row[] = [
  { id: 1, name: "alpha" },
  { id: 2, name: "beta" },
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
})
