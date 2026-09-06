import { describe, expect, it } from "vitest"
import { getScanWorkflowDisplayName } from "@/lib/scan-workflow-display"

describe("getScanWorkflowDisplayName", () => {
  const workflows = [
    {
      id: 1,
      name: "default",
      scanWorkflowId: "default",
      displayName: "默认扫描",
      title: "默认扫描",
    },
  ]

  it("resolves a scan workflow resource reference to its catalog display name", () => {
    expect(getScanWorkflowDisplayName("scanWorkflows/default", workflows)).toBe("默认扫描")
  })

  it("does not expose an unmatched workflow resource reference as display text", () => {
    expect(getScanWorkflowDisplayName("scanWorkflows/retired", workflows)).toBeUndefined()
  })
})
