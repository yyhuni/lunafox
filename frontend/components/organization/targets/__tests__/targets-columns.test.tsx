import { fireEvent, render, screen } from "@testing-library/react"
import type { ReactNode } from "react"
import { describe, expect, it, vi } from "vitest"

import { createTargetColumns } from "@/components/organization/targets/targets-columns"

vi.mock("@/components/shared/data-table/menu-owners", () => ({
  DenseRowActionMenu: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: ReactNode
    onClick?: () => void
  }) => <button type="button" onClick={onClick}>{children}</button>,
}))

describe("organization target row actions", () => {
  it("uses the owning navigation callback for the portalled view-details menu item", () => {
    const navigate = vi.fn()
    const columns = createTargetColumns({
      formatDate: (value) => value,
      navigate,
      handleDelete: vi.fn(),
      t: {
        columns: {
          targetName: "目标名称",
          type: "类型",
          addedOn: "添加时间",
          lastScanned: "最后扫描时间",
        },
        actions: {
          selectAll: "全选",
          selectRow: "选择行",
        },
        tooltips: {
          viewDetails: "查看详情",
          unlinkTarget: "解除关联",
          clickToCopy: "点击复制",
          copied: "已复制",
        },
        types: {
          domain: "域名",
          ip: "IP",
          cidr: "CIDR",
        },
      },
    })
    const actionsColumn = columns.find((column) => column.id === "actions")

    if (!actionsColumn || typeof actionsColumn.cell !== "function") {
      throw new Error("target action column is unavailable")
    }

    render(actionsColumn.cell({
      row: {
        original: { id: 6 },
      },
    } as never))

    fireEvent.click(screen.getByRole("button", { name: "查看详情" }))

    expect(navigate).toHaveBeenCalledWith("/targets/6/overview/")
  })
})
