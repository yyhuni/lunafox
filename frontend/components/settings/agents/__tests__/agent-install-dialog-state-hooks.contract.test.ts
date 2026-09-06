import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-install-dialog-state-hooks.ts"), "utf8")

describe("agent-install-dialog-state-hooks contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useInstallCommandCopy")
    expect(source).toContain("from \"react\"")
    expect(source).toContain('from "@/components/shared/feedback/copy-button"')
    expect(source).toContain("copyTextToClipboard(text)")
    expect(source).not.toContain('toastFeedback.dismiss("agent-token")')
    expect(source).not.toContain("navigator.clipboard")
    expect(source).not.toContain("document.execCommand")
  })
})
