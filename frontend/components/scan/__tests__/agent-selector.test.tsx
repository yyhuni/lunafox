import * as React from "react"
import { fireEvent, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { ScanAgentSelector } from "@/components/scan/agent-selector"
import { Command, CommandList } from "@/components/ui/command"
import { renderWithProviders } from "@/test/utils/render-with-providers"
import type { AgentDetail } from "@/types/agent.types"

const hookMocks = vi.hoisted(() => ({
  useScanAgentPicker: vi.fn(),
  useSelectedAgentDetail: vi.fn(),
  loadNextPage: vi.fn(),
  retryCurrentGeneration: vi.fn(),
  retryNextPage: vi.fn(),
}))

vi.mock("@/hooks/use-scan-agent-picker", () => ({
  useScanAgentPicker: hookMocks.useScanAgentPicker,
}))

vi.mock("@/hooks/use-agents", () => ({
  useSelectedAgentDetail: hookMocks.useSelectedAgentDetail,
}))

vi.mock("@/components/scan/scan-searchable-picker", () => ({
  ScanSearchablePicker: ({
    trigger,
    children,
    onOpenChange,
    onSearchValueChange,
    listRef,
  }: {
    trigger: React.ReactNode
    children: React.ReactNode
    onOpenChange?: (open: boolean) => void
    onSearchValueChange?: (value: string) => void
    listRef?: React.Ref<HTMLDivElement>
  }) => (
    <div>
      <button type="button" aria-label="picker-trigger" onClick={() => onOpenChange?.(true)}>
        {trigger}
      </button>
      <Command shouldFilter={false}>
        <input aria-label="picker-search" onChange={(event) => onSearchValueChange?.(event.currentTarget.value)} />
        <CommandList ref={listRef}>{children}</CommandList>
      </Command>
    </div>
  ),
}))

function detail(id: number, displayName = `selected-${id}`): AgentDetail {
  return {
    id,
    name: displayName,
    resourceName: `agents/${id}`,
    displayName,
    status: "online",
    maxTasks: 4,
    cpuThreshold: 80,
    memThreshold: 80,
    diskThreshold: 90,
    health: { state: "healthy" },
    createdAt: "2026-08-04T00:00:00Z",
    observedIpGeneration: 1,
    locationState: "unknown",
    location: null,
  }
}

function pickerState(open: boolean) {
  return {
    agents: [],
    normalizedSearch: "",
    loadedPageCount: open ? 1 : 0,
    hasNextPage: open,
    canLoadNextPage: open,
    isInitialLoading: false,
    isInitialError: false,
    isRefreshing: false,
    isRefreshError: false,
    isFetchingNextPage: false,
    isNextPageError: false,
    loadNextPage: hookMocks.loadNextPage,
    retryNextPage: hookMocks.retryNextPage,
    retryCurrentGeneration: hookMocks.retryCurrentGeneration,
  }
}

describe("ScanAgentSelector", () => {
  let intersectionCallback: IntersectionObserverCallback | null

  beforeEach(() => {
    vi.clearAllMocks()
    intersectionCallback = null
    class TestIntersectionObserver {
      constructor(callback: IntersectionObserverCallback) {
        intersectionCallback = callback
      }
      observe() {}
      unobserve() {}
      disconnect() {}
      takeRecords() { return [] }
      root = null
      rootMargin = ""
      thresholds = []
    }
    vi.stubGlobal("IntersectionObserver", TestIntersectionObserver)
    hookMocks.useScanAgentPicker.mockImplementation(({ open }: { open: boolean }) => pickerState(open))
    hookMocks.useSelectedAgentDetail.mockReturnValue({
      data: detail(1001, "renamed-selected-agent"),
      isInitialError: false,
    })
  })

  it("restores a selected Agent through detail while the candidate collection stays closed", () => {
    renderWithProviders(<ScanAgentSelector value={1001} onChange={vi.fn()} />)

    expect(hookMocks.useSelectedAgentDetail).toHaveBeenCalledWith("agents/1001")
    expect(hookMocks.useScanAgentPicker).toHaveBeenCalledWith({ open: false, search: "" })
    expect(screen.getByText("renamed-selected-agent")).toBeInTheDocument()
  })

  it("preserves and labels an unavailable canonical selection", () => {
    const onChange = vi.fn()
    hookMocks.useSelectedAgentDetail.mockReturnValue({ data: undefined, isInitialError: true })

    renderWithProviders(<ScanAgentSelector value={1001} onChange={onChange} />)

    expect(screen.getByText("agents/1001")).toBeInTheDocument()
    expect(screen.getByText("unavailableHint")).toBeInTheDocument()
    expect(onChange).not.toHaveBeenCalled()
  })

  it("opens collection search and suppresses duplicate observer notifications", async () => {
    renderWithProviders(<ScanAgentSelector value={null} onChange={vi.fn()} />)

    fireEvent.click(screen.getByRole("button", { name: "picker-trigger" }))
    await waitFor(() => {
      expect(hookMocks.useScanAgentPicker).toHaveBeenLastCalledWith({ open: true, search: "" })
      expect(intersectionCallback).not.toBeNull()
    })

    fireEvent.change(screen.getByRole("textbox", { name: "picker-search" }), { target: { value: "edge" } })
    expect(hookMocks.useScanAgentPicker).toHaveBeenLastCalledWith({ open: true, search: "edge" })

    intersectionCallback?.([{ isIntersecting: true } as IntersectionObserverEntry], {} as IntersectionObserver)
    intersectionCallback?.([{ isIntersecting: true } as IntersectionObserverEntry], {} as IntersectionObserver)
    await waitFor(() => expect(hookMocks.loadNextPage).toHaveBeenCalledTimes(1))
  })

  it("exposes contextual first-page recovery without changing the selection", () => {
    hookMocks.useScanAgentPicker.mockImplementation(({ open }: { open: boolean }) => ({
      ...pickerState(open),
      hasNextPage: false,
      canLoadNextPage: false,
      isInitialError: open,
    }))

    renderWithProviders(<ScanAgentSelector value={1001} onChange={vi.fn()} />)
    fireEvent.click(screen.getByRole("button", { name: "picker-trigger" }))
    fireEvent.click(screen.getByRole("button", { name: "retry" }))

    expect(hookMocks.retryCurrentGeneration).toHaveBeenCalledTimes(1)
    expect(screen.getByText("renamed-selected-agent")).toBeInTheDocument()
  })
})
