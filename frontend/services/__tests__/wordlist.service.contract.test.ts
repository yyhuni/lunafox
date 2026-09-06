import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "services/wordlist.service.ts"), "utf8")

describe("wordlist.service contract", () => {
  it("uses canonical backend wordlist routes", () => {
    expect(source).toContain("/wordlists")
    expect(source).toContain("/wordlistTags")
    expect(source).toContain("/text")
    expect(source).toContain("updateMask")
    expect(source).not.toContain("/content")
    expect(source).not.toContain("apiClient.put")
  })

  it("exposes metadata and tag summary operations", () => {
    expect(source).toContain("getWordlistTags")
    expect(source).toContain("updateWordlistMetadata")
  })

  it("derives upload naming from the selected file instead of multipart name", () => {
    expect(source).toContain("file: File")
    expect(source).toContain('formData.append("file", payload.file)')
    expect(source).not.toContain('formData.append("name"')
    expect(source).not.toContain("name: string")
  })
})
