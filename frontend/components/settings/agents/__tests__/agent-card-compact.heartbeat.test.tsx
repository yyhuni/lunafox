import type { ReactNode } from "react"
import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import type { Agent } from "@/types/agent.types"

vi.mock("next-intl", () => ({
  useNow: () => new Date("2026-08-09T00:00:00Z"),
  useFormatter: () => ({
    dateTime: (value: Date) => value.toISOString(),
    number: (value: number) => String(value),
    relativeTime: () => "just now",
  }),
  useTranslations: () => (key: string) => key,
}))

vi.mock("@/components/ui/card", () => ({
  Card: ({ children }: { children: ReactNode }) => <article>{children}</article>,
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: ({ children }: { children: ReactNode }) => <span>{children}</span>,
}))

vi.mock("@/components/shared/feedback/status", () => ({
  Status: ({ children }: { children: ReactNode }) => <span>{children}</span>,
  StatusLabel: ({ children }: { children: ReactNode }) => <span>{children}</span>,
}))

vi.mock("@/components/shared/data-table/menu-owners", () => ({
  DenseRowActionMenu: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenuItem: ({ children }: { children: ReactNode }) => <button type="button">{children}</button>,
  DropdownMenuLabel: ({ children }: { children: ReactNode }) => <span>{children}</span>,
  DropdownMenuSeparator: () => <hr />,
}))

vi.mock("@/components/shared/metrics/segmented-metric-progress", () => ({
  SegmentedMetricProgress: ({ label }: { label: string }) => <span>{label}</span>,
}))

import { AgentCardCompact } from "../agent-card-compact"

const baseAgent: Agent = {
  id: 7,
  name: "agents/7",
  displayName: "edge-7",
  status: "online",
  observedHostname: "edge-7",
  connectionIp: "192.0.2.7",
  agentVersion: "1.0.0",
  maxTasks: 4,
  cpuThreshold: 90,
  memThreshold: 90,
  diskThreshold: 90,
  lastHeartbeat: "2026-08-08T23:59:00Z",
  health: { state: "healthy" },
  createdAt: "2026-08-01T00:00:00Z",
}

const actions = {
  onConfig: vi.fn(),
  onDelete: vi.fn(),
  onLogs: vi.fn(),
}

describe("AgentCardCompact heartbeat projection", () => {
  it("renders the unavailable state and preserves only the last observation when heartbeat is absent", () => {
    render(<AgentCardCompact agentNode={baseAgent} {...actions} />)

    expect(screen.getByTestId("agent-realtime-metrics-unavailable")).toHaveTextContent("card.realtimeMetricsUnavailable")
    expect(screen.getByText("metrics.lastHeartbeat")).toBeInTheDocument()
    expect(screen.queryByText("metrics.cpu")).not.toBeInTheDocument()
    expect(screen.queryByText("metrics.mem")).not.toBeInTheDocument()
    expect(screen.queryByText("metrics.disk")).not.toBeInTheDocument()
    expect(screen.queryByText("metrics.runningTasks")).not.toBeInTheDocument()
    expect(screen.queryByText("metrics.usedTaskSlots")).not.toBeInTheDocument()
    expect(screen.queryByText("card.uptime")).not.toBeInTheDocument()
  })

  it("renders realtime metrics when the Redis heartbeat projection is available", () => {
    render(
      <AgentCardCompact
        agentNode={{
          ...baseAgent,
          heartbeat: {
            cpu: 11,
            mem: 22,
            disk: 33,
            runningTasks: 1,
            taskSlotsUsed: 2,
            uptime: 120,
            updatedAt: "2026-08-09T00:00:00Z",
          },
        }}
        {...actions}
      />,
    )

    expect(screen.queryByTestId("agent-realtime-metrics-unavailable")).not.toBeInTheDocument()
    expect(screen.getByText("metrics.cpu")).toBeInTheDocument()
    expect(screen.getByText("metrics.mem")).toBeInTheDocument()
    expect(screen.getByText("metrics.disk")).toBeInTheDocument()
    expect(screen.getByText("metrics.runningTasks")).toBeInTheDocument()
    expect(screen.getByText("metrics.usedTaskSlots")).toBeInTheDocument()
    expect(screen.getByText("card.uptime")).toBeInTheDocument()
    expect(screen.getByText("metrics.lastHeartbeat")).toBeInTheDocument()
  })
})
