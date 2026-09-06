import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useTargetsDetailViewState } from "@/components/organization/targets/targets-detail-view-state"

const organizationHooks = vi.hoisted(() => ({
  useOrganization: vi.fn(),
  useOrganizationTargets: vi.fn(),
  useUnlinkTargetsFromOrganization: vi.fn(),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
  useTranslations: () => (key: string) => key,
}))

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock("@/hooks/use-organizations", () => organizationHooks)
vi.mock("@/components/organization/targets/targets-columns", () => ({
  createTargetColumns: vi.fn(() => []),
}))
vi.mock("@/components/route-progress", () => ({
  pushWithRouteProgress: vi.fn(),
}))

function targetListRequests() {
  return organizationHooks.useOrganizationTargets.mock.calls
}

describe("useTargetsDetailViewState cursor pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    organizationHooks.useOrganization.mockReturnValue({
      data: { id: 1 },
      isLoading: false,
      error: null,
    })
    organizationHooks.useUnlinkTargetsFromOrganization.mockReturnValue({ mutate: vi.fn() })
    organizationHooks.useOrganizationTargets.mockImplementation((
      _organizationId: number,
      params?: { pageSize?: number; pageToken?: string },
    ) => ({
      data: params?.pageToken
        ? { results: [], totalSize: 3 }
        : { results: [], totalSize: 3, nextPageToken: "organization-targets-page-two" },
      isLoading: false,
      isPlaceholderData: false,
      error: null,
      refetch: vi.fn(),
    }))
  })

  it("uses the current response continuation and resets it for first page and page-size changes", () => {
    const { result } = renderHook(() => useTargetsDetailViewState({ organizationId: "1" }))

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })

    expect(targetListRequests().at(-1)?.[1]).toEqual(expect.objectContaining({
      pageSize: 10,
      pageToken: "organization-targets-page-two",
    }))
    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: true,
      canPreviousPage: true,
      canNextPage: false,
    })

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 0, pageSize: 10 })
    })

    expect(targetListRequests().at(-1)?.[1]).toEqual(expect.objectContaining({
      pageSize: 10,
      pageToken: undefined,
    }))

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 0, pageSize: 20 })
    })

    expect(targetListRequests().at(-1)?.[1]).toEqual(expect.objectContaining({
      pageSize: 20,
      pageToken: undefined,
    }))
  })
})
