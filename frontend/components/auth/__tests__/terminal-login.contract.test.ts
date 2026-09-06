import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/auth/terminal-login.tsx"), "utf8")

describe("terminal-login contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TerminalLogin")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
