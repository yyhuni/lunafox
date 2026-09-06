import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-log-list-sections.tsx"), "utf8")
const taskProgressSource = readFileSync(
  path.resolve(process.cwd(), "components/scan/task-progress-log-list-sections.tsx"),
  "utf8"
)

describe("scan-log-list-sections contract", () => {
  it("uses semantic surface tokens for loading and empty states", () => {
    expect(source).toContain("export function ScanLogListLoadingState")
    expect(source).toContain("bg-card")
    expect(source).toContain("text-muted-foreground")
    expect(source).not.toContain("#1e1e1e")
    expect(source).not.toContain("#808080")
  })

  it("uses the shared raw-output viewer without depending on settings features", () => {
    for (const logSource of [source, taskProgressSource]) {
      expect(logSource).toContain('from "@/components/shared/visualization/raw-log-viewer"')
      expect(logSource).toContain('from "@/components/shared/visualization/terminal-log-copy-all-button"')
      expect(logSource).toContain("<RawLogViewer")
      expect(logSource).toContain("topRightAction={(")
      expect(logSource).toContain("<TerminalLogCopyAllButton")
      expect(logSource).toContain('copyLabel={t("copyAllLogs")}')
      expect(logSource).not.toContain("components/settings")
      expect(logSource).not.toContain("AnsiLogViewer")
      expect(logSource).not.toContain("TerminalLogToolbar")
      expect(logSource).not.toContain("LiveLogSurface")
    }
  })
})
