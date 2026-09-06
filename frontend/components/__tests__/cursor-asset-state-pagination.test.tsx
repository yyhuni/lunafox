import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useWebSitesViewState } from "@/components/websites/websites-view-state"
import { useSubdomainsDetailViewState } from "@/components/subdomains/subdomains-detail-view-state"
import { useDirectoriesViewState } from "@/components/directories/directories-view-state"
import { useEndpointsDetailViewState } from "@/components/endpoints/endpoints-detail-view-state"
import { useIPAddressesViewState } from "@/components/ip-addresses/ip-addresses-view-state"
import { useVulnerabilitiesDetailViewState } from "@/components/vulnerabilities/vulnerabilities-detail-view-state"

const websiteHooks = vi.hoisted(() => ({
  useBulkDeleteWebSites: vi.fn(),
  useExportWebsites: vi.fn(),
  useTargetWebSites: vi.fn(),
  useScanWebSites: vi.fn(),
  useTargetWebsiteFilterOptions: vi.fn(),
  useScanWebsiteFilterOptions: vi.fn(),
}))

const subdomainHooks = vi.hoisted(() => ({
  useBatchDeleteSubdomains: vi.fn(),
  useExportSubdomains: vi.fn(),
  useTargetSubdomains: vi.fn(),
  useScanSubdomains: vi.fn(),
}))

const directoryHooks = vi.hoisted(() => ({
  useBulkDeleteDirectories: vi.fn(),
  useExportDirectories: vi.fn(),
  useTargetDirectories: vi.fn(),
  useScanDirectories: vi.fn(),
  useTargetDirectoryFilterOptions: vi.fn(),
  useScanDirectoryFilterOptions: vi.fn(),
}))

const endpointHooks = vi.hoisted(() => ({
  useBatchDeleteEndpoints: vi.fn(),
  useDeleteEndpoint: vi.fn(),
  useExportEndpoints: vi.fn(),
  useTargetEndpoints: vi.fn(),
  useScanEndpoints: vi.fn(),
  useTargetEndpointFilterOptions: vi.fn(),
  useScanEndpointFilterOptions: vi.fn(),
}))

const ipHooks = vi.hoisted(() => ({
  useBulkDeleteIPAddresses: vi.fn(),
  useExportIPAddresses: vi.fn(),
  useTargetIPAddresses: vi.fn(),
  useScanIPAddresses: vi.fn(),
  useTargetPortOptions: vi.fn(),
  useScanPortOptions: vi.fn(),
}))

const vulnerabilityHooks = vi.hoisted(() => ({
  useAllVulnerabilities: vi.fn(),
  useBulkDeleteVulnerabilities: vi.fn(),
  useBulkMarkAsReviewed: vi.fn(),
  useBulkMarkAsUnreviewed: vi.fn(),
  useGlobalVulnerabilityFilterOptions: vi.fn(),
  useScanVulnerabilities: vi.fn(),
  useScanVulnerabilityFilterOptions: vi.fn(),
  useTargetVulnerabilities: vi.fn(),
  useTargetVulnerabilityFilterOptions: vi.fn(),
  useTargetVulnerabilityStats: vi.fn(),
  useVulnerabilityStats: vi.fn(),
}))

const targetHooks = vi.hoisted(() => ({
  useTarget: vi.fn(() => ({ data: undefined })),
  useTargetEndpoints: vi.fn(),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
  useTranslations: () => (key: string) => key,
}))

vi.mock("@/hooks/use-targets", () => targetHooks)
vi.mock("@/hooks/use-websites", () => websiteHooks)
vi.mock("@/hooks/use-subdomains", () => subdomainHooks)
vi.mock("@/hooks/use-directories", () => directoryHooks)
vi.mock("@/hooks/use-endpoints", () => endpointHooks)
vi.mock("@/hooks/use-ip-addresses", () => ipHooks)
vi.mock("@/hooks/use-vulnerabilities", () => vulnerabilityHooks)

vi.mock("@/components/websites/websites-columns", () => ({
  useWebSiteTableColumns: vi.fn(() => ({ columns: [], formatDate: vi.fn(() => "") })),
}))
vi.mock("@/components/subdomains/subdomains-columns", () => ({
  createSubdomainColumns: vi.fn(() => []),
}))
vi.mock("@/components/directories/directories-columns", () => ({
  useDirectoryTableColumns: vi.fn(() => ({ columns: [] })),
}))
vi.mock("@/components/endpoints/endpoints-columns", () => ({
  createEndpointColumns: vi.fn(() => []),
}))
vi.mock("@/components/ip-addresses/ip-addresses-columns", () => ({
  createIPAddressColumns: vi.fn(() => []),
}))
vi.mock("@/components/vulnerabilities/vulnerabilities-columns", () => ({
  createVulnerabilityColumns: vi.fn(() => []),
}))

const mutation = () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false })

function response(
  request: { pageToken?: string },
  token: string,
  shape: "results" | "vulnerabilities" = "results",
  isPlaceholderData = false,
) {
  const data = request.pageToken
    ? { [shape]: [], totalSize: 42, total: 42 }
    : { [shape]: [], totalSize: 42, total: 42, nextPageToken: token }
  return {
    data,
    isLoading: false,
    isFetching: isPlaceholderData,
    isPlaceholderData,
    error: null,
    refetch: vi.fn(),
  }
}

function installDefaultResponses() {
  targetHooks.useTargetEndpoints.mockImplementation((_id: number, request: Record<string, unknown>) => response(request, "endpoint-page-two"))
  websiteHooks.useTargetWebSites.mockImplementation((_id, request) => response(request, "website-page-two"))
  websiteHooks.useScanWebSites.mockImplementation((_id, request) => response(request, "scan-website-page-two"))
  subdomainHooks.useTargetSubdomains.mockImplementation((_id, request) => response(request, "subdomain-page-two"))
  subdomainHooks.useScanSubdomains.mockImplementation((_id, request) => response(request, "scan-subdomain-page-two"))
  directoryHooks.useTargetDirectories.mockImplementation((_id, request) => response(request, "directory-page-two"))
  directoryHooks.useScanDirectories.mockImplementation((_id, request) => response(request, "scan-directory-page-two"))
  endpointHooks.useScanEndpoints.mockImplementation((_id, request) => response(request, "scan-endpoint-page-two"))
  ipHooks.useTargetIPAddresses.mockImplementation((_id, request) => response(request, "ip-page-two"))
  ipHooks.useScanIPAddresses.mockImplementation((_id, request) => response(request, "scan-ip-page-two"))
  vulnerabilityHooks.useAllVulnerabilities.mockImplementation((request) => response(request, "vulnerability-page-two", "vulnerabilities"))
  vulnerabilityHooks.useTargetVulnerabilities.mockImplementation((_id, request) => response(request, "target-vulnerability-page-two", "vulnerabilities"))
  vulnerabilityHooks.useScanVulnerabilities.mockImplementation((_id, request) => response(request, "scan-vulnerability-page-two", "vulnerabilities"))

  for (const hook of [
    websiteHooks.useTargetWebsiteFilterOptions,
    websiteHooks.useScanWebsiteFilterOptions,
    directoryHooks.useTargetDirectoryFilterOptions,
    directoryHooks.useScanDirectoryFilterOptions,
    endpointHooks.useTargetEndpointFilterOptions,
    endpointHooks.useScanEndpointFilterOptions,
    vulnerabilityHooks.useGlobalVulnerabilityFilterOptions,
    vulnerabilityHooks.useScanVulnerabilityFilterOptions,
    vulnerabilityHooks.useTargetVulnerabilityFilterOptions,
    ipHooks.useTargetPortOptions,
    ipHooks.useScanPortOptions,
  ]) {
    hook.mockReturnValue({ data: { results: [] } })
  }

  websiteHooks.useBulkDeleteWebSites.mockReturnValue(mutation())
  websiteHooks.useExportWebsites.mockReturnValue(vi.fn())
  subdomainHooks.useBatchDeleteSubdomains.mockReturnValue(mutation())
  subdomainHooks.useExportSubdomains.mockReturnValue(vi.fn())
  directoryHooks.useBulkDeleteDirectories.mockReturnValue(mutation())
  directoryHooks.useExportDirectories.mockReturnValue(vi.fn())
  endpointHooks.useBatchDeleteEndpoints.mockReturnValue(mutation())
  endpointHooks.useDeleteEndpoint.mockReturnValue(mutation())
  endpointHooks.useExportEndpoints.mockReturnValue(vi.fn())
  ipHooks.useBulkDeleteIPAddresses.mockReturnValue(mutation())
  ipHooks.useExportIPAddresses.mockReturnValue(vi.fn())
  vulnerabilityHooks.useBulkDeleteVulnerabilities.mockReturnValue(mutation())
  vulnerabilityHooks.useBulkMarkAsReviewed.mockReturnValue(mutation())
  vulnerabilityHooks.useBulkMarkAsUnreviewed.mockReturnValue(mutation())
  vulnerabilityHooks.useVulnerabilityStats.mockReturnValue({ data: { total: 42 } })
  vulnerabilityHooks.useTargetVulnerabilityStats.mockReturnValue({ data: { total: 42 } })
}

function lastPaginationRequest(hook: ReturnType<typeof vi.fn>) {
  return hook.mock.calls.at(-1)?.find((argument) => (
    typeof argument === "object" &&
    argument !== null &&
    "pageSize" in argument
  ))
}

function assertAdjacentCursor(
  result: { current: { paginationNavigation: { canFirstPage: boolean; canPreviousPage: boolean; canNextPage: boolean }; handlePaginationChange: (value: { pageIndex: number; pageSize: number }) => void } },
  hook: ReturnType<typeof vi.fn>,
) {
  expect(result.current.paginationNavigation).toEqual({ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: true })

  act(() => {
    result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
  })
  expect(lastPaginationRequest(hook)).toEqual(expect.objectContaining({ pageToken: expect.any(String) }))
  expect(result.current.paginationNavigation).toEqual({ mode: "cursor", canFirstPage: true, canPreviousPage: true, canNextPage: false })

  const callsBeforeTerminalAttempt = hook.mock.calls.length
  act(() => {
    result.current.handlePaginationChange({ pageIndex: 2, pageSize: 10 })
  })
  expect(hook).toHaveBeenCalledTimes(callsBeforeTerminalAttempt)

  act(() => {
    result.current.handlePaginationChange({ pageIndex: 0, pageSize: 10 })
  })
  expect(lastPaginationRequest(hook)).toEqual(expect.objectContaining({ pageToken: undefined }))
}

type CursorPaginationState = {
  paginationNavigation: { canPreviousPage: boolean; canNextPage: boolean }
  handlePaginationChange: (value: { pageIndex: number; pageSize: number }) => void
}

type WebsiteScopeProps = {
  websiteScope?: { host: string; url: string; readOnly: true }
}

type CursorPaginationRenderHook<TProps = void> = {
  result: { current: CursorPaginationState }
  rerender: TProps extends void ? () => void : (props: TProps) => void
}

type AssetStateOwner = {
  useTarget: (options: WebsiteScopeProps) => CursorPaginationRenderHook<WebsiteScopeProps>
  useScan: () => CursorPaginationRenderHook
  targetHook: ReturnType<typeof vi.fn>
  scanHook: ReturnType<typeof vi.fn>
  setPlaceholder: (enabled: boolean) => void
}

function createAssetStateOwner(
  useTarget: AssetStateOwner["useTarget"],
  useScan: AssetStateOwner["useScan"],
  targetHook: ReturnType<typeof vi.fn>,
  scanHook: ReturnType<typeof vi.fn>,
): AssetStateOwner {
  let placeholder = false

  targetHook.mockImplementation((_id: number, request: Record<string, unknown>) => (
    response(request, "target-page-two", "results", placeholder)
  ))
  scanHook.mockImplementation((_id: number, request: Record<string, unknown>) => (
    response(request, "scan-page-two", "results", placeholder)
  ))

  return {
    useTarget,
    useScan,
    targetHook,
    scanHook,
    setPlaceholder: (enabled) => {
      placeholder = enabled
    },
  }
}

function assertPlaceholderAndScopeReset(owner: AssetStateOwner) {
  const scope = { host: "first.example", url: "https://first.example", readOnly: true } as const
  const { result, rerender } = owner.useTarget({ websiteScope: scope })

  act(() => {
    result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
  })
  expect(result.current.paginationNavigation.canPreviousPage).toBe(true)

  rerender({ websiteScope: { host: "second.example", url: "https://second.example", readOnly: true } })
  expect(owner.targetHook.mock.calls.at(-1)?.[1]).toEqual(expect.objectContaining({ pageToken: undefined }))
  expect(result.current.paginationNavigation.canPreviousPage).toBe(false)

  owner.setPlaceholder(true)
  rerender({ websiteScope: { host: "second.example", url: "https://second.example", readOnly: true } })
  expect(result.current.paginationNavigation.canNextPage).toBe(false)

  const callsBeforePlaceholderAttempt = owner.targetHook.mock.calls.length
  act(() => {
    result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
  })
  expect(owner.targetHook).toHaveBeenCalledTimes(callsBeforePlaceholderAttempt)

  const scan = owner.useScan()
  expect(scan.result.current.paginationNavigation).toEqual({
    mode: "cursor",
    canFirstPage: false,
    canPreviousPage: false,
    canNextPage: false,
  })
}

describe("cursor asset state owners", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    installDefaultResponses()
  })

  it("keeps website target and scan cursors adjacent and independent", () => {
    const target = renderHook(() => useWebSitesViewState({ targetId: 7 }))
    assertAdjacentCursor(target.result, websiteHooks.useTargetWebSites)

    const scan = renderHook(() => useWebSitesViewState({ scanId: 11 }))
    assertAdjacentCursor(scan.result, websiteHooks.useScanWebSites)
  })

  it("uses cursor navigation for subdomains", () => {
    const target = renderHook(() => useSubdomainsDetailViewState({ targetId: 7 }))
    assertAdjacentCursor(target.result, subdomainHooks.useTargetSubdomains)

    const scan = renderHook(() => useSubdomainsDetailViewState({ scanId: 11 }))
    assertAdjacentCursor(scan.result, subdomainHooks.useScanSubdomains)
  })

  it("uses cursor navigation for directories", () => {
    const target = renderHook(() => useDirectoriesViewState({ targetId: 7 }))
    assertAdjacentCursor(target.result, directoryHooks.useTargetDirectories)

    const scan = renderHook(() => useDirectoriesViewState({ scanId: 11 }))
    assertAdjacentCursor(scan.result, directoryHooks.useScanDirectories)
  })

  it("uses cursor navigation for endpoints", () => {
    const target = renderHook(() => useEndpointsDetailViewState({ targetId: 7 }))
    assertAdjacentCursor(target.result, targetHooks.useTargetEndpoints)

    const scan = renderHook(() => useEndpointsDetailViewState({ scanId: 11 }))
    assertAdjacentCursor(scan.result, endpointHooks.useScanEndpoints)
  })

  it("uses cursor navigation for IP addresses", () => {
    const target = renderHook(() => useIPAddressesViewState({ targetId: 7 }))
    assertAdjacentCursor(target.result, ipHooks.useTargetIPAddresses)

    const scan = renderHook(() => useIPAddressesViewState({ scanId: 11 }))
    assertAdjacentCursor(scan.result, ipHooks.useScanIPAddresses)
  })

  it("uses independent cursor navigation for global, target, and scan vulnerabilities", () => {
    const global = renderHook(() => useVulnerabilitiesDetailViewState({}))
    assertAdjacentCursor(global.result, vulnerabilityHooks.useAllVulnerabilities)

    const target = renderHook(() => useVulnerabilitiesDetailViewState({ targetId: 7 }))
    assertAdjacentCursor(target.result, vulnerabilityHooks.useTargetVulnerabilities)

    const scan = renderHook(() => useVulnerabilitiesDetailViewState({ scanId: 11 }))
    assertAdjacentCursor(scan.result, vulnerabilityHooks.useScanVulnerabilities)
  })

  it("resets website and subdomain parent scopes and ignores placeholder continuations", () => {
    let websitePlaceholder = false
    websiteHooks.useTargetWebSites.mockImplementation((_id, request) => (
      response(request, "website-page-two", "results", websitePlaceholder)
    ))
    const websites = renderHook(
      ({ targetId }: { targetId: number }) => useWebSitesViewState({ targetId }),
      { initialProps: { targetId: 7 } },
    )

    act(() => {
      websites.result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })
    expect(websites.result.current.paginationNavigation.canPreviousPage).toBe(true)

    websites.rerender({ targetId: 8 })
    expect(websiteHooks.useTargetWebSites.mock.calls.at(-1)?.[0]).toBe(8)
    expect(websiteHooks.useTargetWebSites.mock.calls.at(-1)?.[1]).toEqual(expect.objectContaining({ pageToken: undefined }))
    expect(websites.result.current.paginationNavigation.canPreviousPage).toBe(false)

    websitePlaceholder = true
    websites.rerender({ targetId: 8 })
    expect(websites.result.current.paginationNavigation.canNextPage).toBe(false)
    const websiteCallsBeforeAttempt = websiteHooks.useTargetWebSites.mock.calls.length
    act(() => {
      websites.result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })
    expect(websiteHooks.useTargetWebSites).toHaveBeenCalledTimes(websiteCallsBeforeAttempt)

    let subdomainPlaceholder = false
    subdomainHooks.useTargetSubdomains.mockImplementation((_id, request) => (
      response(request, "subdomain-page-two", "results", subdomainPlaceholder)
    ))
    const subdomains = renderHook(
      ({ targetId }: { targetId: number }) => useSubdomainsDetailViewState({ targetId }),
      { initialProps: { targetId: 7 } },
    )

    act(() => {
      subdomains.result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })
    expect(subdomains.result.current.paginationNavigation.canPreviousPage).toBe(true)

    subdomains.rerender({ targetId: 8 })
    expect(subdomainHooks.useTargetSubdomains.mock.calls.at(-1)?.[0]).toBe(8)
    expect(subdomainHooks.useTargetSubdomains.mock.calls.at(-1)?.[1]).toEqual(expect.objectContaining({ pageToken: undefined }))
    expect(subdomains.result.current.paginationNavigation.canPreviousPage).toBe(false)

    subdomainPlaceholder = true
    subdomains.rerender({ targetId: 8 })
    expect(subdomains.result.current.paginationNavigation.canNextPage).toBe(false)
    const subdomainCallsBeforeAttempt = subdomainHooks.useTargetSubdomains.mock.calls.length
    act(() => {
      subdomains.result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })
    expect(subdomainHooks.useTargetSubdomains).toHaveBeenCalledTimes(subdomainCallsBeforeAttempt)
  })

  it("clears vulnerability cursor reachability when switching global, target, and scan scopes", () => {
    const vulnerabilities = renderHook(
      ({ targetId, scanId }: { targetId?: number; scanId?: number }) => (
        useVulnerabilitiesDetailViewState({ targetId, scanId })
      ),
      { initialProps: {} },
    )

    act(() => {
      vulnerabilities.result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })
    expect(vulnerabilities.result.current.paginationNavigation.canPreviousPage).toBe(true)

    vulnerabilities.rerender({ targetId: 7 })
    expect(vulnerabilityHooks.useTargetVulnerabilities.mock.calls.at(-1)?.[1]).toEqual(expect.objectContaining({ pageToken: undefined }))
    expect(vulnerabilities.result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      vulnerabilities.result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })
    expect(vulnerabilities.result.current.paginationNavigation.canPreviousPage).toBe(true)

    vulnerabilities.rerender({ scanId: 11 })
    expect(vulnerabilityHooks.useScanVulnerabilities.mock.calls.at(-1)?.[1]).toEqual(expect.objectContaining({ pageToken: undefined }))
    expect(vulnerabilities.result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })
  })

  it("clears scoped asset caches and ignores placeholder continuations", () => {
    const owners = [
      createAssetStateOwner(
        (options) => renderHook(({ websiteScope }) => useDirectoriesViewState({ targetId: 7, websiteScope }), { initialProps: options }),
        () => renderHook(() => useDirectoriesViewState({ scanId: 11 })),
        directoryHooks.useTargetDirectories,
        directoryHooks.useScanDirectories,
      ),
      createAssetStateOwner(
        (options) => renderHook(({ websiteScope }) => useEndpointsDetailViewState({ targetId: 7, websiteScope }), { initialProps: options }),
        () => renderHook(() => useEndpointsDetailViewState({ scanId: 11 })),
        targetHooks.useTargetEndpoints,
        endpointHooks.useScanEndpoints,
      ),
      createAssetStateOwner(
        (options) => renderHook(({ websiteScope }) => useIPAddressesViewState({ targetId: 7, websiteScope }), { initialProps: options }),
        () => renderHook(() => useIPAddressesViewState({ scanId: 11 })),
        ipHooks.useTargetIPAddresses,
        ipHooks.useScanIPAddresses,
      ),
      createAssetStateOwner(
        (options) => renderHook(({ websiteScope }) => useVulnerabilitiesDetailViewState({ targetId: 7, websiteScope }), { initialProps: options }),
        () => renderHook(() => useVulnerabilitiesDetailViewState({ scanId: 11 })),
        vulnerabilityHooks.useTargetVulnerabilities,
        vulnerabilityHooks.useScanVulnerabilities,
      ),
    ]

    for (const owner of owners) {
      assertPlaceholderAndScopeReset(owner)
    }
  })
})
