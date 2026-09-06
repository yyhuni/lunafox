import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const tsconfig = readFileSync(path.resolve(process.cwd(), "tsconfig.json"), "utf8")
const gitignore = readFileSync(path.resolve(process.cwd(), ".gitignore"), "utf8")

describe("next-env.d contract", () => {
  it("keeps the generated declaration included but untracked", () => {
    expect(tsconfig).toContain('"next-env.d.ts"')
    expect(gitignore).toContain("next-env.d.ts")
  })
})
