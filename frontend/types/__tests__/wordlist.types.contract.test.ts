import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/wordlist.types.ts"), "utf8")

describe("wordlist.types contract", () => {
  it("models canonical wordlist and tag summary resources", () => {
    expect(source).toContain("fileName: string")
    expect(source).toContain("name: string")
    expect(source).toContain("WordlistTagSummary")
    expect(source).toContain("GetWordlistTagsResponse")
    expect(source).toContain("WordlistText")
    expect(source).toContain("nextPageToken")
  })
})
