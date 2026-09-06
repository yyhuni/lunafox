import { act } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  AGENT_INSTALL_POLL_INTERVAL_MS,
  AGENT_INSTALL_POST_EXPIRY_GRACE_MS,
  useAgentInstallConnection,
} from "@/hooks/use-agent-install-connection"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import type {
  Agent,
  RegistrationTokenResource,
  RegistrationTokenResponse,
} from "@/types/agent.types"

const agentServiceMocks = vi.hoisted(() => ({
  getAgents: vi.fn(),
  getRegistrationToken: vi.fn(),
}))

vi.mock("@/services/agent.service", () => ({
  agentService: agentServiceMocks,
}))

const NOW = Date.parse("2026-08-04T08:00:00.000Z")

function createAgent(id: number, status: "online" | "offline"): Agent {
  return {
    id,
    name: `agent-${id}`,
    displayName: `agent-${id}`,
    resourceName: `agents/${id}`,
    status,
    maxTasks: 3,
    cpuThreshold: 80,
    memThreshold: 80,
    diskThreshold: 80,
    health: { state: status === "online" ? "healthy" : "offline" },
    createdAt: `2026-08-04T07:00:${String(id).padStart(2, "0")}Z`,
  }
}

function createToken(
  id: number,
  expiresAt: number,
): RegistrationTokenResponse {
  return {
    id,
    resourceName: `agentRegistrationTokens/${id}`,
    token: `secret-${id}`,
    expiresAt: new Date(expiresAt).toISOString(),
  }
}

function createResource(
  token: RegistrationTokenResponse,
  agents: Agent[],
  state: RegistrationTokenResource["state"] = "active",
): RegistrationTokenResource {
  return {
    id: token.id,
    resourceName: token.resourceName,
    expiresAt: token.expiresAt,
    state,
    agents,
  }
}

async function flushMicrotasks() {
  await Promise.resolve()
  await Promise.resolve()
  await vi.advanceTimersByTimeAsync(1)
}

async function waitForCondition(predicate: () => boolean, attempts = 30) {
  for (let index = 0; index < attempts; index += 1) {
    if (predicate()) return
    await act(flushMicrotasks)
  }
  expect(predicate()).toBe(true)
}

describe("useAgentInstallConnection", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(NOW)
    vi.resetAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("keeps polling the token resource while a valid reusable token has only online Agents", async () => {
    const token = createToken(101, NOW + 60_000)
    agentServiceMocks.getRegistrationToken.mockResolvedValue(
      createResource(token, [createAgent(1, "online")]),
    )

    const { result } = renderHookWithProviders(() => useAgentInstallConnection(token, true))

    await waitForCondition(() => result.current.phase === "online")
    expect(result.current.isLive).toBe(true)
    expect(result.current.endReason).toBeNull()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(AGENT_INSTALL_POLL_INTERVAL_MS)
    })
    await waitForCondition(() => agentServiceMocks.getRegistrationToken.mock.calls.length >= 2)

    expect(agentServiceMocks.getAgents).not.toHaveBeenCalled()
  })

  it("retains the last successful complete projection through a failure and replaces it on recovery", async () => {
    const token = createToken(102, NOW + 60_000)
    agentServiceMocks.getRegistrationToken
      .mockResolvedValueOnce(createResource(token, [createAgent(2, "offline")]))
      .mockRejectedValueOnce(new Error("temporary failure"))
      .mockResolvedValueOnce(createResource(token, [createAgent(2, "online"), createAgent(3, "offline")]))

    const { result } = renderHookWithProviders(() => useAgentInstallConnection(token, true))
    await waitForCondition(() => result.current.agents.length === 1)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(AGENT_INSTALL_POLL_INTERVAL_MS)
    })
    await waitForCondition(() => result.current.hasDetectionError)
    expect(result.current.agents.map((agent) => agent.id)).toEqual([2])
    expect(result.current.isLastKnown).toBe(true)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(AGENT_INSTALL_POLL_INTERVAL_MS)
    })
    await waitForCondition(() => result.current.agents.length === 2 && !result.current.hasDetectionError)

    expect(result.current.agents.map((agent) => agent.id)).toEqual([2, 3])
    expect(result.current.phase).toBe("registered")
  })

  it("ends after expiry on the first non-empty all-online projection and survives close/reopen", async () => {
    const token = createToken(103, NOW + 1_000)
    agentServiceMocks.getRegistrationToken
      .mockResolvedValueOnce(createResource(token, [createAgent(4, "offline")]))
      .mockResolvedValue(createResource(token, [createAgent(4, "online")], "expired"))

    const { result, rerender } = renderHookWithProviders(
      ({ open }: { open: boolean }) => useAgentInstallConnection(token, open),
      { initialProps: { open: true } },
    )
    await waitForCondition(() => result.current.phase === "registered")
    expect(result.current.endReason).toBeNull()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(AGENT_INSTALL_POLL_INTERVAL_MS)
    })
    await waitForCondition(() => result.current.endReason === "allOnline")

    expect(result.current.isLive).toBe(false)
    expect(result.current.isTokenExpired).toBe(true)
    const completedCalls = agentServiceMocks.getRegistrationToken.mock.calls.length

    rerender({ open: false })
    rerender({ open: true })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10_000)
    })

    expect(result.current.agents.map((agent) => agent.id)).toEqual([4])
    expect(result.current.endReason).toBe("allOnline")
    expect(agentServiceMocks.getRegistrationToken).toHaveBeenCalledTimes(completedCalls)
  })

  it("stops at the two-minute post-expiry deadline without discarding the last success", async () => {
    const token = createToken(104, NOW - AGENT_INSTALL_POST_EXPIRY_GRACE_MS + 1_000)
    agentServiceMocks.getRegistrationToken.mockResolvedValue(
      createResource(token, [createAgent(5, "offline")], "expired"),
    )

    const { result } = renderHookWithProviders(() => useAgentInstallConnection(token, true))
    await waitForCondition(() => result.current.agents.length === 1)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1_000)
    })
    await waitForCondition(() => result.current.endReason === "deadline")

    expect(result.current.agents.map((agent) => agent.id)).toEqual([5])
    expect(result.current.isLive).toBe(false)
    expect(result.current.phase).toBe("registered")
    const deadlineCalls = agentServiceMocks.getRegistrationToken.mock.calls.length

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10_000)
    })
    expect(agentServiceMocks.getRegistrationToken).toHaveBeenCalledTimes(deadlineCalls)
  })

  it("cancels in-flight token reads on replacement and close without querying the Agent collection", async () => {
    const first = createToken(105, NOW + 60_000)
    const second = createToken(106, NOW + 60_000)
    const signals = new Map<string, AbortSignal>()
    agentServiceMocks.getRegistrationToken.mockImplementation((resourceName: string, signal: AbortSignal) => {
      signals.set(resourceName, signal)
      return new Promise((_resolve, reject) => {
        signal.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")))
      })
    })

    const { rerender } = renderHookWithProviders(
      ({ token, open }: { token: RegistrationTokenResponse; open: boolean }) => (
        useAgentInstallConnection(token, open)
      ),
      { initialProps: { token: first, open: true } },
    )
    await waitForCondition(() => signals.has(first.resourceName))

    rerender({ token: second, open: true })
    await waitForCondition(() => signals.get(first.resourceName)?.aborted === true)
    await waitForCondition(() => signals.has(second.resourceName))

    rerender({ token: second, open: false })
    await waitForCondition(() => signals.get(second.resourceName)?.aborted === true)

    expect(agentServiceMocks.getAgents).not.toHaveBeenCalled()
  })
})
