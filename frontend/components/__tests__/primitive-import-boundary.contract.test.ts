import { describe, expect, it } from "vitest"
import { existsSync, lstatSync, readFileSync, readdirSync, statSync } from "node:fs"
import path from "node:path"

const frontendRoot = process.cwd()
const scanRoots = ["app", "components", "hooks", "lib"]
const sourceExtensions = new Set([".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"])
const ignoredDirectories = new Set([".next", "coverage", "node_modules", "__tests__"])

function walk(dir: string, files: string[] = []) {
  for (const entry of readdirSync(dir)) {
    if (ignoredDirectories.has(entry)) continue

    const fullPath = path.join(dir, entry)
    const linkStats = lstatSync(fullPath)
    if (linkStats.isSymbolicLink()) continue


    const stats = statSync(fullPath)
    if (stats.isDirectory()) {
      walk(fullPath, files)
      continue
    }

    if (sourceExtensions.has(path.extname(fullPath))) {
      files.push(fullPath)
    }
  }

  return files
}

function isAllowedPrimitiveAdapter(relativePath: string) {
  return relativePath.startsWith("components/ui/") || relativePath.startsWith("lib/ui/")
}

function collectRawPrimitiveImports() {
  const violations: string[] = []

  for (const root of scanRoots) {
    const absoluteRoot = path.join(frontendRoot, root)
    if (!existsSync(absoluteRoot)) continue

    for (const filePath of walk(absoluteRoot)) {
      const relativePath = path.relative(frontendRoot, filePath)
      if (isAllowedPrimitiveAdapter(relativePath)) continue

      const source = readFileSync(filePath, "utf8")
      for (const match of source.matchAll(/from\s+["'](@(?:radix-ui|base-ui)\/[^"']+)["']/g)) {
        const line = source.slice(0, match.index).split("\n").length
        violations.push(`${relativePath}:${line} ${match[1]}`)
      }
    }
  }

  return violations.sort()
}

describe("primitive import boundary", () => {
  it("keeps raw Radix and Base UI imports behind shared UI adapters", () => {
    expect(collectRawPrimitiveImports()).toEqual([])
  })
})
