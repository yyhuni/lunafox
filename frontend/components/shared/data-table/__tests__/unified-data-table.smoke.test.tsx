import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { fireEvent, screen, waitFor } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { UnifiedDataTable } from "@/components/shared/data-table/unified-data-table"
import { Checkbox } from "@/components/ui/checkbox"
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

  it("粘性表头和数据单元格继承各自行表面", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
      {
        id: "actions",
        header: "Actions",
        cell: () => "actions",
        meta: { title: "Actions", stickyRight: true },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        ui={{ hideToolbar: true, hidePagination: true }}
        behavior={{ enableRowSelection: false }}
      />
    )

    expect(container.querySelector('[data-slot="table-header"]')).toHaveClass("[&_tr]:bg-secondary")
    expect(container.querySelector("thead th.sticky")).toHaveClass("right-0", "z-20", "bg-inherit")
    expect(container.querySelector("tbody td.sticky")).toHaveClass("right-0", "z-10", "bg-inherit")
  })

  it("行详情激活不会拦截 role=checkbox 的选择控件", () => {
    const onRowClick = vi.fn()
    const columns: ColumnDef<Row>[] = [
      {
        id: "select",
        header: "Select",
        cell: () => <Checkbox aria-label="Select row" />,
        meta: { title: "Select" },
      },
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
        behavior={{ enableRowSelection: false, onRowClick }}
      />
    )

    fireEvent.click(screen.getByRole("checkbox", { name: "Select row" }))

    expect(onRowClick).not.toHaveBeenCalled()
  })

  it("在不激活详情的情况下发出行级预加载意图", () => {
    const onRowIntent = vi.fn()
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
        behavior={{ enableRowSelection: false, onRowIntent }}
      />
    )

    const row = screen.getByText("alpha").closest("tr")
    expect(row).not.toBeNull()

    fireEvent.pointerEnter(row!)
    fireEvent.focus(row!)

    expect(onRowIntent).toHaveBeenCalledTimes(2)
    expect(onRowIntent).toHaveBeenLastCalledWith({ id: 1, name: "alpha" })
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

  it("游标分页只公开已知可达的前后页", () => {
    const onPaginationChange = vi.fn()
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
        state={{
          pagination: { pageIndex: 0, pageSize: 5 },
          onPaginationChange,
          paginationInfo: { total: 20, totalPages: 4, page: 1, pageSize: 5 },
          paginationNavigation: {
            mode: "cursor",
            canFirstPage: false,
            canPreviousPage: false,
            canNextPage: true,
          },
        }}
        ui={{ hideToolbar: true }}
        behavior={{ enableRowSelection: false }}
      />
    )

    expect(screen.getByRole("button", { name: "first" })).toBeDisabled()
    expect(screen.queryByRole("button", { name: "last" })).not.toBeInTheDocument()
    expect(screen.queryByText(/page:\{/)).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "next" }))

    expect(onPaginationChange).toHaveBeenCalledWith({ pageIndex: 1, pageSize: 5 })
  })

  it("游标分页保留结果摘要，并忽略禁用相邻操作", () => {
    const onPaginationChange = vi.fn()
    const columns: ColumnDef<Row>[] = [{
      accessorKey: "name",
      header: "Name",
      meta: { title: "Name" },
    }]

    renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 5 },
          onPaginationChange,
          cursorPaginationSummary: { total: 37 },
          paginationNavigation: {
            mode: "cursor",
            canFirstPage: false,
            canPreviousPage: false,
            canNextPage: false,
          },
        }}
        ui={{ hideToolbar: true }}
        behavior={{ enableRowSelection: false }}
      />
    )

    const previous = screen.getByRole("button", { name: "previous" })
    const next = screen.getByRole("button", { name: "next" })

    expect(screen.getByText(/total:\{"count":37\}/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "first" })).toBeDisabled()
    expect(screen.queryByRole("button", { name: "last" })).not.toBeInTheDocument()
    expect(screen.queryByText(/page:\{/)).not.toBeInTheDocument()
    expect(previous).toBeDisabled()
    expect(next).toBeDisabled()

    fireEvent.click(previous)
    fireEvent.click(next)

    expect(onPaginationChange).not.toHaveBeenCalled()
  })

  it("将分页保留为表格边框外的自然流区域", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        ui={{ hideToolbar: true }}
        behavior={{ enableRowSelection: false }}
      />
    )

    const tableSurface = container.querySelector<HTMLDivElement>('[data-slot="data-table"]')
    const paginationSurface = container.querySelector<HTMLDivElement>('[data-slot="data-table-pagination-surface"]')
    const tableRoot = container.querySelector<HTMLDivElement>('[data-slot="unified-data-table"]')

    expect(tableSurface).not.toBeNull()
    expect(paginationSurface).not.toBeNull()
    expect(tableSurface).not.toContainElement(paginationSurface)
    expect(paginationSurface?.parentElement).toBe(tableRoot)
    expect(tableRoot).toHaveClass("min-w-0")
    expect(tableSurface).toHaveClass("overflow-x-auto", "rounded-md", "border", "bg-card")
    expect(paginationSurface).not.toHaveClass("shrink-0", "border-t", "border-border", "py-2")
  })

  it("resolved 稀疏表面保持自然表头和内容高度", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        ui={{ hideToolbar: true, stableSurfaceRowCount: 3 }}
        behavior={{ enableRowSelection: false }}
      />
    )

    const root = container.querySelector<HTMLDivElement>('[data-slot="unified-data-table"]')
    const tableSurface = container.querySelector<HTMLDivElement>('[data-slot="data-table"]')
    const tableHeader = container.querySelector<HTMLTableSectionElement>('[data-slot="table-header"]')
    const paginationSurface = container.querySelector<HTMLDivElement>('[data-slot="data-table-pagination-surface"]')

    expect(root).not.toHaveClass("md:flex", "md:min-h-0", "md:flex-1", "md:flex-col")
    expect(tableSurface).not.toHaveClass("md:flex", "md:min-h-0", "md:flex-1", "md:flex-col")
    expect(tableHeader).not.toHaveClass("sticky", "top-0", "z-20")
    expect(tableSurface?.style.minHeight).toBe("")
    expect(container.querySelectorAll("tbody tr")).toHaveLength(1)
    expect(paginationSurface?.parentElement).toBe(root)
    expect(paginationSurface).not.toHaveClass("shrink-0", "border-t")
  })

  it("将虚拟滚动保留在显式的大数据表面", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]
    const data = Array.from({ length: 101 }, (_, index) => ({
      id: index + 1,
      name: `row-${index + 1}`,
    }))

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={data}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 101 },
          paginationInfo: { total: 101, totalPages: 1, page: 1, pageSize: 101 },
        }}
        ui={{ hideToolbar: true }}
        behavior={{ enableRowSelection: false }}
      />
    )

    const tableSurface = container.querySelector<HTMLDivElement>('[data-slot="data-table"]')

    expect(tableSurface).toHaveClass("max-h-150", "overflow-y-auto")
  })

  it("兼容扩展列以明确的列宽填满剩余空间", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
      {
        accessorKey: "id",
        header: "ID",
        meta: { title: "ID" },
      },
    ]

    renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        ui={{ hideToolbar: true, hidePagination: true }}
        behavior={{ enableRowSelection: false, expandColumnIds: ["name"] }}
      />
    )

    const nameHeader = screen.getByText("Name").closest("th")
    const idHeader = screen.getByText("ID").closest("th")

    expect(nameHeader).not.toBeNull()
    expect(idHeader).not.toBeNull()
    expect(nameHeader?.style.minWidth).not.toBe("")
    expect(nameHeader?.style.width).not.toBe("")
    expect(idHeader?.style.width).not.toBe("")
  })

  it("扩展列不使用当前 size 推高 fixed layout 的表格最小宽", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        size: 320,
        minSize: 120,
        meta: { title: "Name" },
      },
      {
        accessorKey: "id",
        header: "ID",
        size: 80,
        minSize: 80,
        meta: { title: "ID" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        ui={{ hideToolbar: true, hidePagination: true }}
        behavior={{
          enableRowSelection: false,
          expandColumnIds: ["name"],
          columnLayout: "fixed",
        }}
      />
    )

    const table = container.querySelector("table")

    expect(table?.style.minWidth).toBe("200px")
  })

  it("fixed layout 使用 colgroup 让固定列保持 px 宽度，扩展列按比例占剩余空间", async () => {
    const originalResizeObserver = globalThis.ResizeObserver
    globalThis.ResizeObserver = class {
      private readonly callback: ResizeObserverCallback

      constructor(callback: ResizeObserverCallback) {
        this.callback = callback
      }

      observe() {
        this.callback([{ contentRect: { width: 500 } } as ResizeObserverEntry], this as ResizeObserver)
      }

      disconnect() {}
      unobserve() {}
    } as typeof ResizeObserver

    try {
      const columns: ColumnDef<Row>[] = [
        {
          accessorKey: "name",
          header: "Name",
          size: 240,
          minSize: 120,
          meta: { title: "Name" },
        },
        {
          accessorKey: "id",
          header: "ID",
          size: 80,
          minSize: 80,
          meta: { title: "ID" },
        },
      ]

      const { container } = renderWithProviders(
        <UnifiedDataTable<Row>
          data={[{ id: 1, name: "alpha" }]}
          columns={columns}
          getRowId={(row) => String(row.id)}
          ui={{ hideToolbar: true, hidePagination: true }}
          behavior={{
            enableRowSelection: false,
            expandColumnIds: ["name"],
            columnLayout: "fixed",
          }}
        />
      )

      const columnElements = Array.from(container.querySelectorAll("col"))

      expect(columnElements).toHaveLength(2)
      await waitFor(() => {
        expect(columnElements[0]?.style.width).toBe("420px")
      })
      expect(columnElements[1]?.style.width).toBe("80px")
    } finally {
      globalThis.ResizeObserver = originalResizeObserver
    }
  })

  it("fixed layout 运行时隐藏列后 colgroup 只保留可见列", async () => {
    function RuntimeColumnVisibilityTable() {
      const [columnVisibility, setColumnVisibility] = React.useState<Record<string, boolean>>({})
      const columns: ColumnDef<Row>[] = [
        {
          accessorKey: "name",
          header: "Name",
          size: 240,
          minSize: 120,
          meta: { title: "Name" },
        },
        {
          accessorKey: "id",
          header: "ID",
          size: 80,
          minSize: 80,
          meta: { title: "ID" },
        },
        {
          id: "extra",
          header: "Extra",
          size: 160,
          minSize: 160,
          cell: () => "extra",
          meta: { title: "Extra" },
        },
        {
          id: "actions",
          header: "Actions",
          size: 56,
          minSize: 56,
          cell: () => "actions",
          meta: { title: "Actions" },
        },
      ]

      return (
        <>
          <button type="button" onClick={() => setColumnVisibility({ extra: false, id: false })}>
            Hide columns
          </button>
          <UnifiedDataTable<Row>
            data={[{ id: 1, name: "alpha" }]}
            columns={columns}
            getRowId={(row) => String(row.id)}
            state={{ columnVisibility, onColumnVisibilityChange: setColumnVisibility }}
            ui={{ hideToolbar: true, hidePagination: true }}
            behavior={{
              enableRowSelection: false,
              expandColumnIds: ["name"],
              columnLayout: "fixed",
            }}
          />
        </>
      )
    }

    const { container } = renderWithProviders(<RuntimeColumnVisibilityTable />)

    expect(container.querySelectorAll("col")).toHaveLength(4)
    fireEvent.click(screen.getByRole("button", { name: "Hide columns" }))

    await waitFor(() => {
      expect(container.querySelectorAll("col")).toHaveLength(2)
    })
    expect(container.querySelectorAll("thead th")).toHaveLength(2)
    expect(container.querySelectorAll("tbody td")).toHaveLength(2)
  })

  it("fixed layout 初始化隐藏列后 colgroup 只保留可见列", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        size: 240,
        minSize: 120,
        meta: { title: "Name" },
      },
      {
        accessorKey: "id",
        header: "ID",
        size: 80,
        minSize: 80,
        meta: { title: "ID" },
      },
      {
        id: "extra",
        header: "Extra",
        size: 160,
        minSize: 160,
        cell: () => "extra",
        meta: { title: "Extra" },
      },
      {
        id: "actions",
        header: "Actions",
        size: 56,
        minSize: 56,
        cell: () => "actions",
        meta: { title: "Actions" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{ columnVisibility: { extra: false, id: false } }}
        ui={{ hideToolbar: true, hidePagination: true }}
        behavior={{
          enableRowSelection: false,
          expandColumnIds: ["name"],
          columnLayout: "fixed",
        }}
      />
    )

    expect(container.querySelectorAll("col")).toHaveLength(2)
    expect(container.querySelectorAll("thead th")).toHaveLength(2)
    expect(container.querySelectorAll("tbody td")).toHaveLength(2)
  })

  it("语义列宽表格可切换到共享 fixed layout 模式", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
      {
        accessorKey: "id",
        header: "ID",
        meta: { title: "ID" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        ui={{ hideToolbar: true, hidePagination: true }}
        behavior={{ enableRowSelection: false, expandColumnIds: ["name"], columnLayout: "fixed" }}
      />
    )

    const shell = container.querySelector('[data-table-variant="shell"]')
    const table = container.querySelector("table")

    expect(shell?.getAttribute("data-column-layout")).toBe("fixed")
    expect(table?.className).toContain("table-fixed")
  })

  it("分页表格数据少于 pageSize 时默认不保留稳定行槽", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 5 },
          paginationInfo: { total: 1, totalPages: 1, page: 1, pageSize: 5 },
        }}
        ui={{ hideToolbar: true }}
        behavior={{ enableRowSelection: false }}
      />
    )

    expect(container.querySelectorAll('[data-slot="data-table-spacer-row"]')).toHaveLength(0)
  })

  it("空分页表格默认不保留稳定行槽", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[]}
        columns={columns}
        state={{
          pagination: { pageIndex: 0, pageSize: 5 },
          paginationInfo: { total: 0, totalPages: 1, page: 1, pageSize: 5 },
        }}
        ui={{ hideToolbar: true }}
        behavior={{ enableRowSelection: false }}
      />
    )

    expect(screen.getByText("No results")).toBeInTheDocument()
    expect(container.querySelectorAll('[data-slot="data-table-spacer-row"]')).toHaveLength(0)
  })

  it("分页表格即使数据少于 pageSize 也不渲染补位空行", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 5 },
          paginationInfo: { total: 1, totalPages: 1, page: 1, pageSize: 5 },
        }}
        ui={{ hideToolbar: true }}
        behavior={{ enableRowSelection: false }}
      />
    )

    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(container.querySelectorAll("tbody tr")).toHaveLength(1)
    expect(container.querySelectorAll('[data-slot="data-table-spacer-row"]')).toHaveLength(0)
  })

  it("仅在 loading 阶段保留稳定表面，并在稀疏结果解析后释放", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const { container, rerender } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 3 },
          paginationInfo: { total: 0, totalPages: 1, page: 1, pageSize: 3 },
        }}
        ui={{
          hideToolbar: true,
          hidePagination: true,
          loading: true,
          loadingRowCount: 3,
          stableSurfaceRowCount: 3,
        }}
        behavior={{ enableRowSelection: false }}
      />
    )

    const tableSurface = container.querySelector<HTMLDivElement>('[data-slot="data-table"]')

    expect(tableSurface).toHaveAttribute("data-stable-surface-row-count", "3")
    expect(tableSurface).toHaveStyle({ minHeight: "184px" })
    expect(container.querySelectorAll("tbody tr")).toHaveLength(3)
    expect(container.querySelectorAll('[data-slot="data-table-spacer-row"]')).toHaveLength(0)

    rerender(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 3 },
          paginationInfo: { total: 1, totalPages: 1, page: 1, pageSize: 3 },
        }}
        ui={{ hideToolbar: true, hidePagination: true, stableSurfaceRowCount: 3 }}
        behavior={{ enableRowSelection: false }}
      />
    )

    expect(tableSurface?.style.minHeight).toBe("")
    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(container.querySelectorAll("tbody tr")).toHaveLength(1)
    expect(container.querySelectorAll('[data-slot="data-table-spacer-row"]')).toHaveLength(0)
  })

  it("按 comfortable 行节奏为 loading 表面保留稳定高度", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 3 },
          paginationInfo: { total: 0, totalPages: 1, page: 1, pageSize: 3 },
        }}
        ui={{
          hideToolbar: true,
          hidePagination: true,
          loading: true,
          loadingRowCount: 3,
          rowDensity: "comfortable",
          stableSurfaceRowCount: 3,
        }}
        behavior={{ enableRowSelection: false }}
      />
    )

    expect(container.querySelector('[data-slot="data-table"]')).toHaveStyle({ minHeight: "232px" })
  })

  it("在 loading 和 resolved 状态为共享表格暴露同一组结构槽位", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const loadingSlots = {
      toolbar: "table-probe-toolbar",
      body: "table-probe-body",
      pagination: "table-probe-pagination",
    }

    const { container, rerender } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 3 },
          paginationInfo: { total: 0, totalPages: 1, page: 1, pageSize: 3 },
        }}
        ui={{
          loading: true,
          loadingPresentation: "initial",
          loadingRowCount: 3,
          loadingSlots,
        }}
        behavior={{ enableRowSelection: false }}
      />
    )

    expect(container.querySelectorAll('[data-loading-slot="table-probe-toolbar"]')).toHaveLength(1)
    expect(container.querySelectorAll('[data-loading-slot="table-probe-body"]')).toHaveLength(1)
    expect(container.querySelectorAll('[data-loading-slot="table-probe-pagination"]')).toHaveLength(1)

    rerender(
      <UnifiedDataTable<Row>
        data={[{ id: 1, name: "alpha" }]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 3 },
          paginationInfo: { total: 1, totalPages: 1, page: 1, pageSize: 3 },
        }}
        ui={{ loadingSlots }}
        behavior={{ enableRowSelection: false }}
      />
    )

    expect(container.querySelectorAll('[data-loading-slot="table-probe-toolbar"]')).toHaveLength(1)
    expect(container.querySelectorAll('[data-loading-slot="table-probe-body"]')).toHaveLength(1)
    expect(container.querySelectorAll('[data-loading-slot="table-probe-pagination"]')).toHaveLength(1)
    expect(container.querySelectorAll("tbody tr")).toHaveLength(1)
    expect(container.querySelectorAll('[data-slot="data-table-spacer-row"]')).toHaveLength(0)
  })

  it("loading rows reuse the shared table rhythm without inline estimated row height", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 3 },
          paginationInfo: { total: 0, totalPages: 1, page: 1, pageSize: 3 },
        }}
        ui={{ hideToolbar: true, hidePagination: true, loading: true, loadingRowCount: 3 }}
        behavior={{ enableRowSelection: false }}
      />
    )

    const bodyRows = Array.from(container.querySelectorAll("tbody tr"))

    expect(bodyRows).toHaveLength(3)
    expect(bodyRows.every((row) => row.className.includes("h-12"))).toBe(true)
    expect(bodyRows.every((row) => (row.getAttribute("style") ?? "") === "")).toBe(true)
    expect(screen.queryByText("No results")).not.toBeInTheDocument()
  })

  it("initial loading hides resolved table chrome and exposes busy semantics", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: { pageIndex: 0, pageSize: 10 },
          paginationInfo: { total: 0, totalPages: 1, page: 1, pageSize: 10 },
        }}
        actions={{ onBulkAdd: () => undefined, bulkAddLabel: "Add" }}
        ui={{ loading: true, loadingPresentation: "initial", loadingRowCount: 2 }}
        behavior={{ enableRowSelection: false }}
      />
    )

    const tableRoot = container.querySelector('[data-slot="unified-data-table"]')

    expect(tableRoot).toHaveAttribute("aria-busy", "true")
    expect(tableRoot).toHaveAttribute("data-loading-presentation", "initial")
    expect(screen.queryByText("Name")).not.toBeInTheDocument()
    expect(screen.queryByText("Add")).not.toBeInTheDocument()
    expect(screen.queryByText("No results")).not.toBeInTheDocument()
    const loadingControls = Array.from(
      container.querySelectorAll<HTMLButtonElement | HTMLInputElement>("button, input, select, textarea")
    )
    expect(loadingControls.length).toBeGreaterThan(0)
    expect(loadingControls.every((control) => control.disabled || control.getAttribute("aria-hidden") === "true")).toBe(true)
    expect(container.querySelector('[data-slot="compact-pagination-skeleton"]')).toBeInTheDocument()
  })

  it("keeps initial filter placeholders as siblings of the search slot", () => {
    const columns: ColumnDef<Row>[] = [
      {
        accessorKey: "name",
        header: "Name",
        meta: { title: "Name" },
      },
    ]

    const { container } = renderWithProviders(
      <UnifiedDataTable<Row>
        data={[]}
        columns={columns}
        getRowId={(row) => String(row.id)}
        ui={{
          loading: true,
          loadingPresentation: "initial",
          initialLoadingToolbarFilterCount: 1,
        }}
        behavior={{ enableRowSelection: false }}
      />
    )

    const leftToolbar = container.querySelector('[data-slot="data-table-initial-toolbar-skeleton"]')?.firstElementChild
    const searchSkeleton = leftToolbar?.querySelector('[data-slot="search-toolbar-skeleton"]')
    const filterSkeletons = leftToolbar?.querySelectorAll('[data-slot="action-skeleton"]') ?? []

    expect(searchSkeleton?.parentElement).toBe(leftToolbar)
    expect(filterSkeletons).toHaveLength(1)
    expect(filterSkeletons[0].parentElement).toBe(leftToolbar)
  })
})
