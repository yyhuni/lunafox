import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const packageJson = JSON.parse(readFileSync(path.resolve(process.cwd(), "package.json"), "utf8")) as {
  dependencies?: Record<string, string>
  devDependencies?: Record<string, string>
}

describe("primitive dependency boundary", () => {
  it("keeps direct Radix and cmdk primitive dependencies out of the frontend package", () => {
    const dependencies = {
      ...packageJson.dependencies,
      ...packageJson.devDependencies,
    }

    expect(Object.keys(dependencies).filter((name) => name.startsWith("@radix-ui/"))).toEqual([])
    expect(dependencies).not.toHaveProperty("cmdk")
  })
})
