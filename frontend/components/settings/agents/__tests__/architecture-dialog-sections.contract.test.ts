import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/architecture-dialog-sections.tsx"), "utf8")

describe("architecture-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ArchitectureDialogHeader")
    expect(source).toContain("className")
    expect(source).toContain('from "./architecture-flow"')
    expect(source).not.toContain("from \"next/dynamic\"")
  })

  it("owns the demo-aligned command center sections", () => {
    expect(source).toContain("export function ArchitectureCommandCenter")
    expect(source).toContain("export function ArchitectureSummaryPanel")
    expect(source).not.toContain("ArchitectureRolesPanel")
    expect(source).not.toContain("ArchitectureExecutionTimeline")
  })

  it("keeps the topology as the direct center content with its relationship note overlaid on the canvas", () => {
    expect(source).toContain("<ArchitectureFlow fillAvailableSpace showDiagramNote />")
    expect(source).toContain("<ArchitectureFlowCanvasLoadingState fillAvailableSpace />")
    expect(source).not.toContain('t("flowDiagramNote")')
  })

  it("uses shared chart role helpers instead of local raw chart utilities", () => {
    expect(source).toContain('from "@/lib/chart-config"')
    expect(source).toContain("getArchitectureFlowRoleDotClassName")
    expect(source).not.toContain('bg-[color:var(--color-chart-2)]')
    expect(source).not.toContain('bg-[color:var(--color-chart-1)]')
  })

  it("renders no bottom execution timeline alongside the topology", () => {
    expect(source).not.toContain("ArchitectureTimelineStep")
    expect(source).not.toContain("flowStepsTitle")
    expect(source).not.toContain("gridTemplateRows")
    expect(source).toContain('className="min-h-0 overflow-hidden"')
  })

  it("uses a structured flow canvas skeleton instead of a raw fullscreen placeholder block", () => {
    expect(source).toContain("function ArchitectureFlowCanvasLoadingState")
    expect(source).toContain("<ArchitectureFlowCanvasLoadingState fillAvailableSpace />")
    expect(source).not.toContain("function ArchitectureFlowCanvasSkeleton")
    expect(source).not.toContain("loading: () => <ArchitectureFlowCanvasSkeleton />")
    expect(source).not.toContain('<Skeleton className="h-full min-h-96 w-full" />')
  })

  it("derives the flow canvas skeleton from the resolved canvas geometry contract", () => {
    expect(source).toContain('from "./architecture-flow-layout"')
    expect(source).toContain("ArchitectureFlowShell")
    expect(source).toContain("ArchitectureFlowViewport")
    expect(source).toContain("serverFrame")
    expect(source).toContain("agentFrames")
    expect(source).toContain("engineFrames")
    expect(source).not.toContain("workerFrames")
    expect(source).not.toContain("grid gap-5 lg:grid-cols-[180px_minmax(0,1fr)]")
  })
})
