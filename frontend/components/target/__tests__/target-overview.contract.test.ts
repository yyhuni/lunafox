import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/target-overview.tsx"), "utf8")
const guide = readFileSync(path.resolve(process.cwd(), "components/target/README.md"), "utf8")

describe("target-overview contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TargetOverview")
    expect(source).toContain("from \"./target-overview-sections\"")
  })

  it("uses shared skeleton handoff instead of hard-cutting from loading to content", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain('owner="target-overview-content"')
    expect(source).toContain("skeleton={<TargetOverviewLoadingState />}")
    expect(source).toContain('from "@/components/shared/feedback/app-error-state"')
    expect(source).toContain("useDetailShellReadySignal")
    expect(source).toContain("detailShellReady?.deferInitialSkeleton")
    expect(source).toContain("if (detailShellReady?.deferInitialSkeleton && isInitialLoading) {")
    expect(source).toContain("<AppErrorState")
    expect(source).not.toContain("ContentReveal")
  })

  it("documents the target overview detail-shell handoff contract", () => {
    expect(guide).toContain("## Target Detail Overview Handoff")
    expect(guide).toContain("TargetDetailShellLoadingState")
    expect(guide).toContain("target-detail-shell-layout.tsx")
    expect(guide).toContain("useDetailShellReadySignal")
    expect(guide).toContain("deferInitialSkeleton")
  })
})
