import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/visualization/terminal-log-toolbar.tsx"), "utf8")

describe("terminal-log-toolbar contract", () => {
  it("owns the shared system and Agent log search/filter controls", () => {
    expect(source).toContain("export function TerminalLogToolbar")
    expect(source).toContain("TERMINAL_LOG_LEVEL_FILTERS")
    expect(source).toContain("StructuredLogLevelFilter")
    expect(source).toContain("ActionSkeleton")
    expect(source).toContain('orientation="vertical"')
    expect(source).toContain('className={cn("w-11 !text-xs", levelFilter !== value && "!text-muted-foreground")}')
  })

  it("makes the terminal toolbar itself the search surface", () => {
    expect(source).toContain('data-slot="terminal-log-toolbar"')
    expect(source).toContain("bg-muted/50")
    expect(source).toContain("!rounded-none !border-0 !bg-transparent")
    expect(source).not.toContain("focus-within:bg-muted/30")
    expect(source).toContain("absolute left-0")
  })

  it("owns the optional log-window selector through the shared select primitive", () => {
    expect(source).toContain("lineWindow?: {")
    expect(source).toContain("<Select")
    expect(source).toContain("lineWindow.options.includes(parsed)")
    expect(source).toContain("lineWindow.onValueChange(parsed)")
  })
})
