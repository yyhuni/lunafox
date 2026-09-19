import { readFileSync } from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"

const source = readFileSync(path.resolve(process.cwd(), "components/system-upgrade-status.tsx"), "utf8")

describe("system-upgrade-status event stream contract", () => {
  it("uses the shared raw viewer and copy action for server-projected upgrade events", () => {
    expect(source).toContain('from "@/components/shared/visualization/raw-log-viewer"')
    expect(source).toContain('from "@/components/shared/visualization/terminal-log-copy-all-button"')
    expect(source).toContain("<RawLogViewer")
    expect(source).toContain("<TerminalLogCopyAllButton")
    expect(source).toContain('toastId="system-upgrade-event-copy"')
    expect(source).toContain("upgradeEventLine")
    expect(source).toContain("entry.message")
  })

  it("keeps upgrade events separate from the system-log data path", () => {
    expect(source).not.toContain("useSystemLogs")
    expect(source).not.toContain("LiveLogSurface")
    expect(source).not.toContain("StructuredLogViewer")
    expect(source).not.toContain("components/settings/system-logs")
    expect(source).not.toContain("loki")
    expect(source).not.toContain("docker compose")
  })
})
