import { readFileSync } from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"

const source = readFileSync(
  path.resolve(process.cwd(), "components/shared/visualization/raw-log-viewer.tsx"),
  "utf8"
)

describe("raw-log-viewer contract", () => {
  it("owns ANSI and plain-text rendering in the shared visualization layer", () => {
    expect(source).toContain("export function RawLogViewer")
    expect(source).toContain("new AnsiToHtml")
    expect(source).toContain("escapeXML: true")
    expect(source).toContain("colorizeLogContent")
    expect(source).toContain("highlightSearch")
  })

  it("uses the shared terminal viewport geometry and near-bottom follow rule", () => {
    expect(source).toContain("TERMINAL_LOG_VIEWPORT_CLASS")
    expect(source).toContain('data-slot="raw-log-viewer"')
    expect(source).toContain("scrollHeight - scrollTop - clientHeight < 30")
    expect(source).toContain("isAtBottomRef.current")
    expect(source).toContain("topRightAction?: React.ReactNode")
    expect(source).toContain('data-slot="raw-log-top-right-action"')
  })
})
