import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/auth/terminal-login-types.ts"), "utf8")

describe("terminal-login-types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export type LoginStep = \"username\" | \"password\" | \"authenticating\" | \"success\" |")
  })
})
