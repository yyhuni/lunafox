import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/visualization/structured-log-viewer.tsx"), "utf8")

describe("structured-log-viewer contract", () => {
  it("owns shared structured log parsing and rendering", () => {
    expect(source).toContain("export function StructuredLogViewer")
    expect(source).toContain("export function parseLogLine")
    expect(source).toContain("LEVEL_LABEL")
    expect(source).not.toContain("LEVEL_BADGE_STYLE")
    expect(source).not.toContain('from "@/components/ui/badge"')
  })

  it("renders all entries in one selectable preformatted text stream", () => {
    expect(source).toContain('data-slot="structured-log-viewer"')
    expect(source).toContain("m-0 min-w-full whitespace-pre-wrap break-all")
    expect(source).toContain('trailingNewline ? "\\n" : null')
    expect(source).not.toContain("space-y-0")
  })

  it("renders bracketed full level names without fixed-width padding or badges", () => {
    expect(source).toContain("LEVEL_LABEL")
    expect(source).toContain("[{LEVEL_LABEL[parsed.level]}]")
    expect(source).not.toContain("inline-block w-7")
    expect(source).not.toContain("text-[10px]")
    expect(source).not.toContain("data-slot=\"badge\"")
    expect(source).not.toContain('from "@/components/ui/badge"')
  })

  it("keeps semantic terminal colors without decorative error-row backgrounds", () => {
    expect(source).toContain('parsed.level === "error" || parsed.level === "fatal"')
    expect(source).toContain('isErrorLevel ? "var(--terminal-log-error)" : "var(--terminal-log-foreground)"')
    expect(source).not.toContain("oklch(from var(--terminal-log-error)")
  })

  it("defers virtualizer rerenders triggered by layout measurement", () => {
    expect(source).toContain("useFlushSync: false")
  })
})
