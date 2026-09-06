import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/shared/loading/detail-shell-ready-context.tsx"),
  "utf8"
)
const guide = readFileSync(path.resolve(process.cwd(), "components/shared/loading/README.md"), "utf8")

describe("detail-shell-ready-context contract", () => {
  it("exports the shared provider and ready-signal hook for detail overview shell handoff", () => {
    expect(source).toContain("export interface DetailShellReadyContextValue")
    expect(source).toContain("export function DetailShellReadyProvider")
    expect(source).toContain("export function useDetailShellReadyContext")
    expect(source).toContain("export function useDetailShellReadySignal")
    expect(source).toContain("window.requestAnimationFrame")
  })

  it("documents the detail-shell overview handoff pattern in the shared loading guide", () => {
    expect(guide).toContain("## Detail Shell Overview Handoff")
    expect(guide).toContain("feature-local shell loading state")
    expect(guide).toContain("HiddenReadinessRouteBoundary")
    expect(guide).toContain("title -> tabs -> content")
    expect(guide).toContain("Do not start child queries earlier")
  })

  it("documents root boot motion continuity for full document reloads", () => {
    expect(guide).toContain("## Root Boot Motion")
    expect(guide).toContain("full document reload")
    expect(guide).toContain("document-local `performance.now()`")
    expect(guide).toContain("wall-clock-derived phase")
    expect(guide).toContain("`Date.now() - performance.now()`")
  })
})
