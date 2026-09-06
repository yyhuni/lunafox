import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { OrganizationSelectionWorkspace } from "@/components/organization/organization-selection-workspace"
import type { Organization } from "@/types/organization.types"

const organizations: Organization[] = [
  {
    id: 1,
    name: "Acme Corporation",
    description: "Cloud security",
    createdAt: "2026-07-01T08:00:00.000Z",
    updatedAt: "2026-07-01T08:00:00.000Z",
    targetCount: 12,
  },
  {
    id: 2,
    name: "TechStart Inc",
    description: "Application security",
    createdAt: "2026-07-02T08:00:00.000Z",
    updatedAt: "2026-07-02T08:00:00.000Z",
    targetCount: 6,
  },
]

function renderWorkspace(
  overrides: Partial<React.ComponentProps<typeof OrganizationSelectionWorkspace>> = {}
) {
  const props: React.ComponentProps<typeof OrganizationSelectionWorkspace> = {
    id: "organization-workspace",
    title: "Choose organization",
    hint: "Choose one or more organizations",
    selectionMode: "multiple",
    organizations,
    selectedOrganizationIds: ["1"],
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
    onToggleOrganization: vi.fn(),
    onClearOrganizations: vi.fn(),
    ...overrides,
  }

  return { ...render(<OrganizationSelectionWorkspace {...props} />), props }
}

describe("OrganizationSelectionWorkspace", () => {
  it("renders first reset and adjacent cursor controls without numbered navigation", () => {
    renderWorkspace()

    expect(screen.queryByRole("button", { name: /goToPage/ })).not.toBeInTheDocument()
    expect(screen.queryByText(/ellipsis/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/page:/i)).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "previousPage" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "nextPage" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "firstPage" })).toBeDisabled()
  })

  it("retains total, page-size, and disabled adjacent controls for an empty cursor response", () => {
    renderWorkspace({ totalCount: 0, canPreviousPage: false, canNextPage: false, organizations: [] })

    expect(screen.getByText("total:{\"count\":0}")).toBeInTheDocument()
    expect(screen.getByText("perPage")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "previousPage" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "nextPage" })).toBeDisabled()
  })

  it("renders controlled multi-selection and delegates toggle and clear actions", () => {
    const { props } = renderWorkspace()

    expect(screen.getByRole("checkbox", { name: /Acme Corporation/ })).toBeChecked()
    expect(screen.getByRole("checkbox", { name: /TechStart Inc/ })).not.toBeChecked()
    expect(screen.getByRole("checkbox", { name: /Acme Corporation/ }).closest("label")).toHaveClass("border-interaction-accent")
    expect(screen.getByRole("checkbox", { name: /TechStart Inc/ }).closest("label")).not.toHaveClass("border-interaction-accent")
    expect(screen.getByText("A").closest('[data-slot="avatar"]')).toHaveClass("border-border")
    expect(screen.getByText("A").closest('[data-slot="avatar-fallback"]')).toHaveClass("text-foreground")
    expect(screen.getByText("T").closest('[data-slot="avatar"]')).not.toHaveClass("border-border")

    fireEvent.click(screen.getByRole("checkbox", { name: /TechStart Inc/ }))
    fireEvent.click(screen.getByRole("button", { name: "clear" }))

    expect(props.onToggleOrganization).toHaveBeenCalledWith(organizations[1])
    expect(props.onClearOrganizations).toHaveBeenCalledTimes(1)
  })

  it("uses the first character as the organization avatar fallback", () => {
    renderWorkspace({
      organizations: [
        { ...organizations[0], name: "阿里云" },
        { ...organizations[1], name: "Acme Corporation" },
      ],
    })

    expect(screen.getByText("阿")).toBeInTheDocument()
    expect(screen.getByText("A")).toBeInTheDocument()
  })

  it("delegates search, first reset, and enabled adjacent pagination to the caller", () => {
    const { props } = renderWorkspace({ canFirstPage: true, canPreviousPage: true })

    fireEvent.change(screen.getByPlaceholderText("searchPlaceholder"), {
      target: { value: "Tech" },
    })
    fireEvent.click(screen.getByRole("button", { name: "nextPage" }))
    fireEvent.click(screen.getByRole("button", { name: "previousPage" }))
    fireEvent.click(screen.getByRole("button", { name: "firstPage" }))

    expect(props.onSearchQueryChange).toHaveBeenCalledWith("Tech")
    expect(props.onNextPage).toHaveBeenCalledTimes(1)
    expect(props.onPreviousPage).toHaveBeenCalledTimes(1)
    expect(props.onFirstPage).toHaveBeenCalledTimes(1)
  })

  it("renders single mode with one controlled selection", () => {
    const { container } = renderWorkspace({
      selectionMode: "single",
      selectedOrganizationIds: ["2"],
    })

    expect(container.querySelector('[data-selection-mode="single"]')).toBeInTheDocument()
    expect(screen.getByRole("checkbox", { name: /TechStart Inc/ })).toBeChecked()
    expect(screen.getByRole("checkbox", { name: /Acme Corporation/ })).not.toBeChecked()
  })

  it("can hide the redundant count while keeping the clear action", () => {
    renderWorkspace({
      selectionMode: "single",
      showSelectionCount: false,
    })

    expect(screen.queryByText("selectedCount")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "clear" })).toBeInTheDocument()
  })

  it("fast-fails multiple controlled IDs in single mode", () => {
    expect(() => renderWorkspace({
      selectionMode: "single",
      selectedOrganizationIds: ["1", "2"],
    })).toThrow("single mode accepts at most one selected organization")
  })
})
