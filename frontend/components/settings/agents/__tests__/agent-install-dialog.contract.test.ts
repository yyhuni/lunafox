import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-install-dialog.tsx"), "utf8")

describe("agent-install-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AgentInstallDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/settings/agents/agent-install-dialog-state\"")
  })

  it("uses semantic warning surface for local-only notice", () => {
    expect(source).toContain("getStatusToneSurfaceClass")
    expect(source).toContain("getStatusToneTextClass")
    expect(source).not.toContain("bg-amber-500/5")
    expect(source).not.toContain("border-amber-500/30")
    expect(source).not.toContain("text-amber-700")
  })

  it("relies on the dialog close icon instead of a duplicate footer action", () => {
    expect(source).not.toContain("<DialogClose")
    expect(source).not.toContain('tActions("close")')
    expect(source).not.toContain('className="flex justify-end pt-1"')
  })
})
