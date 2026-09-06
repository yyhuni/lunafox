import { describe, expect, expectTypeOf, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import type {
  EngineDiagnosticAvailability,
  EngineDiagnosticResultState,
  EngineExecutionDiagnostics,
  ScanInputSource,
  ScanLog,
  ScanRecord,
  ScanTriggerType,
} from "@/types/scan.types"

const source = readFileSync(path.resolve(process.cwd(), "types/scan.types.ts"), "utf8")

describe("scan.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from '@/types/scan-workflow.types'")
  })

  it("requires task-scoped progress log identity for runtime task grouping", () => {
    expect(source).toContain("export interface RuntimeTask")
    expect(source).toContain("name: string")
    expect(source).toContain("runtimeTasks?: RuntimeTask[]")
    expect(source).toContain("taskId: number")
    expect(source).not.toContain("taskId?: number")
    expectTypeOf<ScanLog["taskId"]>().toEqualTypeOf<number>()
  })

  it("separates the bound scan workflow from executed engine identities", () => {
    expect(source).toContain("scanWorkflow?: string")
    expect(source).toContain("plannedEngineIds: string[]")
    expect(source).not.toContain("engineNames: string[]")
    expect(source).toContain("configuration: WorkflowConfigurationValue; scanWorkflow: string")
    expect(source).not.toContain("workflowNames")
  })

  it("keeps trigger provenance as a required closed transport union", () => {
    expect(source).toContain('export type ScanTriggerType = "manual" | "scheduled" | "ai"')
    expect(source).toContain("triggerType: ScanTriggerType")
    expectTypeOf<ScanRecord["triggerType"]>().toEqualTypeOf<ScanTriggerType>()
    expectTypeOf<ScanTriggerType>().toEqualTypeOf<"manual" | "scheduled" | "ai">()
  })

  it("keeps execution input source as a required closed transport union", () => {
    expect(source).toContain('export type ScanInputSource = "scanSnapshot" | "targetInventory"')
    expect(source).toContain("inputSource: ScanInputSource")
    expect(source).toContain("export function isScanInputSource")
    expectTypeOf<ScanRecord["inputSource"]>().toEqualTypeOf<ScanInputSource>()
    expectTypeOf<ScanInputSource>().toEqualTypeOf<"scanSnapshot" | "targetInventory">()
  })

  it("keeps current Agent runtime state in the Scan detail projection", () => {
    expect(source).toContain("agentStatus?: string")
    expect(source).toContain("agentHealthState?: string")
    expect(source).toContain("agentDeleted?: boolean")
    expect(source).toContain('assignmentMode?: "automatic" | "pinned"')
  })

  it("keeps terminal Engine diagnostics as a closed runtime-task detail contract", () => {
    expect(source).toContain("export interface EngineExecutionDiagnostics")
    expect(source).toContain("diagnostics?: EngineExecutionDiagnostics")
    expect(source).toContain('export type EngineDiagnosticAvailability = "available" | "unavailable"')
    expect(source).toContain('export type EngineDiagnosticResultState = "complete" | "partial" | "none" | "unknown"')
    expectTypeOf<EngineExecutionDiagnostics["availability"]>().toEqualTypeOf<EngineDiagnosticAvailability>()
    expectTypeOf<EngineExecutionDiagnostics["resultState"]>().toEqualTypeOf<EngineDiagnosticResultState>()
  })
})
