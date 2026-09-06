import { describe, expect, it } from "vitest"

import { getTerminalLogLevelBadgeStyle, getTerminalLogLevelColor } from "@/lib/log-level"

describe("terminal log level presentation", () => {
  it.each([
    ["debug", "var(--terminal-log-debug)"],
    ["INFO", "var(--terminal-log-info)"],
    ["warning", "var(--terminal-log-warning)"],
    ["fatal", "var(--terminal-log-error)"],
    ["unknown", "var(--terminal-log-foreground)"],
  ])("maps %s to the shared terminal color", (level, color) => {
    expect(getTerminalLogLevelColor(level)).toBe(color)
  })

  it("derives task badges from the same level color", () => {
    const style = getTerminalLogLevelBadgeStyle("error")

    expect(style.color).toBe("var(--terminal-log-error)")
    expect(style.borderColor).toContain("var(--terminal-log-error)")
    expect(style.backgroundColor).toContain("var(--terminal-log-error)")
  })
})
