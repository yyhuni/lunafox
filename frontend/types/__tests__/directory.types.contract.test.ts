import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/directory.types.ts"), "utf8")

describe("directory.types contract", () => {
  it("uses nullable decimal strings for 64-bit values and URL-only identity", () => {
    expect(source).toContain("contentLength: string | null")
    expect(source).toContain("duration: string | null")
    expect(source).not.toContain("contentLength: number")
    expect(source).not.toContain("duration: number")
    expect(source).not.toContain("words:")
    expect(source).not.toContain("lines:")
    expect(source).not.toContain("websiteUrl:")
  })
})
