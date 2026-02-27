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
    expect(command).toContain("/api/agent/install-script/remote?token=token")
    expect(command).not.toContain("mode=")
    expect(command).toContain("| bash")
    expect(command).not.toContain("LUNAFOX_AGENT_REGISTER_URL")
    expect(command).not.toContain("LUNAFOX_AGENT_SERVER_URL")

    const encoded = buildInstallCommand("token with space", "https://example.com/")
    expect(encoded).toContain("/api/agent/install-script/remote?token=token%20with%20space")
    expect(encoded).not.toContain("mode=")
  })
})
