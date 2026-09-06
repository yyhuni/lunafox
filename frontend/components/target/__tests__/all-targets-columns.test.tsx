import * as React from "react"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { createAllTargetsColumns } from "@/components/target/all-targets-columns"

const sonnerMocks = vi.hoisted(() => ({
  toast: {
    success: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: sonnerMocks.toast,
}))

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...props
  }: React.AnchorHTMLAttributes<HTMLAnchorElement> & { href: string }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) => <button {...props}>{children}</button>,
}))

vi.mock("@/components/ui/tooltip", () => ({
  TooltipProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

describe("all-targets-columns copy toast", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window, "isSecureContext", {
      configurable: true,
      value: true,
    })
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  it("连续点击复制时复用固定 toast id，避免重复 toast 渲染异常", async () => {
    const columns = createAllTargetsColumns({
      formatDate: (value) => value,
      navigate: vi.fn(),
      handleDelete: vi.fn(),
      handleInitiateScan: vi.fn(),
      handleScheduleScan: vi.fn(),
      t: {
        columns: {
          target: "目标",
          organization: "组织",
          addedOn: "添加时间",
          lastScanned: "最近扫描",
          actions: "操作",
        },
        actions: {
          scheduleScan: "计划扫描",
          delete: "删除",
          selectAll: "全选",
          selectRow: "选择行",
          openMenu: "打开菜单",
        },
        tooltips: {
          targetDetails: "目标详情",
          targetSummary: "目标摘要",
          initiateScan: "发起扫描",
          clickToCopy: "点击复制",
          copied: "已复制",
        },
        targetTypes: {
          domain: "域名",
          ip: "IP",
          cidr: "CIDR",
        },
      },
    })

    const nameColumn = columns.find(
      (column) => "accessorKey" in column && column.accessorKey === "name"
    )
    if (!nameColumn || typeof nameColumn.cell !== "function") {
      throw new Error("name column cell is not available")
    }

    render(
      <>
        {nameColumn.cell({
          row: {
            getValue: () => "example.com",
            original: { id: 1, type: "domain" },
          },
        } as never)}
      </>
    )

    const button = screen.getByRole("button", { name: "点击复制" })
    fireEvent.click(button)

    await waitFor(() => {
      expect(sonnerMocks.toast.success).toHaveBeenCalledWith("已复制", { id: "target-name-copy" })
    })

    fireEvent.click(button)

    await waitFor(() => {
      expect(sonnerMocks.toast.success).toHaveBeenNthCalledWith(2, "已复制", { id: "target-name-copy" })
    })

    expect(screen.getByRole("img", { name: "域名" })).toBeInTheDocument()
  })
})
