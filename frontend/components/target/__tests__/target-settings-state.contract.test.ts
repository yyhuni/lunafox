import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/target-settings-state.ts"), "utf8")

describe("target-settings-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useTargetSettingsState")
    expect(source).toContain("from \"react\"")
  })

  it("keeps target settings state scoped to scheduled scans", () => {
    expect(source).toContain("useScheduledScans")
    expect(source).not.toContain("useTargetBlacklist")
    expect(source).not.toContain("useUpdateTargetBlacklist")
    expect(source).not.toContain("updateTargetBlacklist")
  })
})
