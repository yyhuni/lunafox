import { readFileSync } from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"

const source = readFileSync(
  path.resolve(process.cwd(), "components/shared/visualization/terminal-log-surface.tsx"),
  "utf8"
)

describe("terminal-log-surface contract", () => {
  it("owns shared live-log body, viewport, focus, jump, and footer geometry", () => {
    expect(source).toContain("export const TERMINAL_LOG_BODY_CLASS")
    expect(source).toContain("export const TERMINAL_LOG_VIEWPORT_CLASS")
    expect(source).toContain("export const TERMINAL_LOG_FOOTER_CLASS")
    expect(source).toContain("flex-wrap")
    expect(source).toContain("sm:flex-nowrap")
    expect(source).toContain('data-slot="live-log-terminal-body"')
    expect(source).toContain('data-slot="live-log-top-right-action"')
    expect(source).toContain('data-slot="live-log-footer"')
    expect(source).toContain("font-mono text-xs leading-normal")
    expect(source).toContain("focus-visible:ring-2")
  })

  it("owns the accessible contextual jump-to-latest affordance", () => {
    expect(source).toContain("showJumpToLatest")
    expect(source).toContain("onJumpToLatest")
    expect(source).toContain("jumpToLatestLabel")
    expect(source).toContain("TooltipTrigger")
    expect(source).toContain("ChevronDownIcon")
    expect(source).toContain('aria-label={jumpToLatestLabel}')
    expect(source).toContain('className="absolute bottom-4 right-4 radius-round"')
  })

  it("provides a shared terminal top-right action slot", () => {
    expect(source).toContain("topRightAction?: React.ReactNode")
    expect(source).toContain("topRightAction &&")
    expect(source).toContain('className="absolute right-3 top-3 z-10"')
  })
})
