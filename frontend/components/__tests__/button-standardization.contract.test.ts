import { describe, expect, it } from "vitest"
import { readFileSync, readdirSync, statSync } from "node:fs"
import path from "node:path"

const componentsRoot = path.resolve(process.cwd(), "components")

function collectSourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const fullPath = path.join(dir, entry)
    const stat = statSync(fullPath)

    if (stat.isDirectory()) {
      if (entry === "ui" || entry === "__tests__") return []
      return collectSourceFiles(fullPath)
    }

    if (!/\.(tsx|ts)$/.test(entry)) return []
    return [fullPath]
  })
}

describe("button standardization contract", () => {
  it("does not duplicate primary button hover styling outside the shared Button", () => {
    const offenders = collectSourceFiles(componentsRoot).filter((filePath) => {
      const source = readFileSync(filePath, "utf8")
      return source.includes("hover:bg-primary/90")
    })

    expect(offenders.map((filePath) => path.relative(process.cwd(), filePath))).toEqual([])
  })
})
