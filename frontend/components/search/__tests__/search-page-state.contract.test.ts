import { act, renderHook } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const hookMocks = vi.hoisted(() => ({
  query: "",
  useAssetSearch: vi.fn(() => ({
    data: undefined,
    isLoading: false,
    error: null,
    isFetching: false,
    refetch: vi.fn(),
  })),
}))

vi.mock("next/navigation", () => ({
  useSearchParams: () => ({
    get: (key: string) => (key === "q" ? hookMocks.query : null),
  }),
}))

vi.mock("@/hooks/use-search", () => ({
  useAssetSearch: hookMocks.useAssetSearch,
}))

import { useSearchPageState } from "../search-page-state"

const source = readFileSync(path.resolve(process.cwd(), "components/search/search-page-state.ts"), "utf8")

describe("search-page-state contract", () => {
  afterEach(() => {
    hookMocks.query = ""
    hookMocks.useAssetSearch.mockClear()
    window.localStorage.clear()
  })

  it("uses a Website-first, token-based search state", () => {
    expect(source).toContain("export function useSearchPageState")
    expect(source).toContain('React.useState<AssetType>("website")')
    expect(source).toContain("pageTokens")
    expect(source).toContain("nextPageToken")
    expect(source).not.toContain("asset_type")
    expect(source).not.toContain("totalPages")
    expect(source).not.toContain("useExportAssetSearch")
  })

  it("validates before a query reaches the search hook", () => {
    expect(source).toContain("isGlobalAssetSearchQueryValid(nextQuery)")
    expect(source).toContain('setQueryError(t("invalidQuery"))')
    expect(source).toContain("setPageTokens([undefined])")
    expect(source).toContain("setPageIndex(0)")
  })

  it("starts a valid URL query synchronously with the original parameters", () => {
    expect(source).toContain('const initialQuery = urlSearchParams.get("q") ?? ""')
    expect(source).toContain("const initialSearchParams: SearchParams | undefined")
    expect(source).toContain('{ q: initialQuery, assetType: "website", pageSize: GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE }')
    expect(source).toContain('initialSearchParams ? "searching" : "initial"')
    expect(source).toContain("useAssetSearch(activeSearchParams")
  })

  it("preserves an exact URL lookup from the browser query string", () => {
    const rawURL = "HTTPS://Example.test/path?payload=%00%zz#fragment "
    hookMocks.query = rawURL

    const { result } = renderHook(() => useSearchPageState())

    expect(result.current.query).toBe(rawURL)
    expect(hookMocks.useAssetSearch).toHaveBeenLastCalledWith(
      { q: rawURL, assetType: "website", pageSize: 10 },
      { enabled: true },
    )
  })

  it("puts a quick tag into the draft query without starting a search", () => {
    const { result } = renderHook(() => useSearchPageState())

    act(() => {
      result.current.handleQuickTagClick('statusCode=="200"')
    })

    expect(result.current.query).toBe('statusCode=="200"')
    expect(result.current.searchState).toBe("initial")
    expect(result.current.recentSearches).toEqual([])
    expect(hookMocks.useAssetSearch).toHaveBeenLastCalledWith(undefined, { enabled: false })
  })

  it("puts a recent search into the draft query without starting a search", () => {
    const recentQuery = 'host="api"'
    window.localStorage.setItem("star_patrol_recent_searches", JSON.stringify([recentQuery]))
    const { result } = renderHook(() => useSearchPageState())

    act(() => {
      result.current.handleRecentSearchClick(recentQuery)
    })

    expect(result.current.query).toBe(recentQuery)
    expect(result.current.searchState).toBe("initial")
    expect(result.current.recentSearches).toEqual([recentQuery])
    expect(window.localStorage.getItem("star_patrol_recent_searches")).toBe(JSON.stringify([recentQuery]))
    expect(hookMocks.useAssetSearch).toHaveBeenLastCalledWith(undefined, { enabled: false })
  })
})
