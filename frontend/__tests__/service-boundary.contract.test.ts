import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync } from "node:fs"
import path from "node:path"

const FRONTEND_ROOT = process.cwd()
const RUNTIME_ROOTS = [
  path.join(FRONTEND_ROOT, "app"),
  path.join(FRONTEND_ROOT, "components"),
]

const SOURCE_FILE_PATTERN = /\.(ts|tsx)$/
const SERVICE_IMPORT_PATTERN = /from\s+["']@\/services\/|import\(\s*["']@\/services\//g

describe("frontend runtime service boundary", () => {
  it("keeps app and components free of runtime service imports", () => {
    const offendingFiles: string[] = []

    for (const root of RUNTIME_ROOTS) {
      walk(root, offendingFiles)
    }

    expect(offendingFiles).toEqual([])
  })
})

function walk(directory: string, offendingFiles: string[]) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    if (entry.name === "__tests__") {
      continue
    }

    const absolutePath = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      walk(absolutePath, offendingFiles)
      continue
    }

    if (!SOURCE_FILE_PATTERN.test(entry.name)) {
      continue
    }

    const source = readFileSync(absolutePath, "utf8")
    if (SERVICE_IMPORT_PATTERN.test(source)) {
      offendingFiles.push(path.relative(FRONTEND_ROOT, absolutePath))
    }
    SERVICE_IMPORT_PATTERN.lastIndex = 0
  }
}
