import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const config = JSON.parse(readFileSync(path.resolve(process.cwd(), "components.json"), "utf8")) as {
  style?: string
}

describe("shadcn registry base", () => {
  it("defaults future registry additions to Base UI-backed components", () => {
    expect(config.style).toMatch(/^base-/)
  })
})
