import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-wordlists.ts"), "utf8")

describe("use-wordlists contract", () => {
  it("keys list queries by backend filter state", () => {
    expect(source).toContain("export function useWordlists")
    expect(source).toContain("list: (params: GetWordlistsParams) => params")
    expect(source).toContain("filter: params?.filter")
    expect(source).toContain("orderBy: params?.orderBy")
    expect(source).toContain("queryFn: () => getWordlists(requestParams)")
    expect(source).toContain("placeholderData: keepPreviousData")
    expect(source).not.toContain("buildWordlistFilter")
    expect(source).not.toContain("tagFilters")
  })

  it("exposes tag summaries and metadata updates", () => {
    expect(source).toContain("useWordlistTags")
    expect(source).toContain("useUpdateWordlistMetadata")
  })

  it("loads the complete Catalog as one fail-closed query generation", () => {
    expect(source).toContain("export function useCompleteWordlistCatalog")
    expect(source).toContain("export async function fetchCompleteWordlistCatalog")
    expect(source).toContain("do {")
    expect(source).toContain("page.nextPageToken")
    expect(source).toContain("byResourceName.has(wordlist.name)")
    expect(source).toContain('orderBy: "fileName"')
    expect(source).toContain("queryFn: fetchCompleteWordlistCatalog")
  })
})
