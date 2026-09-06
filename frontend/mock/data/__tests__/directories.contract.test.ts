import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import { mockDirectories } from "@/mock/data/directories"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/directories.ts"), "utf8")

describe("directories contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockDirectories")
    expect(source).toContain("from '@/types/directory.types'")
  })

  it("uses the exact nullable-string Directory read model without source provenance", () => {
    expect(mockDirectories.some((item) => item.contentLength === null && item.duration === null)).toBe(true)
    expect(mockDirectories.some((item) => item.contentLength === "0")).toBe(true)
    expect(mockDirectories.some((item) => item.contentLength === "9223372036854775807")).toBe(true)

    for (const item of mockDirectories) {
      expect(item.contentLength === null || typeof item.contentLength === "string").toBe(true)
      expect(item.duration === null || typeof item.duration === "string").toBe(true)
      expect(item).not.toHaveProperty("words")
      expect(item).not.toHaveProperty("lines")
      expect(item).not.toHaveProperty("websiteUrl")
    }
  })
})
