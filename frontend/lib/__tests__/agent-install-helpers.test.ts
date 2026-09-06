import { describe, expect, it } from "vitest"
import {
  buildInstallCommand,
  normalizeOrigin,
} from "@/lib/agent-install-helpers"

describe("agent install helpers", () => {
  it("normalizes origins", () => {
    expect(normalizeOrigin("https://example.com/")).toBe("https://example.com")
  })

  it("builds install commands consistently", () => {
    expect(buildInstallCommand("", "https://example.com")).toBe("")
    expect(buildInstallCommand("token", "")).toBe("")

    const command = buildInstallCommand("token", "https://example.com/")
    expect(command).toContain(
      "/v1/agents:downloadInstallScript?registrationToken=token&profile=external",
    )
    expect(command).not.toContain("mode=")
    expect(command).not.toContain("token=")
    expect(command).toContain("| bash")
    expect(command).not.toContain("LUNAFOX_AGENT_REGISTER_URL")
    expect(command).not.toContain("LUNAFOX_AGENT_SERVER_URL")

    const encoded = buildInstallCommand("token with space", "https://example.com/")
    expect(encoded).toContain(
      "/v1/agents:downloadInstallScript?registrationToken=token%20with%20space&profile=external",
    )
    expect(encoded).not.toContain("mode=")
    expect(encoded).not.toContain("token=")
  })
})
