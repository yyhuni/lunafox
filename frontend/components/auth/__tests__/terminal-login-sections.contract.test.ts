import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/auth/terminal-login-sections.tsx"), "utf8")

describe("terminal-login-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AuthBootLog")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("renders a static LUNAFOX fallback before the shuffle bundle loads", () => {
    expect(source).toContain("function TerminalLoginBrandStatic()")
    expect(source).toContain(">LUNAFOX<")
    expect(source).toContain("loading: () => <TerminalLoginBrandStatic />")
  })

  it("does not rely on important typography utilities for the terminal banner", () => {
    expect(source).not.toContain("!font-bold")
    expect(source).not.toContain("!text-4xl")
    expect(source).not.toContain("md:!text-6xl")
    expect(source).not.toContain("sm:!text-5xl")
  })

  it("keeps the auth boot progress bar on transform-based motion", () => {
    expect(source).toContain("origin-left")
    expect(source).toContain("scaleX")
    expect(source).not.toContain("transition-[width]")
    expect(source).not.toContain("width: `${progress}%`")
  })
})
