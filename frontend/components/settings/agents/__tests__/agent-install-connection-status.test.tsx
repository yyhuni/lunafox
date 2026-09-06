import type { ReactNode } from "react"
import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { AgentInstallConnectionStatus } from "@/components/settings/agents/agent-install-connection-status"
import type { AgentInstallConnectionViewState } from "@/hooks/use-agent-install-connection"

vi.mock("framer-motion", () => ({
  AnimatePresence: ({ children }: { children: ReactNode }) => children,
  motion: {
    div: ({ children }: { children: ReactNode }) => (
      <div data-testid="motion-wrapper">{children}</div>
    ),
  },
  useReducedMotion: () => true,
}))

const baseConnection: AgentInstallConnectionViewState = {
  phase: "waiting",
  agents: [],
  hasSuccessfulProjection: false,
  hasDetectionError: false,
  isInitialError: false,
  isLastKnown: false,
  lastSuccessfulAt: null,
  isLive: true,
  isTokenExpired: false,
  endReason: null,
}

describe("AgentInstallConnectionStatus", () => {
  it("renders a non-live deadline result accessibly without motion when reduced motion is requested", () => {
    const { container } = render(
      <AgentInstallConnectionStatus
        connection={{
          ...baseConnection,
          hasSuccessfulProjection: true,
          isLive: false,
          isTokenExpired: true,
          endReason: "deadline",
        }}
      />,
    )

    expect(container.querySelector('[aria-live="polite"]')).toBeInTheDocument()
    expect(screen.getByText("endedTitle")).toBeInTheDocument()
    expect(screen.getByText("noAgentsFinal")).toBeInTheDocument()
    expect(screen.getByText("deadlineDescription")).toBeInTheDocument()
    expect(screen.queryByTestId("motion-wrapper")).not.toBeInTheDocument()
  })

  it("keeps the last successful projection visible with explicit refresh-failure context", () => {
    render(
      <AgentInstallConnectionStatus
        connection={{
          ...baseConnection,
          phase: "registered",
          agents: [{
            id: 7,
            name: "edge-7",
            address: "10.0.0.7",
            status: "registered",
            createdAt: "2026-08-04T07:00:00Z",
          }],
          hasSuccessfulProjection: true,
          hasDetectionError: true,
          isInitialError: false,
          isLastKnown: true,
          lastSuccessfulAt: Date.parse("2026-08-04T07:00:00Z"),
        }}
      />,
    )

    expect(screen.getByText("edge-7")).toBeInTheDocument()
    expect(screen.getByText(/lastKnownError/)).toBeInTheDocument()
  })

  it("shows an explicit empty final state when no token projection ever succeeded", () => {
    render(
      <AgentInstallConnectionStatus
        connection={{
          ...baseConnection,
          hasDetectionError: true,
          isInitialError: true,
          isLive: false,
          isTokenExpired: true,
          endReason: "deadline",
        }}
      />,
    )

    expect(screen.getByText("deadlineEmptyDescription")).toBeInTheDocument()
    expect(screen.queryByText("initialError")).not.toBeInTheDocument()
  })
})
