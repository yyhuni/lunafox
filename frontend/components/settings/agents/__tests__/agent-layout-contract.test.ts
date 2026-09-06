import { describe, expect, it } from "vitest"
import {
  AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS,
  AGENT_CLUSTER_METRICS_CLASS,
  AGENT_CLUSTER_SUMMARY_ROOT_CLASS,
  AGENT_CLUSTER_SUMMARY_CELL_DIVIDER_CLASS,
  AGENT_CLUSTER_SUMMARY_STATE_CLASS,
  AGENT_EXPANSION_SLOT_MIN_HEIGHT_CLASS,
  AGENT_OVERVIEW_HEADER_CLASS,
  AGENT_OVERVIEW_TITLE_GROUP_CLASS,
  AGENT_TOOLBAR_ACTIONS_CLASS,
  AGENT_TOOLBAR_CONTROLS_CLASS,
  AGENT_TOOLBAR_FILTERS_CLASS,
  AGENT_TOOLBAR_ROOT_CLASS,
} from "@/components/settings/agents/agent-layout-contract"

describe("agent-layout-contract", () => {
  it("keeps scheme C as a responsive state-and-capacity summary", () => {
    expect(AGENT_OVERVIEW_HEADER_CLASS).toContain("@4xl/main:flex-row")
    expect(AGENT_OVERVIEW_HEADER_CLASS).toContain("@4xl/main:justify-between")
    expect(AGENT_OVERVIEW_TITLE_GROUP_CLASS).toContain("flex-wrap")
    expect(AGENT_CLUSTER_SUMMARY_ROOT_CLASS).toContain("flex-wrap")
    expect(AGENT_CLUSTER_SUMMARY_ROOT_CLASS).toContain("border-y")
    expect(AGENT_CLUSTER_SUMMARY_ROOT_CLASS).toContain("border-border")
    expect(AGENT_CLUSTER_SUMMARY_ROOT_CLASS).not.toContain("border-border/70")
    expect(AGENT_CLUSTER_SUMMARY_STATE_CLASS).toContain("min-w-0")
    expect(AGENT_CLUSTER_SUMMARY_STATE_CLASS).toContain("flex-col")
    expect(AGENT_CLUSTER_SUMMARY_STATE_CLASS).toContain("justify-center")
    expect(AGENT_CLUSTER_SUMMARY_STATE_CLASS).toContain("@5xl/main:w-1/4")
    expect(AGENT_CLUSTER_SUMMARY_STATE_CLASS).toContain("@5xl/main:items-center")
    expect(AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS).toContain("w-full")
    expect(AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS).toContain("@5xl/main:w-1/4")
    expect(AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS).toContain("@5xl/main:border-border")
    expect(AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS).toContain("flex-col")
    expect(AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS).toContain("justify-center")
    expect(AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS).toContain("@5xl/main:items-center")
    expect(AGENT_CLUSTER_SUMMARY_CELL_DIVIDER_CLASS).toContain("@5xl/main:border-border")
    expect(AGENT_CLUSTER_SUMMARY_CELL_DIVIDER_CLASS).toContain("flex-col")
    expect(AGENT_CLUSTER_SUMMARY_CELL_DIVIDER_CLASS).toContain("justify-center")
    expect(AGENT_CLUSTER_SUMMARY_CELL_DIVIDER_CLASS).toContain("@5xl/main:items-center")
    expect(AGENT_CLUSTER_SUMMARY_CELL_DIVIDER_CLASS).not.toContain("border-border/70")
    expect(AGENT_CLUSTER_METRICS_CLASS).toContain("flex-wrap")
    expect(AGENT_CLUSTER_SUMMARY_ROOT_CLASS).not.toContain("grid-cols")
  })

  it("keeps toolbar controls and actions wrap-safe", () => {
    expect(AGENT_TOOLBAR_ROOT_CLASS).toContain("@5xl/main:items-start")
    expect(AGENT_TOOLBAR_CONTROLS_CLASS).toContain("@4xl/main:flex-row")
    expect(AGENT_TOOLBAR_FILTERS_CLASS).toContain("flex-wrap")
    expect(AGENT_TOOLBAR_ACTIONS_CLASS).toContain("flex-wrap")
    expect(AGENT_TOOLBAR_ACTIONS_CLASS).toContain("@5xl/main:justify-end")
  })

  it("owns the single-node expansion cell height beside the card-grid contract", () => {
    expect(AGENT_EXPANSION_SLOT_MIN_HEIGHT_CLASS).toBe("min-h-[252px]")
  })
})
