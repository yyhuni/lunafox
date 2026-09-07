import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "vitest.config.ts"), "utf8")
const setupSource = readFileSync(path.resolve(process.cwd(), "vitest.setup.ts"), "utf8")

describe("vitest.config contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"node:path\"")
  })

  it("uses Oxc automatic JSX transform for tsx test files", () => {
    expect(source).toContain("oxc:")
    expect(source).toContain('runtime: "automatic"')
    expect(source).toContain('importSource: "react"')
    expect(source).not.toContain("jsxInject")
  })

  it("boots the shared mock platform from vitest setup", () => {
    expect(setupSource).toContain("mock/server")
    expect(setupSource).toContain("beforeAll")
    expect(setupSource).toContain("afterEach")
    expect(setupSource).toContain("afterAll")
  })
})
