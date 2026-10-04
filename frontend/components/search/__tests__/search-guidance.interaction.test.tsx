import { act, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

const mobileState = vi.hoisted(() => ({ value: false }))

vi.mock("@/hooks/use-mobile", () => ({
  useIsMobile: () => mobileState.value,
}))

import { SearchAssetBar } from "../search-page-sections"
import type { SearchPageState } from "../search-page-state"

function createState() {
  return {
    t: (key: string) => key,
    searchState: "initial",
    query: 'host="api"',
    assetType: "website",
    pageSize: 10,
    recentSearches: [],
    queryError: null,
    queryDiagnosticCode: null,
    data: undefined,
    isLoading: false,
    isFetching: false,
    error: null,
    refetch: vi.fn(),
    canFirstPage: false,
    canPreviousPage: false,
    canNextPage: false,
    setQuery: vi.fn(),
    handleSearch: vi.fn(),
    handleQuickTagClick: vi.fn(),
    handleRecentSearchClick: vi.fn(),
    handleRemoveRecentSearch: vi.fn(),
    handleAssetTypeChange: vi.fn(),
    handlePageSizeChange: vi.fn(),
    handlePreviousPage: vi.fn(),
    handleFirstPage: vi.fn(),
    handleNextPage: vi.fn(),
  } as unknown as SearchPageState
}

describe("SearchAssetBar guidance interactions", () => {
  afterEach(() => {
    mobileState.value = false
  })

  it("uses a desktop Dialog, preserves the draft, and restores focus", async () => {
    mobileState.value = false
    const state = createState()
    render(<SearchAssetBar state={state} />)

    const input = screen.getByRole("searchbox")
    fireEvent.focus(input)
    await waitFor(() => expect(screen.getByRole("button", { name: "syntax.manualAriaLabel" })).toBeInTheDocument())
    fireEvent.click(screen.getByRole("button", { name: "syntax.manualAriaLabel" }))

    expect(await screen.findByRole("dialog")).toBeInTheDocument()
    expect(screen.getAllByText("manual.title").length).toBeGreaterThanOrEqual(1)
    expect(state.setQuery).not.toHaveBeenCalled()
    expect(state.handleSearch).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole("button", { name: "close" }))
    await waitFor(() => expect(document.activeElement).toBe(input))
    expect(state.setQuery).not.toHaveBeenCalled()
  }, 15_000)

  it("uses a 390px-style Drawer and keeps example actions draft-only", async () => {
    mobileState.value = true
    const state = createState()
    render(<SearchAssetBar state={state} />)

    const input = screen.getByRole("searchbox")
    fireEvent.focus(input)
    await waitFor(() => expect(screen.getByRole("button", { name: "syntax.manualAriaLabel" })).toBeInTheDocument())
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "syntax.manualAriaLabel" }))
      await new Promise<void>((resolve) => window.requestAnimationFrame(() => resolve()))
    })

    await waitFor(() => expect(document.querySelector('[data-slot="drawer-content"]')).toBeInTheDocument())
    expect(document.querySelector('[data-slot="dialog-content"]')).not.toBeInTheDocument()

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: 'host="api" && statusCode=="200"' }))
      await new Promise<void>((resolve) => window.requestAnimationFrame(() => resolve()))
    })
    expect(state.setQuery).toHaveBeenCalledWith('host="api" && statusCode=="200"')
    expect(state.handleSearch).not.toHaveBeenCalled()
  }, 15_000)
})
