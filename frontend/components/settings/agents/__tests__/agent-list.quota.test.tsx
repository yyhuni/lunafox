import * as React from "react"
import { fireEvent, render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { AgentList } from "@/components/settings/agents/agent-list"
import type { AgentClusterSummary } from "@/types/agent.types"

const hookMocks = vi.hoisted(() => ({
  createTokenMutation: vi.fn(),
  deleteAgentMutation: vi.fn(),
  filterOptions: vi.fn(),
  list: vi.fn(),
  summary: vi.fn(),
}))

vi.mock("next/dynamic", () => ({
  default: () => ({ open }: { open?: boolean }) => (
    open ? <div role="dialog">deferred dialog</div> : null
  ),
}))

vi.mock("@/components/ui/button", () => {
  const Button = React.forwardRef<
    HTMLButtonElement,
    React.ButtonHTMLAttributes<HTMLButtonElement> & { size?: string; variant?: string }
  >(function MockButton({ size: _size, variant: _variant, ...props }, ref) {
    void _size
    void _variant
    return <button ref={ref} {...props} />
  })

  return { Button }
})

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children }: React.PropsWithChildren) => <>{children}</>,
  DialogTrigger: ({ children, render: trigger }: { children: React.ReactNode; render: React.ReactElement }) =>
    React.cloneElement(trigger, undefined, children),
}))

vi.mock("@/components/shared/loading/content-handoff", () => ({
  ContentHandoff: ({ children }: React.PropsWithChildren) => <>{children}</>,
}))

vi.mock("@/components/shared/loading/loading-owner", () => ({
  getLoadingStructureSlotAttributes: () => ({}),
}))

vi.mock("@/components/shared/data-table", () => ({
  DataTableFacetedFilter: () => null,
  DataTableFacetedFilterGroup: ({ children }: React.PropsWithChildren) => <>{children}</>,
}))

vi.mock("@/components/ui/card", () => ({
  Card: ({ children }: React.PropsWithChildren) => <div>{children}</div>,
}))

vi.mock("@/components/shared/feedback/confirm-dialog", () => ({
  ConfirmDialog: () => null,
}))

vi.mock("@/components/shared/search-input", () => ({
  SearchInput: (props: React.InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
}))

vi.mock("@/components/settings/agents/agent-card-compact", () => ({
  AgentCardCompact: () => <div data-testid="agent-card" />,
}))

vi.mock("@/components/settings/agents/agent-list-loading-state", () => ({
  AgentCardsLoadingState: () => null,
  AgentToolbarLoadingState: () => null,
}))

vi.mock("@/components/settings/agents/agent-overview-section", () => ({
  AgentOverviewSection: ({ summary, showLicenseBanner = true }: { summary?: AgentClusterSummary; showLicenseBanner?: boolean }) => summary?.license && showLicenseBanner ? <span>license-capability</span> : null,
}))

vi.mock("@/components/settings/agents/agent-overview-loading-state", () => ({
  AgentOverviewLoadingState: () => null,
}))

vi.mock("@/components/settings/agents/architecture-dialog", () => ({
  ArchitectureDialog: ({ trigger }: { trigger: React.ReactNode }) => <>{trigger}</>,
}))

vi.mock("@/components/settings/agents/agent-results-region", () => ({
  AgentResultsRegion: ({ children }: React.PropsWithChildren) => <>{children}</>,
}))

vi.mock("@/hooks/use-agents", () => ({
  useAgentClusterSummary: hookMocks.summary,
  useAgentFilterOptions: hookMocks.filterOptions,
  useAgents: hookMocks.list,
  useCreateRegistrationToken: () => ({ mutateAsync: hookMocks.createTokenMutation, isPending: false }),
  useDeleteAgent: () => ({ mutateAsync: hookMocks.deleteAgentMutation, isPending: false }),
}))

vi.mock("@/hooks/use-agent-management-refresh", () => ({
  useAgentManagementRefresh: () => ({
    isRefreshing: false,
    lastRefreshedAt: new Date("2026-09-19T00:00:00Z"),
    refresh: vi.fn(),
  }),
}))

vi.mock("@/hooks/use-agent-install-connection", () => ({
  useAgentInstallConnection: () => null,
}))

vi.mock("@/hooks/use-deferred-interaction-mount", () => ({
  deferredInteractionUnmountDelayMs: 240,
  useDeferredInteractionMount: (active: boolean) => active,
}))

function makeSummary(totalNodes: number): AgentClusterSummary {
  return {
    totalNodes,
    agentLimit: 3,
  } as AgentClusterSummary
}

function summaryState(data: AgentClusterSummary | undefined) {
  return {
    data,
    error: null,
    isInitialError: data === undefined,
    isRefetchStale: false,
    isSuccess: data !== undefined,
    lastSuccessfulAt: data ? Date.parse("2026-09-19T00:00:00Z") : null,
    refetch: vi.fn(),
  }
}

function listState(results: Array<{ id: number }> = [{ id: 1 }]) {
  return {
    data: { results, nextPageToken: undefined },
    isSuccess: false,
    refetch: vi.fn(),
  }
}

function getAddButtons() {
  return {
    expansion: screen.getByRole("button", { name: "expansion.title" }),
    primary: screen.getByRole("button", { name: "install.openDialog" }),
  }
}

function getQuotaHint() {
  return document.getElementById("agent-quota-status")
}

function quotaMessage(used: number) {
  return `quota.reached:{"used":${used},"limit":3}`
}

describe("AgentList quota feedback", () => {
  it("exposes the Server license capability on the production page while preserving old Server compatibility", () => {
    hookMocks.summary.mockReturnValue(summaryState({ ...makeSummary(3), license: {
      resourceName: "agentLicenses/current", state: "inactive", agentLimit: 3, months: null, expiresAt: null, licenseId: null,
    } }))
    const view = render(<AgentList />)
    expect(screen.getByText("license-capability")).toBeInTheDocument()
    hookMocks.summary.mockReturnValue(summaryState(makeSummary(3)))
    view.rerender(<AgentList />)
    expect(screen.queryByText("license-capability")).not.toBeInTheDocument()
  })

  it.each([
    { count: 2, state: "inactive", visible: false },
    { count: 3, state: "inactive", visible: true },
    { count: 5, state: "inactive", visible: true },
    { count: 2, state: "active", visible: false },
    { count: 3, state: "active", visible: true },
    { count: 2, state: "expired", visible: true },
  ] as const)("shows the on-demand banner for count=$count state=$state: $visible", ({ count, state, visible }) => {
    hookMocks.summary.mockReturnValue(summaryState({ ...makeSummary(count), license: {
      resourceName: "agentLicenses/current", state, agentLimit: 3, months: null, expiresAt: null, licenseId: null,
    } }))
    render(<AgentList />)
    expect(Boolean(screen.queryByText("license-capability"))).toBe(visible)
    expect(screen.queryByRole("button", { name: /toolbarQuota|reveal|查看授权|额度/ })).not.toBeInTheDocument()
  })

  it("updates automatic banner visibility from the authoritative summary", () => {
    hookMocks.summary.mockReturnValue(summaryState({ ...makeSummary(3), license: {
      resourceName: "agentLicenses/current", state: "inactive", agentLimit: 3, months: null, expiresAt: null, licenseId: null,
    } }))
    const view = render(<AgentList />)
    expect(screen.getByText("license-capability")).toBeInTheDocument()
    hookMocks.summary.mockReturnValue(summaryState({ ...makeSummary(3), agentLimit: 10, license: {
      resourceName: "agentLicenses/current", state: "active", agentLimit: 10, months: 1, expiresAt: "2026-11-19T00:00:00Z", licenseId: "demo",
    } }))
    view.rerender(<AgentList />)
    expect(screen.queryByText("license-capability")).not.toBeInTheDocument()
    expect(getAddButtons().primary).toBeEnabled()
    hookMocks.summary.mockReturnValue(summaryState(makeSummary(3)))
    view.rerender(<AgentList />)
    expect(screen.queryByText("license-capability")).not.toBeInTheDocument()
  })
  beforeEach(() => {
    vi.clearAllMocks()
    hookMocks.filterOptions.mockReturnValue({ data: { results: [] }, refetch: vi.fn() })
    hookMocks.list.mockReturnValue(listState())
    hookMocks.summary.mockReturnValue(summaryState(makeSummary(3)))
  })

  it("keeps known-full and over-limit add entries disabled without opening installation or creating a token", () => {
    const view = render(<AgentList />)
    const initialButtons = getAddButtons()

    expect(getQuotaHint()).toHaveAccessibleName(quotaMessage(3))
    expect(initialButtons.primary).toBeDisabled()
    expect(initialButtons.expansion).toBeDisabled()
    expect(initialButtons.primary).toHaveAccessibleDescription(quotaMessage(3))
    expect(initialButtons.expansion).toHaveAccessibleDescription(quotaMessage(3))

    fireEvent.click(initialButtons.primary)
    fireEvent.click(initialButtons.expansion)
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    expect(hookMocks.createTokenMutation).not.toHaveBeenCalled()

    hookMocks.summary.mockReturnValue(summaryState(makeSummary(5)))
    view.rerender(<AgentList />)

    expect(getQuotaHint()).toHaveAccessibleName(quotaMessage(5))
    expect(getAddButtons().primary).toHaveAccessibleDescription(quotaMessage(5))
    expect(getAddButtons().primary).toBeDisabled()
    expect(getAddButtons().expansion).toBeDisabled()
  })

  it("keeps add entries unavailable without fabricating full-quota feedback when the summary is missing", () => {
    hookMocks.summary.mockReturnValue(summaryState(undefined))

    render(<AgentList />)

    const buttons = getAddButtons()
    expect(buttons.primary).toBeDisabled()
    expect(buttons.expansion).toBeDisabled()
    expect(getQuotaHint()).toBeNull()
    expect(buttons.primary).not.toHaveAttribute("aria-describedby")
    expect(buttons.expansion).not.toHaveAttribute("aria-describedby")
  })

  it("removes quota feedback and restores installation after authoritative capacity is released", () => {
    const view = render(<AgentList />)

    hookMocks.summary.mockReturnValue(summaryState(makeSummary(2)))
    view.rerender(<AgentList />)

    const buttons = getAddButtons()
    expect(getQuotaHint()).toBeNull()
    expect(buttons.primary).toBeEnabled()
    expect(buttons.expansion).toBeEnabled()
    expect(buttons.primary).not.toHaveAttribute("aria-describedby")
    expect(buttons.expansion).not.toHaveAttribute("aria-describedby")

    fireEvent.click(buttons.expansion)
    expect(screen.getByRole("dialog")).toBeInTheDocument()
    expect(hookMocks.createTokenMutation).not.toHaveBeenCalled()
  })
})
