import { beforeEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const apiClientMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
}))
vi.mock("@/lib/api-client", () => ({ default: apiClientMocks }))

import { getWordlists, updateWordlistMetadata, uploadWordlist } from "@/services/wordlist.service"

const source = readFileSync(path.resolve(process.cwd(), "services/wordlist.service.ts"), "utf8")
const wordlist = {
  id: 7,
  name: "wordlists/7",
  fileName: "untagged.txt",
  description: "",
  tags: [],
  fileSize: 12,
  lineCount: 2,
  fileHash: "hash",
  createdAt: "2026-09-22T00:00:00Z",
  updatedAt: "2026-09-22T00:00:00Z",
}

describe("wordlist.service contract", () => {
  beforeEach(() => vi.clearAllMocks())

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

  it("rejects malformed required tag arrays instead of coercing them", async () => {
    apiClientMocks.get.mockResolvedValue({
      data: { results: [{ ...wordlist, tags: null }], totalSize: 1 },
    })
    await expect(getWordlists()).rejects.toThrow("Invalid Wordlist response: tags must be an array")

    apiClientMocks.post.mockResolvedValue({ data: { ...wordlist, tags: null } })
    await expect(uploadWordlist({ file: new File(["admin\n"], "untagged.txt") })).rejects.toThrow(
      "Invalid Wordlist response: tags must be an array"
    )

    apiClientMocks.patch.mockResolvedValue({ data: { ...wordlist, tags: null } })
    await expect(updateWordlistMetadata({
      id: 7,
      name: "wordlists/7",
      tags: [],
      updateMask: "tags",
    })).rejects.toThrow("Invalid Wordlist response: tags must be an array")

    expect(source).not.toContain("tags ?? []")
  })

  it("accepts and preserves a valid empty tag array", async () => {
    apiClientMocks.get.mockResolvedValue({ data: { results: [wordlist], totalSize: 1 } })

    await expect(getWordlists()).resolves.toMatchObject({
      results: [{ name: "wordlists/7", tags: [] }],
    })
  })
})
