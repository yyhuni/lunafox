import { render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { AgentLocationMap } from "@/components/overview/agent-location-map"
import type { AgentClusterSummary, AgentLocationMap as AgentLocationMapProjection } from "@/types/agent.types"

const hookMocks = vi.hoisted(() => ({
  summary: vi.fn(),
  locationMap: vi.fn(),
}))

vi.mock("@/hooks/use-agents", () => ({
  useAgentClusterSummary: hookMocks.summary,
  useAgentLocationMap: hookMocks.locationMap,
}))

vi.mock("@/components/overview/world-map", () => ({
  default: ({ markers = [], dots = [] }: {
    markers?: Array<{
      id: string
      locationState?: string
      ariaLabel?: string
      details?: string[]
    }>
    dots?: Array<{ id?: string; active?: boolean }>
  }) => (
    <div data-testid="world-map" data-marker-count={markers.length} data-connection-count={dots.length}>
      {markers.map((marker) => (
        <span
          key={marker.id}
          data-testid={`marker-${marker.id}`}
          data-location-state={marker.locationState}
          aria-label={marker.ariaLabel}
        >
          {marker.details?.join(" | ")}
        </span>
      ))}
      {dots.map((dot) => (
        <span key={dot.id} data-testid="map-connection" data-active={dot.active ? "true" : "false"} />
      ))}
    </div>
  ),
}))

function queryState<T>(data: T) {
  return {
    data,
    error: null,
    isError: false,
    isInitialError: false,
    isPending: false,
    isRefetchStale: false,
    lastSuccessfulAt: Date.parse("2026-08-04T07:00:00Z"),
    refetch: vi.fn(),
  }
}

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
    staleAgentCount: 0,
    executionCapacity: {
      configuredSlots: 6000,
      occupiedSlots: 1000,
      availableSlots: 4200,
      unavailableSlots: 800,
      overcommittedSlots: 0,
    },
    clusterState: "needsAttention",
    reasonCodes: ["offline_agents"],
    locationCoverage: { positionedCount: 3, unpositionedCount: 1198 },
    ...overrides,
  }
}

function makeMap(serverLocation: AgentLocationMapProjection["serverLocation"]): AgentLocationMapProjection {
  return {
    resourceName: "agentLocationMaps/current",
    generatedAt: "2026-08-04T07:00:00Z",
    serverLocation,
    agents: [
      {
        id: 1001,
        resourceName: "agents/1001",
        displayName: "edge-current",
        status: "online",
        healthState: "healthy",
        taskSlotsUsed: 2,
        location: {
          state: "current",
          latitude: 39.9042,
          longitude: 116.4074,
          accuracyRadiusKm: null,
          sourceObservedIp: "1.1.1.1",
          providerKey: "freeipapi",
          resolvedAt: "2026-08-02T07:00:00Z",
        },
      },
      {
        id: 1002,
        resourceName: "agents/1002",
        displayName: "edge-expired",
        status: "online",
        healthState: "warning",
        taskSlotsUsed: 0,
        location: {
          state: "expired",
          latitude: 35.6762,
          longitude: 139.6503,
          accuracyRadiusKm: 25,
          sourceObservedIp: "8.8.8.8",
          providerKey: "freeipapi",
          resolvedAt: "2026-07-01T07:00:00Z",
        },
      },
      {
        id: 1003,
        resourceName: "agents/1003",
        displayName: "edge-offline",
        status: "offline",
        healthState: "critical",
        taskSlotsUsed: 4,
        location: {
          state: "current",
          latitude: 40.7128,
          longitude: -74.006,
          accuracyRadiusKm: null,
          sourceObservedIp: "9.9.9.9",
          providerKey: "freeipapi",
          resolvedAt: "2026-08-01T07:00:00Z",
        },
      },
    ],
  }
}

describe("AgentLocationMap", () => {
  beforeEach(() => {
    hookMocks.summary.mockReturnValue(queryState(makeSummary()))
  })

  it("renders every positioned Agent in Agent-only mode when Server location is unknown", () => {
    hookMocks.locationMap.mockReturnValue(queryState(makeMap(null)))

    render(<AgentLocationMap />)

    expect(screen.getByTestId("world-map")).toHaveAttribute("data-marker-count", "3")
    expect(screen.getByTestId("world-map")).toHaveAttribute("data-connection-count", "0")
    expect(screen.queryByTestId("marker-server-location")).not.toBeInTheDocument()
    expect(screen.queryByText("serverUnknown")).not.toBeInTheDocument()
    expect(screen.queryByText("expiredAgentLocations")).not.toBeInTheDocument()
    expect(screen.getByText(/runtimeUnreported/)).toHaveTextContent('"count":1')
    expect(screen.queryByText("status.unknown")).not.toBeInTheDocument()
    expect(screen.getByTestId("marker-agent-location-1002")).toHaveAttribute("data-location-state", "expired")
  })

  it("keeps expired/offline markers, draws all eligible connections, and activates only used slots", () => {
    hookMocks.summary.mockReturnValue(queryState(makeSummary({
      totalNodes: 1200,
      unknownCount: 0,
      locationCoverage: { positionedCount: 3, unpositionedCount: 1197 },
    })))
    hookMocks.locationMap.mockReturnValue(queryState(makeMap({
      state: "expired",
      observedEgressIp: "4.2.2.2",
      latitude: 31.2304,
      longitude: 121.4737,
      accuracyRadiusKm: null,
      providerKey: "freeipapi",
      resolvedAt: "2026-07-01T07:00:00Z",
    })))

    render(<AgentLocationMap />)

    expect(screen.getByTestId("world-map")).toHaveAttribute("data-marker-count", "4")
    expect(screen.getByTestId("world-map")).toHaveAttribute("data-connection-count", "2")
    expect(screen.getByTestId("marker-server-location")).toHaveAttribute("data-location-state", "expired")
    expect(screen.getByTestId("marker-agent-location-1003")).toBeInTheDocument()
    expect(screen.getAllByTestId("map-connection").filter((connection) => connection.dataset.active === "true")).toHaveLength(1)
    expect(screen.queryByText(/runtimeUnreported/)).not.toBeInTheDocument()
    expect(screen.queryByText("status.unknown")).not.toBeInTheDocument()
    expect(screen.getByTestId("marker-agent-location-1001")).toHaveTextContent("inference")
    expect(screen.getByText(/coverageValue/)).toHaveTextContent("positioned")
  })
})
