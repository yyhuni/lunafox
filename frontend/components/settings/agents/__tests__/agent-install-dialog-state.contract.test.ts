import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-install-dialog-state.ts"), "utf8")

describe("agent-install-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useAgentInstallDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("shows the local-network warning only for a generated local command", () => {
    expect(source).toContain("return hasToken && (")
    expect(source).toContain('hostname === "localhost"')
    expect(source).toContain('hostname === "server" && port === "8080"')
  })
})
