import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/mcp-access-popover.tsx"), "utf8")

describe("MCP access popover contract", () => {
  it("uses the shared header popover, confirmation dialog, and canonical icons", () => {
    expect(source).toContain('from "@/components/ui/popover"')
    expect(source).toContain('from "@/components/shared/feedback/confirm-dialog"')
    expect(source).toContain("semanticIcons.concept.mcp")
    expect(source).toContain("shellOverlaySideOffsets.header")
    expect(source).toContain('aria-label={t("open")}')
  })

  it("keeps the same-origin address copyable and owns one-time key clearing", () => {
    expect(source).toContain('const MCP_PATH = "/mcp"')
    expect(source).toContain("window.location.origin")
    expect(source).toContain('id="mcp-popover-url"')
    expect(source).toContain('id="mcp-popover-key"')
    expect(source).toContain("clearOneTimeKey")
    expect(source).toContain("generateMcpKey.reset()")
    expect(source).toContain("setRevealedKey(null)")
    expect(source).not.toContain("localStorage")
    expect(source).not.toContain("crypto.randomUUID")
  })

  it("uses the hook and never leaves a pending key or environment placeholder", () => {
    expect(source).toContain('from "@/hooks/use-mcp-key"')
    expect(source).toContain("useMcpKeyStatus(open)")
    expect(source).toContain("useGenerateMcpKey")
    expect(source).toContain("regenerationConfirmationOpen")
    expect(source).not.toContain("PENDING_KEY_VALUE")
    expect(source).not.toContain("LUNAFOX_MCP_KEY")
    expect(source).not.toContain("bearer_token_env_var")
  })

  it("builds all supported client configurations with the current Bearer key", () => {
    expect(source).toContain("getMcpClientConfiguration")
    expect(source).toContain('"vscode", "cursor", "claudeCode", "codex"')
    expect(source).toContain('type: "http"')
    expect(source).toContain('type: "streamable-http"')
    expect(source).toContain("http_headers")
    expect(source).toContain("Authorization")
    expect(source).toContain("Bearer ${key}")
    expect(source).toContain('id="mcp-popover-client"')
    expect(source).not.toContain("?key=")
  })
})
