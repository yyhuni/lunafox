import { describe, expect, it } from "vitest"

import en from "@/messages/en.json"
import zh from "@/messages/zh.json"

const statuses = [
  "queued",
  "stopping",
  "preflight",
  "updating",
  "migrating",
  "restarting",
  "agent_verifying",
  "verifying",
  "succeeded",
  "failed",
  "needs_recovery",
  "needs_attention",
] as const

describe("system upgrade messages", () => {
  it("defines a badge label for every server operation status", () => {
    for (const status of statuses) {
      expect(en.systemUpgrade.status[status]).toBeTruthy()
      expect(zh.systemUpgrade.status[status]).toBeTruthy()
    }
  })
})
