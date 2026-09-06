import { describe, expect, it } from "vitest"
import {
  agentRegistrationTokenName,
  parseAgentName,
  parseAgentRegistrationTokenName,
} from "@/lib/resource-name"

describe("Agent resource-name contract", () => {
  it("round-trips canonical Agent and registration-token identities", () => {
    expect(parseAgentName("agents/42")).toBe(42)
    expect(agentRegistrationTokenName(101)).toBe("agentRegistrationTokens/101")
    expect(parseAgentRegistrationTokenName("agentRegistrationTokens/101")).toBe(101)
  })

  it.each([
    "agents/01",
    "agents/-1",
    "agentRegistrationTokens/01",
    "agentRegistrationTokens/secret-value",
  ])("rejects non-canonical or secret-like identity %s", (value) => {
    const parse = value.startsWith("agents/") ? parseAgentName : parseAgentRegistrationTokenName
    expect(() => parse(value)).toThrow()
  })
})
