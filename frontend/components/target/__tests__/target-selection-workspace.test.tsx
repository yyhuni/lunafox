import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { TargetSelectionWorkspace } from "@/components/target/target-selection-workspace"
import type { Target } from "@/types/target.types"

const targets: Target[] = [
  {
    id: 1,
    name: "acme.com",
    type: "domain",
    createdAt: "2026-07-01T08:00:00.000Z",
    organizations: [{ id: 1, name: "Acme Corporation" }],
  },
  {
    id: 2,
    name: "192.168.1.0/24",
    type: "cidr",
    createdAt: "2026-07-02T08:00:00.000Z",
    organizations: [{ id: 2, name: "Global Finance Ltd" }],
  },
]

function renderWorkspace(
  overrides: Partial<React.ComponentProps<typeof TargetSelectionWorkspace>> = {}
) {
  const props: React.ComponentProps<typeof TargetSelectionWorkspace> = {
    id: "target-workspace",
    title: "Choose target",
    hint: "Choose one target",
    targets,
    selectedTargetId: 1,
    totalCount: 4,
    canFirstPage: false,
    canPreviousPage: false,
    canNextPage: true,
    pageSize: 2,
    pageSizeOptions: [2, 4],
    searchQuery: "",
    isLoading: false,
    onSearchQueryChange: vi.fn(),
    onFirstPage: vi.fn(),
    onPreviousPage: vi.fn(),
    onNextPage: vi.fn(),
    onPageSizeChange: vi.fn(),
    onToggleTarget: vi.fn(),
    onClearTarget: vi.fn(),
    ...overrides,
  }

  return { ...render(<TargetSelectionWorkspace {...props} />), props }
}

describe("TargetSelectionWorkspace", () => {
  it("disables previous navigation on the first cursor response", () => {
    renderWorkspace()

    expect(screen.getByRole("button", { name: "previousPage" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "nextPage" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "firstPage" })).toBeDisabled()
  })

  it("retains total, page-size, and disabled adjacent controls for an empty cursor response", () => {
    renderWorkspace({ totalCount: 0, canPreviousPage: false, canNextPage: false, targets: [] })

    expect(screen.getByText("total:{\"count\":0}")).toBeInTheDocument()
    expect(screen.getByText("perPage")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "previousPage" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "nextPage" })).toBeDisabled()
  })

  it("renders one controlled target and delegates toggle and clear actions", () => {
    const { props } = renderWorkspace()

    expect(screen.getByRole("checkbox", { name: /acme.com/ })).toBeChecked()
    expect(screen.getByRole("checkbox", { name: /192.168.1.0\/24/ })).not.toBeChecked()
    expect(screen.getByRole("checkbox", { name: /acme.com/ }).closest("label")).toHaveClass("border-interaction-accent")
    expect(screen.getByRole("checkbox", { name: /192.168.1.0\/24/ }).closest("label")).not.toHaveClass("border-interaction-accent")

    fireEvent.click(screen.getByRole("checkbox", { name: /192.168.1.0\/24/ }))
    fireEvent.click(screen.getByRole("button", { name: "clear" }))

    expect(props.onToggleTarget).toHaveBeenCalledWith(targets[1])
    expect(props.onClearTarget).toHaveBeenCalledTimes(1)
  })

  it("omits the redundant single-selection count", () => {
    renderWorkspace()

    expect(screen.queryByText("selectedCount")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "clear" })).toBeInTheDocument()
  })

  it("renders first reset and adjacent cursor controls and delegates enabled actions", () => {
    const { props } = renderWorkspace({ canFirstPage: true, canPreviousPage: true })

    expect(screen.queryByRole("button", { name: /goToPage/ })).not.toBeInTheDocument()
    expect(screen.queryByText(/page:/i)).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "previousPage" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "nextPage" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "firstPage" })).toBeEnabled()

    fireEvent.change(screen.getByPlaceholderText("searchPlaceholder"), {
      target: { value: "acme" },
    })
    fireEvent.click(screen.getByRole("button", { name: "nextPage" }))
    fireEvent.click(screen.getByRole("button", { name: "previousPage" }))
    fireEvent.click(screen.getByRole("button", { name: "firstPage" }))

    expect(props.onSearchQueryChange).toHaveBeenCalledWith("acme")
    expect(props.onNextPage).toHaveBeenCalledTimes(1)
    expect(props.onPreviousPage).toHaveBeenCalledTimes(1)
    expect(props.onFirstPage).toHaveBeenCalledTimes(1)
  })
})
