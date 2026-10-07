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

const hostActions = [
  "preflight",
  "pull_images",
  "update_services",
  "database_migration",
  "update_resident_agent",
  "wait_for_service_health",
  "verify_runtime_images",
  "verify_frontend_container",
  "verify_frontend_edge",
] as const

describe("system upgrade messages", () => {
  it("defines a badge label for every server operation status", () => {
    for (const status of statuses) {
      expect(en.systemUpgrade.status[status]).toBeTruthy()
      expect(zh.systemUpgrade.status[status]).toBeTruthy()
    }
  })

  it("defines localized Agent diagnostic labels at the status-surface namespace", () => {
    for (const messages of [en, zh]) {
      expect(messages.systemUpgrade.agentDiagnostics.title).toBeTruthy()
      expect(messages.systemUpgrade.agentDiagnostics.description).toBeTruthy()
      expect(messages.systemUpgrade.agentDiagnostics.empty).toBeTruthy()
      expect(messages.systemUpgrade.agentDiagnostics.reconnecting).toBeTruthy()
      expect(messages.systemUpgrade.agentDiagnostics.terminal).toBeTruthy()
      expect(messages.systemUpgrade.agentDiagnostics.source.agent_heartbeat).toBeTruthy()
      expect(messages.systemUpgrade.agentDiagnostics.source.server_observation).toBeTruthy()
      expect(messages.systemUpgrade.agentDiagnostics.source.historical).toBeTruthy()
    }
  })

  it("defines localized labels for every bounded host activity fact", () => {
    for (const messages of [en, zh]) {
      expect(messages.systemUpgrade.hostActivity.title).toBeTruthy()
      expect(messages.systemUpgrade.hostActivity.waiting).toBeTruthy()
      expect(messages.systemUpgrade.hostActivity.action).toBeTruthy()
      expect(messages.systemUpgrade.hostActivity.startedAt).toBeTruthy()
      expect(messages.systemUpgrade.hostActivity.lastHeartbeatAt).toBeTruthy()
      expect(messages.systemUpgrade.timeline.current).toBeTruthy()
      expect(messages.systemUpgrade.timeline.complete).toBeTruthy()
      expect(messages.systemUpgrade.diagnostics.title).toBeTruthy()
      expect(messages.systemUpgrade.facts.drawerTitle).toBeTruthy()
      expect(messages.systemUpgrade.actions.showFacts).toBeTruthy()
      expect(messages.systemUpgrade.actions.hideFacts).toBeTruthy()
      expect(messages.systemUpgrade.mock.mode).toBeTruthy()
      expect(messages.systemUpgrade.mock.live).toBeTruthy()
      for (const action of hostActions) expect(messages.systemUpgrade.hostActivity.actions[action]).toBeTruthy()
    }
  })
})
