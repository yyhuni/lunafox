import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { parseGlobalAssetSearchQuery, GLOBAL_ASSET_SEARCH_FIELDS } from "@/lib/global-asset-search-query"
import {
  appendGlobalAssetSearchCondition,
  getGlobalAssetSearchInlineCompletion,
  GLOBAL_ASSET_SEARCH_SYNTAX_EXAMPLES,
  SearchSyntaxGuidance,
} from "../search-syntax-guidance"

describe("SearchSyntaxGuidance", () => {
  it("uses the parser whitelist and keeps every suggested example valid", () => {
    expect(GLOBAL_ASSET_SEARCH_FIELDS).toEqual(["url", "host", "title", "statusCode", "tech"])
    expect(GLOBAL_ASSET_SEARCH_SYNTAX_EXAMPLES).toHaveLength(3)
    expect(() => GLOBAL_ASSET_SEARCH_SYNTAX_EXAMPLES.forEach(parseGlobalAssetSearchQuery)).not.toThrow()
  })

  it("builds an editable condition without replacing the existing structured draft", () => {
    expect(appendGlobalAssetSearchCondition("", "host")).toBe('host=""')
    expect(appendGlobalAssetSearchCondition('tech="nginx"', "statusCode")).toBe('tech="nginx" && statusCode=""')
  })

  it("offers only draft-editing shortcuts for allowed syntax", () => {
    const onSelectField = vi.fn()
    const onSelectExample = vi.fn()

    render(
      <SearchSyntaxGuidance
        t={(key) => key}
        onSelectField={onSelectField}
        onSelectExample={onSelectExample}
      />
    )

    fireEvent.click(screen.getByRole("button", { name: 'host=""' }))
    fireEvent.click(screen.getByRole("button", { name: 'statusCode=="200"' }))

    expect(onSelectField).toHaveBeenCalledWith("host")
    expect(onSelectExample).toHaveBeenCalledWith('statusCode=="200"')
    expect(screen.queryByText("!=")).not.toBeInTheDocument()
    expect(screen.queryByText("||")).not.toBeInTheDocument()
  })

  it("completes only strict field and connector syntax", () => {
    expect(getGlobalAssetSearchInlineCompletion("ho")).toBe('st="')
    expect(getGlobalAssetSearchInlineCompletion("host")).toBe('="')
    expect(getGlobalAssetSearchInlineCompletion('statusCode==')).toBe('"')
    expect(getGlobalAssetSearchInlineCompletion('host="api')).toBe('"')
    expect(getGlobalAssetSearchInlineCompletion('host="api"')).toBe(" && ")
    expect(getGlobalAssetSearchInlineCompletion('host="api" && te')).toBe('ch="')
    expect(getGlobalAssetSearchInlineCompletion("domain")).toBe("")
    expect(getGlobalAssetSearchInlineCompletion("||")).toBe("")
  })
})
