import { describe, expect, it } from "vitest"

import { getAgentConnectionIpDisplay } from "../agent-connection-ip"

const labels = {
  offline: "未连接",
  unobserved: "未观测到",
}

describe("Agent connection IP display", () => {
  it("uses the current connection address when one is available", () => {
    expect(getAgentConnectionIpDisplay({ status: "online", connectionIp: "172.20.0.5" }, labels)).toBe("172.20.0.5")
  })

  it("distinguishes a disconnected Agent from an online Agent without a safe observation", () => {
    expect(getAgentConnectionIpDisplay({ status: "offline" }, labels)).toBe("未连接")
    expect(getAgentConnectionIpDisplay({ status: "online" }, labels)).toBe("未观测到")
  })
})
