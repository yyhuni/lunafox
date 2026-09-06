import * as React from "react"
import { fireEvent, render, screen, within } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { VulnerabilitiesVerticalHeader } from "@/components/vulnerabilities/vulnerabilities-vertical-header"

describe("VulnerabilitiesVerticalHeader", () => {
  it("aligns toolbar shell and control rhythm with the compact list-control baseline", () => {
    const onPaginationChange = vi.fn()

    const { container } = render(
      React.createElement(VulnerabilitiesVerticalHeader, {
        reviewFilter: "all",
        onReviewFilterChange: vi.fn(),
        severityFilter: [],
        onSeverityFilterChange: vi.fn(),
        filterQuery: "",
        onFilterChange: vi.fn(),
        totalCount: 18,
        pendingCount: 9,
        reviewedCount: 9,
        selectedCount: 0,
        onBulkMarkAsReviewed: vi.fn(),
        onBulkMarkAsPending: vi.fn(),
        onClearSelection: vi.fn(),
        pagination: { pageIndex: 0, pageSize: 20 },
        cursorPaginationSummary: { total: 18 },
        paginationNavigation: { mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: true },
        onPaginationChange,
      })
    )

    const shell = container.firstElementChild
    const tablist = screen.getByRole("tablist")
    const tabs = within(tablist).getAllByRole("tab")
    const searchbox = screen.getByRole("searchbox")
    const filterButton = screen.getByRole("button", { name: /Filter|筛选|filter/ })
    const pageSizeCombobox = screen.getByRole("combobox")
    const paginationLabel = screen.getByText("rowsPerPage")
    const previousButton = screen.getByRole("button", { name: "previous" })
    const nextButton = screen.getByRole("button", { name: "next" })

    expect(shell?.className).toContain("shrink-0")
    expect(shell?.className).not.toContain("bg-background")
    expect(shell?.className).not.toContain("bg-card")
    expect(shell?.className).not.toContain("border-b")
    expect(tablist.className).toContain("bg-muted")
    expect(tablist.className).toContain("h-8")
    expect(tablist.className).toContain("p-[2px]")

    for (const tab of tabs) {
      expect(tab).toHaveAttribute("data-size", "sm")
      expect(tab.className).not.toContain("h-11")
      expect(tab.className).not.toContain("md:h-7")
    }

    expect(searchbox).toHaveAttribute("data-input-size", "sm")
    expect(searchbox.className).toContain("h-8")
    expect(searchbox.className).toContain("pl-9")
    expect(searchbox.className).not.toContain("md:h-8")
    expect(searchbox.className).not.toContain("bg-background/50")

    expect(paginationLabel).toBeInTheDocument()
    expect(filterButton).toHaveAttribute("data-size", "sm")
    expect(filterButton.className).not.toContain("border-dashed")
    expect(pageSizeCombobox).toHaveAttribute("data-size", "sm")
    expect(pageSizeCombobox.className).toContain("data-[size=sm]:h-8")
    expect(pageSizeCombobox.className).toContain("w-24")

    expect(pageSizeCombobox).toHaveTextContent("20")
    expect(previousButton).toBeDisabled()
    expect(nextButton).toBeEnabled()
    expect(screen.getByRole("button", { name: "first" })).toBeDisabled()
    expect(screen.queryByRole("button", { name: "last" })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "1" })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "3" })).not.toBeInTheDocument()
    expect(container.querySelector('[aria-current="page"]')).toBeNull()
    fireEvent.click(nextButton)
    expect(onPaginationChange).toHaveBeenLastCalledWith({ pageIndex: 1, pageSize: 20 })

    expect(previousButton.className).toContain("size-8")
    expect(nextButton.className).toContain("size-8")
  })

  it("keeps terminal cursor navigation disabled and delegates only the cached predecessor", () => {
    const onPaginationChange = vi.fn()

    render(
      React.createElement(VulnerabilitiesVerticalHeader, {
        reviewFilter: "all",
        onReviewFilterChange: vi.fn(),
        severityFilter: [],
        onSeverityFilterChange: vi.fn(),
        filterQuery: "",
        onFilterChange: vi.fn(),
        totalCount: 18,
        pendingCount: 9,
        reviewedCount: 9,
        selectedCount: 0,
        onBulkMarkAsReviewed: vi.fn(),
        onBulkMarkAsPending: vi.fn(),
        onClearSelection: vi.fn(),
        pagination: { pageIndex: 1, pageSize: 20 },
        cursorPaginationSummary: { total: 18 },
        paginationNavigation: { mode: "cursor", canFirstPage: true, canPreviousPage: true, canNextPage: false },
        onPaginationChange,
      })
    )

    const previousButton = screen.getByRole("button", { name: "previous" })
    const nextButton = screen.getByRole("button", { name: "next" })
    expect(previousButton).toBeEnabled()
    expect(nextButton).toBeDisabled()
    expect(screen.getByRole("button", { name: "first" })).toBeEnabled()
    expect(screen.queryByRole("button", { name: "last" })).not.toBeInTheDocument()

    fireEvent.click(previousButton)
    fireEvent.click(nextButton)
    fireEvent.click(screen.getByRole("button", { name: "first" }))

    expect(onPaginationChange).toHaveBeenCalledTimes(2)
    expect(onPaginationChange).toHaveBeenCalledWith({ pageIndex: 0, pageSize: 20 })
  })
})
