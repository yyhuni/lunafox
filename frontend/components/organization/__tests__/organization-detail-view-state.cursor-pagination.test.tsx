import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useOrganizationDetailViewState } from "@/components/organization/organization-detail-view-state"

const organizationHooks = vi.hoisted(() => ({
  useOrganization: vi.fn(),
  useOrganizationTargets: vi.fn(),
  useUnlinkTargetsFromOrganization: vi.fn(),
}))

const scheduledScanHooks = vi.hoisted(() => ({
  useDeleteScheduledScan: vi.fn(),
  useScheduledScans: vi.fn(),
  useToggleScheduledScan: vi.fn(),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
  useTranslations: () => Object.assign(
    (key: string) => key,
    { raw: (key: string) => key.endsWith("weekdays") ? [] : key }
  ),
}))

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock("@/hooks/use-organizations", () => organizationHooks)
vi.mock("@/hooks/use-scheduled-scans", () => scheduledScanHooks)
vi.mock("@/components/organization/targets/targets-columns", () => ({
  createTargetColumns: vi.fn(() => []),
}))
vi.mock("@/components/scan/scheduled/scheduled-scan-columns", () => ({
  createScheduledScanColumns: vi.fn(() => []),
}))
vi.mock("@/components/route-progress", () => ({
  pushWithRouteProgress: vi.fn(),
}))
vi.mock("@/components/shared/loading/use-interaction-open-loader", () => ({
  useInteractionOpenLoader: () => ({
    pendingInteraction: null,
    cancelPending: vi.fn(),
    openAfterLoad: vi.fn(),
  }),
}))

function createMutation() {
  return { mutate: vi.fn() }
}

function targetListRequests() {
  return organizationHooks.useOrganizationTargets.mock.calls.filter(
    ([, params]) => params?.pageSize === 10
  )
}

describe("useOrganizationDetailViewState cursor pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    organizationHooks.useOrganization.mockReturnValue({
      data: { id: 5, targetCount: 42 },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    })
    organizationHooks.useUnlinkTargetsFromOrganization.mockReturnValue(createMutation())
    organizationHooks.useOrganizationTargets.mockImplementation((
      _organizationId: number,
      params?: { pageSize?: number; pageToken?: string },
    ) => ({
      data: params?.pageSize === 1000
        ? { results: [], total: 42, totalSize: 42 }
        : params?.pageToken
          ? { results: [], total: 42, totalSize: 42 }
          : { results: [], total: 42, totalSize: 42, nextPageToken: "organization-targets-page-two" },
      isLoading: false,
      isFetching: false,
      isPlaceholderData: false,
      error: null,
      refetch: vi.fn(),
    }))
    scheduledScanHooks.useDeleteScheduledScan.mockReturnValue(createMutation())
    scheduledScanHooks.useToggleScheduledScan.mockReturnValue(createMutation())
    scheduledScanHooks.useScheduledScans.mockReturnValue({
      data: { scheduledScans: [] },
      isLoading: false,
      isFetching: false,
      refetch: vi.fn(),
    })
  })

  it("uses the service continuation for next and never derives a last page from totalSize", () => {
    const { result } = renderHook(() => useOrganizationDetailViewState({ organizationId: "5" }))

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })
    expect(targetListRequests().at(-1)?.[1]).toEqual(expect.objectContaining({
      pageSize: 10,
      pageToken: undefined,
    }))

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

    const callsBeforeTerminalAttempt = targetListRequests().length
    act(() => {
      result.current.handlePaginationChange({ pageIndex: 2, pageSize: 10 })
    })
    expect(targetListRequests()).toHaveLength(callsBeforeTerminalAttempt)

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 0, pageSize: 10 })
    })
    expect(targetListRequests().at(-1)?.[1]).toEqual(expect.objectContaining({
      pageToken: undefined,
    }))
  })

  it("keeps the first detail frame pending until summary sources settle", () => {
    organizationHooks.useOrganizationTargets.mockImplementation((
      _organizationId: number,
      params?: { pageSize?: number; pageToken?: string },
    ) => ({
      data: params?.pageSize === 1000
        ? undefined
        : { results: [], total: 42, totalSize: 42, nextPageToken: "organization-targets-page-two" },
      isLoading: params?.pageSize === 1000,
      isFetching: false,
      isPlaceholderData: false,
      error: null,
      refetch: vi.fn(),
    }))
    scheduledScanHooks.useScheduledScans.mockReturnValue({
      data: undefined,
      isLoading: true,
      isFetching: false,
      refetch: vi.fn(),
    })

    const { result } = renderHook(() => useOrganizationDetailViewState({ organizationId: "5" }))

    expect(result.current.isLoading).toBe(false)
    expect(result.current.isInitialContentLoading).toBe(true)
  })
})
