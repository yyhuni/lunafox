import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { AgentOverviewSection } from "@/components/settings/agents/agent-overview-section"
import type { AgentClusterSummary } from "@/types/agent.types"

vi.mock("next/navigation", () => ({
  useRouter: () => ({ back: vi.fn() }),
}))

function makeSummary(overrides: Partial<AgentClusterSummary> = {}): AgentClusterSummary {
  return {
    resourceName: "agentClusterSummaries/current",
    generatedAt: "2026-08-04T07:00:00Z",
    executionFreshnessSeconds: 15,
    totalNodes: 1201,
    healthyCount: 1000,
    warningCount: 100,
    offlineCount: 100,
    unknownCount: 1,
    staleAgentCount: 3,
    executionCapacity: {
      configuredSlots: 6000,
      occupiedSlots: 1200,
      availableSlots: 4000,
      unavailableSlots: 800,
      overcommittedSlots: 2,
    },
    clusterState: "needsAttention",
    reasonCodes: ["offline_agents", "stale_runtime_observations", "overcommitted_slots"],
    locationCoverage: {
      positionedCount: 1100,
      unpositionedCount: 101,
    },
    ...overrides,
  }
}

const baseProps = {
  error: null,
  isInitialError: false,
  isRefetchStale: false,
  lastSuccessfulAt: Date.parse("2026-08-04T07:00:00Z"),
  lastRefreshedAt: new Date("2026-08-04T07:00:00Z"),
  isRefreshing: false,
  onRefresh: vi.fn(),
  onRetry: vi.fn(),
}

describe("AgentOverviewSection", () => {
  it("renders the authoritative full-cluster counts and Server-owned conclusion", () => {
    render(<AgentOverviewSection {...baseProps} summary={makeSummary()} />)

    expect(screen.getByText("1201")).toBeInTheDocument()
    expect(screen.getByText("overview.clusterState.needsAttention")).toBeInTheDocument()
    expect(screen.getByText(/overview\.clusterState\.reasons\.offline_agents/)).toBeInTheDocument()
    expect(screen.getByText(/overview\.clusterState\.reasons\.stale_runtime_observations/)).toBeInTheDocument()
    expect(screen.getByText(/overview\.overcommittedSlots/)).toBeInTheDocument()
  })

  it("renders an actionable section error when no successful summary exists", () => {
    const onRetry = vi.fn()
    render(
      <AgentOverviewSection
        {...baseProps}
        summary={undefined}
        error={new Error("summary unavailable")}
        isInitialError
        onRetry={onRetry}
      />,
    )

    expect(screen.getByTestId("agent-overview-error-state")).toBeInTheDocument()
    expect(screen.queryByTestId("agent-overview-section")).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "Retry" }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it("retains a visibly stale snapshot and clears the marker atomically on recovery", () => {
    const onRetry = vi.fn()
    const { rerender } = render(
      <AgentOverviewSection
        {...baseProps}
        summary={makeSummary()}
        isRefetchStale
        onRetry={onRetry}
      />,
    )

    expect(screen.getByTestId("agent-overview-stale-alert")).toBeInTheDocument()
    expect(screen.getAllByText(/2026\/08\/04/)).not.toHaveLength(0)
    fireEvent.click(screen.getByRole("button", { name: "overview.stale.retry" }))
    expect(onRetry).toHaveBeenCalledTimes(1)

    rerender(
      <AgentOverviewSection
        {...baseProps}
        summary={makeSummary({ totalNodes: 1202, healthyCount: 1001 })}
      />,
    )

    expect(screen.queryByTestId("agent-overview-stale-alert")).not.toBeInTheDocument()
    expect(screen.getByText("1202")).toBeInTheDocument()
  })

  it("renders the explicit empty cluster state without treating it as a query failure", () => {
    render(
      <AgentOverviewSection
        {...baseProps}
        summary={makeSummary({
          totalNodes: 0,
          healthyCount: 0,
          warningCount: 0,
          offlineCount: 0,
          unknownCount: 0,
          staleAgentCount: 0,
          clusterState: "empty",
          reasonCodes: ["no_agents"],
          executionCapacity: {
            configuredSlots: 0,
            occupiedSlots: 0,
            availableSlots: 0,
            unavailableSlots: 0,
            overcommittedSlots: 0,
          },
          locationCoverage: { positionedCount: 0, unpositionedCount: 0 },
        })}
      />,
    )

    expect(screen.getByText("overview.clusterState.empty")).toBeInTheDocument()
    expect(screen.queryByTestId("agent-overview-error-state")).not.toBeInTheDocument()
  })
})
