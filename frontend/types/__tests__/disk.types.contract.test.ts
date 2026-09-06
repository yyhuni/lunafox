import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/disk.types.ts"), "utf8")

describe("disk.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export interface DiskStats {")
  })
})
