import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/wordlists.ts"), "utf8")

describe("wordlists contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockWordlists")
    expect(source).toContain("from '@/types/wordlist.types'")
  })

  it("keeps mock filtering aligned with multi-tag OR filters", () => {
    expect(source).toContain("matchAll(/tags==")
    expect(source).toContain("tags.some")
  })
})
