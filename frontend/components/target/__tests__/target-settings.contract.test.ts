import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/target-settings.tsx"), "utf8")

describe("target-settings contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TargetSettings")
    expect(source).toContain("from \"./target-settings-sections\"")
    expect(source).toContain("<BlacklistSettingsWorkspace embedded targetId={targetId} />")
  })

  it("uses shared skeleton handoff instead of hard-cutting from loading to content", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain('owner="target-settings-content"')
    expect(source).toContain("getDataTableSkeletonRowCount")
    expect(source).toContain("skeleton={<TargetSettingsLoadingState state={state} rowCount={loadingRowCount} />}")
    expect(source).not.toContain("ContentReveal")
  })
})
