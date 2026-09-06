import { DEFAULT_MOCK_SCENARIO, resolveMockScenarioFromEnv } from "../config"

export const mockScenarioIds = ["happy", "empty", "stress", "edge", "error"] as const

export type MockScenarioId = (typeof mockScenarioIds)[number]

let activeMockScenario = resolveScenario(resolveMockScenarioFromEnv())

export function resolveScenario(input?: string | null): MockScenarioId {
  if (!input) {
    return DEFAULT_MOCK_SCENARIO as MockScenarioId
  }

  if (mockScenarioIds.includes(input as MockScenarioId)) {
    return input as MockScenarioId
  }

  return DEFAULT_MOCK_SCENARIO as MockScenarioId
}

export function getMockScenario(): MockScenarioId {
  return activeMockScenario
}

export function setMockScenario(nextScenario: string | null | undefined): MockScenarioId {
  activeMockScenario = resolveScenario(nextScenario)
  return activeMockScenario
}

export function resetMockScenario(): MockScenarioId {
  activeMockScenario = resolveScenario(resolveMockScenarioFromEnv())
  return activeMockScenario
}
