import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { fireEvent, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { renderWithProviders } from "@/test/utils/render-with-providers"

type Row = {
  id: number
  name: string
}

describe("UnifiedDataTable", () => {
  it("使用 grouped props 能正常渲染数据行", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        ui={{ hideToolbar: true, hidePagination: true }}
        behavior={{ enableRowSelection: false }}
      />
    )

    expect(screen.getByText("Name")).toBeInTheDocument()
    expect(screen.getByText("alpha")).toBeInTheDocument()
  })

  it("grouped 分页/搜索/选择配置可协同工作", () => {
    const onSearch = vi.fn()
    const setPagination = vi.fn()
    const onPaginationChange = vi.fn()
    const onRowSelectionChange = vi.fn()

    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    renderWithProviders(
      <UnifiedDataTable<Row>
        data={[
          { id: 1, name: "alpha" },
          { id: 2, name: "beta" },
        ]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          searchValue: "",
          pagination: { pageIndex: 1, pageSize: 5 },
          setPagination,
          onPaginationChange,
          paginationInfo: { total: 20, totalPages: 4, page: 2, pageSize: 5 },
          rowSelection: { "1": true },
          onRowSelectionChange,
        }}
        ui={{
          searchPlaceholder: "Search assets",
          hideToolbar: false,
          hidePagination: false,
        }}
        behavior={{
          enableRowSelection: true,
          onSearch,
        }}
      />
    )

    const searchInput = screen.getByPlaceholderText("Search assets")
    fireEvent.change(searchInput, { target: { value: "alpha" } })
    fireEvent.keyDown(searchInput, { key: "Enter" })
    const nextPageLabel = screen.getByText("next")
    const nextPageButton = nextPageLabel.closest("button")
    expect(nextPageButton).not.toBeNull()
    fireEvent.click(nextPageButton!)

    expect(onSearch).toHaveBeenCalledWith("alpha")
    expect(onPaginationChange).toHaveBeenCalledWith({ pageIndex: 2, pageSize: 5 })
    expect(screen.getByText(/page:\{"current":2,"total":4\}/)).toBeInTheDocument()
    expect(screen.getByText(/selected:\{"count":1\}/)).toBeInTheDocument()
  })
})
