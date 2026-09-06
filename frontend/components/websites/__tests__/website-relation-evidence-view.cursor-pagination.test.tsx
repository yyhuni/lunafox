import { fireEvent, render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { TargetWebsiteEvidenceView } from "@/components/websites/website-relation-evidence-view"

const websiteState = vi.hoisted(() => ({
  current: undefined as undefined | Record<string, unknown>,
}))

vi.mock("next/navigation", () => ({
  useSearchParams: () => new URLSearchParams(),
}))

vi.mock("@/components/websites/websites-view-state", () => ({
  useWebSitesViewState: () => websiteState.current,
}))

vi.mock("@/components/websites/websites-view-sections", () => ({
  WebSitesManagementOverlays: () => null,
}))

vi.mock("@/hooks/use-screenshots", () => ({
  useScreenshotImageUrlResolver: () => (id: number) => `/screenshots/${id}`,
}))

const website = {
  id: 1,
  resourceName: "targets/7/websites/1",
  url: "https://admin.example",
  host: "admin.example",
  location: "",
  title: "Admin",
  webserver: "nginx",
  contentType: "text/html",
  statusCode: 200,
  contentLength: 128,
  responseBody: "",
  responseHeaders: "server: nginx",
  tech: [],
  vhost: false,
  subdomain: "",
  createdAt: "2026-01-01T00:00:00.000Z",
}

function createState() {
  const handlePaginationChange = vi.fn()
  return {
    query: { pageIndex: 1, pageSize: 10, filters: {} },
    pageTokens: { 1: undefined },
    data: { results: [website], totalSize: 42 },
    error: null,
    isLoading: false,
    isSearching: false,
    refetch: vi.fn(),
    websites: [website],
    filterQuery: "",
    commitFilterSearch: vi.fn(),
    statusCodeFilter: [],
    statusCodeOptions: [],
    techFilter: [],
    techOptions: [],
    webserverFilter: [],
    webserverOptions: [],
    contentTypeFilter: [],
    contentTypeOptions: [],
    vhostFilter: [],
    vhostOptions: [],
    handleStatusCodeFilterChange: vi.fn(),
    handleTechFilterChange: vi.fn(),
    handleWebserverFilterChange: vi.fn(),
    handleContentTypeFilterChange: vi.fn(),
    handleVhostFilterChange: vi.fn(),
    handleExportAll: vi.fn(),
    handleExportSelected: vi.fn(),
    selectedWebSites: [],
    setBulkAddDialogOpen: vi.fn(),
    setDeleteDialogOpen: vi.fn(),
    handleSelectionChange: vi.fn(),
    pagination: { pageIndex: 0, pageSize: 10 },
    cursorPaginationSummary: { total: 42 },
    paginationNavigation: { mode: "cursor" as const, canPreviousPage: false, canNextPage: true },
    handlePaginationChange,
  }
}

describe("TargetWebsiteEvidenceView cursor pagination", () => {
  beforeEach(() => {
    websiteState.current = createState()
  })

  it("forwards only adjacent cursor actions to the website state owner", () => {
    const state = websiteState.current as ReturnType<typeof createState>
    const { container, rerender } = render(<TargetWebsiteEvidenceView targetId={7} />)

    expect(screen.getByRole("button", { name: "previous" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "next" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "first" })).toBeDisabled()
    expect(screen.queryByRole("button", { name: "last" })).not.toBeInTheDocument()
    expect(container.querySelector('[aria-current="page"]')).toBeNull()

    fireEvent.click(screen.getByRole("button", { name: "next" }))
    expect(state.handlePaginationChange).toHaveBeenCalledWith({ pageIndex: 1, pageSize: 10 })

    websiteState.current = {
      ...state,
      pagination: { pageIndex: 1, pageSize: 10 },
      paginationNavigation: { mode: "cursor", canPreviousPage: true, canNextPage: false },
    }
    rerender(<TargetWebsiteEvidenceView targetId={7} />)

    expect(screen.getByRole("button", { name: "previous" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "next" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "first" })).toBeEnabled()
    fireEvent.click(screen.getByRole("button", { name: "previous" }))
    fireEvent.click(screen.getByRole("button", { name: "next" }))

    expect(state.handlePaginationChange).toHaveBeenLastCalledWith({ pageIndex: 0, pageSize: 10 })
    expect(state.handlePaginationChange).toHaveBeenCalledTimes(2)
  })

  it("keeps cached previous navigation available on an empty terminal cursor response", () => {
    const state = websiteState.current as ReturnType<typeof createState>
    websiteState.current = {
      ...state,
      data: { results: [], totalSize: 42 },
      websites: [],
      pagination: { pageIndex: 1, pageSize: 10 },
      paginationNavigation: { mode: "cursor", canPreviousPage: true, canNextPage: false },
    }

    render(<TargetWebsiteEvidenceView targetId={7} />)

    expect(screen.getByRole("button", { name: "previous" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "next" })).toBeDisabled()
    expect(screen.getByText('selected:{"count":0} / total:{"count":42}')).toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "previous" }))
    expect(state.handlePaginationChange).toHaveBeenCalledWith({ pageIndex: 0, pageSize: 10 })
  })
})
