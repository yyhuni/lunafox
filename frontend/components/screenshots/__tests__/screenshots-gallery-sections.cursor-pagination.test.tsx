import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { ScreenshotsGalleryContent } from "@/components/screenshots/screenshots-gallery-sections"
import type { ScreenshotsGalleryState } from "@/components/screenshots/screenshots-gallery-state"

function createState(overrides: Partial<ScreenshotsGalleryState> = {}) {
  const handlePaginationChange = vi.fn()
  return {
    t: (key: string) => key,
    tCommon: (key: string) => key,
    targetId: 7,
    scanId: undefined,
    screenshots: [],
    filterQuery: "",
    commitFilterSearch: vi.fn(),
    isSearching: false,
    statusCodeFilter: [],
    statusCodeOptions: [],
    handleStatusCodeFilterChange: vi.fn(),
    sorting: [],
    handleSortingChange: vi.fn(),
    selectedIds: new Set<number>(),
    selectAll: vi.fn(),
    toggleSelect: vi.fn(),
    openLightbox: vi.fn(),
    getImageUrl: vi.fn(),
    pagination: { pageIndex: 0, pageSize: 12 },
    cursorPaginationSummary: { total: 36 },
    paginationNavigation: { mode: "cursor" as const, canFirstPage: false, canPreviousPage: false, canNextPage: true },
    handlePaginationChange,
    pageSizeOptions: [12, 24, 48],
    lightboxIndex: 0,
    lightboxOpen: false,
    setLightboxOpen: vi.fn(),
    prevImage: vi.fn(),
    nextImage: vi.fn(),
    deleteDialogOpen: false,
    setDeleteDialogOpen: vi.fn(),
    isDeleting: false,
    handleBulkDelete: vi.fn(),
    clearSelection: vi.fn(),
    ...overrides,
  } as unknown as ScreenshotsGalleryState
}

describe("ScreenshotsGalleryContent cursor pagination", () => {
  it("renders only adjacent actions and disables terminal next navigation", () => {
    const initial = createState()
    const { container, rerender } = render(<ScreenshotsGalleryContent state={initial} />)

    expect(screen.getByText('total:{"count":36}')).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "previous" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "next" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "first" })).toBeDisabled()
    expect(screen.queryByRole("button", { name: "last" })).not.toBeInTheDocument()
    expect(container.querySelector('[aria-current="page"]')).toBeNull()

    fireEvent.click(screen.getByRole("button", { name: "next" }))
    expect(initial.handlePaginationChange).toHaveBeenCalledWith({ pageIndex: 1, pageSize: 12 })

    const terminal = createState({
      pagination: { pageIndex: 1, pageSize: 12 },
      paginationNavigation: { mode: "cursor", canFirstPage: true, canPreviousPage: true, canNextPage: false },
    })
    rerender(<ScreenshotsGalleryContent state={terminal} />)

    expect(screen.getByRole("button", { name: "previous" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "first" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "next" })).toBeDisabled()
    fireEvent.click(screen.getByRole("button", { name: "previous" }))
    fireEvent.click(screen.getByRole("button", { name: "next" }))
    fireEvent.click(screen.getByRole("button", { name: "first" }))

    expect(terminal.handlePaginationChange).toHaveBeenCalledTimes(2)
    expect(terminal.handlePaginationChange).toHaveBeenCalledWith({ pageIndex: 0, pageSize: 12 })
  })
})
