import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const readSource = (relativePath: string) => readFileSync(
  path.resolve(process.cwd(), relativePath),
  "utf8",
)

const componentSource = readSource("components/settings/agents/agent-install-connection-status.tsx")
const dialogSource = readSource("components/settings/agents/agent-install-dialog.tsx")
const hookSource = readSource("hooks/use-agent-install-connection.ts")
const localRules = readSource("components/settings/agents/README.md")

describe("Agent installation connection feedback contract", () => {
  it("uses the canonical non-secret token resource on a bounded two-second cadence", () => {
    expect(hookSource).toContain("AGENT_INSTALL_POLL_INTERVAL_MS = 2_000")
    expect(hookSource).toContain("AGENT_INSTALL_POST_EXPIRY_GRACE_MS = 2 * 60_000")
    expect(hookSource).toContain("useRegistrationToken")
    expect(hookSource).toContain("agentKeys.registrationToken")
    expect(hookSource).not.toContain("getAgents")
    expect(hookSource).not.toContain("pageSize")
    expect(hookSource).not.toContain("baseline")
  })

  it("keeps page-owned feedback available for expired-token close/reopen", () => {
    expect(dialogSource).toContain("connection={connection}")
    expect(dialogSource).toContain("<AgentInstallConnectionStatus")
    expect(dialogSource).not.toContain("key={token.token}")
  })

  it("keeps the approved inline status and removes the prototype controls", () => {
    expect(componentSource).toContain("export function AgentInstallConnectionStatus")
    expect(componentSource).toContain('aria-live="polite"')
    expect(componentSource).toContain("AnimatePresence")
    expect(componentSource).toContain("motion-reduce:animate-none")
    expect(componentSource).not.toContain("installDemo")
    expect(componentSource).not.toContain("PrototypeSwitcher")
    expect(dialogSource).not.toContain("AgentInstallConnectionLab")
  })

  it("documents exact token attribution and bounded post-expiry observation", () => {
    expect(localRules).toContain("agentRegistrationTokens/{registrationToken}")
    expect(localRules).toContain("两分钟")
    expect(localRules).not.toContain("列表接口不提供令牌归因")
  })
})
