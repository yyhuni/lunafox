import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync } from "node:fs"
import path from "node:path"

const MOCK_SOURCE_ROOT = "mock"
const ADDITIONAL_MOCK_SOURCE_FILES = ["lib/mock-vulnerabilities.ts"]
const HAN_CHARACTER_PATTERN = /[\u3400-\u9fff]/u
const MOCK_SOURCE_EXTENSIONS = new Set([".ts", ".tsx", ".js", ".mjs", ".json"])

function collectMockSourceFiles(relativeDirectory: string): string[] {
  const absoluteDirectory = path.resolve(process.cwd(), relativeDirectory)

  return readdirSync(absoluteDirectory, { withFileTypes: true }).flatMap((entry) => {
    if (entry.name === "__tests__") return []

    const relativePath = path.join(relativeDirectory, entry.name)
    if (entry.isDirectory()) return collectMockSourceFiles(relativePath)
    if (!entry.isFile()) return []
    if (entry.name.endsWith(".test.ts") || entry.name.endsWith(".test.tsx")) return []
    return MOCK_SOURCE_EXTENSIONS.has(path.extname(entry.name)) ? [relativePath] : []
  })
}

describe("runtime mock data language contract", () => {
  it("keeps all runtime mock data in English", () => {
    const sourceFiles = [
      ...collectMockSourceFiles(MOCK_SOURCE_ROOT),
      ...ADDITIONAL_MOCK_SOURCE_FILES,
    ]

    expect(sourceFiles.length).toBeGreaterThan(0)

    for (const relativePath of sourceFiles) {
      const source = readFileSync(path.resolve(process.cwd(), relativePath), "utf8")
      expect(source, relativePath).not.toMatch(HAN_CHARACTER_PATTERN)
    }
  })
})
