import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-disk.ts"), "utf8")

describe("use-disk contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useDiskStats")
    expect(source).toContain("from '@tanstack/react-query'")
  })
})
