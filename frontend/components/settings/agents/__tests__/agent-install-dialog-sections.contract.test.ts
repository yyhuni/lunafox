import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-install-dialog-sections.tsx"), "utf8")

describe("agent-install-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AgentInstallTokenCard")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/ui/button\"")
    expect(source).toContain('from "@/components/shared/feedback/copy-button"')
    expect(source).toContain("<CopyButton")
    expect(source).toContain("loading={isGenerating}")
    expect(source).toContain("loadingLabel=")
  })

  it("uses shared status tone helpers instead of emerald and rose patches", () => {
    expect(source).toContain("getStatusToneBadgeClass")
    expect(source).toContain("getStatusToneBgClass")
    expect(source).not.toContain("border-emerald-500/50")
    expect(source).not.toContain("text-emerald-700")
    expect(source).not.toContain("bg-emerald-500/5")
    expect(source).not.toContain("border-rose-500/50")
    expect(source).not.toContain("text-rose-700")
    expect(source).not.toContain("bg-rose-500/5")
  })

  it("keeps the command copy control inside the command output surface", () => {
    expect(source).toContain('className="bg-muted/30 border p-3 relative rounded-lg"')
    expect(source).toContain('className="absolute right-0 top-0 z-10"')
    expect(source).toContain("overflow-y-auto pr-12 text-xs")
    expect(source).toContain('className="pr-12 text-muted-foreground text-xs"')
  })

  it("leaves installation prerequisites to the dialog-level description", () => {
    expect(source).not.toContain('t("install.commandDesc")')
  })

  it("keeps only the command output as a framed inner surface", () => {
    expect(source).not.toContain('rounded-lg border p-3 transition-[background-color,border-color]')
    expect(source).not.toContain('bg-background border p-3 rounded-lg space-y-3')
    expect(source).not.toContain('bg-muted/20 border p-4 rounded-lg text-xs')
    expect(source).toContain('className="bg-muted/30 border p-3 relative rounded-lg"')
  })

  it("keeps token usage beside the expiry metadata", () => {
    expect(source).toContain('• {t("install.commandExpires", { time: formatRelativeTime(token.expiresAt) })}')
    expect(source).toContain('className="text-muted-foreground text-xs">• {t("install.tokenUsage")}</span>')
    expect(source).not.toContain('className="text-[11px] text-muted-foreground"')
  })
})
